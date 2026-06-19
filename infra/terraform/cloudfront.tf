resource "aws_cloudfront_origin_access_control" "media" {
  name                              = "${local.name_prefix}-media-oac"
  description                       = "OAC for ${aws_s3_bucket.media.id}"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# Cache policy for HLS playlists: short TTL so updates propagate.
resource "aws_cloudfront_cache_policy" "hls_playlist" {
  name        = "${local.name_prefix}-hls-playlist"
  default_ttl = 5
  max_ttl     = 30
  min_ttl     = 0

  parameters_in_cache_key_and_forwarded_to_origin {
    enable_accept_encoding_brotli = true
    enable_accept_encoding_gzip   = true
    cookies_config { cookie_behavior = "none" }
    headers_config { header_behavior = "none" }
    query_strings_config { query_string_behavior = "none" }
  }
}

# Response headers policy: CORS so hls.js can play from any origin.
resource "aws_cloudfront_response_headers_policy" "cors" {
  name = "${local.name_prefix}-cors"
  cors_config {
    access_control_allow_credentials = false
    access_control_allow_methods { items = ["GET", "HEAD", "OPTIONS"] }
    access_control_allow_origins { items = ["*"] }
    access_control_allow_headers { items = ["*"] }
    access_control_expose_headers { items = ["ETag"] }
    access_control_max_age_sec = 3000
    origin_override            = true
  }
}

resource "aws_cloudfront_distribution" "media" {
  enabled         = true
  is_ipv6_enabled = true
  comment         = "${local.name_prefix} HLS distribution"
  price_class     = "PriceClass_200"

  aliases = local.cloudfront_alias

  origin {
    domain_name              = aws_s3_bucket.media.bucket_regional_domain_name
    origin_id                = "s3-media"
    origin_access_control_id = aws_cloudfront_origin_access_control.media.id
  }

  # Default behaviour: long-lived cache for segments and other assets.
  default_cache_behavior {
    target_origin_id           = "s3-media"
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    compress                   = true
    cache_policy_id            = data.aws_cloudfront_cache_policy.caching_optimized.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.cors.id
  }

  # Override for playlists — short TTL.
  ordered_cache_behavior {
    path_pattern               = "*.m3u8"
    target_origin_id           = "s3-media"
    viewer_protocol_policy     = "redirect-to-https"
    allowed_methods            = ["GET", "HEAD", "OPTIONS"]
    cached_methods             = ["GET", "HEAD"]
    compress                   = true
    cache_policy_id            = aws_cloudfront_cache_policy.hls_playlist.id
    response_headers_policy_id = aws_cloudfront_response_headers_policy.cors.id
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = length(local.cloudfront_alias) == 0
    acm_certificate_arn            = length(local.cloudfront_alias) > 0 ? local.cloudfront_certificate_arn : null
    ssl_support_method             = length(local.cloudfront_alias) > 0 ? "sni-only" : null
    minimum_protocol_version       = length(local.cloudfront_alias) > 0 ? "TLSv1.2_2021" : "TLSv1"
  }
}

data "aws_cloudfront_cache_policy" "caching_optimized" {
  name = "Managed-CachingOptimized"
}
