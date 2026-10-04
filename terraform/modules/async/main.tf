##############################################################################
# Async module — Image Worker Lambda + SQS trigger + S3 event notification
#
# Lambda runtime : provided.al2023 (custom runtime for Go)
# Architecture   : arm64 (Graviton2)
# Binary path    : ../../backend/build/image-worker/bootstrap
#                  Built with: make build-image-worker (from backend/)
#
# Pipeline flow  : raw S3 PUT → S3 event notification → SQS → Lambda
#                  (upload-api also enqueues directly for immediate processing)
##############################################################################

locals {
  name_prefix     = "${var.project}-${var.environment}"
  binary_path     = "${path.root}/../backend/build/image-worker/bootstrap"
  zip_output_path = "${path.root}/../backend/build/image-worker.zip"
}

##############################################################################
# Lambda package
##############################################################################

data "archive_file" "image_worker" {
  type        = "zip"
  source_file = local.binary_path
  output_path = local.zip_output_path
}

##############################################################################
# CloudWatch Log Group
##############################################################################

resource "aws_cloudwatch_log_group" "image_worker" {
  name              = "/aws/lambda/${local.name_prefix}-image-worker"
  retention_in_days = var.log_retention_days

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# IAM Role + Policy — image-worker Lambda (least privilege)
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

resource "aws_iam_role" "image_worker" {
  name               = "${local.name_prefix}-image-worker-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

data "aws_iam_policy_document" "image_worker" {
  # CloudWatch Logs
  statement {
    sid    = "Logs"
    effect = "Allow"
    actions = [
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["${aws_cloudwatch_log_group.image_worker.arn}:*"]
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

  # SQS — receive and delete messages (required for event source mapping)
  statement {
    sid    = "SQSConsume"
    effect = "Allow"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
    ]
    resources = [var.queue_arn]
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

resource "aws_iam_role_policy" "image_worker" {
  name   = "${local.name_prefix}-image-worker-policy"
  role   = aws_iam_role.image_worker.id
  policy = data.aws_iam_policy_document.image_worker.json
}

##############################################################################
# Lambda Function — image-worker
##############################################################################

resource "aws_lambda_function" "image_worker" {
  function_name = "${local.name_prefix}-image-worker"
  description   = "Consumes SQS jobs, copies raw media to output bucket, updates job status"
  role          = aws_iam_role.image_worker.arn

  filename         = data.archive_file.image_worker.output_path
  source_code_hash = data.archive_file.image_worker.output_base64sha256

  runtime       = "provided.al2023"
  architectures = ["arm64"]
  handler       = "bootstrap"
  memory_size   = var.image_worker_memory_mb
  timeout       = var.image_worker_timeout_sec

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
    aws_cloudwatch_log_group.image_worker,
    aws_iam_role_policy.image_worker,
  ]

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# SQS → Lambda event source mapping
#
# batch_size=1         — process one job per invocation (simpler error isolation)
# bisect_on_error=true — on batch failure, split and retry halves individually
##############################################################################

resource "aws_lambda_event_source_mapping" "sqs_to_image_worker" {
  event_source_arn = var.queue_arn
  function_name    = aws_lambda_function.image_worker.arn
  batch_size       = var.sqs_batch_size
  enabled          = true

  function_response_types = ["ReportBatchItemFailures"]

  scaling_config {
    maximum_concurrency = 5
  }
}

##############################################################################
# S3 → SQS event notifications (raw/images/* -> image queue, raw/videos/* -> video queue)
##############################################################################

data "aws_iam_policy_document" "sqs_s3_notification" {
  statement {
    sid    = "AllowS3ToSendMessage"
    effect = "Allow"
    principals {
      type        = "Service"
      identifiers = ["s3.amazonaws.com"]
    }
    actions   = ["sqs:SendMessage"]
    resources = [var.queue_arn, var.video_queue_arn]
    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = [var.raw_bucket_arn]
    }
  }
}

resource "aws_sqs_queue_policy" "s3_notification" {
  queue_url = var.queue_url
  policy    = data.aws_iam_policy_document.sqs_s3_notification.json
}

resource "aws_sqs_queue_policy" "video_s3_notification" {
  queue_url = var.video_queue_url
  policy    = data.aws_iam_policy_document.sqs_s3_notification.json
}

resource "aws_s3_bucket_notification" "raw_to_sqs" {
  bucket = var.raw_bucket_name

  queue {
    id            = "raw-images-to-sqs"
    queue_arn     = var.queue_arn
    events        = ["s3:ObjectCreated:Put"]
    filter_prefix = "raw/images/"
  }

  queue {
    id            = "raw-videos-to-sqs"
    queue_arn     = var.video_queue_arn
    events        = ["s3:ObjectCreated:Put"]
    filter_prefix = "raw/videos/"
  }

  depends_on = [
    aws_sqs_queue_policy.s3_notification,
    aws_sqs_queue_policy.video_s3_notification
  ]
}
