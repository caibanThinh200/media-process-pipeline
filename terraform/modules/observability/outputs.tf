output "dashboard_name" {
  description = "Name of the CloudWatch dashboard"
  value       = aws_cloudwatch_dashboard.pipeline.dashboard_name
}

output "dashboard_url" {
  description = "Direct URL to the CloudWatch dashboard in AWS Console"
  value       = "https://${var.aws_region}.console.aws.amazon.com/cloudwatch/home?region=${var.aws_region}#dashboards:name=${aws_cloudwatch_dashboard.pipeline.dashboard_name}"
}

output "image_dlq_alarm_arn" {
  description = "ARN of the image DLQ CloudWatch alarm"
  value       = aws_cloudwatch_metric_alarm.image_dlq.arn
}

output "video_dlq_alarm_arn" {
  description = "ARN of the video DLQ CloudWatch alarm"
  value       = aws_cloudwatch_metric_alarm.video_dlq.arn
}

output "xray_group_arn" {
  description = "ARN of the X-Ray group"
  value       = aws_xray_group.pipeline.arn
}
