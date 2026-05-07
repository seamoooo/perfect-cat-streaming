# --- EventBridge → Lambda → ECS UpdateService on ECR push ---

data "archive_file" "redeploy_lambda" {
  type        = "zip"
  source_file = "${path.module}/lambda/redeploy.py"
  output_path = "${path.module}/lambda/redeploy.zip"
}

resource "aws_iam_role" "redeploy_lambda" {
  name = "${local.name_prefix}-redeploy-lambda"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

data "aws_iam_policy_document" "redeploy_lambda" {
  statement {
    sid    = "Logs"
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["arn:aws:logs:*:*:*"]
  }
  statement {
    sid    = "UpdateECSService"
    effect = "Allow"
    actions = [
      "ecs:UpdateService",
      "ecs:DescribeServices",
    ]
    resources = [
      aws_ecs_service.backend.id,
      aws_ecs_service.frontend.id,
    ]
  }
}

resource "aws_iam_role_policy" "redeploy_lambda" {
  role   = aws_iam_role.redeploy_lambda.id
  policy = data.aws_iam_policy_document.redeploy_lambda.json
}

resource "aws_cloudwatch_log_group" "redeploy_lambda" {
  name              = "/aws/lambda/${local.name_prefix}-redeploy"
  retention_in_days = var.log_retention_days
}

resource "aws_lambda_function" "redeploy" {
  function_name    = "${local.name_prefix}-redeploy"
  role             = aws_iam_role.redeploy_lambda.arn
  filename         = data.archive_file.redeploy_lambda.output_path
  source_code_hash = data.archive_file.redeploy_lambda.output_base64sha256
  handler          = "redeploy.handler"
  runtime          = "python3.12"
  timeout          = 30
  memory_size      = 128

  environment {
    variables = {
      CLUSTER = aws_ecs_cluster.main.name
      # repository-name → service-name pairs, comma-separated
      REPO_TO_SERVICE = "${aws_ecr_repository.backend.name}=${aws_ecs_service.backend.name},${aws_ecr_repository.frontend.name}=${aws_ecs_service.frontend.name}"
    }
  }

  depends_on = [aws_cloudwatch_log_group.redeploy_lambda]
}

# EventBridge rule: ECR push for our two repos with the watched tag.
resource "aws_cloudwatch_event_rule" "ecr_push" {
  name        = "${local.name_prefix}-ecr-push"
  description = "Forward ECR push events for backend/frontend to redeploy lambda"
  event_pattern = jsonencode({
    source      = ["aws.ecr"]
    detail-type = ["ECR Image Action"]
    detail = {
      action-type     = ["PUSH"]
      result          = ["SUCCESS"]
      repository-name = [aws_ecr_repository.backend.name, aws_ecr_repository.frontend.name]
      image-tag       = [var.image_tag]
    }
  })
}

resource "aws_cloudwatch_event_target" "ecr_push_lambda" {
  rule      = aws_cloudwatch_event_rule.ecr_push.name
  target_id = "redeploy-lambda"
  arn       = aws_lambda_function.redeploy.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.redeploy.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.ecr_push.arn
}
