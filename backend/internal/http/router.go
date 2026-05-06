package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/http/handlers"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/repository"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/storage"
	"github.com/seamoooo/perfect-cat-streaming/backend/internal/transcoder"
)

func NewRouter(cfg config.Config, repo repository.Repository, stg storage.Storage, queue *transcoder.Queue) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger())
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	upload := &handlers.Upload{Cfg: cfg, Repo: repo, Stg: stg, Queue: queue}
	videos := &handlers.Videos{Cfg: cfg, Repo: repo}
	stream := &handlers.Stream{Stg: stg}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/meow", handlers.Meow)

	r.Route("/api", func(r chi.Router) {
		r.Get("/videos", videos.List)
		r.Get("/videos/{id}", videos.Get)
		r.Post("/videos", upload.Handle)
	})

	r.Get("/media/{id}/*", stream.ServeHLS)

	return r
}
