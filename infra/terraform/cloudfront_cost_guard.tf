# --- CloudFront bandwidth cost guard ---
#
# Detect (and optionally auto-stop) runaway CloudFront egress, the one usage-based
# cost that scales with a traffic spike:
#   alarm (BytesDownloaded > N GB/hour) -> SNS -> email  (detection)
#                                              \-> Lambda (disable distribution)
#
# CloudFront default metrics only exist in us-east-1, so the alarm + SNS + Lambda
# all live there via the aws.us_east_1 provider alias. There is no native
# CloudFront spend cap; the Lambda disabling the distribution is the closest
# thing to a hard ceiling (it stops video delivery until re-enabled manually).

locals {
  cost_alert_email           = var.cost_alert_email != "" ? var.cost_alert_email : var.new_relic_alert_email
  cost_guard_enabled         = var.cloudfront_cost_guard_enabled
  cost_guard_auto_disable    = var.cloudfront_cost_guard_enabled && var.cloudfront_auto_disable
  cloudfront_bytes_threshold = var.cloudfront_bytes_alarm_gb_per_hour * 1024 * 1024 * 1024
}

# --- SNS topic (us-east-1) ---
resource "aws_sns_topic" "cf_cost" {
  count    = local.cost_guard_enabled ? 1 : 0
  provider = aws.us_east_1
  name     = "${local.name_prefix}-cf-cost-guard"
}

resource "aws_sns_topic_subscription" "cf_cost_email" {
  count     = local.cost_guard_enabled && local.cost_alert_email != "" ? 1 : 0
  provider  = aws.us_east_1
  topic_arn = aws_sns_topic.cf_cost[0].arn
  protocol  = "email"
  endpoint  = local.cost_alert_email
}

# --- Bandwidth alarm (us-east-1) ---
resource "aws_cloudwatch_metric_alarm" "cf_bytes" {
  count             = local.cost_guard_enabled ? 1 : 0
  provider          = aws.us_east_1
  alarm_name        = "${local.name_prefix}-cloudfront-bytes-high"
  alarm_description = "CloudFront BytesDownloaded > ${var.cloudfront_bytes_alarm_gb_per_hour} GB/hour — possible runaway egress cost."

  namespace   = "AWS/CloudFront"
  metric_name = "BytesDownloaded"
  dimensions = {
    DistributionId = aws_cloudfront_distribution.media.id
    Region         = "Global"
  }
  statistic           = "Sum"
  period              = 3600
  evaluation_periods  = 1
  threshold           = local.cloudfront_bytes_threshold
  comparison_operator = "GreaterThanThreshold"
  treat_missing_data  = "notBreaching"

  alarm_actions = [aws_sns_topic.cf_cost[0].arn]
  ok_actions    = [aws_sns_topic.cf_cost[0].arn]
}

# --- Circuit-breaker Lambda (us-east-1) — optional ---
data "archive_file" "cf_disable" {
  count       = local.cost_guard_auto_disable ? 1 : 0
  type        = "zip"
  source_file = "${path.module}/lambda/cloudfront_disable.py"
  output_path = "${path.module}/lambda/cloudfront_disable.zip"
}

resource "aws_iam_role" "cf_disable" {
  count = local.cost_guard_auto_disable ? 1 : 0
  name  = "${local.name_prefix}-cf-disable-lambda"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

data "aws_iam_policy_document" "cf_disable" {
  count = local.cost_guard_auto_disable ? 1 : 0
  statement {
    sid       = "Logs"
    effect    = "Allow"
    actions   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["arn:aws:logs:*:*:*"]
  }
  statement {
    sid       = "DisableDistribution"
    effect    = "Allow"
    actions   = ["cloudfront:GetDistributionConfig", "cloudfront:UpdateDistribution"]
    resources = [aws_cloudfront_distribution.media.arn]
  }
}

resource "aws_iam_role_policy" "cf_disable" {
  count  = local.cost_guard_auto_disable ? 1 : 0
  role   = aws_iam_role.cf_disable[0].id
  policy = data.aws_iam_policy_document.cf_disable[0].json
}

resource "aws_cloudwatch_log_group" "cf_disable" {
  count             = local.cost_guard_auto_disable ? 1 : 0
  provider          = aws.us_east_1
  name              = "/aws/lambda/${local.name_prefix}-cf-disable"
  retention_in_days = var.log_retention_days
}

resource "aws_lambda_function" "cf_disable" {
  count            = local.cost_guard_auto_disable ? 1 : 0
  provider         = aws.us_east_1
  function_name    = "${local.name_prefix}-cf-disable"
  role             = aws_iam_role.cf_disable[0].arn
  filename         = data.archive_file.cf_disable[0].output_path
  source_code_hash = data.archive_file.cf_disable[0].output_base64sha256
  handler          = "cloudfront_disable.handler"
  runtime          = "python3.12"
  timeout          = 30
  memory_size      = 128

  environment {
    variables = {
      DISTRIBUTION_ID = aws_cloudfront_distribution.media.id
    }
  }

  depends_on = [aws_cloudwatch_log_group.cf_disable]
}

resource "aws_sns_topic_subscription" "cf_disable" {
  count     = local.cost_guard_auto_disable ? 1 : 0
  provider  = aws.us_east_1
  topic_arn = aws_sns_topic.cf_cost[0].arn
  protocol  = "lambda"
  endpoint  = aws_lambda_function.cf_disable[0].arn
}

resource "aws_lambda_permission" "cf_disable_sns" {
  count         = local.cost_guard_auto_disable ? 1 : 0
  provider      = aws.us_east_1
  statement_id  = "AllowSNSInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cf_disable[0].function_name
  principal     = "sns.amazonaws.com"
  source_arn    = aws_sns_topic.cf_cost[0].arn
}
