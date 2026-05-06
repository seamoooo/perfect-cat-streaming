resource "aws_cloudwatch_log_group" "backend" {
  name              = "/${local.name_prefix}/backend"
  retention_in_days = var.log_retention_days
}

resource "aws_cloudwatch_log_group" "frontend" {
  name              = "/${local.name_prefix}/frontend"
  retention_in_days = var.log_retention_days
}
