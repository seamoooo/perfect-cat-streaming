# --- New Relic AWS infrastructure monitoring via CloudWatch Metric Streams ---
#
# Pushes AWS infra metrics (RDS, ALB, ECS service-level, S3, WAF) to New Relic
# WITHOUT any agent inside ECS — CloudWatch Metric Stream -> Kinesis Firehose ->
# New Relic metric endpoint. Only created when a New Relic license key is set.

variable "new_relic_region" {
  description = "New Relic data center for the metric stream endpoint (US or EU)."
  type        = string
  default     = "US"
}

locals {
  nr_aws_integration = var.new_relic_license_key != "" ? 1 : 0
  nr_metrics_endpoint = upper(var.new_relic_region) == "EU" ? (
    "https://aws-api.eu.newrelic.com/cloudwatch-metrics/v1"
  ) : "https://aws-api.newrelic.com/cloudwatch-metrics/v1"
}

# Backup bucket for Firehose delivery failures.
resource "aws_s3_bucket" "firehose_backup" {
  count         = local.nr_aws_integration
  bucket        = "${local.name_prefix}-nr-fh-${random_id.suffix.hex}"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "firehose_backup" {
  count                   = local.nr_aws_integration
  bucket                  = aws_s3_bucket.firehose_backup[0].id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# --- Firehose role (write to backup bucket) ---
data "aws_iam_policy_document" "firehose_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["firehose.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "firehose_nr" {
  count              = local.nr_aws_integration
  name               = "${local.name_prefix}-nr-firehose"
  assume_role_policy = data.aws_iam_policy_document.firehose_assume.json
}

data "aws_iam_policy_document" "firehose_nr" {
  count = local.nr_aws_integration
  statement {
    effect = "Allow"
    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObject",
    ]
    resources = [
      aws_s3_bucket.firehose_backup[0].arn,
      "${aws_s3_bucket.firehose_backup[0].arn}/*",
    ]
  }
}

resource "aws_iam_role_policy" "firehose_nr" {
  count  = local.nr_aws_integration
  name   = "firehose-s3"
  role   = aws_iam_role.firehose_nr[0].id
  policy = data.aws_iam_policy_document.firehose_nr[0].json
}

# --- Firehose delivery stream -> New Relic ---
resource "aws_kinesis_firehose_delivery_stream" "newrelic" {
  count       = local.nr_aws_integration
  name        = "${local.name_prefix}-nr-metrics"
  destination = "http_endpoint"

  http_endpoint_configuration {
    url                = local.nr_metrics_endpoint
    name               = "New Relic"
    access_key         = var.new_relic_license_key
    buffering_size     = 1
    buffering_interval = 60
    role_arn           = aws_iam_role.firehose_nr[0].arn
    s3_backup_mode     = "FailedDataOnly"

    request_configuration {
      content_encoding = "GZIP"
    }

    s3_configuration {
      role_arn           = aws_iam_role.firehose_nr[0].arn
      bucket_arn         = aws_s3_bucket.firehose_backup[0].arn
      buffering_size     = 5
      buffering_interval = 300
      compression_format = "GZIP"
    }
  }
}

# --- Metric stream role (write to Firehose) ---
data "aws_iam_policy_document" "metric_stream_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["streams.metrics.cloudwatch.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "metric_stream_nr" {
  count              = local.nr_aws_integration
  name               = "${local.name_prefix}-nr-metric-stream"
  assume_role_policy = data.aws_iam_policy_document.metric_stream_assume.json
}

data "aws_iam_policy_document" "metric_stream_nr" {
  count = local.nr_aws_integration
  statement {
    effect    = "Allow"
    actions   = ["firehose:PutRecord", "firehose:PutRecordBatch"]
    resources = [aws_kinesis_firehose_delivery_stream.newrelic[0].arn]
  }
}

resource "aws_iam_role_policy" "metric_stream_nr" {
  count  = local.nr_aws_integration
  name   = "metric-stream-firehose"
  role   = aws_iam_role.metric_stream_nr[0].id
  policy = data.aws_iam_policy_document.metric_stream_nr[0].json
}

# --- The metric stream (infra namespaces; no ECS container internals) ---
resource "aws_cloudwatch_metric_stream" "newrelic" {
  count         = local.nr_aws_integration
  name          = "${local.name_prefix}-nr-metric-stream"
  role_arn      = aws_iam_role.metric_stream_nr[0].arn
  firehose_arn  = aws_kinesis_firehose_delivery_stream.newrelic[0].arn
  output_format = "opentelemetry0.7"

  include_filter {
    namespace = "AWS/RDS"
  }
  include_filter {
    namespace = "AWS/ApplicationELB"
  }
  include_filter {
    namespace = "AWS/ECS"
  }
  include_filter {
    namespace = "AWS/S3"
  }
  include_filter {
    namespace = "AWS/WAFV2"
  }
}
