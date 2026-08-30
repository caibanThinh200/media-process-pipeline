variable "project" {
  description = "Project name prefix"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

variable "raw_bucket_name" {
  description = "Name of the S3 raw uploads bucket"
  type        = string
}

variable "raw_bucket_arn" {
  description = "ARN of the S3 raw uploads bucket (used in IAM policy)"
  type        = string
}

variable "output_bucket_name" {
  description = "Name of the S3 output bucket"
  type        = string
}

variable "dynamodb_table_name" {
  description = "DynamoDB jobs table name"
  type        = string
}

variable "dynamodb_table_arn" {
  description = "DynamoDB jobs table ARN (used in IAM policy)"
  type        = string
}

variable "queue_url" {
  description = "SQS queue URL (injected as Lambda env var)"
  type        = string
}

variable "queue_arn" {
  description = "SQS queue ARN (used in IAM policy)"
  type        = string
}

variable "upload_api_memory_mb" {
  description = "Lambda memory allocation for the upload-api in MB"
  type        = number
  default     = 256
}

variable "upload_api_timeout_sec" {
  description = "Lambda timeout for the upload-api in seconds"
  type        = number
  default     = 30
}

variable "max_upload_bytes" {
  description = "Maximum allowed upload size in bytes (enforced in Go handler)"
  type        = number
  default     = 524288000 # 500 MB
}

variable "log_retention_days" {
  description = "CloudWatch log group retention in days"
  type        = number
  default     = 14
}
