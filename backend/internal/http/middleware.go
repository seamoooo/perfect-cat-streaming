package httpx

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, ww.Status(), time.Since(start))
		})
	}
}

// NewRelicTxn wraps each request in a New Relic web transaction. The txn name
// is upgraded to chi's route pattern (e.g. "GET /api/videos/{id}") after the
// handler runs so cardinality stays bounded. When app is nil or disabled the
// middleware is a no-op.
func NewRelicTxn(app *newrelic.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if app == nil {
				next.ServeHTTP(w, r)
				return
			}
			txn := app.StartTransaction(r.Method + " " + r.URL.Path)
			defer txn.End()
			txn.SetWebRequestHTTP(r)
			ww := txn.SetWebResponse(w)
			r = newrelic.RequestWithTransactionContext(r, txn)

			next.ServeHTTP(ww, r)

			if rc := chi.RouteContext(r.Context()); rc != nil {
				if pat := rc.RoutePattern(); pat != "" {
					txn.SetName(r.Method + " " + pat)
				}
			}
		})
	}
}
