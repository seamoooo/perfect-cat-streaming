package transcoder

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"

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
	CatName  string // domain context for New Relic custom attributes
	Breed    string // siamese | bengal | other
}

type Queue struct {
	jobs  chan Job
	tx    *FFmpeg
	repo  repository.Repository
	stg   storage.Storage
	pub   publisher.Publisher   // optional; nil = local mode
	nrApp *newrelic.Application // optional; nil = no APM
}

func NewQueue(tx *FFmpeg, repo repository.Repository, stg storage.Storage, pub publisher.Publisher, nrApp *newrelic.Application, buffer int) *Queue {
	if buffer <= 0 {
		buffer = 16
	}
	return &Queue{
		jobs:  make(chan Job, buffer),
		tx:    tx,
		repo:  repo,
		stg:   stg,
		pub:   pub,
		nrApp: nrApp,
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

	// Background transaction so the whole pipeline (ffmpeg → ffprobe → S3 →
	// MySQL UPDATE) appears as one trace in New Relic.
	publishTarget := "local"
	if q.pub != nil {
		publishTarget = "s3"
	}

	var txn *newrelic.Transaction
	if q.nrApp != nil {
		txn = q.nrApp.StartTransaction("transcoder.job")
		defer txn.End()
		// Domain custom attributes for slicing transcode performance in NR.
		txn.AddAttribute("videoID", j.VideoID)
		txn.AddAttribute("video.id", j.VideoID)
		txn.AddAttribute("cat.name", j.CatName)
		txn.AddAttribute("cat.breed", j.Breed)
		txn.AddAttribute("transcode.publish_target", publishTarget)
		if fi, err := os.Stat(j.SrcPath); err == nil {
			txn.AddAttribute("transcode.input_bytes", fi.Size())
		}
		ctx = newrelic.NewContext(ctx, txn)
	}

	if err := q.repo.UpdateStatus(ctx, j.VideoID, domain.StatusProcessing, ""); err != nil {
		log.Printf("[transcoder] failed to mark processing: %v", err)
	}

	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	transcodeStart := time.Now()
	if err := q.tx.TranscodeToHLS(jobCtx, j.SrcPath, j.OutDir); err != nil {
		log.Printf("[transcoder] transcode failed videoID=%s: %v", j.VideoID, err)
		if txn != nil {
			txn.NoticeError(err)
		}
		_ = q.repo.UpdateStatus(ctx, j.VideoID, domain.StatusError, err.Error())
		return
	}

	dur, err := q.tx.Probe(jobCtx, j.SrcPath)
	if err != nil {
		log.Printf("[transcoder] probe failed (non-fatal) videoID=%s: %v", j.VideoID, err)
		if txn != nil {
			txn.NoticeError(err)
		}
	}

	// Netflix-style poster frame next to the HLS output, so it gets published
	// to S3 / served from /media alongside index.m3u8. Non-fatal on failure.
	posterPath := filepath.Join(j.OutDir, "poster.jpg")
	if err := q.tx.Thumbnail(jobCtx, j.SrcPath, posterPath); err != nil {
		log.Printf("[transcoder] thumbnail failed (non-fatal) videoID=%s: %v", j.VideoID, err)
		if txn != nil {
			txn.NoticeError(err)
		}
	}

	playlistURL := j.Playlist
	if q.pub != nil {
		log.Printf("[transcoder] publishing videoID=%s to S3", j.VideoID)
		published, err := q.pub.PublishHLS(jobCtx, j.OutDir, j.VideoID)
		if err != nil {
			log.Printf("[transcoder] publish failed videoID=%s: %v", j.VideoID, err)
			if txn != nil {
				txn.NoticeError(err)
			}
			_ = q.repo.UpdateStatus(ctx, j.VideoID, domain.StatusError, "publish: "+err.Error())
			return
		}
		playlistURL = published
		log.Printf("[transcoder] published videoID=%s url=%s", j.VideoID, playlistURL)
	}

	if err := q.repo.UpdateAfterTranscode(ctx, j.VideoID, dur, playlistURL); err != nil {
		log.Printf("[transcoder] failed to mark ready: %v", err)
		if txn != nil {
			txn.NoticeError(err)
		}
		return
	}
	if txn != nil {
		transcodeWall := time.Since(transcodeStart)
		segments, outBytes := hlsOutputStats(j.OutDir)
		txn.AddAttribute("durationSec", dur)
		txn.AddAttribute("transcode.video_duration_sec", dur)
		txn.AddAttribute("transcode.wall_sec", transcodeWall.Seconds())
		txn.AddAttribute("transcode.segment_count", segments)
		txn.AddAttribute("transcode.output_bytes", outBytes)
		// Real-time factor: wall-clock transcode time per second of video.
		// <1 = faster than real time. A key efficiency signal for the demo.
		if dur > 0 {
			txn.AddAttribute("transcode.realtime_factor", transcodeWall.Seconds()/dur)
		}
	}
	log.Printf("[transcoder] worker#%d done videoID=%s duration=%.2fs", workerID, j.VideoID, dur)
}

// hlsOutputStats counts the .ts segments and sums all bytes written to the HLS
// output directory.
func hlsOutputStats(dir string) (segments int, totalBytes int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".ts") {
			segments++
		}
		if info, err := e.Info(); err == nil {
			totalBytes += info.Size()
		}
	}
	return segments, totalBytes
}
