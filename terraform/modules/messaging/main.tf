##############################################################################
# Messaging module — SQS queues + DLQs for Images and Videos
##############################################################################

locals {
  name_prefix = "${var.project}-${var.environment}"
}

# --- Image Processing Queue & DLQ ---

resource "aws_sqs_queue" "image_dlq" {
  name                      = "${local.name_prefix}-image-dlq"
  message_retention_seconds = var.dlq_retention_seconds

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

resource "aws_sqs_queue" "image_queue" {
  name                       = "${local.name_prefix}-image-queue"
  visibility_timeout_seconds = var.visibility_timeout_seconds
  message_retention_seconds  = var.message_retention_seconds

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.image_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

# --- Video Processing Queue & DLQ ---

resource "aws_sqs_queue" "video_dlq" {
  name                      = "${local.name_prefix}-video-dlq"
  message_retention_seconds = var.dlq_retention_seconds

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

resource "aws_sqs_queue" "video_queue" {
  name                       = "${local.name_prefix}-video-queue"
  visibility_timeout_seconds = var.video_visibility_timeout_seconds
  message_retention_seconds  = var.message_retention_seconds

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.video_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}
