// Package janitor periodically frees the local ephemeral disk.
//
// In S3 mode the backend transcodes into a local working directory and then
// publishes the HLS output (and poster) to S3. After a successful publish the
// local copies are redundant — playback is served from CloudFront — but they
// linger on the Fargate task's ephemeral storage until the video is deleted.
//
// To keep that disk from filling up, the janitor sweeps the upload and HLS
// roots on a schedule (daily by default) and removes any entry older than the
// configured TTL. An in-progress transcode keeps a fresh mtime, so the
// age-based cutoff never deletes work that's still running.
//
// IMPORTANT: only start this in S3 mode. In local mode the HLS files ARE the
// served media; deleting them would break playback.
package janitor

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type Janitor struct {
	UploadRoot string
	HLSRoot    string
	TTL        time.Duration
	Interval   time.Duration
	nrApp      *newrelic.Application // optional; nil = no APM
}

func New(uploadRoot, hlsRoot string, ttl, interval time.Duration, nrApp *newrelic.Application) *Janitor {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Janitor{
		UploadRoot: uploadRoot,
		HLSRoot:    hlsRoot,
		TTL:        ttl,
		Interval:   interval,
		nrApp:      nrApp,
	}
}

// Start launches the background sweeper. It runs one sweep shortly after boot
// (to clear leftovers from a previous task generation) and then every Interval
// until ctx is cancelled.
func (j *Janitor) Start(ctx context.Context) {
	go func() {
		j.runSweep()
		t := time.NewTicker(j.Interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Printf("[janitor] stopping")
				return
			case <-t.C:
				j.runSweep()
			}
		}
	}()
}

func (j *Janitor) runSweep() {
	var txn *newrelic.Transaction
	if j.nrApp != nil {
		txn = j.nrApp.StartTransaction("janitor.sweep")
		defer txn.End()
	}
	removed := j.sweep()
	if txn != nil {
		txn.AddAttribute("removed", removed)
		txn.AddAttribute("ttlHours", j.TTL.Hours())
	}
	log.Printf("[janitor] sweep done removed=%d ttl=%s", removed, j.TTL)
}

// sweep deletes upload/HLS entries whose mtime is older than the TTL.
// RemoveAll handles both the flat upload files and the per-video HLS dirs.
func (j *Janitor) sweep() int {
	cutoff := time.Now().Add(-j.TTL)
	removed := 0
	for _, root := range []string{j.UploadRoot, j.HLSRoot} {
		entries, err := os.ReadDir(root)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[janitor] read %s: %v", root, err)
			}
			continue
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				p := filepath.Join(root, e.Name())
				if err := os.RemoveAll(p); err != nil {
					log.Printf("[janitor] remove %s: %v", p, err)
					continue
				}
				removed++
			}
		}
	}
	return removed
}
