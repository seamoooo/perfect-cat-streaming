package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
)

type Videos struct {
	Cfg  config.Config
	Repo repository.Repository
}

func (h *Videos) List(w http.ResponseWriter, r *http.Request) {
	items := h.Repo.List()
	for i := range items {
		items[i] = withPlaylistURL(items[i], h.Cfg.PublicBaseURL)
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Videos) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, ok := h.Repo.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, withPlaylistURL(v, h.Cfg.PublicBaseURL))
}

func withPlaylistURL(v domain.Video, base string) domain.Video {
	if v.Status == domain.StatusReady && v.PlaylistURL == "" {
		v.PlaylistURL = fmt.Sprintf("%s/media/%s/index.m3u8", strings.TrimRight(base, "/"), v.ID)
	}
	return v
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
