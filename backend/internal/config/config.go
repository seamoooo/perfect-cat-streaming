package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppPort        string
	UploadDir      string
	HLSDir         string
	PublicBaseURL  string
	AllowedOrigins []string
	FFmpegBin      string

	// S3 publishing (optional). When S3Bucket is non-empty the transcoder
	// uploads HLS to S3 after each job and stores the CloudFront playlist URL
	// on the Video. Otherwise the local /media route serves the files.
	S3Bucket     string
	S3Prefix     string
	S3Region     string
	MediaBaseURL string // e.g. https://d1234.cloudfront.net

	// Postgres DSN. When set, the backend uses RDS-backed metadata storage.
	// Empty = in-memory + JSON (local dev mode).
	DatabaseURL string

	// Local ephemeral-disk janitor. In S3 mode the local upload/HLS files are
	// redundant after publish, so a periodic sweep deletes anything older than
	// the TTL to keep Fargate ephemeral storage from filling up. Only runs in
	// S3 mode (deleting local files in local mode would break playback).
	LocalCleanupDisabled      bool // force-off even in S3 mode
	LocalCleanupTTLHours      int  // delete local files older than this
	LocalCleanupIntervalHours int  // sweep cadence

	// New Relic APM (optional). When NewRelicLicenseKey is empty, the
	// observability layer is disabled and the app runs unchanged.
	NewRelicAppName    string
	NewRelicLicenseKey string
	NewRelicLogLevel   string // "debug" | "info" | "" (default: warn)
	AppEnv             string // "dev" | "stg" | "prod" — appended to NR app name

	// AI monitoring (recommended by the NR Go agent install flow). When
	// enabled, ConfigAIMonitoringEnabled is passed to the agent and the
	// CustomInsightsEvents buffer is grown to the configured sample cap.
	NewRelicAIMonitoringEnabled    bool
	NewRelicCustomEventsMaxSamples int

	// Chaos engineering — periodic perf-degradation for NR troubleshooting
	// practice. Off by default; see internal/chaos for full semantics.
	ChaosEnabled              bool
	ChaosPeriodSec            int // window cadence
	ChaosWindowSec            int // window duration
	ChaosLatencyChancePercent int // % of reqs during window that get latency
	ChaosLatencyMinMs         int
	ChaosLatencyMaxMs         int
	ChaosErrorChancePercent   int // % of all reqs returned 500 (independent of window)
	ChaosDBSlowPeriodSec      int // 0 = off
	ChaosDBSlowQuerySec       int
	ChaosDBSlowConcurrency    int
}

func Load() Config {
	return Config{
		AppPort:        getenv("APP_PORT", "8080"),
		UploadDir:      getenv("STORAGE_UPLOAD_DIR", "/var/app/data/uploads"),
		HLSDir:         getenv("STORAGE_HLS_DIR", "/var/app/data/hls"),
		PublicBaseURL:  getenv("PUBLIC_BASE_URL", "http://localhost:8080"),
		AllowedOrigins: splitCSV(getenv("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8000")),
		FFmpegBin:      getenv("FFMPEG_BIN", "ffmpeg"),
		S3Bucket:       getenv("S3_BUCKET", ""),
		S3Prefix:       getenv("S3_HLS_PREFIX", "hls"),
		S3Region:       getenv("S3_REGION", ""),
		MediaBaseURL:   getenv("MEDIA_BASE_URL", ""),
		DatabaseURL:    getenv("DATABASE_URL", ""),

		LocalCleanupDisabled:      getenvBool("LOCAL_CLEANUP_DISABLED", false),
		LocalCleanupTTLHours:      getenvInt("LOCAL_CLEANUP_TTL_HOURS", 24),
		LocalCleanupIntervalHours: getenvInt("LOCAL_CLEANUP_INTERVAL_HOURS", 24),

		NewRelicAppName:    getenv("NEW_RELIC_APP_NAME", "PerfectCatStreaming"),
		NewRelicLicenseKey: getenv("NEW_RELIC_LICENSE_KEY", ""),
		NewRelicLogLevel:   getenv("NEW_RELIC_LOG_LEVEL", ""),
		AppEnv:             getenv("APP_ENV", "dev"),

		NewRelicAIMonitoringEnabled:    getenvBool("NEW_RELIC_AI_MONITORING_ENABLED", false),
		NewRelicCustomEventsMaxSamples: getenvInt("NEW_RELIC_CUSTOM_INSIGHTS_EVENTS_MAX_SAMPLES_STORED", 10000),

		ChaosEnabled:              getenvBool("CHAOS_ENABLED", false),
		ChaosPeriodSec:            getenvInt("CHAOS_PERIOD_SEC", 300),
		ChaosWindowSec:            getenvInt("CHAOS_WINDOW_SEC", 60),
		ChaosLatencyChancePercent: getenvInt("CHAOS_LATENCY_CHANCE_PERCENT", 40),
		ChaosLatencyMinMs:         getenvInt("CHAOS_LATENCY_MIN_MS", 500),
		ChaosLatencyMaxMs:         getenvInt("CHAOS_LATENCY_MAX_MS", 2000),
		ChaosErrorChancePercent:   getenvInt("CHAOS_ERROR_CHANCE_PERCENT", 0),
		ChaosDBSlowPeriodSec:      getenvInt("CHAOS_DB_SLOW_PERIOD_SEC", 0),
		ChaosDBSlowQuerySec:       getenvInt("CHAOS_DB_SLOW_QUERY_SEC", 5),
		ChaosDBSlowConcurrency:    getenvInt("CHAOS_DB_SLOW_CONCURRENCY", 3),
	}
}

func getenvBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(strings.ToLower(v))
	if err != nil {
		return def
	}
	return b
}

func getenvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
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
