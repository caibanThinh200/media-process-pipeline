output "api_endpoint" {
  description = "HTTP API invoke URL — use this as API_BASE_URL in the frontend"
  value       = aws_apigatewayv2_stage.default.invoke_url
}

output "upload_api_function_name" {
  description = "Upload API Lambda function name"
  value       = aws_lambda_function.upload_api.function_name
}

output "upload_api_function_arn" {
  description = "Upload API Lambda function ARN"
  value       = aws_lambda_function.upload_api.arn
}

# Alias used by the networking module to attach Lambda as ALB target
output "upload_api_lambda_arn" {
  description = "Upload API Lambda ARN — registered as the ALB target group target"
  value       = aws_lambda_function.upload_api.arn
}
