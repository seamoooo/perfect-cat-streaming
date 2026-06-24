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
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/chaos"
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

	// Optional custom thumbnail image. When present it becomes the gallery
	// poster instead of an auto-extracted video frame.
	thumbPath := ""
	if tf, tfh, terr := r.FormFile("thumbnail"); terr == nil {
		defer tf.Close()
		tdst := h.Stg.UploadPath(id, "thumb_"+tfh.Filename)
		if tout, err := os.Create(tdst); err == nil {
			if _, err := io.Copy(tout, tf); err == nil {
				thumbPath = tdst
			}
			tout.Close()
		}
	}

	now := time.Now().UTC()
	v := domain.Video{
		ID:          id,
		Title:       formOr(r, "title", strings.TrimSuffix(fh.Filename, ".mp4")),
		Description: r.FormValue("description"),
		CatName:     formOr(r, "catName", "ねこ"),
		Breed:       parseBreed(r.FormValue("breed")),
		Tags:        splitCSV(r.FormValue("tags")),
		Status:      domain.StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.Repo.Create(r.Context(), v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Demo: deliberately inefficient metadata registration (redundant
	// UPDATE/SELECT loop) so New Relic flags the slow-query / N+1 pattern on the
	// upload transaction. Runs in the request context so the DB segments land on
	// this web transaction. Off unless CHAOS_DB_INEFFICIENT_LOOPS > 0.
	if n := h.Cfg.ChaosDBInefficientLoops; n > 0 {
		if err := h.Repo.InefficientMetaChurn(r.Context(), id, n); err != nil {
			log.Printf("[upload] inefficient churn err videoID=%s: %v", id, err)
		}
		if txn := newrelic.FromContext(r.Context()); txn != nil {
			txn.AddAttribute("chaos.injected", "inefficient_db")
			txn.AddAttribute("chaos.db_churn_loops", n)
		}
	}

	h.Queue.Enqueue(transcoder.Job{
		VideoID:       id,
		SrcPath:       dst,
		OutDir:        h.Stg.HLSDir(id),
		Playlist:      fmt.Sprintf("%s/media/%s/index.m3u8", strings.TrimRight(h.Cfg.PublicBaseURL, "/"), id),
		CatName:       v.CatName,
		Breed:         string(v.Breed),
		ThumbnailPath: thumbPath,
		// Developer demo: an "SRE" keyword in the description degrades transcode
		// throughput so the slowdown is observable in New Relic APM.
		ChaosSlow: chaos.Directive(v.Description) == chaos.ModeSRE,
	})

	// Domain custom attributes on the upload web transaction.
	if txn := newrelic.FromContext(r.Context()); txn != nil {
		txn.AddAttribute("video.id", id)
		txn.AddAttribute("cat.name", v.CatName)
		txn.AddAttribute("cat.breed", string(v.Breed))
		txn.AddAttribute("upload.size_bytes", fh.Size)
		txn.AddAttribute("upload.content_type", fh.Header.Get("Content-Type"))
		txn.AddAttribute("video.tag_count", len(v.Tags))
	}

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

// parseBreed accepts any cat-breed slug from the frontend list (e.g. "siamese",
// "scottish_fold", "maine_coon"). Empty falls back to "other".
func parseBreed(s string) domain.Breed {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return domain.BreedOther
	}
	return domain.Breed(s)
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
