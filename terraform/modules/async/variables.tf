variable "project" {
  description = "Project name used in resource naming."
  type        = string
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)."
  type        = string
}

# Storage
variable "raw_bucket_name" {
  description = "Name of the raw S3 bucket (source for image-worker)."
  type        = string
}

variable "raw_bucket_arn" {
  description = "ARN of the raw S3 bucket."
  type        = string
}

variable "output_bucket_name" {
  description = "Name of the output S3 bucket (destination for processed media)."
  type        = string
}

variable "output_bucket_arn" {
  description = "ARN of the output S3 bucket."
  type        = string
}

# Database
variable "dynamodb_table_name" {
  description = "DynamoDB jobs table name."
  type        = string
}

variable "dynamodb_table_arn" {
  description = "DynamoDB jobs table ARN."
  type        = string
}

# Messaging
variable "queue_url" {
  description = "SQS media queue URL (QUEUE_URL env var for image-worker)."
  type        = string
}

variable "queue_arn" {
  description = "SQS media queue ARN (event source mapping + IAM)."
  type        = string
}

# Tuning
variable "image_worker_memory_mb" {
  description = "Memory allocated to the image-worker Lambda (MB)."
  type        = number
  default     = 256
}

variable "image_worker_timeout_sec" {
  description = "Timeout for the image-worker Lambda (seconds)."
  type        = number
  default     = 30
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days."
  type        = number
  default     = 14
}

variable "sqs_batch_size" {
  description = "Number of SQS messages per Lambda invocation."
  type        = number
  default     = 1
}
