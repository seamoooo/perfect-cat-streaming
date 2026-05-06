package transcoder

import (
	"context"
	"log"
	"time"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/domain"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/publisher"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
)

type Job struct {
	VideoID  string
	SrcPath  string
	OutDir   string
	Playlist string // public URL of the resulting playlist (used when no Publisher)
}

type Queue struct {
	jobs chan Job
	tx   *FFmpeg
	repo repository.Repository
	stg  storage.Storage
	pub  publisher.Publisher // optional; nil = local mode
}

func NewQueue(tx *FFmpeg, repo repository.Repository, stg storage.Storage, pub publisher.Publisher, buffer int) *Queue {
	if buffer <= 0 {
		buffer = 16
	}
	return &Queue{
		jobs: make(chan Job, buffer),
		tx:   tx,
		repo: repo,
		stg:  stg,
		pub:  pub,
	}
}

func (q *Queue) Start(ctx context.Context, workers int) {
	if workers <= 0 {
		workers = 2
	}
	for i := 0; i < workers; i++ {
		go q.worker(ctx, i)
	}
}

func (q *Queue) Enqueue(j Job) {
	q.jobs <- j
}

func (q *Queue) worker(ctx context.Context, id int) {
	log.Printf("[transcoder] worker#%d started", id)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[transcoder] worker#%d stopping", id)
			return
		case j := <-q.jobs:
			q.process(ctx, id, j)
		}
	}
}

func (q *Queue) process(ctx context.Context, workerID int, j Job) {
	log.Printf("[transcoder] worker#%d picked up videoID=%s", workerID, j.VideoID)
	if err := q.repo.UpdateStatus(j.VideoID, domain.StatusProcessing, ""); err != nil {
		log.Printf("[transcoder] failed to mark processing: %v", err)
	}

	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	if err := q.tx.TranscodeToHLS(jobCtx, j.SrcPath, j.OutDir); err != nil {
		log.Printf("[transcoder] transcode failed videoID=%s: %v", j.VideoID, err)
		_ = q.repo.UpdateStatus(j.VideoID, domain.StatusError, err.Error())
		return
	}

	dur, err := q.tx.Probe(jobCtx, j.SrcPath)
	if err != nil {
		log.Printf("[transcoder] probe failed (non-fatal) videoID=%s: %v", j.VideoID, err)
	}

	playlistURL := j.Playlist
	if q.pub != nil {
		log.Printf("[transcoder] publishing videoID=%s to S3", j.VideoID)
		published, err := q.pub.PublishHLS(jobCtx, j.OutDir, j.VideoID)
		if err != nil {
			log.Printf("[transcoder] publish failed videoID=%s: %v", j.VideoID, err)
			_ = q.repo.UpdateStatus(j.VideoID, domain.StatusError, "publish: "+err.Error())
			return
		}
		playlistURL = published
		log.Printf("[transcoder] published videoID=%s url=%s", j.VideoID, playlistURL)
	}

	if err := q.repo.UpdateAfterTranscode(j.VideoID, dur, playlistURL); err != nil {
		log.Printf("[transcoder] failed to mark ready: %v", err)
		return
	}
	log.Printf("[transcoder] worker#%d done videoID=%s duration=%.2fs", workerID, j.VideoID, dur)
}
