resource "aws_ecs_cluster" "main" {
  name = "${local.name_prefix}-cluster"
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

# --- Backend task ---
resource "aws_ecs_task_definition" "backend" {
  family                   = "${local.name_prefix}-backend"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.backend_cpu
  memory                   = var.backend_memory
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.backend_task.arn

  ephemeral_storage {
    size_in_gib = 30
  }

  container_definitions = jsonencode([{
    name         = "backend"
    image        = var.backend_image
    essential    = true
    portMappings = [{ containerPort = 8080, hostPort = 8080, protocol = "tcp" }]
    environment = [
      { name = "APP_PORT", value = "8080" },
      { name = "STORAGE_UPLOAD_DIR", value = "/var/app/data/uploads" },
      { name = "STORAGE_HLS_DIR", value = "/var/app/data/hls" },
      { name = "PUBLIC_BASE_URL", value = local.use_https ? "https://${var.domain_name}" : "http://${aws_lb.main.dns_name}" },
      { name = "ALLOWED_ORIGINS", value = local.use_https ? "https://${var.domain_name}" : "http://${aws_lb.main.dns_name}" },
      { name = "FFMPEG_BIN", value = "ffmpeg" },
      { name = "S3_BUCKET", value = aws_s3_bucket.media.id },
      { name = "S3_HLS_PREFIX", value = "hls" },
      { name = "S3_REGION", value = var.region },
      { name = "MEDIA_BASE_URL", value = length(local.cloudfront_alias) > 0 ? "https://${local.cloudfront_alias[0]}" : "https://${aws_cloudfront_distribution.media.domain_name}" },
      { name = "NEW_RELIC_APP_NAME", value = var.new_relic_app_name },
      { name = "APP_ENV", value = var.environment },
      { name = "NEW_RELIC_AI_MONITORING_ENABLED", value = tostring(var.new_relic_ai_monitoring_enabled) },
      { name = "NEW_RELIC_CUSTOM_INSIGHTS_EVENTS_MAX_SAMPLES_STORED", value = tostring(var.new_relic_custom_events_max_samples) },
      { name = "NEW_RELIC_AI_MONITORING_STREAMING_ENABLED", value = tostring(var.new_relic_ai_monitoring_streaming_enabled) },
    ]
    secrets = concat(
      [{ name = "DATABASE_URL", valueFrom = aws_secretsmanager_secret.database_url.arn }],
      local.new_relic_enabled ? [
        { name = "NEW_RELIC_LICENSE_KEY", valueFrom = aws_secretsmanager_secret.new_relic_license[0].arn },
      ] : [],
    )
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.backend.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "backend"
      }
    }
    healthCheck = {
      command     = ["CMD-SHELL", "wget -q -O- http://localhost:8080/healthz || exit 1"]
      interval    = 30
      timeout     = 5
      retries     = 3
      startPeriod = 30
    }
  }])
}

resource "aws_ecs_service" "backend" {
  name            = "${local.name_prefix}-backend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.backend.arn
  desired_count   = var.backend_desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.ecs_tasks.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.backend.arn
    container_name   = "backend"
    container_port   = 8080
  }

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  depends_on = [aws_lb_listener.http, aws_db_instance.main]
}

# --- Frontend task ---
resource "aws_ecs_task_definition" "frontend" {
  family                   = "${local.name_prefix}-frontend"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = var.frontend_cpu
  memory                   = var.frontend_memory
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.frontend_task.arn

  container_definitions = jsonencode([{
    name         = "frontend"
    image        = var.frontend_image
    essential    = true
    portMappings = [{ containerPort = 80, hostPort = 80, protocol = "tcp" }]
    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.frontend.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "frontend"
      }
    }
  }])
}

resource "aws_ecs_service" "frontend" {
  name            = "${local.name_prefix}-frontend"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.frontend.arn
  desired_count   = var.frontend_desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = aws_subnet.private[*].id
    security_groups = [aws_security_group.ecs_tasks.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.frontend.arn
    container_name   = "frontend"
    container_port   = 80
  }

  deployment_circuit_breaker {
    enable   = true
    rollback = true
  }

  depends_on = [aws_lb_listener.http]
}
