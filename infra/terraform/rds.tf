# --- MySQL on RDS ---

resource "random_password" "db" {
  length  = 32
  special = false # avoid URL-encoding hassles in DSN
}

# --- Enhanced Monitoring role ---
# Lets RDS publish OS-level metrics (per-process CPU/mem, disk, net) to
# CloudWatch Logs, which New Relic surfaces alongside the AWS/RDS metric stream.
data "aws_iam_policy_document" "rds_monitoring_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["monitoring.rds.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "rds_monitoring" {
  name               = "${local.name_prefix}-rds-monitoring"
  assume_role_policy = data.aws_iam_policy_document.rds_monitoring_assume.json
}

resource "aws_iam_role_policy_attachment" "rds_monitoring" {
  role       = aws_iam_role.rds_monitoring.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonRDSEnhancedMonitoringRole"
}

resource "aws_db_subnet_group" "main" {
  name       = "${local.name_prefix}-db-subnets"
  subnet_ids = aws_subnet.private[*].id
  tags       = { Name = "${local.name_prefix}-db-subnets" }
}

resource "aws_security_group" "rds" {
  name        = "${local.name_prefix}-rds"
  description = "RDS MySQL - accept only from ECS tasks"
  vpc_id      = aws_vpc.main.id

  ingress {
    description     = "MySQL from ECS tasks"
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_tasks.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_db_instance" "main" {
  identifier                 = "${local.name_prefix}-mysql"
  engine                     = "mysql"
  engine_version             = var.db_engine_version
  instance_class             = var.db_instance_class
  allocated_storage          = var.db_allocated_storage
  storage_type               = "gp3"
  storage_encrypted          = true
  db_name                    = var.db_name
  username                   = var.db_username
  password                   = random_password.db.result
  port                       = 3306
  db_subnet_group_name       = aws_db_subnet_group.main.name
  vpc_security_group_ids     = [aws_security_group.rds.id]
  publicly_accessible        = false
  backup_retention_period    = 7
  skip_final_snapshot        = true # convenience for dev; flip in prod
  deletion_protection        = false
  apply_immediately          = true
  auto_minor_version_upgrade = true

  # --- New Relic RDS instrumentation (CloudWatch path) ---
  # AWS/RDS standard metrics already flow via the CloudWatch Metric Stream
  # (newrelic_aws_integration.tf). These two add depth that NR can surface:
  #   - Performance Insights: query-level DB load (top SQL, wait events).
  #   - Enhanced Monitoring : OS-level metrics at 60s granularity.
  # PI 7-day retention is free; db.t4g.micro supports both.
  performance_insights_enabled          = true
  performance_insights_retention_period = 7
  monitoring_interval                   = 60
  monitoring_role_arn                   = aws_iam_role.rds_monitoring.arn

  tags = { Name = "${local.name_prefix}-mysql" }
}

# Build the DSN once; store in Secrets Manager so the ECS task can fetch it.
# tls=skip-verify gives encrypted-but-no-cert-verify connections to RDS, which
# is fine for MVP. Switch to bundled RDS CA verification when you formalise.
locals {
  database_url = "mysql://${aws_db_instance.main.username}:${random_password.db.result}@${aws_db_instance.main.endpoint}/${aws_db_instance.main.db_name}?parseTime=true&tls=skip-verify"
}

resource "aws_secretsmanager_secret" "database_url" {
  name                    = "${local.name_prefix}/database-url"
  description             = "MySQL DSN for the perfect-cat backend"
  recovery_window_in_days = 0 # immediate delete on destroy (dev)
}

resource "aws_secretsmanager_secret_version" "database_url" {
  secret_id     = aws_secretsmanager_secret.database_url.id
  secret_string = local.database_url
}
