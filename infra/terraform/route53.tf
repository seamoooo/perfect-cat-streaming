# --- DNS records pointing the custom domain at ALB / CloudFront ---
#
# Alias records (free, no CNAME-at-apex restriction). Created only when a
# custom domain is configured. The zone is the one created by Route53 domain
# registration (looked up in acm.tf as data.aws_route53_zone.main).

# apex (e.g. example.com) → ALB (frontend + /api/*)
resource "aws_route53_record" "frontend" {
  count   = var.domain_name != "" ? 1 : 0
  zone_id = data.aws_route53_zone.main[0].zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_lb.main.dns_name
    zone_id                = aws_lb.main.zone_id
    evaluate_target_health = true
  }
}

# www.<domain> → ALB as well, so both work.
resource "aws_route53_record" "frontend_www" {
  count   = var.domain_name != "" ? 1 : 0
  zone_id = data.aws_route53_zone.main[0].zone_id
  name    = "www.${var.domain_name}"
  type    = "A"

  alias {
    name                   = aws_lb.main.dns_name
    zone_id                = aws_lb.main.zone_id
    evaluate_target_health = true
  }
}

# media.<domain> → CloudFront (HLS distribution)
resource "aws_route53_record" "media" {
  count   = var.domain_name != "" ? 1 : 0
  zone_id = data.aws_route53_zone.main[0].zone_id
  name    = "media.${var.domain_name}"
  type    = "A"

  alias {
    name = aws_cloudfront_distribution.media.domain_name
    # CloudFront's hosted zone ID is a fixed global constant.
    zone_id                = "Z2FDTNDATAQYW2"
    evaluate_target_health = false
  }
}
