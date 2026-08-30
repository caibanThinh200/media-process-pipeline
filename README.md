# Media Processing Pipeline

A **serverless, cloud-native media processing platform** built with Next.js, Go (AWS Lambda), S3, SQS, DynamoDB, and Terraform. Upload an image directly from the browser — it gets processed asynchronously in the cloud and delivered back via CloudFront CDN.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Data Flow](#data-flow)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
  - [1. Infrastructure (Terraform)](#1-infrastructure-terraform)
  - [2. Backend (Go Lambdas)](#2-backend-go-lambdas)
  - [3. Frontend (Next.js)](#3-frontend-nextjs)
- [Configuration Reference](#configuration-reference)
- [Available Make Targets](#available-make-targets)
- [Delivery Phases](#delivery-phases)
- [Design Principles](#design-principles)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Browser (Next.js)                           │
│  Upload page ─── Drag & drop ─── Presign request ─── Direct S3 PUT  │
│  Gallery page ─── Polling ─────────────────────── CloudFront CDN     │
└────────────────────────────┬────────────────────────────────────────┘
                             │
          ┌──────────────────▼──────────────────┐
          │         API Gateway (HTTP API)        │
          └──────────────────┬──────────────────┘
                             │
          ┌──────────────────▼──────────────────┐
          │       upload-api Lambda (Go)          │
          │  • generates presigned S3 PUT URL     │
          │  • creates job record (PENDING)        │
          │  • enqueues job message to SQS        │
          └───────┬──────────────────┬───────────┘
                  │                  │
    ┌─────────────▼──┐     ┌─────────▼──────────┐
    │  S3 raw bucket  │     │  SQS media queue    │
    │  (raw/jobId/…)  │     │  + Dead-Letter Queue│
    └─────────────────┘     └──────────┬──────────┘
                                       │  triggers
                     ┌─────────────────▼──────────────────┐
                     │     image-worker Lambda (Go)         │
                     │  • idempotency check (DynamoDB)      │
                     │  • downloads raw object from S3      │
                     │  • processes (copy / resize / …)     │
                     │  • writes output to S3 output bucket │
                     │  • sets job status → COMPLETE        │
                     └──────────────┬─────────────────────┘
                                    │
              ┌─────────────────────▼───────────────────┐
              │  S3 output bucket  ←→  CloudFront CDN    │
              │  (output/jobId/…)       (HTTPS delivery) │
              └────────────────────────────────────────┘
                                    │
                         ┌──────────▼──────────┐
                         │  DynamoDB jobs table  │
                         │  (PENDING → COMPLETE) │
                         └───────────────────────┘
```

---

## Tech Stack

| Layer | Technology |
|---|---|
| **Frontend** | Next.js 14, React 18, TypeScript, Tailwind CSS v4 |
| **UI Components** | shadcn/ui, Base UI, Lucide React |
| **Backend** | Go 1.22, AWS Lambda (arm64) |
| **API** | AWS API Gateway HTTP API |
| **Storage** | AWS S3 (raw + output buckets) |
| **CDN** | AWS CloudFront |
| **Database** | AWS DynamoDB (PAY_PER_REQUEST) |
| **Messaging** | AWS SQS + Dead-Letter Queue |
| **IaC** | Terraform >= 1.6, AWS provider ~5.0 |

---

## Project Structure

```
media-processing-pipeline/
├── Makefile                        # Top-level dev commands
├── agent/                          # Architecture & roadmap skill files
│   └── roadmap.md                  # Phase gate tracker
│
├── terraform/                      # All infrastructure as code
│   ├── main.tf                     # Root module — wires all child modules
│   ├── variables.tf                # Tuneable knobs (region, sizes, TTLs…)
│   ├── outputs.tf                  # Outputs fed into frontend .env
│   ├── terraform.tfvars.example    # Template — copy to terraform.tfvars
│   ├── iam-policies/               # Least-privilege IAM JSON policies
│   └── modules/
│       ├── storage/                # S3 raw & output buckets + CloudFront
│       ├── database/               # DynamoDB jobs table + GSI + TTL
│       ├── messaging/              # SQS queue + DLQ
│       ├── compute/                # upload-api Lambda + API Gateway
│       ├── async/                  # image-worker Lambda + SQS trigger
│       └── networking/             # (Phase 5) VPC + ALB
│
├── backend/                        # Go Lambda source code
│   ├── go.mod
│   ├── Makefile                    # Build targets for each Lambda
│   ├── build/                      # Compiled zip artifacts (git-ignored)
│   ├── cmd/
│   │   ├── upload-api/             # POST /upload/presign handler
│   │   ├── image-worker/           # SQS consumer — image passthrough/process
│   │   └── video-worker/           # (Phase 6) FFmpeg video transcoder
│   └── internal/
│       ├── models/                 # Shared domain types (Job, JobMessage)
│       ├── db/                     # DynamoDB client wrapper
│       ├── storage/                # S3 client wrapper
│       └── queue/                  # SQS client wrapper
│
└── frontend/                       # Next.js application
    ├── .env.local                  # Local env vars (git-ignored)
    ├── app/
    │   ├── layout.tsx              # Root layout + global nav
    │   ├── page.tsx                # Upload page (drag-and-drop)
    │   ├── gallery/page.tsx        # Gallery of processed media
    │   └── api/job-status/[jobId]/ # Server-side polling proxy
    └── components/
        ├── UploadZone.tsx          # File picker + S3 direct upload + polling
        ├── MediaCard.tsx           # Job result preview card
        └── ProcessingStatus.tsx    # Live status badge (PENDING → COMPLETE)
```

---

## Data Flow

```
1. User drops a file in the browser.
2. Browser calls POST /upload/presign → upload-api Lambda
   → returns { jobId, uploadUrl, objectKey }
   → creates DynamoDB record (status: PENDING)
   → publishes JobMessage to SQS
3. Browser PUTs the file directly to S3 via the presigned URL.
4. image-worker Lambda is triggered by SQS:
   → idempotency check (skip if already COMPLETE/FAILED)
   → sets DynamoDB status to PROCESSING
   → downloads object from raw bucket
   → writes processed output to output/jobId/filename
   → sets DynamoDB status to COMPLETE with outputKey
5. Frontend polls GET /api/job-status/[jobId] every 2 s until done.
6. On COMPLETE, the MediaCard renders the image via the
   CloudFront URL: https://<distribution>/output/jobId/filename
```

---

## Prerequisites

| Tool | Minimum version | Install |
|---|---|---|
| Go | 1.22 | [go.dev/dl](https://go.dev/dl) |
| Node.js | 18.17 (22 recommended) | [nvm](https://github.com/nvm-sh/nvm) |
| pnpm | 9+ | `npm i -g pnpm` |
| Terraform | 1.6 | [developer.hashicorp.com](https://developer.hashicorp.com/terraform/install) |
| AWS CLI | v2 | [aws.amazon.com/cli](https://aws.amazon.com/cli/) |

You also need an **AWS account** with an IAM user/role that has permissions to create Lambda, API Gateway, S3, CloudFront, SQS, and DynamoDB resources. See [`terraform/iam-policies/`](./terraform/iam-policies/) for the least-privilege policy documents.

---

## Quick Start

### 1. Infrastructure (Terraform)

```bash
# Copy and fill in the example vars
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
# Edit terraform.tfvars — set cors_allowed_origins to http://localhost:3000 for dev

# Build Lambda binaries first (Terraform packages them)
make build-upload-api
make build-image-worker

# Deploy
cd terraform
terraform init
terraform apply
```

After apply, note the outputs you need for the frontend:

```bash
terraform output api_endpoint           # → API_BASE_URL in frontend .env.local
terraform output cloudfront_domain_name # → NEXT_PUBLIC_CLOUDFRONT_URL
```

### 2. Backend (Go Lambdas)

The Makefile compiles each Lambda for **Linux arm64** and zips the binary:

```bash
# Build upload-api
make build-upload-api

# Build image-worker
make build-image-worker

# Run unit tests (no AWS credentials required)
make test-unit
```

Lambda binaries land in `backend/build/` and are referenced by Terraform's `archive_file` data sources — just run `terraform apply` after rebuilding.

### 3. Frontend (Next.js)

```bash
# Requires Node.js >= 18.17; use nvm:
nvm use 22

cd frontend

# Create your env file
cp .env.local.example .env.local   # then fill in values from terraform output

# Install & run dev server
pnpm install
pnpm dev
```

App is available at http://localhost:3000.

---

## Configuration Reference

### Frontend — `frontend/.env.local`

| Variable | Source | Description |
|---|---|---|
| `API_BASE_URL` | `terraform output api_endpoint` | API Gateway invoke URL |
| `NEXT_PUBLIC_CLOUDFRONT_URL` | `terraform output cloudfront_domain_name` | CloudFront domain (no `https://`) |

### Backend — Lambda environment variables (set by Terraform)

| Variable | Lambda(s) | Description |
|---|---|---|
| `RAW_BUCKET` | upload-api, image-worker | S3 bucket for raw uploads |
| `OUTPUT_BUCKET` | image-worker | S3 bucket for processed media |
| `QUEUE_URL` | upload-api | SQS queue URL |
| `DYNAMODB_TABLE` | all | DynamoDB jobs table name |
| `MAX_UPLOAD_BYTES` | upload-api | Maximum accepted file size (bytes) |

### Key Terraform Variables — `terraform/terraform.tfvars`

| Variable | Default | Description |
|---|---|---|
| `project` | `media-pipeline` | Resource name prefix |
| `environment` | `dev` | Environment tag |
| `aws_region` | `ap-southeast-1` | AWS region |
| `cors_allowed_origins` | `["*"]` | Origins allowed for S3 presigned PUTs |
| `raw_expiry_days` | `7` | Days until raw uploads are auto-deleted |
| `cloudfront_price_class` | `PriceClass_100` | Edge location tier |
| `job_ttl_days` | `30` | Days before DynamoDB job records expire |
| `upload_api_memory_mb` | `256` | Upload Lambda memory |
| `image_worker_memory_mb` | `256` | Image worker Lambda memory |

---

## Available Make Targets

Run `make help` to see all targets:

| Target | Description |
|---|---|
| `build-upload-api` | Compile upload-api Lambda binary for Linux arm64 |
| `build-image-worker` | Compile image-worker Lambda binary for Linux arm64 |
| `test-unit` | Run all Go unit tests (no AWS credentials required) |
| `deploy-phase3` | Build image-worker then run `terraform apply` |
| `tf-plan` | Run `terraform plan` with the dev AWS profile |
| `tf-apply` | Run `terraform apply` with the dev AWS profile |
| `frontend-dev` | Start Next.js dev server (Node 22 via nvm) |
| `frontend-build` | Build the Next.js production bundle |
| `policy-update-core` | Push a new version of the Core IAM policy |
| `policy-update-platform` | Push a new version of the Platform IAM policy |

---

## Delivery Phases

The project is structured around six delivery phases, each with explicit acceptance criteria (phase gates).

| # | Phase | Status | What was built |
|---|---|---|---|
| 1 | **Storage Foundation** | ✅ Complete | S3 raw + output buckets, CloudFront distribution, DynamoDB jobs table with GSI |
| 2 | **Upload Flow** | ✅ Complete | `POST /upload/presign` Lambda, API Gateway HTTP API, presigned S3 PUT URL, job creation |
| 3 | **Async Pipeline** | ✅ Complete | SQS queue + DLQ, S3 event trigger, image-worker Lambda with idempotency |
| 4 | **Frontend** | ✅ Complete | Next.js upload page, drag-and-drop, live status polling, gallery, error handling |
| 5 | **Networking Layer** | 🔜 Planned | VPC, ALB, private subnets, security groups |
| 6 | **Video + Polish** | 🔜 Planned | FFmpeg Lambda layer, video transcoding, CloudWatch alarms, X-Ray tracing |

---

## Design Principles

- **IAM least privilege** — every Lambda role has only the actions it needs on specific resource ARNs. Policies are stored in [`terraform/iam-policies/`](./terraform/iam-policies/) and versioned via Make.
- **Structured logging** — all log lines include `jobId`, `objectKey`, and `workerType` for easy CloudWatch Insights querying.
- **Idempotency** — the image worker checks the existing job status before processing; duplicate SQS deliveries (at-least-once semantics) are silently skipped.
- **Cost awareness** — DynamoDB uses `PAY_PER_REQUEST`, Lambdas are `arm64`, CloudFront uses `PriceClass_100` in dev.
- **No secrets in code** — all ARNs, bucket names, and URLs are injected as environment variables from Terraform outputs; `terraform.tfvars` and `.env.local` are git-ignored.
- **Direct browser-to-S3 uploads** — the app server is never in the media data path; the browser PUTs directly to S3 via a short-lived presigned URL, keeping Lambda costs and latency minimal.
