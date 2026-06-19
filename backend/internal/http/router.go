package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/chaos"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/http/handlers"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/publisher"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/transcoder"
)

func NewRouter(cfg config.Config, nrApp *newrelic.Application, ch *chaos.Chaos, repo repository.Repository, stg storage.Storage, pub publisher.Publisher, queue *transcoder.Queue) http.Handler {
	r := chi.NewRouter()
	r.Use(NewRelicTxn(nrApp))
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger())
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	upload := &handlers.Upload{Cfg: cfg, Repo: repo, Stg: stg, Queue: queue}
	videos := &handlers.Videos{Cfg: cfg, Repo: repo, Stg: stg, Pub: pub}
	stream := &handlers.Stream{Stg: stg}

	// Healthz must stay fast & 200 — never inject chaos here.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Apply chaos middleware to everything else when enabled. No-op otherwise.
	r.Group(func(r chi.Router) {
		if ch != nil {
			r.Use(ch.Middleware)
		}
		r.Get("/meow", handlers.Meow)
		r.Route("/api", func(r chi.Router) {
			r.Get("/videos", videos.List)
			r.Get("/videos/{id}", videos.Get)
			r.Post("/videos", upload.Handle)
			r.Patch("/videos/{id}", videos.Patch)
			r.Delete("/videos/{id}", videos.Delete)
		})
		r.Get("/media/{id}/*", stream.ServeHLS)
	})

	return r
}
