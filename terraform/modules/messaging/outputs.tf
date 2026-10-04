# Backward-compatible outputs (defaulting to image queue)
output "queue_arn" {
  description = "ARN of the image SQS queue (backward compatibility)"
  value       = aws_sqs_queue.image_queue.arn
}

output "queue_url" {
  description = "URL of the image SQS queue (backward compatibility)"
  value       = aws_sqs_queue.image_queue.url
}

output "dlq_arn" {
  description = "ARN of the image DLQ (backward compatibility)"
  value       = aws_sqs_queue.image_dlq.arn
}

output "dlq_name" {
  description = "Name of the image DLQ (backward compatibility)"
  value       = aws_sqs_queue.image_dlq.name
}

# Image queue outputs
output "image_queue_arn" {
  description = "ARN of the image processing queue"
  value       = aws_sqs_queue.image_queue.arn
}

output "image_queue_url" {
  description = "URL of the image processing queue"
  value       = aws_sqs_queue.image_queue.url
}

output "image_queue_name" {
  description = "Name of the image processing queue"
  value       = aws_sqs_queue.image_queue.name
}

output "image_dlq_arn" {
  description = "ARN of the image dead-letter queue"
  value       = aws_sqs_queue.image_dlq.arn
}

output "image_dlq_name" {
  description = "Name of the image dead-letter queue"
  value       = aws_sqs_queue.image_dlq.name
}

output "image_dlq_url" {
  description = "URL of the image dead-letter queue"
  value       = aws_sqs_queue.image_dlq.url
}

# Video queue outputs
output "video_queue_arn" {
  description = "ARN of the video processing queue"
  value       = aws_sqs_queue.video_queue.arn
}

output "video_queue_url" {
  description = "URL of the video processing queue"
  value       = aws_sqs_queue.video_queue.url
}

output "video_queue_name" {
  description = "Name of the video processing queue"
  value       = aws_sqs_queue.video_queue.name
}

output "video_dlq_arn" {
  description = "ARN of the video dead-letter queue"
  value       = aws_sqs_queue.video_dlq.arn
}

output "video_dlq_name" {
  description = "Name of the video dead-letter queue"
  value       = aws_sqs_queue.video_dlq.name
}

output "video_dlq_url" {
  description = "URL of the video dead-letter queue"
  value       = aws_sqs_queue.video_dlq.url
}
