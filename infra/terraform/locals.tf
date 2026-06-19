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

  # HTTPS is enabled whenever a custom domain is set. ACM certs are created and
  # DNS-validated automatically in acm.tf (using the Route53 zone created by
  # domain registration) unless you bring your own cert ARNs via the
  # acm_certificate_arn_* vars.
  use_https        = var.domain_name != ""
  cloudfront_alias = var.domain_name != "" ? ["media.${var.domain_name}"] : []

  # Which cert to wire into the ALB listener / CloudFront distribution:
  # an explicitly provided ARN wins, otherwise the auto-created+validated cert.
  alb_certificate_arn        = var.acm_certificate_arn_alb != "" ? var.acm_certificate_arn_alb : one(aws_acm_certificate_validation.alb[*].certificate_arn)
  cloudfront_certificate_arn = var.acm_certificate_arn_cloudfront != "" ? var.acm_certificate_arn_cloudfront : one(aws_acm_certificate_validation.cloudfront[*].certificate_arn)

  # Create certs only when a domain is set AND no override ARN was supplied.
  create_alb_cert        = var.domain_name != "" && var.acm_certificate_arn_alb == ""
  create_cloudfront_cert = var.domain_name != "" && var.acm_certificate_arn_cloudfront == ""
}
