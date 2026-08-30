---
name: "media-pipeline-upload-api"
description: "Builds Phase 2 of the media pipeline: the Go Upload API Lambda, API Gateway, and IAM roles. Invoke when implementing or reviewing the presign endpoint, job record creation, input validation, or the Terraform wiring for API Gateway and Lambda."
---

# Media Pipeline — Phase 2: Upload Flow

Use this skill when working on the Go Upload API Lambda and its Terraform infrastructure.

## Phase scope

- Go: `backend/cmd/upload-api/main.go` — Lambda handler
- Go: `backend/internal/` — shared storage, db, queue clients
- Terraform: `terraform/modules/compute/` — Lambda function, IAM role, API Gateway, event source

## Invoke when

- The user asks for the presign endpoint, job creation logic, or upload API scaffolding.
- You need to wire API Gateway HTTP API to the upload Lambda in Terraform.
- You are reviewing input validation, error responses, or IAM least privilege for the upload role.
- You need to return stable job identifiers to the browser and write PENDING records to DynamoDB.

## API contract

```
POST /upload/presign
Content-Type: application/json

Request:
{
  "fileName":  "photo.jpg",
  "fileType":  "image",          // "image" | "video"
  "mediaType": "image/jpeg",     // MIME type
  "sizeBytes": 1048576
}

Response 200:
{
  "jobId":        "uuid-v4",
  "uploadUrl":    "https://s3.amazonaws.com/...",  // presigned PUT, 15 min TTL
  "objectKey":    "raw/<jobId>/photo.jpg",
  "expiresInSec": 900
}

Response 400: { "error": "..." }   // missing fields, unsupported type, size exceeded
Response 500: { "error": "..." }   // downstream failure
```

## Implementation rules

- Generate `jobId` with `uuid.NewString()` before any downstream call.
- Object key pattern: `raw/<jobId>/<originalFileName>` — preserves extension for worker type detection.
- Write to DynamoDB with `attribute_not_exists(jobId)` condition for idempotency.
- Presign TTL: 15 minutes (`900s`). Reject files larger than a configurable max (default 500 MB).
- Return the presigned URL and jobId in one response — the browser should not need a second call.
- Enqueue to SQS after DynamoDB write succeeds; a failed enqueue is non-fatal (sweeper can recover).

## Terraform additions for this phase

```hcl
# modules/compute/ additions:
resource "aws_lambda_function" "upload_api" { ... }
resource "aws_iam_role" "upload_api" { ... }
resource "aws_iam_role_policy" "upload_api" { ... }   # s3:PutObject on raw/*, dynamodb:PutItem, sqs:SendMessage
resource "aws_apigatewayv2_api" "http" { ... }        # HTTP API (not REST)
resource "aws_apigatewayv2_integration" "upload" { ... }
resource "aws_apigatewayv2_route" "presign" { ... }   # POST /upload/presign
resource "aws_lambda_permission" "apigw" { ... }
```

## IAM policy for upload Lambda role (least privilege)

```json
{
  "Statement": [
    { "Action": ["s3:PutObject"], "Resource": "arn:aws:s3:::*-raw/raw/*" },
    { "Action": ["dynamodb:PutItem", "dynamodb:GetItem"], "Resource": "<jobs-table-arn>" },
    { "Action": ["sqs:SendMessage"], "Resource": "<queue-arn>" },
    { "Action": ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"], "Resource": "*" }
  ]
}
```

## Environment variables (injected by Terraform)

| Variable | Source |
|---|---|
| `RAW_BUCKET` | `module.storage.raw_bucket_name` |
| `DYNAMODB_TABLE` | `module.database.jobs_table_name` |
| `QUEUE_URL` | `module.messaging.queue_url` |
| `MAX_UPLOAD_BYTES` | variable default `524288000` (500 MB) |

## Gate criteria (Phase 2 done when)

- `POST /upload/presign` returns `{jobId, uploadUrl}` with a valid presigned URL
- `curl -X PUT <uploadUrl> --upload-file test.jpg` succeeds (HTTP 200)
- DynamoDB shows a `PENDING` record for the jobId
- Lambda CloudWatch logs include structured jobId + objectKey
- 400 is returned for missing `fileName` or unsupported `fileType`

## Example prompts

- `Implement the Go upload API Lambda for presigned S3 uploads.`
- `Write the Terraform for API Gateway HTTP API wired to the upload Lambda.`
- `Review this upload handler for input validation and idempotent DynamoDB writes.`
