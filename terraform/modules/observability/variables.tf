variable "project" {
  description = "Project name prefix"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-southeast-1"
}

variable "upload_api_function_name" {
  description = "Name of the upload-api Lambda function"
  type        = string
}

variable "image_worker_function_name" {
  description = "Name of the image-worker Lambda function"
  type        = string
}

variable "video_worker_function_name" {
  description = "Name of the video-worker Lambda function"
  type        = string
}

variable "image_queue_name" {
  description = "Name of the image SQS queue"
  type        = string
}

variable "video_queue_name" {
  description = "Name of the video SQS queue"
  type        = string
}

variable "image_dlq_name" {
  description = "Name of the image dead-letter queue"
  type        = string
}

variable "video_dlq_name" {
  description = "Name of the video dead-letter queue"
  type        = string
}

variable "alarm_topic_arn" {
  description = "Optional SNS topic ARN for alarm notifications"
  type        = string
  default     = ""
}
