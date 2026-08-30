# Media Processing Pipeline

A cloud-native media processing platform built with **Next.js**, **Go (Lambda)**, **AWS S3**, **SQS**, **DynamoDB**, and **Terraform**.

## Project Structure

```
media-processing-pipeline/
├── agent/                  # Architecture & agent skill files
├── terraform/              # Infrastructure as Code (Terraform)
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   └── modules/
│       ├── networking/
│       ├── storage/        # S3 raw & output buckets
│       ├── messaging/      # SQS queue + DLQ
│       ├── database/       # DynamoDB jobs table
│       └── compute/        # Lambda + API Gateway + CloudFront
├── backend/                # Go Lambda functions
│   ├── go.mod
│   ├── cmd/
│   │   ├── upload-api/     # Presign URL + job creation
│   │   ├── image-worker/   # Image resize/compress/watermark
│   │   └── video-worker/   # Transcode + thumbnail generation
│   └── internal/
│       ├── models/         # Shared domain types (Job)
│       ├── storage/        # S3 client wrapper
│       ├── queue/          # SQS client wrapper
│       └── db/             # DynamoDB client wrapper
└── frontend/               # Next.js 14 application
    ├── package.json
    ├── app/
    │   ├── page.tsx                    # Upload flow
    │   ├── gallery/page.tsx            # Processed media gallery
    │   └── api/job-status/[jobId]/     # Status polling proxy route
    └── components/
        ├── UploadZone.tsx              # Drag-and-drop uploader
        ├── MediaCard.tsx               # Media preview card
        └── ProcessingStatus.tsx        # Live job status badge
```

## Data Flow

```
Browser → PUT presigned URL → S3 (raw/)
        → POST /upload/presign → upload-api Lambda → DynamoDB (PENDING)
                                                    → SQS (job message)
SQS → image-worker / video-worker Lambda
    → S3 (output/) + DynamoDB (COMPLETE)
CloudFront → S3 (output/) → Browser
```

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+
- Terraform 1.6+
- AWS CLI configured

### Infrastructure

```bash
cd terraform
terraform init
terraform apply -var="environment=dev"
```

### Backend

```bash
cd backend
go mod tidy
GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/upload-api
zip upload-api.zip bootstrap
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Environment Variables

| Variable         | Used by              | Description                     |
|-----------------|---------------------|---------------------------------|
| `RAW_BUCKET`    | upload-api, workers | S3 bucket for raw uploads       |
| `OUTPUT_BUCKET` | workers             | S3 bucket for processed media   |
| `QUEUE_URL`     | upload-api          | SQS queue URL                   |
| `DYNAMODB_TABLE`| all backend         | DynamoDB jobs table name        |
| `API_BASE_URL`  | frontend            | Base URL of the upload-api      |
