##############################################################################
# Observability module — CloudWatch Alarms, X-Ray Group, Dashboard
##############################################################################

locals {
  name_prefix = "${var.project}-${var.environment}"
}

##############################################################################
# CloudWatch Metric Alarms — DLQ Backlog Detection
# Triggers immediately if any message lands in the Dead-Letter Queue
##############################################################################

resource "aws_cloudwatch_metric_alarm" "image_dlq" {
  alarm_name          = "${local.name_prefix}-image-dlq-messages"
  alarm_description   = "Triggers when a message lands in the image processing Dead-Letter Queue"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 60
  statistic           = "Sum"
  threshold           = 0
  treat_missing_data  = "notBreaching"

  dimensions = {
    QueueName = var.image_dlq_name
  }

  alarm_actions = var.alarm_topic_arn != "" ? [var.alarm_topic_arn] : []

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

resource "aws_cloudwatch_metric_alarm" "video_dlq" {
  alarm_name          = "${local.name_prefix}-video-dlq-messages"
  alarm_description   = "Triggers when a message lands in the video processing Dead-Letter Queue"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 60
  statistic           = "Sum"
  threshold           = 0
  treat_missing_data  = "notBreaching"

  dimensions = {
    QueueName = var.video_dlq_name
  }

  alarm_actions = var.alarm_topic_arn != "" ? [var.alarm_topic_arn] : []

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# AWS X-Ray Group
##############################################################################

resource "aws_xray_group" "pipeline" {
  group_name        = "${local.name_prefix}-pipeline"
  filter_expression = "service(\"${var.upload_api_function_name}\") OR service(\"${var.image_worker_function_name}\") OR service(\"${var.video_worker_function_name}\")"

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# CloudWatch Dashboard — Centralized Pipeline Overview
##############################################################################

resource "aws_cloudwatch_dashboard" "pipeline" {
  dashboard_name = "${local.name_prefix}-pipeline-overview"

  dashboard_body = jsonencode({
    widgets = [
      # Widget 1: Lambda Invocations & Errors
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6
        properties = {
          metrics = [
            ["AWS/Lambda", "Invocations", "FunctionName", var.upload_api_function_name, { stat = "Sum", label = "Upload API Invocations" }],
            [".", "Errors", ".", ".", { stat = "Sum", label = "Upload API Errors", color = "#d62728" }],
            [".", "Invocations", "FunctionName", var.image_worker_function_name, { stat = "Sum", label = "Image Worker Invocations" }],
            [".", "Errors", ".", ".", { stat = "Sum", label = "Image Worker Errors", color = "#ff7f0e" }],
            [".", "Invocations", "FunctionName", var.video_worker_function_name, { stat = "Sum", label = "Video Worker Invocations" }],
            [".", "Errors", ".", ".", { stat = "Sum", label = "Video Worker Errors", color = "#e377c2" }]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "Lambda Invocations & Errors"
          period  = 60
        }
      },
      # Widget 2: Lambda Latency / Duration
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 12
        height = 6
        properties = {
          metrics = [
            ["AWS/Lambda", "Duration", "FunctionName", var.upload_api_function_name, { stat = "Average", label = "Upload API Avg (ms)" }],
            [".", ".", ".", var.image_worker_function_name, { stat = "Average", label = "Image Worker Avg (ms)" }],
            [".", ".", ".", var.video_worker_function_name, { stat = "Average", label = "Video Worker Avg (ms)" }],
            [".", ".", ".", var.video_worker_function_name, { stat = "p95", label = "Video Worker p95 (ms)", color = "#8c564b" }]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "Lambda Duration (Avg & p95)"
          period  = 60
        }
      },
      # Widget 3: SQS Queue Depth & Messages Sent
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 12
        height = 6
        properties = {
          metrics = [
            ["AWS/SQS", "ApproximateNumberOfMessagesVisible", "QueueName", var.image_queue_name, { stat = "Average", label = "Image Queue Backlog" }],
            [".", "NumberOfMessagesSent", ".", ".", { stat = "Sum", label = "Image Messages Sent" }],
            [".", "ApproximateNumberOfMessagesVisible", "QueueName", var.video_queue_name, { stat = "Average", label = "Video Queue Backlog" }],
            [".", "NumberOfMessagesSent", ".", ".", { stat = "Sum", label = "Video Messages Sent" }]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "SQS Queue Backlog & Throughput"
          period  = 60
        }
      },
      # Widget 4: Dead-Letter Queue Depth
      {
        type   = "metric"
        x      = 12
        y      = 6
        width  = 12
        height = 6
        properties = {
          metrics = [
            ["AWS/SQS", "ApproximateNumberOfMessagesVisible", "QueueName", var.image_dlq_name, { stat = "Sum", label = "Image DLQ Messages", color = "#d62728" }],
            [".", "ApproximateNumberOfMessagesVisible", "QueueName", var.video_dlq_name, { stat = "Sum", label = "Video DLQ Messages", color = "#ff7f0e" }]
          ]
          view    = "timeSeries"
          stacked = false
          region  = var.aws_region
          title   = "Dead-Letter Queue (DLQ) Depth"
          period  = 60
        }
      }
    ]
  })
}
