variable "project" {
  description = "Project name prefix"
  type        = string
}

variable "environment" {
  description = "Deployment environment"
  type        = string
}

variable "enable_pitr" {
  description = "Enable Point-in-Time Recovery for the jobs table"
  type        = bool
  default     = true
}

variable "job_ttl_days" {
  description = "Days after job creation before the record expires (TTL)"
  type        = number
  default     = 30
}
