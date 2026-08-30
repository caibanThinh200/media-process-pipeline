output "image_worker_function_name" {
  description = "Image-worker Lambda function name."
  value       = aws_lambda_function.image_worker.function_name
}

output "image_worker_function_arn" {
  description = "Image-worker Lambda function ARN."
  value       = aws_lambda_function.image_worker.arn
}
