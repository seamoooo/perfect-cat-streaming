variable "project" {
  description = "Project name; used as a prefix for AWS resources."
  type        = string
  default     = "perfect-cat"
}

variable "environment" {
  description = "Environment name (e.g. dev, stg, prod)."
  type        = string
  default     = "dev"
}

variable "region" {
  description = "AWS region for ECS, ALB, S3."
  type        = string
  default     = "ap-northeast-1"
}

variable "vpc_cidr" {
  type    = string
  default = "10.42.0.0/16"
}

variable "public_subnet_cidrs" {
  type    = list(string)
  default = ["10.42.0.0/24", "10.42.1.0/24"]
}

variable "private_subnet_cidrs" {
  type    = list(string)
  default = ["10.42.10.0/24", "10.42.11.0/24"]
}

variable "azs" {
  description = "AZs to spread subnets across. If empty, the first 2 AZs of the region are used."
  type        = list(string)
  default     = []
}

variable "backend_image" {
  description = "Full image URI for the backend (Bincho/Kanpachi API). e.g. <acct>.dkr.ecr.<region>.amazonaws.com/perfect-cat-backend:<tag>"
  type        = string
}

variable "frontend_image" {
  description = "Full image URI for the frontend (nginx + built Vite assets)."
  type        = string
}

variable "backend_cpu" {
  type    = number
  default = 1024 # 1 vCPU — ffmpeg is CPU-heavy
}

variable "backend_memory" {
  type    = number
  default = 2048
}

variable "frontend_cpu" {
  type    = number
  default = 256
}

variable "frontend_memory" {
  type    = number
  default = 512
}

variable "backend_desired_count" {
  type    = number
  default = 1
}

variable "frontend_desired_count" {
  type    = number
  default = 1
}

variable "log_retention_days" {
  type    = number
  default = 14
}

# --- RDS ---
variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_allocated_storage" {
  type    = number
  default = 20
}

variable "db_engine_version" {
  description = "MySQL engine version on RDS."
  type        = string
  default     = "8.0.46"
}

variable "db_name" {
  type    = string
  default = "perfectcat"
}

variable "db_username" {
  type    = string
  default = "perfectcat"
}

variable "image_tag" {
  description = "Tag pushed to ECR; auto-redeploy watches for this tag."
  type        = string
  default     = "latest"
}

# --- New Relic APM (optional) ---
variable "new_relic_license_key" {
  description = "New Relic ingest license key. Leave empty to ship the backend with APM disabled."
  type        = string
  default     = ""
  sensitive   = true
}

variable "new_relic_app_name" {
  description = "App name shown in New Relic. The environment suffix is appended automatically."
  type        = string
  default     = "PerfectCatStreaming"
}

variable "new_relic_ai_monitoring_enabled" {
  description = "Toggle ConfigAIMonitoringEnabled in the Go agent (LLM observability)."
  type        = bool
  default     = false
}

variable "new_relic_custom_events_max_samples" {
  description = "Sample cap for the CustomInsightsEvents buffer (NR recommends 10000 for AI workloads)."
  type        = number
  default     = 10000
}

variable "new_relic_ai_monitoring_streaming_enabled" {
  description = "AIMonitoring.Streaming.Enabled — picked up directly by the agent from this env var."
  type        = bool
  default     = false
}

variable "new_relic_user_api_key" {
  description = "Optional User key (NRAK-*) — not used by the Go agent; stored in Secrets Manager for NerdGraph / Terraform NR provider use. Required to manage alerts in Terraform."
  type        = string
  default     = ""
  sensitive   = true
}

variable "new_relic_account_id" {
  description = "New Relic account ID for the Terraform provider (alerts/notifications)."
  type        = number
  default     = 6729598
}

variable "new_relic_alert_email" {
  description = "Email address that receives New Relic alert notifications. Empty = no email destination/workflow created."
  type        = string
  default     = ""
}

# Optional HTTPS — set both to enable TLS on the ALB and CloudFront alias.
variable "domain_name" {
  description = "Custom domain (e.g. cats.example.com). Empty = HTTP-only ALB and *.cloudfront.net for CDN."
  type        = string
  default     = ""
}

variable "acm_certificate_arn_alb" {
  description = "ACM cert ARN in the ALB region. Required if domain_name is set."
  type        = string
  default     = ""
}

variable "acm_certificate_arn_cloudfront" {
  description = "ACM cert ARN in us-east-1 (CloudFront only). Required if domain_name is set."
  type        = string
  default     = ""
}
