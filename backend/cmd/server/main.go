package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/chaos"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	httpx "github.com/seamoooo/perfect-cat-streaming/backend/internal/http"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/janitor"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/observability"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/publisher"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/transcoder"
)

func main() {
	cfg := config.Load()

	nrApp, err := observability.NewRelic(cfg)
	if err != nil {
		log.Fatalf("newrelic init failed: %v", err)
	}
	defer nrApp.Shutdown(10 * time.Second)

	stg := storage.NewLocal(cfg.UploadDir, cfg.HLSDir)
	if err := stg.EnsureDirs(); err != nil {
		log.Fatalf("storage init failed: %v", err)
	}

	var repo repository.Repository
	if cfg.DatabaseURL != "" {
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 60*time.Second)
		my, err := repository.NewMySQL(dbCtx, cfg.DatabaseURL)
		dbCancel()
		if err != nil {
			log.Fatalf("mysql init failed: %v", err)
		}
		repo = my
		log.Printf("[repository] mysql mode")
	} else {
		repoFile := filepath.Join(filepath.Dir(cfg.UploadDir), "metadata.json")
		mem, err := repository.NewMemory(repoFile)
		if err != nil {
			log.Fatalf("repository init failed: %v", err)
		}
		repo = mem
		log.Printf("[repository] in-memory mode (file=%s)", repoFile)
	}

	tx := transcoder.New(cfg.FFmpegBin)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var pub publisher.Publisher
	if cfg.S3Bucket != "" {
		s3pub, err := publisher.NewS3(rootCtx, publisher.S3Config{
			Bucket:    cfg.S3Bucket,
			Prefix:    cfg.S3Prefix,
			Region:    cfg.S3Region,
			MediaBase: cfg.MediaBaseURL,
		})
		if err != nil {
			log.Fatalf("S3 publisher init failed: %v", err)
		}
		pub = s3pub
		log.Printf("[publisher] S3 mode bucket=%s prefix=%s media=%s", cfg.S3Bucket, cfg.S3Prefix, cfg.MediaBaseURL)
	} else {
		log.Printf("[publisher] local mode (no S3)")
	}

	queue := transcoder.NewQueue(tx, repo, stg, pub, nrApp, 32, time.Duration(cfg.TranscodeTimeoutSec)*time.Second)
	queue.Start(rootCtx, cfg.TranscodeWorkers)

	// Daily janitor for the local ephemeral disk. Only in S3 mode, where the
	// published HLS/poster files are redundant locally; in local mode these ARE
	// the served media so we must never delete them.
	if pub != nil && !cfg.LocalCleanupDisabled {
		jan := janitor.New(
			cfg.UploadDir, cfg.HLSDir,
			time.Duration(cfg.LocalCleanupTTLHours)*time.Hour,
			time.Duration(cfg.LocalCleanupIntervalHours)*time.Hour,
			nrApp,
		)
		jan.Start(rootCtx)
		log.Printf("[janitor] enabled ttl=%dh interval=%dh", cfg.LocalCleanupTTLHours, cfg.LocalCleanupIntervalHours)
	}

	// Chaos: opt-in synthetic perf degradation, labelled in NR for easy
	// drill-down. No-op when CHAOS_ENABLED is false.
	ch := chaos.New(chaos.Config{
		Enabled:              cfg.ChaosEnabled,
		Period:               time.Duration(cfg.ChaosPeriodSec) * time.Second,
		Window:               time.Duration(cfg.ChaosWindowSec) * time.Second,
		LatencyChancePercent: cfg.ChaosLatencyChancePercent,
		LatencyMin:           time.Duration(cfg.ChaosLatencyMinMs) * time.Millisecond,
		LatencyMax:           time.Duration(cfg.ChaosLatencyMaxMs) * time.Millisecond,
		ErrorChancePercent:   cfg.ChaosErrorChancePercent,
		DBSlowPeriod:         time.Duration(cfg.ChaosDBSlowPeriodSec) * time.Second,
		DBSlowQuerySec:       cfg.ChaosDBSlowQuerySec,
		DBSlowConcurrency:    cfg.ChaosDBSlowConcurrency,
	}, nrApp)
	if cfg.ChaosEnabled {
		log.Printf("[chaos] ENABLED — synthetic perf degradation active")
		ch.StartSpikeScheduler(rootCtx)
		if my, ok := repo.(*repository.MySQL); ok {
			ch.StartDBPressure(rootCtx, my.DB())
		}
	}

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           httpx.NewRouter(cfg, nrApp, ch, repo, stg, pub, queue),
		ReadHeaderTimeout: 10 * time.Second,
	}

	banner(cfg)

	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("[server] shutdown requested")
		ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = srv.Shutdown(ctx)
		cancel()
		close(idleConnsClosed)
	}()

	log.Printf("[server] listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
	<-idleConnsClosed
	log.Println("[server] bye 🐾")
}

func banner(cfg config.Config) {
	log.Println("┌────────────────────────────────────────────────┐")
	log.Println("│  perfect-cat-streaming                         │")
	log.Println("│  Bincho (シャム) × Kanpachi (ベンガル)          │")
	log.Println("└────────────────────────────────────────────────┘")
	log.Printf("[config] uploadDir=%s hlsDir=%s base=%s", cfg.UploadDir, cfg.HLSDir, cfg.PublicBaseURL)
}
