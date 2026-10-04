##############################################################################
# Video module — Video Worker Lambda + FFmpeg Layer + SQS trigger
##############################################################################

locals {
  name_prefix       = "${var.project}-${var.environment}"
  binary_path       = "${path.root}/../backend/build/video-worker/bootstrap"
  zip_output_path   = "${path.root}/../backend/build/video-worker.zip"
  layer_zip_path    = var.ffmpeg_layer_zip_path != "" ? var.ffmpeg_layer_zip_path : "${path.root}/../backend/build/ffmpeg-layer.zip"
}

##############################################################################
# Lambda package
##############################################################################

data "archive_file" "video_worker" {
  type        = "zip"
  source_file = local.binary_path
  output_path = local.zip_output_path
}

##############################################################################
# CloudWatch Log Group
##############################################################################

resource "aws_cloudwatch_log_group" "video_worker" {
  name              = "/aws/lambda/${local.name_prefix}-video-worker"
  retention_in_days = var.log_retention_days

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# FFmpeg Lambda Layer
##############################################################################

resource "aws_lambda_layer_version" "ffmpeg" {
  filename                 = local.layer_zip_path
  layer_name               = "${local.name_prefix}-ffmpeg"
  compatible_runtimes      = ["provided.al2023"]
  compatible_architectures = ["arm64"]
  description              = "FFmpeg static binary for ARM64"
  source_code_hash         = filebase64sha256(local.layer_zip_path)
}

##############################################################################
# IAM Role + Policy — video-worker Lambda (least privilege)
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

resource "aws_iam_role" "video_worker" {
  name               = "${local.name_prefix}-video-worker-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

data "aws_iam_policy_document" "video_worker" {
  # CloudWatch Logs
  statement {
    sid    = "Logs"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["${aws_cloudwatch_log_group.video_worker.arn}:*"]
  }

  # S3 — read from raw bucket, write to output bucket
  statement {
    sid     = "S3ReadRaw"
    effect  = "Allow"
    actions = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      var.raw_bucket_arn,
      "${var.raw_bucket_arn}/raw/*"
    ]
  }

  statement {
    sid     = "S3WriteOutput"
    effect  = "Allow"
    actions = ["s3:PutObject"]
    resources = ["${var.output_bucket_arn}/output/*"]
  }

  # DynamoDB — read and update job records
  statement {
    sid    = "DynamoDB"
    effect = "Allow"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:UpdateItem",
    ]
    resources = [var.dynamodb_table_arn]
  }

  # SQS — receive and delete messages
  statement {
    sid    = "SQSConsume"
    effect = "Allow"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
    ]
    resources = [var.video_queue_arn]
  }

  # AWS X-Ray tracing
  statement {
    sid    = "XRay"
    effect = "Allow"
    actions = [
      "xray:PutTraceSegments",
      "xray:PutTelemetryRecords",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "video_worker" {
  name   = "${local.name_prefix}-video-worker-policy"
  role   = aws_iam_role.video_worker.id
  policy = data.aws_iam_policy_document.video_worker.json
}

##############################################################################
# Lambda Function — video-worker
##############################################################################

resource "aws_lambda_function" "video_worker" {
  function_name = "${local.name_prefix}-video-worker"
  description   = "Consumes video SQS jobs, transcodes media via FFmpeg, updates job status"
  role          = aws_iam_role.video_worker.arn

  filename         = data.archive_file.video_worker.output_path
  source_code_hash = data.archive_file.video_worker.output_base64sha256

  runtime       = "provided.al2023"
  architectures = ["arm64"]
  handler       = "bootstrap"
  memory_size   = var.video_worker_memory_mb
  timeout       = var.video_worker_timeout_sec

  layers = [aws_lambda_layer_version.ffmpeg.arn]

  ephemeral_storage {
    size = var.video_worker_ephemeral_storage_mb
  }

  tracing_config {
    mode = "Active"
  }

  environment {
    variables = {
      RAW_BUCKET     = var.raw_bucket_name
      OUTPUT_BUCKET  = var.output_bucket_name
      DYNAMODB_TABLE = var.dynamodb_table_name
    }
  }

  depends_on = [
    aws_cloudwatch_log_group.video_worker,
    aws_iam_role_policy.video_worker,
    aws_lambda_layer_version.ffmpeg,
  ]

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# SQS → Lambda event source mapping
##############################################################################

resource "aws_lambda_event_source_mapping" "sqs_to_video_worker" {
  event_source_arn = var.video_queue_arn
  function_name    = aws_lambda_function.video_worker.arn
  batch_size       = var.sqs_batch_size
  enabled          = true

  function_response_types = ["ReportBatchItemFailures"]

  scaling_config {
    maximum_concurrency = 5
  }
}
