package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/transcoder"
)

const maxUploadBytes = 2 << 30 // 2 GiB

type Upload struct {
	Cfg   config.Config
	Repo  repository.Repository
	Stg   storage.Storage
	Queue *transcoder.Queue
}

func (h *Upload) Handle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "invalid multipart: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, fh, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing 'file' field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	id := uuid.NewString()
	dst := h.Stg.UploadPath(id, fh.Filename)
	out, err := os.Create(dst)
	if err != nil {
		http.Error(w, "cannot create upload file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		_ = os.Remove(dst)
		http.Error(w, "cannot write upload: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out.Close()

	now := time.Now().UTC()
	v := domain.Video{
		ID:          id,
		Title:       formOr(r, "title", strings.TrimSuffix(fh.Filename, ".mp4")),
		Description: r.FormValue("description"),
		CatName:     formOr(r, "catName", "Bincho"),
		Breed:       parseBreed(r.FormValue("breed")),
		Tags:        splitCSV(r.FormValue("tags")),
		Status:      domain.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.Repo.Create(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Queue.Enqueue(transcoder.Job{
		VideoID:  id,
		SrcPath:  dst,
		OutDir:   h.Stg.HLSDir(id),
		Playlist: fmt.Sprintf("%s/media/%s/index.m3u8", strings.TrimRight(h.Cfg.PublicBaseURL, "/"), id),
	})

	log.Printf("[upload] queued videoID=%s catName=%s breed=%s size=%d", id, v.CatName, v.Breed, fh.Size)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(v)
}

func formOr(r *http.Request, key, def string) string {
	if v := r.FormValue(key); v != "" {
		return v
	}
	return def
}

func parseBreed(s string) domain.Breed {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "siamese", "bincho":
		return domain.BreedSiamese
	case "bengal", "kanpachi":
		return domain.BreedBengal
	case "":
		return domain.BreedOther
	default:
		return domain.BreedOther
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
