##############################################################################
# Compute module — Upload API Lambda + API Gateway HTTP API
#
# Lambda runtime : provided.al2023 (custom runtime for Go)
# Architecture   : arm64 (Graviton2 — ~20% cheaper than x86_64)
# Binary path    : ../../backend/build/upload-api/bootstrap
#                  Built with: make build-upload-api (from backend/)
#
# API Gateway    : HTTP API (v2) — cheaper and simpler than REST API (v1)
#                  Routes:
#                    POST /upload/presign  → handlePresign
#                    GET  /jobs/{jobId}    → handleGetJob
#                    GET  /jobs            → handleListJobs (dev only)
##############################################################################

locals {
  name_prefix      = "${var.project}-${var.environment}"
  binary_path      = "${path.root}/../backend/build/upload-api/bootstrap"
  zip_output_path  = "${path.root}/../backend/build/upload-api.zip"
}

##############################################################################
# Lambda package — zip the pre-built binary
##############################################################################

data "archive_file" "upload_api" {
  type        = "zip"
  source_file = local.binary_path
  output_path = local.zip_output_path
}

##############################################################################
# CloudWatch Log Group
##############################################################################

resource "aws_cloudwatch_log_group" "upload_api" {
  name              = "/aws/lambda/${local.name_prefix}-upload-api"
  retention_in_days = var.log_retention_days

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# IAM Role + Policy — upload-api Lambda
##############################################################################

data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "upload_api" {
  name               = "${local.name_prefix}-upload-api-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

data "aws_iam_policy_document" "upload_api" {
  # CloudWatch Logs
  statement {
    sid    = "Logs"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["${aws_cloudwatch_log_group.upload_api.arn}:*"]
  }

  # S3 — generate presigned PUT URLs for the raw bucket only
  statement {
    sid     = "S3PresignRaw"
    effect  = "Allow"
    actions = ["s3:PutObject"]
    resources = ["${var.raw_bucket_arn}/raw/*"]
  }

  # DynamoDB — create and read job records
  statement {
    sid    = "DynamoDB"
    effect = "Allow"
    actions = [
      "dynamodb:PutItem",
      "dynamodb:GetItem",
    ]
    resources = [var.dynamodb_table_arn]
  }

  # SQS — send job messages
  statement {
    sid     = "SQSSend"
    effect  = "Allow"
    actions = ["sqs:SendMessage"]
    resources = [var.queue_arn]
  }
}

resource "aws_iam_role_policy" "upload_api" {
  name   = "${local.name_prefix}-upload-api-policy"
  role   = aws_iam_role.upload_api.id
  policy = data.aws_iam_policy_document.upload_api.json
}

##############################################################################
# Lambda Function — upload-api
##############################################################################

resource "aws_lambda_function" "upload_api" {
  function_name = "${local.name_prefix}-upload-api"
  description   = "Issues presigned S3 upload URLs and creates DynamoDB job records"
  role          = aws_iam_role.upload_api.arn

  filename         = data.archive_file.upload_api.output_path
  source_code_hash = data.archive_file.upload_api.output_base64sha256

  runtime       = "provided.al2023"
  architectures = ["arm64"]
  handler       = "bootstrap"
  memory_size   = var.upload_api_memory_mb
  timeout       = var.upload_api_timeout_sec

  environment {
    variables = {
      RAW_BUCKET        = var.raw_bucket_name
      OUTPUT_BUCKET     = var.output_bucket_name
      DYNAMODB_TABLE    = var.dynamodb_table_name
      QUEUE_URL         = var.queue_url
      MAX_UPLOAD_BYTES  = tostring(var.max_upload_bytes)
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.upload_api,
    aws_iam_role_policy.upload_api,
  ]

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# API Gateway HTTP API (v2)
##############################################################################

resource "aws_apigatewayv2_api" "http" {
  name          = "${local.name_prefix}-api"
  protocol_type = "HTTP"
  description   = "Media pipeline HTTP API"

  cors_configuration {
    allow_origins = ["*"] # tighten to frontend origin in staging/prod
    allow_methods = ["GET", "POST", "OPTIONS"]
    allow_headers = ["Content-Type", "Authorization"]
    max_age       = 300
  }

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

resource "aws_cloudwatch_log_group" "api_gateway" {
  name              = "/aws/apigateway/${local.name_prefix}-api"
  retention_in_days = var.log_retention_days
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gateway.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      routeKey       = "$context.routeKey"
      status         = "$context.status"
      responseLength = "$context.responseLength"
      durationMs     = "$context.responseLatency"
      sourceIp       = "$context.identity.sourceIp"
      errorMessage   = "$context.error.message"
    })
  }
}

resource "aws_apigatewayv2_integration" "upload_api" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.upload_api.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "presign" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "POST /upload/presign"
  target    = "integrations/${aws_apigatewayv2_integration.upload_api.id}"
}

resource "aws_apigatewayv2_route" "get_job" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /jobs/{jobId}"
  target    = "integrations/${aws_apigatewayv2_integration.upload_api.id}"
}

resource "aws_apigatewayv2_route" "list_jobs" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "GET /jobs"
  target    = "integrations/${aws_apigatewayv2_integration.upload_api.id}"
}

# Allow API Gateway to invoke the Lambda
resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.upload_api.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}
