package main

import (
	"context"
	"github.com/Tomassalgueiro/gorss/internal/config"
	"github.com/Tomassalgueiro/gorss/internal/database"
	"github.com/Tomassalgueiro/gorss/internal/feed"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.New()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize db, %v", err)
	}
	err = database.Migrate(db)
	if err != nil {
		log.Fatalf("failed to migrate db, %v", err)
	}
	defer db.Close()

/*	repo := feed.NewRepository(db)

	testFeed := &feed.Feed{
		FeedURL: "https://news.ycombinator.com/rss",
		SiteURL: "https://news.ycombinator.com",
		Title:   "Hacker News",
	}

	if err := repo.CreateFeed(ctx, testFeed); err != nil {
		log.Printf("create feed failed (might already exist): %v", err)
	} else {
		log.Printf("Created feed ID=%d, Title=%s, CreatedAt=%s", testFeed.ID, testFeed.Title, testFeed.CreatedAt)
	}

	feeds, err := repo.ListFeeds(ctx)
	if err != nil {
		log.Fatalf("failed to list feeds: %v", err)
	}
	log.Printf("Total feeds in database: %d", len(feeds))
	*/

	log.Println("Service starting...")
	log.Printf("port: %s\ndburl: %s", cfg.ServerPort, cfg.DatabaseURL)

	<-ctx.Done()
	log.Println("Signal received, shutting down gracefully")
}
