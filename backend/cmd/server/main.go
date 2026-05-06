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

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	httpx "github.com/seamoooo/perfect-cat-streaming/backend/internal/http"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/publisher"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/transcoder"
)

func main() {
	cfg := config.Load()

	stg := storage.NewLocal(cfg.UploadDir, cfg.HLSDir)
	if err := stg.EnsureDirs(); err != nil {
		log.Fatalf("storage init failed: %v", err)
	}

	repoFile := filepath.Join(filepath.Dir(cfg.UploadDir), "metadata.json")
	repo, err := repository.NewMemory(repoFile)
	if err != nil {
		log.Fatalf("repository init failed: %v", err)
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

	queue := transcoder.NewQueue(tx, repo, stg, pub, 32)
	queue.Start(rootCtx, 2)

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           httpx.NewRouter(cfg, repo, stg, queue),
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
