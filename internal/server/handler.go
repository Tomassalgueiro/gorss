package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tomassalgueiro/gorss/internal/feed"
)

type Handler struct {
	feedRepo *feed.Repository
}

type createFeedRequest struct {
	FeedURL string `json:"feed_url"`
}

func NewHandler(feedRepo *feed.Repository) *Handler {
	return &Handler{feedRepo: feedRepo}
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

	f := &feed.Feed{
		FeedURL: req.FeedURL,
	}

	if err := h.feedRepo.CreateFeed(r.Context(), f); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			http.Error(w, "feed already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create feed", http.StatusInternalServerError)
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
