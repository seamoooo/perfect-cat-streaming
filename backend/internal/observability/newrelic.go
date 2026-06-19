// Package observability sets up the New Relic Go agent.
//
// Design goals:
//   - No secrets in source. License key & related settings flow from env vars
//     (config.Config), populated locally via .env (gitignored) and in AWS via
//     Secrets Manager -> ECS task `secrets`.
//   - Optional / graceful degradation. Missing license key → the agent is
//     instantiated with ConfigEnabled(false) so the rest of the code can call
//     txn := app.StartTransaction(...) etc. with no nil checks and no data.
//   - Extensible. Options are built by small composable helpers
//     (baseOptions / licenseOptions / aiMonitoringOptions / loggingOptions);
//     adding a new section (e.g. distributed tracing tweaks, custom labels,
//     security agent) is a one-function diff.
package observability

import (
	"log"
	"os"
	"time"

	"github.com/newrelic/go-agent/v3/newrelic"

	"github.com/seamoooo/perfect-cat-streaming/backend/internal/config"
)

const defaultCustomEventsMaxSamples = 10000

// AppName composes the final app name including environment suffix when
// the environment isn't production. Exposed so other packages (e.g. log
// forwarders) can tag emitted data with the same identity.
func AppName(cfg config.Config) string {
	name := cfg.NewRelicAppName
	if name == "" {
		name = "PerfectCatStreaming"
	}
	if cfg.AppEnv != "" && cfg.AppEnv != "prod" {
		name = name + " (" + cfg.AppEnv + ")"
	}
	return name
}

// baseOptions returns the always-on options that don't depend on optional
// features (license / AI monitoring). Distributed tracing and app log
// forwarding are on by default so the agent emits the richest signal set.
func baseOptions(cfg config.Config) []newrelic.ConfigOption {
	return []newrelic.ConfigOption{
		newrelic.ConfigAppName(AppName(cfg)),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	}
}

// licenseOptions wires the ingest credentials. Empty license → explicitly
// disabled application so downstream code keeps the same shape.
func licenseOptions(cfg config.Config) []newrelic.ConfigOption {
	if cfg.NewRelicLicenseKey == "" {
		return []newrelic.ConfigOption{newrelic.ConfigEnabled(false)}
	}
	return []newrelic.ConfigOption{newrelic.ConfigLicense(cfg.NewRelicLicenseKey)}
}

// aiMonitoringOptions enables AI monitoring features and grows the
// CustomInsightsEvents buffer to retain enough samples for analysis. Off by
// default to avoid bloating event volume on services that don't call LLMs.
func aiMonitoringOptions(cfg config.Config) []newrelic.ConfigOption {
	if !cfg.NewRelicAIMonitoringEnabled {
		return nil
	}
	max := cfg.NewRelicCustomEventsMaxSamples
	if max <= 0 {
		max = defaultCustomEventsMaxSamples
	}
	return []newrelic.ConfigOption{
		newrelic.ConfigAIMonitoringEnabled(true),
		newrelic.ConfigCustomInsightsEventsMaxSamplesStored(max),
		// AIMonitoring.Streaming.Enabled is controlled via the
		// NEW_RELIC_AI_MONITORING_STREAMING_ENABLED env var which the agent
		// picks up directly — no programmatic option is required.
	}
}

// loggingOptions selects the appropriate stdlib log writer for the agent's
// internal diagnostics. Default (empty) keeps it quiet at WARN level.
func loggingOptions(cfg config.Config) []newrelic.ConfigOption {
	switch cfg.NewRelicLogLevel {
	case "debug":
		return []newrelic.ConfigOption{newrelic.ConfigDebugLogger(os.Stdout)}
	case "info":
		return []newrelic.ConfigOption{newrelic.ConfigInfoLogger(os.Stdout)}
	default:
		return nil
	}
}

// NewRelic builds a *newrelic.Application from config. Callers are expected
// to `defer app.Shutdown(10 * time.Second)` once the server is shutting down
// so buffered events are flushed.
func NewRelic(cfg config.Config) (*newrelic.Application, error) {
	opts := make([]newrelic.ConfigOption, 0, 8)
	opts = append(opts, baseOptions(cfg)...)
	opts = append(opts, licenseOptions(cfg)...)
	opts = append(opts, aiMonitoringOptions(cfg)...)
	opts = append(opts, loggingOptions(cfg)...)

	if cfg.NewRelicLicenseKey == "" {
		log.Printf("[newrelic] disabled (no license key)")
	} else {
		log.Printf("[newrelic] enabled app=%q ai_monitoring=%t max_events=%d",
			AppName(cfg), cfg.NewRelicAIMonitoringEnabled, cfg.NewRelicCustomEventsMaxSamples)
	}

	app, err := newrelic.NewApplication(opts...)
	if err != nil {
		return nil, err
	}
	if cfg.NewRelicLicenseKey != "" {
		if err := app.WaitForConnection(5 * time.Second); err != nil {
			log.Printf("[newrelic] connection wait: %v (continuing)", err)
		} else {
			log.Printf("[newrelic] connected")
		}
	}
	return app, nil
}
