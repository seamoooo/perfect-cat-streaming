# --- New Relic alerts (managed via terraform-provider-newrelic) ---
#
# Three signals, one policy, optional email workflow:
#   1. Backend throughput  — transcode efficiency (realtime_factor) + web req rate
#   2. Browser Core Web Vitals — LCP p75
#   3. Video agent — player error (CONTENT_ERROR)
#
# Everything is gated on local.nr_alerts_enabled, so with no User key (NRAK-*)
# the New Relic provider is never invoked and `plan` still works. Set
# new_relic_user_api_key (and optionally new_relic_alert_email) in the gitignored
# secret.auto.tfvars to turn these on.

locals {
  nr_alerts_enabled = var.new_relic_user_api_key != "" && var.new_relic_account_id != 0
  nr_email_enabled  = local.nr_alerts_enabled && var.new_relic_alert_email != ""
}

resource "newrelic_alert_policy" "perfect_cat" {
  count               = local.nr_alerts_enabled ? 1 : 0
  account_id          = var.new_relic_account_id
  name                = "${var.new_relic_app_name} alerts"
  incident_preference = "PER_CONDITION"
}

# 1a. Backend — transcode throughput degraded.
# realtime_factor = wall_sec / video_duration_sec; <1 is faster than real time.
# The SRE chaos demo (an "SRE" keyword in a description) pushes this from ~0.1
# toward ~1.0+, so "above 0.6" cleanly separates degraded from healthy.
resource "newrelic_nrql_alert_condition" "transcode_throughput" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.perfect_cat[0].id
  type                         = "static"
  name                         = "Backend transcode throughput degraded"
  description                  = "transcode.realtime_factor high = transcoding slower than real time (e.g. SRE chaos)."
  enabled                      = true
  aggregation_window           = 60
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT average(`transcode.realtime_factor`) FROM Transaction WHERE appName = '${var.new_relic_app_name}' AND `transcode.realtime_factor` IS NOT NULL"
  }
  critical {
    operator              = "above"
    threshold             = 0.6
    threshold_duration    = 300
    threshold_occurrences = "all"
  }
}

# 1b. Backend — web request throughput drop, as a baseline so it never
# false-fires on a naturally idle demo. lower_only => alert only on drops below
# the learned baseline (threshold is in standard deviations).
resource "newrelic_nrql_alert_condition" "web_throughput" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.perfect_cat[0].id
  type                         = "baseline"
  baseline_direction           = "lower_only"
  name                         = "Backend web throughput drop"
  description                  = "Web request throughput fell well below the learned baseline."
  enabled                      = true
  aggregation_window           = 60
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT rate(count(*), 1 minute) FROM Transaction WHERE appName = '${var.new_relic_app_name}' AND transactionType = 'Web'"
  }
  critical {
    operator              = "above"
    threshold             = 3
    threshold_duration    = 300
    threshold_occurrences = "all"
  }
}

# 2. Browser — Core Web Vitals (LCP). PageViewTiming.largestContentfulPaint is
# in seconds; 2.5s is the CWV "good" upper bound. Account-scoped — if this NR
# account hosts more than one browser app, add `AND appName = '...'`.
resource "newrelic_nrql_alert_condition" "browser_cwv_lcp" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.perfect_cat[0].id
  type                         = "static"
  name                         = "Browser Core Web Vitals - LCP poor (p75)"
  description                  = "p75 Largest Contentful Paint above the 2.5s CWV 'good' threshold."
  enabled                      = true
  aggregation_window           = 300
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT percentile(largestContentfulPaint, 75) FROM PageViewTiming"
  }
  critical {
    operator              = "above"
    threshold             = 2.5
    threshold_duration    = 300
    threshold_occurrences = "all"
  }
}

# 3. Video agent — player error. The NR Video agent emits CONTENT_ERROR on a
# fatal player error (our "player" chaos keyword forces one). Stored as a
# PageAction; account-scoped. Fires if any error lands in a 5-min window.
resource "newrelic_nrql_alert_condition" "video_player_error" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.perfect_cat[0].id
  type                         = "static"
  name                         = "Video player error (CONTENT_ERROR)"
  description                  = "New Relic Video agent reported a fatal player error."
  enabled                      = true
  aggregation_window           = 60
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT count(*) FROM PageAction WHERE actionName = 'CONTENT_ERROR'"
  }
  critical {
    operator              = "above"
    threshold             = 0
    threshold_duration    = 300
    threshold_occurrences = "at_least_once"
  }
}

# --- Email notification (destination -> channel -> workflow) ---
# Only when new_relic_alert_email is set. The workflow forwards every issue from
# the policy above to the email channel.
resource "newrelic_notification_destination" "email" {
  count      = local.nr_email_enabled ? 1 : 0
  account_id = var.new_relic_account_id
  name       = "${var.new_relic_app_name} email"
  type       = "EMAIL"
  property {
    key   = "email"
    value = var.new_relic_alert_email
  }
}

resource "newrelic_notification_channel" "email" {
  count          = local.nr_email_enabled ? 1 : 0
  account_id     = var.new_relic_account_id
  name           = "${var.new_relic_app_name} email channel"
  type           = "EMAIL"
  destination_id = newrelic_notification_destination.email[0].id
  product        = "IINT"
  property {
    key   = "subject"
    value = "[New Relic] {{ issueTitle }}"
  }
}

resource "newrelic_workflow" "email" {
  count                 = local.nr_email_enabled ? 1 : 0
  account_id            = var.new_relic_account_id
  name                  = "${var.new_relic_app_name} workflow"
  muting_rules_handling = "NOTIFY_ALL_ISSUES"

  issues_filter {
    name = "policy-filter"
    type = "FILTER"
    predicate {
      attribute = "labels.policyIds"
      operator  = "EXACTLY_MATCHES"
      values    = [newrelic_alert_policy.perfect_cat[0].id]
    }
  }

  destination {
    channel_id = newrelic_notification_channel.email[0].id
  }
}
