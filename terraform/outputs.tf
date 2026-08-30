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
