##############################################################################
# Messaging module — SQS queue + DLQ
##############################################################################

locals {
  name_prefix = "${var.project}-${var.environment}"
}

resource "aws_sqs_queue" "media_dlq" {
  name                      = "${local.name_prefix}-media-dlq"
  message_retention_seconds = var.dlq_retention_seconds

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

resource "aws_sqs_queue" "media_queue" {
  name                       = "${local.name_prefix}-media-queue"
  visibility_timeout_seconds = var.visibility_timeout_seconds
  message_retention_seconds  = var.message_retention_seconds

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.media_dlq.arn
    maxReceiveCount     = var.max_receive_count
  })

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}
