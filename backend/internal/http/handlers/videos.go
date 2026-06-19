package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/publisher"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
)

type Videos struct {
	Cfg  config.Config
	Repo repository.Repository
	Stg  storage.Storage
	Pub  publisher.Publisher // optional; nil = local mode
}

func (h *Videos) List(w http.ResponseWriter, r *http.Request) {
	items := h.Repo.List(r.Context())
	for i := range items {
		items[i] = withPlaylistURL(items[i], h.Cfg.PublicBaseURL)
	}
	if txn := newrelic.FromContext(r.Context()); txn != nil {
		txn.AddAttribute("video.count", len(items))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Videos) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	v, ok := h.Repo.Get(r.Context(), id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if txn := newrelic.FromContext(r.Context()); txn != nil {
		txn.AddAttribute("video.id", v.ID)
		txn.AddAttribute("cat.breed", string(v.Breed))
		txn.AddAttribute("video.status", string(v.Status))
	}
	writeJSON(w, http.StatusOK, withPlaylistURL(v, h.Cfg.PublicBaseURL))
}

// Patch updates mutable fields on a Video. Currently supports `tags` only.
// Body: {"tags": ["a","b"]}
func (h *Videos) Patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.Repo.Get(r.Context(), id); !ok {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Tags *[]string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Tags != nil {
		if err := h.Repo.UpdateTags(r.Context(), id, *body.Tags); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	v, ok := h.Repo.Get(r.Context(), id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, withPlaylistURL(v, h.Cfg.PublicBaseURL))
}

// Delete removes the video row, then best-effort cleans up local files and S3.
// 204 on success, 404 if the row didn't exist.
func (h *Videos) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.Repo.Get(r.Context(), id); !ok {
		http.NotFound(w, r)
		return
	}
	if txn := newrelic.FromContext(r.Context()); txn != nil {
		txn.AddAttribute("video.id", id)
	}
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if h.Stg != nil {
		if err := h.Stg.RemoveVideo(id); err != nil {
			log.Printf("[delete] local cleanup failed videoID=%s: %v", id, err)
		}
	}
	if h.Pub != nil {
		if err := h.Pub.DeleteHLS(r.Context(), id); err != nil {
			log.Printf("[delete] S3 cleanup failed videoID=%s: %v", id, err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
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
