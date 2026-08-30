###############################################################################
# Core
###############################################################################

variable "project" {
  description = "Project name used as a prefix for all resources"
  type        = string
  default     = "media-pipeline"
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "aws_region" {
  description = "AWS region to deploy resources"
  type        = string
  default     = "ap-southeast-1"
}

variable "aws_profile" {
  description = "AWS CLI named profile to use for authentication. Leave null to use env vars (AWS_ACCESS_KEY_ID etc.) or instance role."
  type        = string
  default     = null
}

###############################################################################
# Phase 1 — Storage: S3 + CloudFront
###############################################################################

variable "raw_bucket_force_destroy" {
  description = "Destroy raw bucket even when non-empty — set true only for dev/test"
  type        = bool
  default     = false
}

variable "output_bucket_force_destroy" {
  description = "Destroy output bucket even when non-empty — set true only for dev/test"
  type        = bool
  default     = false
}

variable "raw_expiry_days" {
  description = "Days before raw upload objects are automatically deleted"
  type        = number
  default     = 7
}

variable "cloudfront_price_class" {
  description = "CloudFront price class — PriceClass_100 (US/EU), PriceClass_200 (+Asia), PriceClass_All (global)"
  type        = string
  default     = "PriceClass_100"

  validation {
    condition     = contains(["PriceClass_100", "PriceClass_200", "PriceClass_All"], var.cloudfront_price_class)
    error_message = "Must be one of: PriceClass_100, PriceClass_200, PriceClass_All."
  }
}

variable "cors_allowed_origins" {
  description = "Origins permitted to PUT objects to the raw bucket via presigned URLs"
  type        = list(string)
  default     = ["*"]
}

###############################################################################
# Phase 1 — Database: DynamoDB
###############################################################################

variable "enable_pitr" {
  description = "Enable Point-in-Time Recovery on the jobs DynamoDB table"
  type        = bool
  default     = true
}

variable "job_ttl_days" {
  description = "Days after job creation before the DynamoDB record expires"
  type        = number
  default     = 30
}

###############################################################################
# Phase 2 — Compute: Lambda + API Gateway
###############################################################################

variable "upload_api_memory_mb" {
  description = "Lambda memory for the upload-api in MB (128–10240)"
  type        = number
  default     = 256
}

variable "upload_api_timeout_sec" {
  description = "Lambda timeout for the upload-api in seconds (max 900)"
  type        = number
  default     = 30
}

variable "max_upload_bytes" {
  description = "Maximum file size the upload-api accepts, in bytes (default 500 MB)"
  type        = number
  default     = 524288000
}

variable "log_retention_days" {
  description = "CloudWatch log group retention for Lambda and API Gateway logs"
  type        = number
  default     = 14
}

###############################################################################
# Phase 3 — Async: Image Worker Lambda
###############################################################################

variable "image_worker_memory_mb" {
  description = "Lambda memory for the image-worker in MB (128–10240)"
  type        = number
  default     = 256
}

variable "image_worker_timeout_sec" {
  description = "Lambda timeout for the image-worker in seconds (max 900)"
  type        = number
  default     = 30
}
