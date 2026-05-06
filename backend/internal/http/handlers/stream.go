package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
)

type Stream struct {
	Stg storage.Storage
}

// ServeHLS streams .m3u8 / .ts files from the storage HLS dir for the videoID.
// Mounted at /media/{id}/* — chi will give us the trailing path.
func (h *Stream) ServeHLS(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rest := chi.URLParam(r, "*") // matches the wildcard
	if id == "" || rest == "" {
		http.NotFound(w, r)
		return
	}
	// Defensive: prevent traversal.
	cleaned := filepath.Clean(rest)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "../") {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	full := filepath.Join(h.Stg.HLSDir(id), cleaned)

	switch {
	case strings.HasSuffix(cleaned, ".m3u8"):
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache")
	case strings.HasSuffix(cleaned, ".ts"):
		w.Header().Set("Content-Type", "video/mp2t")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeFile(w, r, full)
}
