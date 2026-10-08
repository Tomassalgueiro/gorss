package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"log"
	"strings"

	"github.com/Tomassalgueiro/gorss/internal/feed"
	"github.com/Tomassalgueiro/gorss/internal/parser"
)

type Handler struct {
	feedRepo *feed.Repository
	fetcher *parser.Fetcher
}

func NewHandler(feedRepo *feed.Repository, fetcher *parser.Fetcher) *Handler {
	return &Handler{
		feedRepo: feedRepo,
		fetcher: fetcher,
	}
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

	mux.HandleFunc("POST /v1/feeds", h.createFeed)
	mux.HandleFunc("GET /v1/feeds", h.listFeeds)
	mux.HandleFunc("GET /v1/feeds/{id}", h.getFeed)

	return mux
}
