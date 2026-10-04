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
  description = "Name of the raw S3 bucket."
  type        = string
}

variable "raw_bucket_arn" {
  description = "ARN of the raw S3 bucket."
  type        = string
}

variable "output_bucket_name" {
  description = "Name of the output S3 bucket."
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
variable "video_queue_url" {
  description = "SQS video processing queue URL."
  type        = string
}

variable "video_queue_arn" {
  description = "SQS video processing queue ARN."
  type        = string
}

# Layer
variable "ffmpeg_layer_zip_path" {
  description = "Path to the pre-packaged FFmpeg layer zip."
  type        = string
  default     = ""
}

# Tuning
variable "video_worker_memory_mb" {
  description = "Memory allocated to video-worker Lambda (MB)."
  type        = number
  default     = 1024
}

variable "video_worker_timeout_sec" {
  description = "Timeout for video-worker Lambda (seconds)."
  type        = number
  default     = 300
}

variable "video_worker_ephemeral_storage_mb" {
  description = "Ephemeral storage (/tmp) for video transcoding (MB)."
  type        = number
  default     = 2048
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
