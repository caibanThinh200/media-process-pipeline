terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "~> 2.0"
    }
  }

  # Uncomment and configure when you have a remote state bucket.
  # backend "s3" {
  #   bucket = "your-terraform-state-bucket"
  #   key    = "media-pipeline/terraform.tfstate"
  #   region = "ap-southeast-1"
  # }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

##############################################################################
# Phase 1 — Storage Foundation
##############################################################################

module "storage" {
  source      = "./modules/storage"
  project     = var.project
  environment = var.environment

  raw_bucket_force_destroy    = var.raw_bucket_force_destroy
  output_bucket_force_destroy = var.output_bucket_force_destroy
  raw_expiry_days             = var.raw_expiry_days
  cloudfront_price_class      = var.cloudfront_price_class
  cors_allowed_origins        = var.cors_allowed_origins
}

module "database" {
  source      = "./modules/database"
  project     = var.project
  environment = var.environment

  enable_pitr  = var.enable_pitr
  job_ttl_days = var.job_ttl_days
}

##############################################################################
# Phase 2 — Upload Flow
##############################################################################

module "messaging" {
  source      = "./modules/messaging"
  project     = var.project
  environment = var.environment
}

module "compute" {
  source      = "./modules/compute"
  project     = var.project
  environment = var.environment

  # Storage
  raw_bucket_name    = module.storage.raw_bucket_name
  raw_bucket_arn     = module.storage.raw_bucket_arn
  output_bucket_name = module.storage.output_bucket_name

  # Database
  dynamodb_table_name = module.database.jobs_table_name
  dynamodb_table_arn  = module.database.jobs_table_arn

  # Messaging
  queue_url = module.messaging.queue_url
  queue_arn = module.messaging.queue_arn

  # Tuning
  upload_api_memory_mb   = var.upload_api_memory_mb
  upload_api_timeout_sec = var.upload_api_timeout_sec
  max_upload_bytes       = var.max_upload_bytes
  log_retention_days     = var.log_retention_days
}

##############################################################################
# Phase 3 — Async Pipeline
##############################################################################

module "async" {
  source      = "./modules/async"
  project     = var.project
  environment = var.environment

  # Storage
  raw_bucket_name    = module.storage.raw_bucket_name
  raw_bucket_arn     = module.storage.raw_bucket_arn
  output_bucket_name = module.storage.output_bucket_name
  output_bucket_arn  = module.storage.output_bucket_arn

  # Database
  dynamodb_table_name = module.database.jobs_table_name
  dynamodb_table_arn  = module.database.jobs_table_arn

  # Messaging
  queue_url       = module.messaging.image_queue_url
  queue_arn       = module.messaging.image_queue_arn
  video_queue_url = module.messaging.video_queue_url
  video_queue_arn = module.messaging.video_queue_arn

  # Tuning
  image_worker_memory_mb   = var.image_worker_memory_mb
  image_worker_timeout_sec = var.image_worker_timeout_sec
  log_retention_days       = var.log_retention_days
}

##############################################################################
# Phase 5 — Networking (VPC + ALB → Lambda target group)
##############################################################################

module "networking" {
  source      = "./modules/networking"
  project     = var.project
  environment = var.environment

  vpc_cidr          = var.vpc_cidr
  az_count          = var.az_count
  upload_lambda_arn = module.compute.upload_api_lambda_arn
  certificate_arn   = var.certificate_arn

  depends_on = [module.compute]
}

##############################################################################
# Phase 6 — Video Worker & Observability
##############################################################################

module "video" {
  source      = "./modules/video"
  project     = var.project
  environment = var.environment

  # Storage
  raw_bucket_name    = module.storage.raw_bucket_name
  raw_bucket_arn     = module.storage.raw_bucket_arn
  output_bucket_name = module.storage.output_bucket_name
  output_bucket_arn  = module.storage.output_bucket_arn

  # Database
  dynamodb_table_name = module.database.jobs_table_name
  dynamodb_table_arn  = module.database.jobs_table_arn

  # Messaging
  video_queue_url = module.messaging.video_queue_url
  video_queue_arn = module.messaging.video_queue_arn

  # Tuning
  video_worker_memory_mb            = var.video_worker_memory_mb
  video_worker_timeout_sec          = var.video_worker_timeout_sec
  video_worker_ephemeral_storage_mb = var.video_worker_ephemeral_storage_mb
  log_retention_days                = var.log_retention_days
}

module "observability" {
  source      = "./modules/observability"
  project     = var.project
  environment = var.environment
  aws_region  = var.aws_region

  # Lambdas
  upload_api_function_name   = module.compute.upload_api_function_name
  image_worker_function_name = module.async.image_worker_function_name
  video_worker_function_name = module.video.video_worker_function_name

  # Queues & DLQs
  image_queue_name = module.messaging.image_queue_name
  video_queue_name = module.messaging.video_queue_name
  image_dlq_name   = module.messaging.image_dlq_name
  video_dlq_name   = module.messaging.video_dlq_name

  # Notifications
  alarm_topic_arn = var.alarm_topic_arn
}
