##############################################################################
# Storage module — S3 (raw + output) + CloudFront OAC
#
# Raw bucket  : receives direct browser uploads via presigned PUT URLs.
#               Versioning is disabled to control cost; lifecycle deletes
#               objects after var.raw_expiry_days so raw files don't linger.
#
# Output bucket : holds processed media served privately through CloudFront.
#                 Versioning is enabled so re-processing doesn't lose previous
#                 outputs. Objects are served only via CloudFront using OAC.
#
# CloudFront  : Origin Access Control (not legacy OAI) keeps the output
#               bucket fully private while delivering processed assets globally.
##############################################################################

locals {
  name_prefix = "${var.project}-${var.environment}"
}

###############################################################################
# RAW BUCKET
###############################################################################

resource "aws_s3_bucket" "raw" {
  bucket        = "${local.name_prefix}-raw"
  force_destroy = var.raw_bucket_force_destroy

  tags = {
    Project     = var.project
    Environment = var.environment
    Purpose     = "raw-uploads"
  }
}

# Block all public access — uploads happen via presigned PUT, not public ACLs.
resource "aws_s3_bucket_public_access_block" "raw" {
  bucket                  = aws_s3_bucket.raw.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# CORS — required so browsers can PUT directly to S3 with a presigned URL.
resource "aws_s3_bucket_cors_configuration" "raw" {
  bucket = aws_s3_bucket.raw.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT"]
    allowed_origins = var.cors_allowed_origins
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}

# Lifecycle — delete raw objects after N days to control storage cost.
resource "aws_s3_bucket_lifecycle_configuration" "raw" {
  bucket = aws_s3_bucket.raw.id

  rule {
    id     = "expire-raw-uploads"
    status = "Enabled"

    filter {
      prefix = "raw/"
    }

    expiration {
      days = var.raw_expiry_days
    }

    # Clean up incomplete multipart uploads quickly.
    abort_incomplete_multipart_upload {
      days_after_initiation = 1
    }
  }
}

###############################################################################
# OUTPUT BUCKET
###############################################################################

resource "aws_s3_bucket" "output" {
  bucket        = "${local.name_prefix}-output"
  force_destroy = var.output_bucket_force_destroy

  tags = {
    Project     = var.project
    Environment = var.environment
    Purpose     = "processed-media"
  }
}

# Block all public access — delivery goes through CloudFront only.
resource "aws_s3_bucket_public_access_block" "output" {
  bucket                  = aws_s3_bucket.output.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Versioning — preserve previous processed outputs when a job re-runs.
resource "aws_s3_bucket_versioning" "output" {
  bucket = aws_s3_bucket.output.id

  versioning_configuration {
    status = "Enabled"
  }
}

###############################################################################
# CLOUDFRONT — Origin Access Control (OAC)
###############################################################################

resource "aws_cloudfront_origin_access_control" "output" {
  name                              = "${local.name_prefix}-output-oac"
  description                       = "OAC for ${local.name_prefix} output bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "cdn" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = "${local.name_prefix} CDN"
  default_root_object = ""
  price_class         = var.cloudfront_price_class
  wait_for_deployment = false

  origin {
    domain_name              = aws_s3_bucket.output.bucket_regional_domain_name
    origin_id                = "S3OutputOrigin"
    origin_access_control_id = aws_cloudfront_origin_access_control.output.id
  }

  default_cache_behavior {
    target_origin_id       = "S3OutputOrigin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    # 24h default TTL; workers can set Cache-Control on objects to override.
    min_ttl     = 0
    default_ttl = 86400
    max_ttl     = 31536000
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }

  tags = {
    Project     = var.project
    Environment = var.environment
  }
}

# Bucket policy — grants CloudFront OAC GetObject access to the output bucket.
data "aws_iam_policy_document" "output_bucket_policy" {
  statement {
    sid    = "AllowCloudFrontOAC"
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.output.arn}/*"]

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.cdn.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "output" {
  bucket = aws_s3_bucket.output.id
  policy = data.aws_iam_policy_document.output_bucket_policy.json

  # The public access block must be in place before attaching a policy.
  depends_on = [aws_s3_bucket_public_access_block.output]
}
