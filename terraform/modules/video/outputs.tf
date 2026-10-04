output "video_worker_lambda_arn" {
  description = "ARN of the video-worker Lambda function"
  value       = aws_lambda_function.video_worker.arn
}

output "video_worker_function_name" {
  description = "Name of the video-worker Lambda function"
  value       = aws_lambda_function.video_worker.function_name
}

output "ffmpeg_layer_arn" {
  description = "ARN of the FFmpeg Lambda layer version"
  value       = aws_lambda_layer_version.ffmpeg.arn
}
