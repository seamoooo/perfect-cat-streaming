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
