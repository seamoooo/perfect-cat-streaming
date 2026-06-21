# --- AWS WAF (WAFv2) on the ALB ---
#
# Protects the public app/API entry point with AWS managed rule groups plus a
# per-IP rate limit. REGIONAL scope = Application Load Balancer.

variable "waf_enabled" {
  description = "Attach AWS WAF to the ALB."
  type        = bool
  default     = true
}

variable "waf_rate_limit" {
  description = "Max requests per 5-minute window per IP before blocking."
  type        = number
  default     = 2000
}

resource "aws_wafv2_web_acl" "alb" {
  count       = var.waf_enabled ? 1 : 0
  name        = "${local.name_prefix}-alb-waf"
  description = "Managed protections + rate limit for the perfect-cat ALB"
  scope       = "REGIONAL"

  default_action {
    allow {}
  }

  # AWS Managed: broad common protections (XSS, LFI, bad bots, etc.)
  rule {
    name     = "AWSCommon"
    priority = 1
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"

        # Video uploads send large multipart bodies; don't let the default
        # 8KB body-size rule block them (count instead of block).
        rule_action_override {
          name = "SizeRestrictions_BODY"
          action_to_use {
            count {}
          }
        }
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSCommon"
      sampled_requests_enabled   = true
    }
  }

  # AWS Managed: known malicious inputs.
  rule {
    name     = "AWSKnownBadInputs"
    priority = 2
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesKnownBadInputsRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSKnownBadInputs"
      sampled_requests_enabled   = true
    }
  }

  # AWS Managed: SQL injection.
  rule {
    name     = "AWSSQLi"
    priority = 3
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesSQLiRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSSQLi"
      sampled_requests_enabled   = true
    }
  }

  # AWS Managed: Amazon IP reputation list (known bad sources).
  rule {
    name     = "AWSIpReputation"
    priority = 4
    override_action {
      none {}
    }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesAmazonIpReputationList"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSIpReputation"
      sampled_requests_enabled   = true
    }
  }

  # Per-IP rate limit (DDoS / brute-force dampening).
  rule {
    name     = "RateLimit"
    priority = 5
    action {
      block {}
    }
    statement {
      rate_based_statement {
        limit              = var.waf_rate_limit
        aggregate_key_type = "IP"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "RateLimit"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "${local.name_prefix}-alb-waf"
    sampled_requests_enabled   = true
  }
}

resource "aws_wafv2_web_acl_association" "alb" {
  count        = var.waf_enabled ? 1 : 0
  resource_arn = aws_lb.main.arn
  web_acl_arn  = aws_wafv2_web_acl.alb[0].arn
}
