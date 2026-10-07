package server

import (
	"encoding/json"
	"github.com/Tomassalgueiro/gorss/internal/feed"
	"net/http"
)

type Handler struct {
	feedRepo *feed.Repository
}

func NewHandler(feedRepo *feed.Repository) *Handler {
	return &Handler{feedRepo: feedRepo}
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func ReadJSON (r http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/feeds", h.createFeed)
	mux.HandleFunc("GET /v1/feeds", h.listFeeds)
	mux.HandleFunc("GET /v1/feeds{id}", h.getFeed)

	return mux
}
