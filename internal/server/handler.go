package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"log"
	"strings"

	"github.com/Tomassalgueiro/gorss/internal/article"
	"github.com/Tomassalgueiro/gorss/internal/feed"
	"github.com/Tomassalgueiro/gorss/internal/parser"
	"github.com/Tomassalgueiro/gorss/internal/opml"
)

type Handler struct {
	feedRepo *feed.Repository
	articleRepo *article.Repository
	fetcher *parser.Fetcher
}

func NewHandler(feedRepo *feed.Repository, articleRepo *article.Repository, fetcher *parser.Fetcher) *Handler {
	return &Handler{
		feedRepo: feedRepo,
		articleRepo: articleRepo,
		fetcher: fetcher,
	}
}

type updateArticleRequest struct {
	IsRead    *bool `json:"is_read"`
	IsStarred *bool `json:"is_starred"`
}

type createFeedRequest struct {
	FeedURL string `json:"feed_url"`
}

func (h* Handler) createFeed(w http.ResponseWriter, r *http.Request) {
	var req createFeedRequest

	if err := ReadJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.FeedURL = strings.TrimSpace(req.FeedURL)
	if req.FeedURL == "" {
		http.Error(w, "feed_url is required", http.StatusBadRequest)
		return
	}

	resp, err := h.fetcher.Fetch(r.Context(), req.FeedURL, "", "")
	if err != nil {
		log.Printf("fetch feed failed for %s: %v", req.FeedURL, err)
		http.Error(w, "failed to reach remote feed URL", http.StatusBadGateway)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "feed returned non-200 status code", http.StatusBadGateway)
		return
	}

	parsed, err := parser.Parse(resp.Body)
	if err != nil {
		http.Error(w, "invalid feed format"+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	log.Printf("[DEBUG] Parsed title: %q, total items found: %d", parsed.Title, len(parsed.Items))

	f := &feed.Feed{
		FeedURL: req.FeedURL,
		SiteURL: parsed.SiteURL,
		Title: parsed.Title,
		ETag: resp.ETag,
		LastModified: resp.LastModified,
	}

	if err := h.feedRepo.CreateFeed(r.Context(), f); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") { 
			http.Error(w, "feed already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to save feed", http.StatusInternalServerError)
		return
	}

	if len(parsed.Items) > 0 {
		articles := make([]article.Article, 0, len(parsed.Items))
		for _, item := range parsed.Items {
			articles = append(articles, article.Article{
				FeedID: f.ID,
				GUID: item.GUID,
				URL: item.URL,
				Title: item.Title,
				Content: item.Content,
				PublishedAt: item.PublishedAt,
			})
		}

		if err := h.articleRepo.CreateArticles(r.Context(), articles); err != nil {
			log.Printf("failed to save articles for feed %d: %v", f.ID, err)	
		} else {
			log.Printf("[DEBUG] Saved %d aricles for feed %d", len(articles), f.ID)
		}
	} else {
		log.Printf("[WARN] parsed.Items was empty for %s", req.FeedURL)
	}

	WriteJSON(w, http.StatusCreated, f)
}

func (h* Handler) listFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.feedRepo.ListFeeds(r.Context())
	if err != nil {
		http.Error(w, "failed to list feeds", http.StatusInternalServerError)
		return
	}

	if feeds == nil {
		feeds = []*feed.Feed{}
	}

	WriteJSON(w, http.StatusOK, feeds)

}

func (h* Handler) getFeed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}

	f, err := h.feedRepo.GetFeedByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, feed.ErrNotFound) {
			http.Error(w, "feed not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get feed", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, f)

}

