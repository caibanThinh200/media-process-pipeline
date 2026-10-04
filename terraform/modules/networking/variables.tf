###############################################################################
# Networking module — variables
###############################################################################

variable "project" {
  description = "Project name prefix (e.g. media-pipeline)"
  type        = string
}

variable "environment" {
  description = "Deployment environment (dev, staging, prod)"
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "az_count" {
  description = "Number of Availability Zones to spread subnets across (1–3)"
  type        = number
  default     = 2

  validation {
    condition     = var.az_count >= 1 && var.az_count <= 3
    error_message = "az_count must be between 1 and 3."
  }
}

variable "upload_lambda_arn" {
  description = "ARN of the upload-api Lambda to register as an ALB Lambda target"
  type        = string
}

variable "certificate_arn" {
  description = "ACM certificate ARN for HTTPS. Leave empty to use HTTP only (dev/test)."
  type        = string
  default     = ""
}
