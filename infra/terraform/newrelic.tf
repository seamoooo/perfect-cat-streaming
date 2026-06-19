# --- New Relic license key in Secrets Manager ---
# Only materialised when var.new_relic_license_key is non-empty. The ECS task
# reads NEW_RELIC_LICENSE_KEY from this secret via the `secrets` block in
# ecs.tf so the value never lives in task-definition JSON or env exports.

locals {
  new_relic_enabled = var.new_relic_license_key != ""
}

resource "aws_secretsmanager_secret" "new_relic_license" {
  count                   = local.new_relic_enabled ? 1 : 0
  name                    = "${local.name_prefix}/new-relic-license-key"
  description             = "New Relic ingest license key for the perfect-cat backend"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "new_relic_license" {
  count         = local.new_relic_enabled ? 1 : 0
  secret_id     = aws_secretsmanager_secret.new_relic_license[0].id
  secret_string = var.new_relic_license_key
}

# Extend the exec role's secrets-read policy to include the NR key when enabled.
data "aws_iam_policy_document" "ecs_secrets_newrelic" {
  count = local.new_relic_enabled ? 1 : 0
  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.new_relic_license[0].arn]
  }
}

resource "aws_iam_role_policy" "ecs_execution_newrelic_secret" {
  count  = local.new_relic_enabled ? 1 : 0
  name   = "secrets-read-newrelic"
  role   = aws_iam_role.ecs_execution.id
  policy = data.aws_iam_policy_document.ecs_secrets_newrelic[0].json
}

# --- (optional) New Relic User key (NRAK-*) ---
# NOT used by the Go agent. Stored only when var.new_relic_user_api_key is
# set, so future tooling — e.g. terraform-provider-newrelic for dashboards/
# alerts, or NerdGraph CLI from an admin workflow — can read it.
locals {
  new_relic_user_key_enabled = var.new_relic_user_api_key != ""
}

resource "aws_secretsmanager_secret" "new_relic_user_key" {
  count                   = local.new_relic_user_key_enabled ? 1 : 0
  name                    = "${local.name_prefix}/new-relic-user-api-key"
  description             = "New Relic User key (NRAK-*) for management APIs"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "new_relic_user_key" {
  count         = local.new_relic_user_key_enabled ? 1 : 0
  secret_id     = aws_secretsmanager_secret.new_relic_user_key[0].id
  secret_string = var.new_relic_user_api_key
}
