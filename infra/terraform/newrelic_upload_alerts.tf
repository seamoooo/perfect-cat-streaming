# --- "Upload pipeline" alerts (the important ones: errors + job degradation) ---
#
# A dedicated policy + workflow so Slack (and a manually-attached SRE Agent) can
# be wired to just these high-signal alerts. Signals chosen from telemetry that
# is actually flowing — AWS/ECS CloudWatch metrics are NOT reaching New Relic, so
# CPU/memory degradation is detected via APM proxies instead:
#   A. transcode.job errors  → ffmpeg failures (e.g. HEVC exit status 1)
#   B. transcode.realtime_factor high → transcode struggling (CPU/mem pressure)
#   C. web latency spike     → CPU contention symptom (normal p95 ~2.6ms)
#
# Gated on local.nr_alerts_enabled like the rest. Slack activates only once
# slack_webhook_url is set in secret.auto.tfvars.

locals {
  upload_workflow_enabled = local.nr_alerts_enabled && local.nr_email_enabled
}

resource "newrelic_alert_policy" "upload_pipeline" {
  count               = local.nr_alerts_enabled ? 1 : 0
  account_id          = var.new_relic_account_id
  name                = "${var.new_relic_app_name} - upload pipeline"
  incident_preference = "PER_CONDITION"
}

# A. Upload/transcode failure — ffmpeg died (HEVC etc.). The transcoder records
# txn.NoticeError on the background "transcoder.job" transaction.
resource "newrelic_nrql_alert_condition" "transcode_error" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.upload_pipeline[0].id
  type                         = "static"
  name                         = "Upload/transcode error (ffmpeg failed)"
  description                  = "A transcoder.job transaction errored (e.g. ffmpeg exit status 1 on HEVC)."
  enabled                      = true
  aggregation_window           = 60
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT count(*) FROM TransactionError WHERE appName = '${var.new_relic_app_name}' AND transactionName = 'OtherTransaction/Go/transcoder.job'"
  }
  critical {
    operator              = "above"
    threshold             = 0
    threshold_duration    = 300
    threshold_occurrences = "at_least_once"
  }
}

# B. Transcode job degradation — wall time per second of video. Healthy HEVC is
# ~1.8x; sustained >5x means the job is starved (CPU/memory pressure).
resource "newrelic_nrql_alert_condition" "transcode_degraded" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.upload_pipeline[0].id
  type                         = "static"
  name                         = "Transcode job degraded (realtime_factor)"
  description                  = "transcode.realtime_factor sustained high — ffmpeg starved for CPU/memory."
  enabled                      = true
  aggregation_window           = 300
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT average(`transcode.realtime_factor`) FROM Transaction WHERE appName = '${var.new_relic_app_name}' AND `transcode.realtime_factor` IS NOT NULL"
  }
  critical {
    operator              = "above"
    threshold             = 5
    threshold_duration    = 300
    threshold_occurrences = "all"
  }
}

# C. Backend latency spike — web requests normally finish in a few ms; a request
# over 10s means the shared CPU is being monopolised (transcode contention).
resource "newrelic_nrql_alert_condition" "backend_latency_spike" {
  count                        = local.nr_alerts_enabled ? 1 : 0
  account_id                   = var.new_relic_account_id
  policy_id                    = newrelic_alert_policy.upload_pipeline[0].id
  type                         = "static"
  name                         = "Backend latency spike (CPU contention)"
  description                  = "A web request took >10s (normal p95 ~2.6ms) — likely CPU starvation during transcode."
  enabled                      = true
  aggregation_window           = 60
  aggregation_method           = "event_flow"
  aggregation_delay            = 120
  violation_time_limit_seconds = 3600

  nrql {
    query = "SELECT count(*) FROM Transaction WHERE appName = '${var.new_relic_app_name}' AND transactionType = 'Web' AND duration > 10"
  }
  critical {
    operator              = "above"
    threshold             = 0
    threshold_duration    = 300
    threshold_occurrences = "at_least_once"
  }
}

# Dedicated email channel for this workflow. New Relic ties each channel to a
# single workflow, so we can't reuse the channel from the email-only workflow —
# but we can point a fresh channel at the same email destination.
resource "newrelic_notification_channel" "upload_email" {
  count          = local.nr_email_enabled ? 1 : 0
  account_id     = var.new_relic_account_id
  name           = "${var.new_relic_app_name} upload-pipeline email channel"
  type           = "EMAIL"
  destination_id = newrelic_notification_destination.email[0].id
  product        = "IINT"
  property {
    key   = "subject"
    value = "[New Relic] {{ issueTitle }}"
  }
}

# --- Workflow: route the upload-pipeline policy to email ---
# Slack (native OAuth) and the preview "SRE Agent" are attached to THIS workflow
# in the New Relic UI. lifecycle.ignore_changes on `destination` below means a
# later `terraform apply` won't strip those GUI-added destinations.
resource "newrelic_workflow" "upload_pipeline" {
  count                 = local.upload_workflow_enabled ? 1 : 0
  account_id            = var.new_relic_account_id
  name                  = "${var.new_relic_app_name} - upload pipeline workflow"
  muting_rules_handling = "NOTIFY_ALL_ISSUES"

  issues_filter {
    name = "upload-pipeline-policy"
    type = "FILTER"
    predicate {
      attribute = "labels.policyIds"
      operator  = "EXACTLY_MATCHES"
      values    = [newrelic_alert_policy.upload_pipeline[0].id]
    }
  }

  dynamic "destination" {
    for_each = local.nr_email_enabled ? [newrelic_notification_channel.upload_email[0].id] : []
    content {
      channel_id = destination.value
    }
  }

  # Slack + SRE Agent are added to this workflow in the New Relic UI; don't let
  # Terraform revert those GUI-managed destinations on the next apply.
  lifecycle {
    ignore_changes = [destination]
  }
}
