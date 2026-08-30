---
name: "media-pipeline-video-polish"
description: "Builds Phase 6 of the media pipeline: FFmpeg-based video worker, DLQ alarms, CloudWatch dashboards, and X-Ray tracing. Invoke when implementing video transcoding, adding observability, or reviewing failure handling for the complete pipeline."
---

# Media Pipeline — Phase 6: Video + Polish

Use this skill when working on the video processing path, operational observability, and system hardening.

## Phase scope

- Go: `backend/cmd/video-worker/main.go` — FFmpeg Lambda handler
- Terraform: FFmpeg Lambda Layer, video-worker function, DLQ CloudWatch alarm
- Terraform: CloudWatch dashboard, X-Ray tracing enablement

## Invoke when

- The user asks to add video transcoding or thumbnail generation.
- You are attaching the FFmpeg Lambda Layer in Terraform.
- You need to set up CloudWatch alarms for DLQ depth, Lambda errors, or duration.
- You are enabling X-Ray tracing across the upload API and workers.
- You need a CloudWatch dashboard for the full pipeline health.

## Video worker flow

```
SQS message (fileType = "video")
  │
  ▼
video-worker Lambda (arm64, 3008 MB, 900s timeout)
  │  1. parse message → jobId, objectKey
  │  2. check DynamoDB status — skip if not PENDING/UPLOADED (idempotency)
  │  3. update → PROCESSING
  │  4. download raw video from S3 to /tmp/<jobId>-input.<ext>
  │  5. run FFmpeg:
  │     - transcode to H.264 MP4 (720p, CRF 23)
  │     - extract thumbnail at 00:00:02 as JPEG
  │  6. upload output/<jobId>/video.mp4 and output/<jobId>/thumb.jpg to output bucket
  │  7. update DynamoDB → COMPLETE, set outputKey + thumbKey
  │  8. on error → update → FAILED with ffmpeg stderr snippet
```

## FFmpeg Lambda Layer

```hcl
resource "aws_lambda_layer_version" "ffmpeg" {
  layer_name          = "${var.project}-${var.environment}-ffmpeg"
  s3_bucket           = var.lambda_assets_bucket
  s3_key              = "layers/ffmpeg-arm64.zip"
  compatible_runtimes = ["provided.al2023"]
  compatible_architectures = ["arm64"]
}
```

> The FFmpeg static binary must be built for `linux/arm64` and placed at `bin/ffmpeg` inside the zip.
> Publicly available builds: https://johnvansickle.com/ffmpeg/ (use the arm64 static build)

## FFmpeg commands (run via `exec.CommandContext`)

```bash
# Transcode to H.264 MP4
ffmpeg -i /tmp/input.mp4 \
  -vf "scale=-2:720" \
  -c:v libx264 -crf 23 -preset fast \
  -c:a aac -b:a 128k \
  -movflags +faststart \
  /tmp/output.mp4

# Extract thumbnail at 2 seconds
ffmpeg -i /tmp/input.mp4 \
  -ss 00:00:02 -vframes 1 \
  -vf "scale=-2:360" \
  /tmp/thumb.jpg
```

## Routing image vs video jobs

The SQS event source mapping filter (or in-handler logic) routes by `fileType`:
- `fileType = "image"` → image-worker
- `fileType = "video"` → video-worker

Two separate SQS event source mappings each with a filter policy:
```hcl
filter_criteria {
  filter { pattern = jsonencode({ body = { fileType = ["image"] } }) }
}
```

## CloudWatch alarms

```hcl
# DLQ depth alarm — fire when any message lands in the dead-letter queue
resource "aws_cloudwatch_metric_alarm" "dlq_depth" {
  alarm_name          = "${var.project}-${var.environment}-dlq-messages"
  comparison_operator = "GreaterThanThreshold"
  threshold           = 0
  evaluation_periods  = 1
  metric_name         = "ApproximateNumberOfMessagesVisible"
  namespace           = "AWS/SQS"
  period              = 60
  statistic           = "Maximum"
  dimensions          = { QueueName = aws_sqs_queue.media_dlq.name }
  alarm_actions       = [var.sns_alert_topic_arn]
}

# Lambda error rate alarm
resource "aws_cloudwatch_metric_alarm" "worker_errors" { ... }

# Lambda P95 duration alarm
resource "aws_cloudwatch_metric_alarm" "worker_duration" { ... }
```

## X-Ray tracing

```hcl
# Enable on each Lambda
resource "aws_lambda_function" "video_worker" {
  tracing_config { mode = "Active" }
}
```

In Go, use `github.com/aws/aws-xray-sdk-go` to instrument AWS SDK calls:
```go
import "github.com/aws/aws-xray-sdk-go/xray"
s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.HTTPClient = xray.Client(nil)
})
```

## CloudWatch dashboard

Create a single dashboard with widgets for:
- Upload API: invocations, errors, duration (p95)
- Image Worker: invocations, errors, duration (p95)
- Video Worker: invocations, errors, duration (p95)
- SQS queue depth (media-queue + DLQ)
- DynamoDB read/write capacity consumed

## Gate criteria (Phase 6 done when)

- Video upload triggers video-worker (not image-worker); image uploads still route to image-worker
- Transcoded `.mp4` and `thumb.jpg` appear in the output bucket
- CloudFront serves both `output/<jobId>/video.mp4` and `output/<jobId>/thumb.jpg`
- DLQ CloudWatch alarm triggers within 2 minutes of a message landing in the DLQ
- X-Ray service map shows API → SQS → Worker → S3 spans
- CloudWatch dashboard shows green (no alarm) for all Lambdas under normal load

## Example prompts

- `Implement the Go video worker Lambda using the FFmpeg layer.`
- `Add CloudWatch alarms for DLQ depth and Lambda errors in Terraform.`
- `Enable X-Ray tracing on all Lambdas and instrument the Go AWS SDK calls.`
- `Build a CloudWatch dashboard covering upload API, image worker, and video worker.`
