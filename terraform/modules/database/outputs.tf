output "jobs_table_name" {
  description = "DynamoDB jobs table name"
  value       = aws_dynamodb_table.jobs.name
}

output "jobs_table_arn" {
  description = "DynamoDB jobs table ARN (used in IAM policies for workers)"
  value       = aws_dynamodb_table.jobs.arn
}

output "jobs_table_stream_arn" {
  description = "DynamoDB Streams ARN (empty if streams are disabled)"
  value       = aws_dynamodb_table.jobs.stream_arn
}
