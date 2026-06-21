# --- New Relic Synthetics: uptime (death) monitoring ---
#
# Two SIMPLE (ping) monitors from Tokyo, every 5 min, plus a NRQL condition on
# the shared policy so a failed check pages through the same email workflow.
#   - frontend root  (HEAD; nginx serves the SPA)
#   - backend /healthz (GET; chi route returns "ok", so HEAD would 405 → bypass)
#
# Gated on local.nr_alerts_enabled (User key + account id), same as the alerts.

locals {
  # Same public base URL the app advertises (see outputs.tf / ecs.tf).
  public_base_url = local.use_https ? "https://${var.domain_name}" : "http://${aws_lb.main.dns_name}"
}

resource "newrelic_synthetics_monitor" "frontend" {
  count            = local.nr_alerts_enabled ? 1 : 0
  account_id       = var.new_relic_account_id
  name             = "${var.new_relic_app_name} frontend ping"
  type             = "SIMPLE"
  uri              = local.public_base_url
  period           = "EVERY_5_MINUTES"
  status           = "ENABLED"
  locations_public = ["AP_NORTHEAST_1"]
  verify_ssl       = local.use_https
}

resource "newrelic_synthetics_monitor" "healthz" {
  count            = local.nr_alerts_enabled ? 1 : 0
  account_id       = var.new_relic_account_id
  name             = "${var.new_relic_app_name} healthz ping"
  type             = "SIMPLE"
  uri              = "${local.public_base_url}/healthz"
  period           = "EVERY_5_MINUTES"
  status           = "ENABLED"
  locations_public = ["AP_NORTHEAST_1"]
  verify_ssl       = local.use_https
  # /healthz is GET-only and returns "ok"; use GET and assert the body.
  bypass_head_request = true
  validation_string   = "ok"
}

# Uptime alert: any FAILED synthetic check on either monitor in a 5-min window.
# Lands on the shared policy, so the email workflow forwards it.
resource "newrelic_nrql_alert_condition" "synthetics_down" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.perfect_cat[0].id
  type                         = "static"
  name                         = "Uptime - synthetics ping failing"
  description                  = "A New Relic Synthetics ping check failed (frontend or backend healthz)."
  enabled                      = true
  aggregation_window           = 60
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT count(*) FROM SyntheticCheck WHERE result = 'FAILED' AND monitorName IN ('${newrelic_synthetics_monitor.frontend[0].name}', '${newrelic_synthetics_monitor.healthz[0].name}')"
  }
  critical {
    operator              = "above"
    threshold             = 0
    threshold_duration    = 300
    threshold_occurrences = "at_least_once"
  }
}
