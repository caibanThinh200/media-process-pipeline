variable "project" {
  description = "Project name prefix"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

variable "visibility_timeout_seconds" {
  description = "SQS visibility timeout — must be >= the worker Lambda timeout"
  type        = number
  default     = 300
}

variable "message_retention_seconds" {
  description = "How long messages are retained in the processing queue"
  type        = number
  default     = 86400 # 1 day
}

variable "dlq_retention_seconds" {
  description = "How long messages are retained in the DLQ"
  type        = number
  default     = 1209600 # 14 days
}

variable "max_receive_count" {
  description = "Number of times a message is delivered before being moved to the DLQ"
  type        = number
  default     = 3
}
