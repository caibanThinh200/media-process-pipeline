###############################################################################
# Phase 1 — Storage outputs
###############################################################################

output "raw_bucket_name" {
  description = "S3 bucket name for raw uploads (used by upload-api presigner)"
  value       = module.storage.raw_bucket_name
}

output "raw_bucket_arn" {
  description = "ARN of the raw S3 bucket"
  value       = module.storage.raw_bucket_arn
}

output "output_bucket_name" {
  description = "S3 bucket name for processed media (used by workers)"
  value       = module.storage.output_bucket_name
}

output "output_bucket_arn" {
  description = "ARN of the processed-media S3 bucket"
  value       = module.storage.output_bucket_arn
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID (used for cache invalidations)"
  value       = module.storage.cloudfront_distribution_id
}

output "cloudfront_domain_name" {
  description = "CloudFront domain name — base URL for processed asset delivery"
  value       = module.storage.cloudfront_domain_name
}

###############################################################################
# Phase 1 — Database outputs
###############################################################################

output "jobs_table_name" {
  description = "DynamoDB jobs table name (DYNAMODB_TABLE env var)"
  value       = module.database.jobs_table_name
}

output "jobs_table_arn" {
  description = "DynamoDB jobs table ARN (used in Lambda IAM policies)"
  value       = module.database.jobs_table_arn
}

###############################################################################
# Phase 2 — Messaging outputs
###############################################################################

output "sqs_queue_url" {
  description = "SQS media queue URL (QUEUE_URL env var for upload-api Lambda)"
  value       = module.messaging.queue_url
}

output "sqs_queue_arn" {
  description = "SQS media queue ARN"
  value       = module.messaging.queue_arn
}

output "sqs_dlq_arn" {
  description = "SQS dead-letter queue ARN"
  value       = module.messaging.dlq_arn
}

output "sqs_dlq_name" {
  description = "SQS dead-letter queue name"
  value       = module.messaging.dlq_name
}


###############################################################################
# Phase 2 — Compute outputs
###############################################################################

output "api_endpoint" {
  description = "HTTP API invoke URL — set as API_BASE_URL in the frontend .env"
  value       = module.compute.api_endpoint
}

output "upload_api_function_name" {
  description = "Upload API Lambda function name (for manual invocations and CI deploys)"
  value       = module.compute.upload_api_function_name
}

###############################################################################
# Phase 3 — Async outputs
###############################################################################

output "image_worker_function_name" {
  description = "Image-worker Lambda function name"
  value       = module.async.image_worker_function_name
}

###############################################################################
# Phase 5 — Networking outputs
###############################################################################

output "alb_dns_name" {
  description = "ALB DNS name — use as API_BASE_URL in staging/prod (replaces direct APIGW URL)"
  value       = try(module.networking[0].alb_dns_name, null)
}

output "vpc_id" {
  description = "VPC ID created by the networking module"
  value       = try(module.networking[0].vpc_id, null)
}

output "alb_arn" {
  description = "ARN of the Application Load Balancer"
  value       = try(module.networking[0].alb_arn, null)
}

###############################################################################
# Phase 6 — Video Worker & Observability outputs
###############################################################################

output "video_worker_function_name" {
  description = "Video-worker Lambda function name"
  value       = module.video.video_worker_function_name
}

output "video_queue_url" {
  description = "URL of the SQS video processing queue"
  value       = module.messaging.video_queue_url
}

output "video_dlq_url" {
  description = "URL of the video dead-letter queue"
  value       = module.messaging.video_dlq_url
}

output "cloudwatch_dashboard_url" {
  description = "Direct URL to the CloudWatch overview dashboard"
  value       = module.observability.dashboard_url
}

output "image_dlq_alarm_arn" {
  description = "ARN of the CloudWatch alarm for image DLQ"
  value       = module.observability.image_dlq_alarm_arn
}

output "video_dlq_alarm_arn" {
  description = "ARN of the CloudWatch alarm for video DLQ"
  value       = module.observability.video_dlq_alarm_arn
}

output "xray_group_arn" {
  description = "ARN of the AWS X-Ray group"
  value       = module.observability.xray_group_arn
}

