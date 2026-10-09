package worker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Tomassalgueiro/gorss/internal/article"
	"github.com/Tomassalgueiro/gorss/internal/feed"
	"github.com/Tomassalgueiro/gorss/internal/parser"
)

type Worker struct {
	feedRepo    *feed.Repository
	articleRepo *article.Repository
	fetcher     *parser.Fetcher
	interval    time.Duration
	batchSize   int
}

func New(feedRepo *feed.Repository, articleRepo *article.Repository, fetcher *parser.Fetcher, interval time.Duration, batchSize int, ) *Worker {
	return &Worker{
		feedRepo:    feedRepo,
		articleRepo: articleRepo,
		fetcher:     fetcher,
		interval:    interval,
		batchSize:   batchSize,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("[worker] started with interval %s", w.interval)

	w.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[worker] stopping background worker...")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *Worker) runOnce(ctx context.Context) {
	feeds, err := w.feedRepo.GetFeedsToFetch(ctx, w.interval, w.batchSize)
	if err != nil {
		log.Printf("[worker] failed to load feeds to fetch: %v", err)
		return
	}

	if len(feeds) == 0 {
		return
	}

	log.Printf("[worker] refreshing %d feeds", len(feeds))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, f := range feeds {
		wg.Add(1)
		go func(f *feed.Feed) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := w.refreshFeed(ctx, f); err != nil {
				log.Printf("[worker] error refreshing feed %d (%s): %v", f.ID, f.FeedURL, err)
			}
		}(f)
	}

	wg.Wait()
}

func (w *Worker) refreshFeed(ctx context.Context, f *feed.Feed) error {
	fetchCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	resp, err := w.fetcher.Fetch(fetchCtx, f.FeedURL, f.ETag, f.LastModified)
	if err != nil {
		_ = w.feedRepo.UpdateFeedFetchStatus(ctx, f.ID, "", "", err)
		return fmt.Errorf("fetch: %w", err)
	}

	if resp.StatusCode == http.StatusNotModified {
		log.Printf("[worker] feed %d (%s): 304 not modified", f.ID, f.Title)
		return w.feedRepo.UpdateFeedFetchStatus(ctx, f.ID, resp.ETag, resp.LastModified, nil)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("bad status code: %d", resp.StatusCode)
		_ = w.feedRepo.UpdateFeedFetchStatus(ctx, f.ID, "", "", err)
		return err
	}

	parsed, err := parser.Parse(resp.Body)
	if err != nil {
		_ = w.feedRepo.UpdateFeedFetchStatus(ctx, f.ID, "", "", err)
		return fmt.Errorf("parse: %w", err)
	}

	if len(parsed.Items) > 0 {
		articles := make([]article.Article, 0, len(parsed.Items))
		for _, item := range parsed.Items {
			articles = append(articles, article.Article{
				FeedID:      f.ID,
				GUID:        item.GUID,
				URL:         item.URL,
				Title:       item.Title,
				Content:     item.Content,
				PublishedAt: item.PublishedAt,
			})
		}

		if err := w.articleRepo.CreateArticles(ctx, articles); err != nil {
			log.Printf("[worker] failed saving articles for feed %d: %v", f.ID, err)
		}
	}

	return w.feedRepo.UpdateFeedFetchStatus(ctx, f.ID, resp.ETag, resp.LastModified, nil)
}
