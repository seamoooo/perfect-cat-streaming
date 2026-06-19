// Package chaos injects controlled performance degradation into the backend
// so operators can practice troubleshooting in New Relic.
//
// All behaviour is opt-in via env vars (CHAOS_ENABLED=true) and labelled in
// NR with custom segments + transaction attributes so the noise is easy to
// distinguish from real incidents:
//
//   Segment           "chaos.delay"            (synthetic latency span)
//   Background txn    "chaos.db.sleep"         (DB pool pressure)
//   Attribute         chaos.injected = latency | error_500
//                     chaos.delay_ms          (ms of synthetic delay)
//                     chaos.db_sleep_sec
//                     chaos.db_concurrency
//
// Three independent knobs:
//   1. HTTP latency injection — periodic spike window pattern
//   2. HTTP error injection   — steady low rate
//   3. DB pool contention     — periodic SELECT SLEEP() bursts
//
// Default: disabled; flipping CHAOS_ENABLED=true with the sample env yields
// visible response-time degradation within ~5 min in NR APM.
package chaos

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"
)

type Config struct {
	Enabled bool

	// --- HTTP latency in spike windows ---
	// Every `Period` the chaos manager opens a spike window for `Window`. While
	// open, each incoming request has `LatencyChancePercent` % probability of
	// being delayed by a uniform draw in [LatencyMin, LatencyMax].
	Period               time.Duration
	Window               time.Duration
	LatencyChancePercent int
	LatencyMin           time.Duration
	LatencyMax           time.Duration

	// --- HTTP error injection (independent of windows) ---
	ErrorChancePercent int

	// --- DB connection pool contention ---
	// Every `DBSlowPeriod` we fire `DBSlowConcurrency` parallel
	// `SELECT SLEEP(DBSlowQuerySec)` queries. These tie up MySQL connections,
	// indirectly slowing legitimate queries — a realistic production scenario.
	DBSlowPeriod      time.Duration
	DBSlowQuerySec    int
	DBSlowConcurrency int
}

type Chaos struct {
	cfg   Config
	app   *newrelic.Application
	spike atomic.Bool
	rnd   *rand.Rand
	rndMu sync.Mutex
}

func New(cfg Config, app *newrelic.Application) *Chaos {
	return &Chaos{
		cfg: cfg,
		app: app,
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Middleware injects latency / errors based on the chaos config. Always cheap
// when disabled (returns next directly).
func (c *Chaos) Middleware(next http.Handler) http.Handler {
	if !c.cfg.Enabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Errors are independent of the spike window so the error-rate signal
		// is steady and easy to alert on.
		if c.cfg.ErrorChancePercent > 0 && c.roll() < c.cfg.ErrorChancePercent {
			if txn := newrelic.FromContext(r.Context()); txn != nil {
				txn.AddAttribute("chaos.injected", "error_500")
				txn.NoticeError(errors.New("chaos: simulated 500"))
			}
			http.Error(w, "chaos: simulated 500", http.StatusInternalServerError)
			return
		}

		// Latency is gated on the spike window so the response-time signal has
		// a clear "incident shape" (calm → spike → calm).
		if c.spike.Load() && c.roll() < c.cfg.LatencyChancePercent {
			delta := int64(c.cfg.LatencyMax - c.cfg.LatencyMin)
			latency := c.cfg.LatencyMin
			if delta > 0 {
				latency += time.Duration(c.rndInt64(delta))
			}
			var seg *newrelic.Segment
			if txn := newrelic.FromContext(r.Context()); txn != nil {
				seg = txn.StartSegment("chaos.delay")
				txn.AddAttribute("chaos.injected", "latency")
				txn.AddAttribute("chaos.delay_ms", latency.Milliseconds())
			}
			select {
			case <-r.Context().Done():
			case <-time.After(latency):
			}
			if seg != nil {
				seg.End()
			}
		}

		next.ServeHTTP(w, r)
	})
}

// StartSpikeScheduler toggles the spike window on/off on a fixed cadence.
// Runs until ctx is done.
func (c *Chaos) StartSpikeScheduler(ctx context.Context) {
	if !c.cfg.Enabled || c.cfg.Period <= 0 || c.cfg.Window <= 0 {
		return
	}
	go func() {
		log.Printf("[chaos] spike scheduler running period=%s window=%s chance=%d%% latency=%s..%s",
			c.cfg.Period, c.cfg.Window, c.cfg.LatencyChancePercent, c.cfg.LatencyMin, c.cfg.LatencyMax)
		ticker := time.NewTicker(c.cfg.Period)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.openSpikeWindow(ctx)
			}
		}
	}()
}

func (c *Chaos) openSpikeWindow(ctx context.Context) {
	c.spike.Store(true)
	log.Printf("[chaos] 🐾 spike window OPEN (latency injection ON for %s)", c.cfg.Window)
	defer func() {
		c.spike.Store(false)
		log.Printf("[chaos] spike window CLOSED")
	}()
	select {
	case <-ctx.Done():
	case <-time.After(c.cfg.Window):
	}
}

// StartDBPressure periodically fires concurrent SELECT SLEEP() queries to
// monopolise MySQL connections. Each batch is wrapped in a NR background
// transaction so it's filterable in APM.
func (c *Chaos) StartDBPressure(ctx context.Context, db *sql.DB) {
	if !c.cfg.Enabled || c.cfg.DBSlowPeriod <= 0 || db == nil {
		return
	}
	go func() {
		log.Printf("[chaos] DB pressure running period=%s concurrency=%d sleep=%ds",
			c.cfg.DBSlowPeriod, c.cfg.DBSlowConcurrency, c.cfg.DBSlowQuerySec)
		ticker := time.NewTicker(c.cfg.DBSlowPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.runDBSlow(ctx, db)
			}
		}
	}()
}

func (c *Chaos) runDBSlow(ctx context.Context, db *sql.DB) {
	var txn *newrelic.Transaction
	queryCtx := ctx
	if c.app != nil {
		txn = c.app.StartTransaction("chaos.db.sleep")
		defer txn.End()
		txn.AddAttribute("chaos.db_sleep_sec", c.cfg.DBSlowQuerySec)
		txn.AddAttribute("chaos.db_concurrency", c.cfg.DBSlowConcurrency)
		queryCtx = newrelic.NewContext(ctx, txn)
	}
	log.Printf("[chaos] 💤 DB pressure firing %d × SELECT SLEEP(%d)", c.cfg.DBSlowConcurrency, c.cfg.DBSlowQuerySec)

	var wg sync.WaitGroup
	for i := 0; i < c.cfg.DBSlowConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := db.ExecContext(queryCtx, "SELECT SLEEP(?)", c.cfg.DBSlowQuerySec); err != nil {
				log.Printf("[chaos] db sleep err: %v", err)
			}
		}()
	}
	wg.Wait()
}

func (c *Chaos) roll() int {
	c.rndMu.Lock()
	defer c.rndMu.Unlock()
	return c.rnd.Intn(100)
}

func (c *Chaos) rndInt64(n int64) int64 {
	if n <= 0 {
		return 0
	}
	c.rndMu.Lock()
	defer c.rndMu.Unlock()
	return c.rnd.Int63n(n)
}
