variable "project" {
  description = "Project name prefix for all resources"
  type        = string
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
}

variable "raw_bucket_force_destroy" {
  description = "Allow destroying raw bucket even when non-empty (useful for dev)"
  type        = bool
  default     = false
}

variable "output_bucket_force_destroy" {
  description = "Allow destroying output bucket even when non-empty (useful for dev)"
  type        = bool
  default     = false
}

variable "raw_expiry_days" {
  description = "Days before raw upload objects are deleted by lifecycle rule"
  type        = number
  default     = 7
}

variable "cloudfront_price_class" {
  description = "CloudFront price class (PriceClass_100 = US/EU, PriceClass_All = global)"
  type        = string
  default     = "PriceClass_100"
}

variable "cors_allowed_origins" {
  description = "Origins allowed to PUT to the raw bucket via presigned URLs"
  type        = list(string)
  default     = ["*"]
}
