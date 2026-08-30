---
name: "media-pipeline-go-lambdas"
description: "Implements Go Lambda handlers for the media pipeline. Invoke when building or reviewing presign APIs, SQS workers, DynamoDB status updates, S3 access, and media processing logic."
---

# Media Pipeline Go Lambdas

Use this skill for Go services running in AWS Lambda within the media processing system.

## Invoke when

- The user asks for Go Lambda scaffolding or handler implementation.
- You need to build the upload API, image worker, or video worker.
- You are wiring AWS SDK v2 clients for S3, SQS, DynamoDB, CloudFront-related URL generation, or configuration loading.
- You need to review error handling, retries, idempotency, or batch processing behavior.

## Service boundaries

- `cmd/upload-api/`: generate presigned URLs, validate inputs, create job record, enqueue or trigger downstream state
- `cmd/image-worker/`: process image jobs, resize, compress, watermark, persist outputs
- `cmd/video-worker/`: transcode with FFmpeg layer, generate thumbnails, persist outputs
- `internal/`: config, shared models, AWS clients, repository helpers, queue helpers, media utilities

## Implementation guidance

- Use AWS SDK v2 and initialize clients once per execution environment when possible.
- Parse event payloads into explicit typed structs.
- Model job transitions clearly and write defensive status updates.
- Build workers to safely handle duplicate deliveries.
- Store both source key and output key in DynamoDB so the frontend can poll stable state.
- For SQS batches, decide whether to fail whole-batch or partial-batch and implement accordingly.

## Error-handling rules

- Validate media type before processing.
- Log structured context including job id, object key, and worker type.
- Distinguish retryable failures from permanent failures.
- Update DynamoDB to `FAILED` with a reason on terminal errors.
- Avoid partial success without an explicit final status update.

## Performance guidance

- Reuse temporary files carefully within Lambda `/tmp`.
- Keep packages small to reduce cold starts.
- Use memory settings appropriate for image libraries and video transcoding.
- Prefer streaming or staged temporary files over loading very large objects fully into memory.

## Expected outputs

- Go handler files
- shared internal packages
- event models
- status update helpers
- worker processing flow

## Example prompts

- `Scaffold the Go upload API Lambda for presigned S3 uploads and DynamoDB job creation.`
- `Implement the image worker Lambda in Go using AWS SDK v2.`
- `Review this Go Lambda worker for SQS idempotency and failure handling.`
