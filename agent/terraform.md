---
name: "media-pipeline-terraform"
description: "Builds Terraform for the AWS media pipeline. Invoke when creating or reviewing IaC for S3, CloudFront, SQS, Lambda, API Gateway, DynamoDB, IAM, networking, and environment structure."
---

# Media Pipeline Terraform

Use this skill for infrastructure-as-code work related to the media processing platform.

## Invoke when

- The user asks to scaffold Terraform modules or environment composition.
- You need to create AWS resources for storage, messaging, compute, database, networking, or CDN delivery.
- You are reviewing IAM policies, module boundaries, variables, outputs, or remote state layout.
- You need to wire event sources such as S3 notifications and SQS-triggered Lambda workers.

## Terraform standards

- Keep reusable resources under `terraform/modules/`.
- Keep environment composition shallow in `terraform/` with `main.tf`, `variables.tf`, and `outputs.tf`.
- Use explicit variables for bucket names, TTLs, queue timeouts, Lambda memory, and domain or certificate settings.
- Prefer least-privilege IAM policies scoped to resource ARNs and exact actions.
- Make dependencies explicit when event wiring order matters.

## Suggested modules

- `networking`: VPC, subnets, security groups, ALB, ACM integration points
- `storage`: raw bucket, processed bucket, lifecycle rules, CloudFront, origin access control
- `messaging`: SQS queue, DLQ, redrive policy
- `compute`: Lambda functions, roles, permissions, log groups, event source mappings
- `database`: DynamoDB table, GSIs if needed, TTL configuration

## Resource guidance

- Raw uploads should land in a dedicated bucket or prefix with tightly scoped upload permissions.
- Processed output should use a separate bucket or prefix optimized for delivery and cache control.
- CloudFront should use Origin Access Control and keep S3 private.
- SQS should include a DLQ and a visibility timeout larger than the worker processing time.
- Lambda packaging should account for image libraries and FFmpeg layer usage.
- API Gateway and Lambda permissions must be wired explicitly.

## Review checklist

- Are modules small and composable?
- Are names, tags, and outputs environment-safe?
- Are bucket policies and CloudFront origin settings consistent?
- Are Lambda roles over-permissioned?
- Are alarms, log retention, and retries defined where needed?

## Expected outputs

- Terraform file structure
- module skeletons
- variable and output definitions
- IAM policy guidance
- event wiring recommendations

## Example prompts

- `Scaffold Terraform modules for this media processing platform.`
- `Write the storage and messaging Terraform for S3, CloudFront, and SQS.`
- `Review this Terraform for least-privilege IAM and event source wiring.`
