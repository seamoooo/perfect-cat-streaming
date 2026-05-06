data "aws_availability_zones" "available" {
  state = "available"
}

resource "random_id" "suffix" {
  byte_length = 3
}

locals {
  name_prefix = "${var.project}-${var.environment}"
  azs         = length(var.azs) > 0 ? var.azs : slice(data.aws_availability_zones.available.names, 0, 2)

  # S3 bucket name must be globally unique. Append a short random suffix.
  media_bucket_name = "${local.name_prefix}-media-${random_id.suffix.hex}"

  # Path-based routing on the ALB
  api_path_patterns = ["/api/*", "/healthz", "/meow"]
  # /media/* stays on the backend so local dev can still preview;
  # in production the frontend will hit CloudFront directly via MEDIA_BASE_URL.

  use_https        = var.domain_name != "" && var.acm_certificate_arn_alb != ""
  cloudfront_alias = var.domain_name != "" && var.acm_certificate_arn_cloudfront != "" ? ["media.${var.domain_name}"] : []
}
