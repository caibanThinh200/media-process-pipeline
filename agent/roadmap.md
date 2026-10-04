---
name: "media-pipeline-roadmap"
description: "Master orchestrator for the media pipeline project. Invoke when planning delivery order, selecting which phase to work on next, checking phase gate criteria, or deciding which specialist skill to use."
---

# Media Pipeline Roadmap — Multi-Agent Orchestrator

This skill coordinates all six delivery phases of the media processing platform.
It routes work to specialist skills and enforces phase gate criteria before advancing.

## Phase map

| # | Phase | Layer | Specialist skill |
|---|-------|-------|-----------------|
| 1 | Storage Foundation | Terraform: S3 + CloudFront + DynamoDB | `media-pipeline-terraform`, `media-pipeline-architect` |
| 2 | Upload Flow | Go Lambda + API Gateway + IAM | `media-pipeline-upload-api` |
| 3 | Async Pipeline | SQS + S3 event + Image Worker Go | `media-pipeline-async-pipeline` |
| 4 | Frontend | Next.js upload UI + polling | `media-pipeline-nextjs` |
| 5 | Networking Layer | Terraform VPC + ALB + routing | `media-pipeline-networking` |
| 6 | Video + Polish | FFmpeg Lambda + DLQ + CloudWatch + X-Ray | `media-pipeline-video-polish` |

## Invoke when

- The user asks what to build next, which phase to tackle, or how phases relate.
- A task spans more than one layer (e.g. "add the Go worker and its Terraform trigger").
- You need to determine which specialist skill to load for a given task.
- You are checking whether a phase is complete before starting the next one.

## Routing rules

Use the table below to decide which skill to invoke:

```
User intent                              → Invoke skill
─────────────────────────────────────────────────────────
Plan architecture or delivery order      → media-pipeline-architect
Any Terraform / IaC question             → media-pipeline-terraform
S3 presign, job record, upload API Go    → media-pipeline-upload-api
SQS, S3 event trigger, image worker Go  → media-pipeline-async-pipeline
Next.js UI, upload UX, polling, gallery  → media-pipeline-nextjs
VPC, ALB, routing, networking IaC        → media-pipeline-networking
FFmpeg, video worker, DLQ, alarms, trace → media-pipeline-video-polish
```

## Phase gate criteria

A phase is complete only when ALL criteria in its gate are met.
Do not begin the next phase until the current gate passes.

### Gate 1 — Storage Foundation ✅ (current status: terraform plan clean)
- [ ] `terraform apply` succeeds with 0 errors
- [ ] Two S3 buckets exist: `*-raw` and `*-output`
- [ ] CloudFront distribution is `Deployed` and returns 403 for missing keys
- [ ] DynamoDB `*-jobs` table exists with `status-createdAt-index` GSI
- [ ] `terraform output` shows all five values

### Gate 2 — Upload Flow ✅
- [x] `POST /upload/presign` resazzsxzaaturns `{jobId, uploadUrl, objectKey}`
- [x] Presigned PUT URL works from curl against the raw bucket
- [x] Job record exists in DynamoDB with status `PENDING` after presign call
- [x] Lambda logs show structured job context (jobId, key, fileType)
- [x] API Gateway returns 400 for missing or invalid input

### Gate 3 — Async Pipeline ✅
- [x] Uploading a file to the raw bucket triggers an SQS message
- [x] Image worker Lambda consumes the message and updates status to `PROCESSING` then `COMPLETE`
- [x] Output object appears in the output bucket under `output/<jobId>/`
- [x] DLQ receives message after 3 failed processing attempts
- [x] Duplicate message delivery produces no duplicate output (idempotency)


### Gate 4 — Frontend ✅
- [x] Upload page renders; user can drag-and-drop or click to pick a file
- [x] File uploads directly to S3 (no app server in the data path)
- [x] Processing status polls and updates live until `COMPLETE` or `FAILED`
- [x] Gallery shows processed image from CloudFront URL
- [x] Error states are displayed for upload failures and job failures

### Gate 5 — Networking Layer ✅
- [x] VPC with public and private subnets exists in `terraform apply`
- [x] ALB is reachable on HTTP and routes to the upload Lambda via target group
- [x] Security groups block all direct access to Lambda; ALB is the only ingress
- [x] Health check on ALB returns 200 from the upload API

### Gate 6 — Video + Polish
- [ ] Video files trigger the video-worker Lambda, not the image-worker
- [ ] FFmpeg layer is attached; transcoded output appears in the output bucket
- [ ] DLQ alarm fires in CloudWatch when a message lands in the dead-letter queue
- [ ] X-Ray traces show the full path: API → SQS → Worker → S3
- [ ] CloudWatch dashboard exists with key metrics for all Lambdas

## Cross-cutting concerns (apply in every phase)

- **IAM least privilege** — every Lambda role gets only the actions it needs on specific ARNs.
- **Structured logging** — always include `jobId`, `objectKey`, and `workerType` in log lines.
- **Idempotency** — workers check for existing output before processing; DynamoDB writes use condition expressions.
- **Cost awareness** — choose PAY_PER_REQUEST for DynamoDB, arm64 for Lambda, and PriceClass_100 for CloudFront in dev.
- **Secrets** — never hard-code ARNs or keys; use environment variables injected by Terraform outputs.

## Example prompts

- `What phase should I work on next and what does it require?`
- `Check if Phase 2 gate criteria are met before I start Phase 3.`
- `I need to build the image worker — which skill handles that?`
- `Give me the full milestone list with acceptance criteria for all phases.`
