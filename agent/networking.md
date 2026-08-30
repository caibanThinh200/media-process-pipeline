---
name: "media-pipeline-networking"
description: "Builds Phase 5 of the media pipeline: VPC, ALB, and routing rules. Invoke when setting up private subnets, attaching Lambda or ECS to an ALB, or reviewing network security for the media platform."
---

# Media Pipeline — Phase 5: Networking Layer

Use this skill when working on VPC infrastructure and ALB-based routing for the media pipeline.

## Phase scope

- Terraform: `terraform/modules/networking/` — VPC, subnets, security groups, ALB, ACM

## Invoke when

- The user asks to add a VPC, ALB, or private subnet to the media pipeline.
- You need to expose the upload API through an ALB instead of (or in addition to) API Gateway.
- You are reviewing security groups, NACLs, or routing rules.
- You need TLS termination at the ALB with an ACM certificate.

## Network topology

```
Internet
  │  HTTPS 443
  ▼
ALB (public subnets, AZ-a + AZ-b)
  │  security group: allow 443 inbound from 0.0.0.0/0
  │
  ├─ Listener rule: /upload/* → Target Group → Lambda upload-api
  └─ Listener rule: /health   → fixed 200 response
       │
       ▼ (future Phase 5 extension)
  ECS / EC2 in private subnets (if workers move off Lambda)

Private subnets: Lambda, RDS, ElastiCache (not needed until Phase 5+)
NAT Gateway: allows private resources to reach AWS APIs
```

## Terraform structure

```hcl
# modules/networking/main.tf

resource "aws_vpc" "main" { cidr_block = "10.0.0.0/16" }

resource "aws_subnet" "public" {
  # Two public subnets (AZ-a, AZ-b) for ALB
  count             = 2
  cidr_block        = cidrsubnet(aws_vpc.main.cidr_block, 8, count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]
}

resource "aws_subnet" "private" {
  # Two private subnets for Lambda / future services
  count             = 2
  cidr_block        = cidrsubnet(aws_vpc.main.cidr_block, 8, count.index + 2)
  availability_zone = data.aws_availability_zones.available.names[count.index]
}

resource "aws_internet_gateway" "main" { vpc_id = aws_vpc.main.id }
resource "aws_nat_gateway" "main" { ... }   # One NAT GW in public subnet

resource "aws_lb" "main" {
  load_balancer_type = "application"
  subnets            = aws_subnet.public[*].id
  security_groups    = [aws_security_group.alb.id]
}

resource "aws_lb_listener" "https" {
  port            = 443
  protocol        = "HTTPS"
  certificate_arn = var.acm_certificate_arn
  ...
}

resource "aws_lb_target_group" "upload_api" {
  target_type = "lambda"
  ...
}

resource "aws_lambda_permission" "alb" {
  principal  = "elasticloadbalancing.amazonaws.com"
  source_arn = aws_lb_target_group.upload_api.arn
}
```

## Security group rules

| SG | Inbound | Outbound |
|---|---|---|
| `alb-sg` | TCP 443 from `0.0.0.0/0` | All to `lambda-sg` |
| `lambda-sg` | TCP 443 from `alb-sg` only | All to AWS APIs (via NAT) |

## Variables needed

```hcl
variable "acm_certificate_arn" {
  description = "ARN of ACM certificate for ALB HTTPS listener"
  type        = string
}
variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}
```

## Gate criteria (Phase 5 done when)

- VPC and subnets exist; `terraform output vpc_id` is populated
- ALB DNS name returns HTTP 301 → HTTPS (TLS terminates at ALB)
- `POST https://<alb-dns>/upload/presign` routes to upload Lambda and returns `{jobId, uploadUrl}`
- Direct Lambda invocation URL (if any) is disabled or restricted to ALB only
- Security group blocks all traffic to Lambda except from the ALB security group

## Example prompts

- `Scaffold the VPC and ALB Terraform for this media pipeline.`
- `Wire the ALB target group to the upload Lambda and set the correct security groups.`
- `Review these security groups for least-privilege network access.`