func (h *Handler) listFeedArticles(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}

	if _, err := h.feedRepo.GetFeedByID(r.Context(), id); err != nil {
		if errors.Is(err, feed.ErrNotFound){
			http.Error(w, "feed not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get feed", http.StatusInternalServerError)
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		} 
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset > 0 {
			offset = parsedOffset	
		}
	}

	articles, err := h.articleRepo.ListByFeed(r.Context(), id, limit, offset)
	if err != nil {
		http.Error(w, "failed to list articles", http.StatusInternalServerError)
		return
	}

	if articles == nil {
		articles = []*article.Article{}
	}

	WriteJSON(w, http.StatusOK, articles)
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func ReadJSON (r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/feeds", h.listFeeds)
	mux.HandleFunc("GET /v1/feeds/{id}", h.getFeed)
	mux.HandleFunc("GET /v1/feeds/{id}/articles", h.listFeedArticles)
	mux.HandleFunc("GET /v1/articles", h.listArticles)
	mux.HandleFunc("GET /v1/opml/export", h.exportOPML)
	mux.HandleFunc("POST /v1/feeds", h.createFeed)
	mux.HandleFunc("POST /v1/articles/{id}/mark-all-read", h.markFeedAsRead)
	mux.HandleFunc("POST /v1/opml/import", h.importOPML)
	mux.HandleFunc("PATCH /v1/articles/{id}", h.updateArticle)
	mux.HandleFunc("DELETE /v1/feeds/{id}", h.deleteFeed)

	return mux
}

func (h *Handler) updateArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid article id", http.StatusBadRequest)
		return
	}

	var req updateArticleRequest
	if err := ReadJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.IsRead == nil && req.IsStarred == nil {
		http.Error(w, "at least one of is_read or is_starred must be provided", http.StatusBadRequest)
		return
	}

	updated, err := h.articleRepo.UpdateStatus(r.Context(), id, req.IsRead, req.IsStarred)
	if err != nil {
		if errors.Is(err, article.ErrNotFound) {
			http.Error(w, "article not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to update article", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

func (h *Handler) markFeedAsRead(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}

	if _, err := h.feedRepo.GetFeedByID(r.Context(), id); err != nil {
		if errors.Is(err, feed.ErrNotFound) {
			http.Error(w, "feed not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get feed", http.StatusInternalServerError)
		return
	}

	if err := h.articleRepo.MarkFeedAsRead(r.Context(), id); err != nil {
		http.Error(w, "failed to mark feed articles as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listArticles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := article.ListFilter{
		UnreadOnly:  q.Get("unread") == "true" || q.Get("unread") == "1",
		StarredOnly: q.Get("starred") == "true" || q.Get("starred") == "1",
		Limit:       50,
		Offset:      0,
	}

	if l := q.Get("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
			filter.Limit = parsedLimit
		}
	}
	if o := q.Get("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil && parsedOffset >= 0 {
			filter.Offset = parsedOffset
		}
	}

	articles, err := h.articleRepo.ListAll(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to list articles", http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, articles)
}

func (h *Handler) deleteFeed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}

	if err := h.feedRepo.DeleteFeed(r.Context(), id); err != nil {
		if errors.Is(err, feed.ErrNotFound) {
			http.Error(w, "feed not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete feed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) exportOPML(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.feedRepo.ListFeeds(r.Context())
	if err != nil {
		http.Error(w, "failed to load feeds", http.StatusInternalServerError)
		return
	}

	items := make([]opml.FeedItem, 0, len(feeds))
	for _, f := range feeds {
		items = append(items, opml.FeedItem{
			Title:   f.Title,
			FeedURL: f.FeedURL,
			SiteURL: f.SiteURL,
		})
	}

	data, err := opml.Generate("GoRSS Subscriptions", items)
	if err != nil {
		http.Error(w, "failed to generate opml", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"subscriptions.opml\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

type opmlImportResponse struct {
	TotalParsed int `json:"total_parsed"`
	Created     int `json:"created"`
	Skipped     int `json:"skipped"`
}

func (h *Handler) importOPML(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024)

	urls, err := opml.Parse(r.Body)
	if err != nil {
		http.Error(w, "invalid opml file: "+err.Error(), http.StatusBadRequest)
		return
	}

	var created, skipped int
	for _, rawURL := range urls {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}

		f := &feed.Feed{FeedURL: rawURL}
		if err := h.feedRepo.CreateFeed(r.Context(), f); err != nil {
			skipped++
			continue
		}
		created++
	}

	WriteJSON(w, http.StatusOK, opmlImportResponse{
		TotalParsed: len(urls),
		Created:     created,
		Skipped:     skipped,
	})
}
