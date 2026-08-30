---
name: "media-pipeline-async-pipeline"
description: "Builds Phase 3 of the media pipeline: SQS queue, S3 event notification, and the Go Image Worker Lambda. Invoke when wiring the async processing path, implementing image transforms, or setting up SQS event source mappings."
---

# Media Pipeline — Phase 3: Async Pipeline

Use this skill when working on the SQS-driven image processing path and its Terraform infrastructure.

## Phase scope

- Terraform: `terraform/modules/messaging/` — SQS queue, DLQ, S3 event notification
- Terraform: `terraform/modules/compute/` — image-worker Lambda, IAM role, event source mapping
- Go: `backend/cmd/image-worker/main.go` — SQS consumer handler
- Go: `backend/internal/` — shared clients reused from Phase 2

## Invoke when

- The user asks to wire S3 → SQS or SQS → Lambda event triggers.
- You are building or reviewing the image resize, compress, or watermark logic.
- You need to implement idempotent SQS batch processing with partial failure support.
- You are setting DLQ thresholds, visibility timeouts, or retry policies.

## Event flow

```
S3 raw bucket (PutObject event)
  │  s3:ObjectCreated:Put on prefix raw/
  ▼
SQS media-queue  ──(3 retries)──▶  DLQ
  │  EventSourceMapping batchSize=1, bisectOnError=true
  ▼
image-worker Lambda
  │  1. parse SQS message → extract jobId, objectKey, fileType
  │  2. check DynamoDB: if status != PENDING/UPLOADED → skip (idempotency)
  │  3. update status → PROCESSING
  │  4. download from raw bucket
  │  5. resize + compress (imaging lib)
  │  6. upload to output bucket: output/<jobId>/result.<ext>
  │  7. update DynamoDB → COMPLETE, set outputKey
  │  8. on error → update → FAILED with reason
  ▼
DynamoDB jobs table
```

## SQS configuration

| Setting | Value | Reason |
|---|---|---|
| `visibility_timeout_seconds` | `300` | Must be > Lambda timeout (5 min max for image work) |
| `message_retention_seconds` | `86400` | 1 day — stale jobs shouldn't block the queue |
| `maxReceiveCount` (DLQ) | `3` | Retry 3× before dead-lettering |
| DLQ `message_retention_seconds` | `1209600` | 14 days to inspect failures |
| `batch_size` (ESM) | `1` | Process one job per invocation for simplicity |
| `bisect_batch_on_function_error` | `true` | Isolate a bad message without failing siblings |

## Image processing rules

- Use `github.com/disintegration/imaging` for resize, crop, and compression.
- Output format: JPEG for photos, PNG for images with transparency (detect from MIME).
- Default resize: longest edge 1920px, preserve aspect ratio.
- Always write to a temp file in `/tmp` then stream upload to S3.
- Never modify the raw object — always write to `output/<jobId>/`.

## Terraform additions for this phase

```hcl
# modules/messaging/ — already scaffolded; ensure these exist:
resource "aws_sqs_queue" "media_queue" { ... }
resource "aws_sqs_queue" "media_dlq" { ... }

# S3 → SQS notification (goes in modules/storage/ or modules/messaging/)
resource "aws_s3_bucket_notification" "raw_uploads" {
  bucket = aws_s3_bucket.raw.id
  queue { queue_arn = var.sqs_queue_arn; events = ["s3:ObjectCreated:Put"]; filter_prefix = "raw/" }
}

# modules/compute/ additions:
resource "aws_lambda_function" "image_worker" { ... }
resource "aws_iam_role" "image_worker" { ... }
resource "aws_lambda_event_source_mapping" "image_sqs" {
  event_source_arn                   = var.sqs_queue_arn
  function_name                      = aws_lambda_function.image_worker.arn
  batch_size                         = 1
  bisect_batch_on_function_error     = true
}
```

## IAM policy for image-worker Lambda role

```json
{
  "Statement": [
    { "Action": ["s3:GetObject"], "Resource": "arn:aws:s3:::*-raw/raw/*" },
    { "Action": ["s3:PutObject"], "Resource": "arn:aws:s3:::*-output/output/*" },
    { "Action": ["dynamodb:GetItem", "dynamodb:UpdateItem"], "Resource": "<jobs-table-arn>" },
    { "Action": ["sqs:ReceiveMessage", "sqs:DeleteMessage", "sqs:GetQueueAttributes"], "Resource": "<queue-arn>" },
    { "Action": ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"], "Resource": "*" }
  ]
}
```

## Gate criteria (Phase 3 done when)

- Uploading to `raw/` prefix triggers an SQS message (check CloudWatch metrics)
- Image worker processes the message: status transitions `PENDING → PROCESSING → COMPLETE`
- Output object exists at `output/<jobId>/result.jpg` in the output bucket
- CloudFront serves the output object via `https://<cf-domain>/output/<jobId>/result.jpg`
- A bad message (corrupt body) lands in the DLQ after 3 attempts without crashing the queue
- Sending the same SQS message twice produces only one output object (idempotency)

## Example prompts

- `Wire the SQS queue to the image worker Lambda with proper event source mapping.`
- `Implement the Go image worker that resizes and saves to S3.`
- `Review this SQS batch handler for idempotency and partial failure behavior.`
