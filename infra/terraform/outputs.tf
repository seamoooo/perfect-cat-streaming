output "alb_dns_name" {
  description = "ALB hostname — point your domain or open directly."
  value       = aws_lb.main.dns_name
}

output "cloudfront_domain" {
  description = "CloudFront distribution domain. Use as MEDIA_BASE_URL."
  value       = aws_cloudfront_distribution.media.domain_name
}

output "cloudfront_distribution_id" {
  value = aws_cloudfront_distribution.media.id
}

output "media_bucket" {
  description = "Private S3 bucket used by the backend publisher."
  value       = aws_s3_bucket.media.id
}

output "ecr_backend_url" {
  value = aws_ecr_repository.backend.repository_url
}

output "ecr_frontend_url" {
  value = aws_ecr_repository.frontend.repository_url
}

output "ecs_cluster" {
  value = aws_ecs_cluster.main.name
}

output "frontend_url" {
  description = "Open this in a browser."
  value       = local.use_https ? "https://${var.domain_name}" : "http://${aws_lb.main.dns_name}"
}

output "rds_endpoint" {
  description = "MySQL endpoint (host:port). Stored in Secrets Manager as DATABASE_URL."
  value       = aws_db_instance.main.endpoint
}

output "rds_secret_arn" {
  description = "Secrets Manager ARN holding DATABASE_URL."
  value       = aws_secretsmanager_secret.database_url.arn
}

output "redeploy_lambda" {
  description = "Lambda that redeploys ECS services on ECR push."
  value       = aws_lambda_function.redeploy.function_name
}
