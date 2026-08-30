output "queue_arn" {
  description = "ARN of the SQS processing queue (used in Lambda event source mapping)"
  value       = aws_sqs_queue.media_queue.arn
}

output "queue_url" {
  description = "URL of the SQS processing queue (used by upload-api Lambda env var)"
  value       = aws_sqs_queue.media_queue.url
}

output "dlq_arn" {
  description = "ARN of the dead-letter queue (used in CloudWatch alarms in Phase 6)"
  value       = aws_sqs_queue.media_dlq.arn
}

output "dlq_name" {
  description = "Name of the dead-letter queue"
  value       = aws_sqs_queue.media_dlq.name
}
