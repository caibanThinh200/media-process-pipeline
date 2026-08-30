##############################################################################
# Database module — DynamoDB jobs table
#
# Primary key : jobId (String) — stable, globally unique identifier set by
#               the upload-api at job creation.
#
# GSI 1 — status-createdAt-index
#   Partition key : status   (query all jobs in a given state)
#   Sort key      : createdAt (scan in time order within a status bucket)
#   This replaces the initial status-only index so callers can paginate
#   results without a full table scan.
#
# TTL — expiresAt (Unix epoch seconds).  Workers write this on job
#        creation; DynamoDB removes expired records automatically so
#        the table doesn't accumulate stale entries indefinitely.
#
# PITR — enabled by default so completed job records can be restored if
#         the table is accidentally mutated or corrupted.
##############################################################################

locals {
  table_name = "${var.project}-${var.environment}-jobs"
}

resource "aws_dynamodb_table" "jobs" {
  name         = local.table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "jobId"

  # ── Attributes required by the table or a GSI ──────────────────────────
  attribute {
    name = "jobId"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  attribute {
    name = "createdAt"
    type = "S"
  }

  # ── Global Secondary Index ──────────────────────────────────────────────
  # Query: "give me all PROCESSING jobs ordered by creation time"
  global_secondary_index {
    name            = "status-createdAt-index"
    hash_key        = "status"
    range_key       = "createdAt"
    projection_type = "ALL"
  }

  # ── TTL ────────────────────────────────────────────────────────────────
  ttl {
    attribute_name = "expiresAt"
    enabled        = true
  }

  # ── Point-in-Time Recovery ─────────────────────────────────────────────
  point_in_time_recovery {
    enabled = var.enable_pitr
  }

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}
