output "raw_bucket_name" {
  description = "Name of the S3 raw uploads bucket"
  value       = aws_s3_bucket.raw.bucket
}

output "raw_bucket_arn" {
  description = "ARN of the S3 raw uploads bucket"
  value       = aws_s3_bucket.raw.arn
}

output "output_bucket_name" {
  description = "Name of the S3 processed-media bucket"
  value       = aws_s3_bucket.output.bucket
}

output "output_bucket_arn" {
  description = "ARN of the S3 processed-media bucket"
  value       = aws_s3_bucket.output.arn
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID"
  value       = aws_cloudfront_distribution.cdn.id
}

output "cloudfront_domain_name" {
  description = "CloudFront distribution domain name (used to build asset URLs)"
  value       = aws_cloudfront_distribution.cdn.domain_name
}
