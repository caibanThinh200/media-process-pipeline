###############################################################################
# Networking module — VPC + ALB
#
# Topology
#   VPC (10.0.0.0/16)
#   ├── Public subnets  (one per AZ) — attached to Internet Gateway
#   │     10.0.0.0/24, 10.0.1.0/24, [10.0.2.0/24]
#   └── Private subnets (one per AZ) — no IGW route (future: NAT GW / workers)
#         10.0.100.0/24, 10.0.101.0/24, [10.0.102.0/24]
#
# ALB (internet-facing, public subnets)
#   ├── Security group: allow inbound 80 (443 when cert provided) from 0.0.0.0/0
#   ├── Listener :80  → forward to Lambda target group (HTTP-only dev mode)
#   │   OR redirect :80 → :443, Listener :443 → forward (when cert_arn set)
#   └── Target group  → type=lambda → upload-api Lambda ARN
###############################################################################

locals {
  name_prefix = "${var.project}-${var.environment}"
  has_cert    = var.certificate_arn != ""
}

##############################################################################
# Data — available AZs in the current region
##############################################################################

data "aws_availability_zones" "available" {
  state = "available"
}

##############################################################################
# VPC
##############################################################################

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name        = "${local.name_prefix}-vpc"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Internet Gateway
##############################################################################

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name        = "${local.name_prefix}-igw"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Public Subnets (one per AZ)
# CIDR: 10.0.{index}.0/24
##############################################################################

resource "aws_subnet" "public" {
  count = var.az_count

  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name        = "${local.name_prefix}-public-${count.index + 1}"
    Tier        = "public"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Private Subnets (one per AZ)
# CIDR: 10.0.{100+index}.0/24
##############################################################################

resource "aws_subnet" "private" {
  count = var.az_count

  vpc_id            = aws_vpc.main.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 8, 100 + count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name        = "${local.name_prefix}-private-${count.index + 1}"
    Tier        = "private"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Route Tables — Public subnets route through IGW
##############################################################################

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name        = "${local.name_prefix}-rtb-public"
    Project     = var.project
    Environment = var.environment
  }
}

resource "aws_route_table_association" "public" {
  count          = var.az_count
  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

##############################################################################
# Security Group — ALB
#   Inbound:  port 80 from anywhere (+ port 443 if cert provided)
#   Outbound: all traffic (ALB needs to reach Lambda service endpoint)
##############################################################################

resource "aws_security_group" "alb" {
  name        = "${local.name_prefix}-alb-sg"
  description = "Allow HTTP/HTTPS inbound to ALB"
  vpc_id      = aws_vpc.main.id

  ingress {
    description = "HTTP from internet"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  dynamic "ingress" {
    for_each = local.has_cert ? [1] : []
    content {
      description = "HTTPS from internet"
      from_port   = 443
      to_port     = 443
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
    }
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "${local.name_prefix}-alb-sg"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Application Load Balancer (internet-facing)
##############################################################################

resource "aws_lb" "main" {
  name               = "${local.name_prefix}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id

  enable_deletion_protection = false # set true in prod

  tags = {
    Name        = "${local.name_prefix}-alb"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Target Group — Lambda type
# No port/protocol needed for lambda targets; health check uses Lambda response
##############################################################################

resource "aws_lb_target_group" "upload_api" {
  name        = "${local.name_prefix}-upload-tg"
  target_type = "lambda"

  health_check {
    enabled             = true
    path                = "/health"
    healthy_threshold   = 2
    unhealthy_threshold = 2
    interval            = 30
    # Note: protocol, timeout, and matcher are NOT valid for target_type = "lambda"
  }

  tags = {
    Name        = "${local.name_prefix}-upload-tg"
    Project     = var.project
    Environment = var.environment
  }
}

##############################################################################
# Target Group Attachment — bind the Lambda ARN
##############################################################################

resource "aws_lb_target_group_attachment" "upload_api" {
  target_group_arn = aws_lb_target_group.upload_api.arn
  target_id        = var.upload_lambda_arn
  depends_on       = [aws_lb_target_group.upload_api]
}

##############################################################################
# Listener — HTTP :80
#   • No cert  → forward directly to Lambda target group (dev mode)
#   • Has cert → redirect to HTTPS
##############################################################################

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = local.has_cert ? "redirect" : "forward"

    dynamic "redirect" {
      for_each = local.has_cert ? [1] : []
      content {
        protocol    = "HTTPS"
        port        = "443"
        status_code = "HTTP_301"
      }
    }

    dynamic "forward" {
      for_each = local.has_cert ? [] : [1]
      content {
        target_group {
          arn = aws_lb_target_group.upload_api.arn
        }
      }
    }
  }
}

##############################################################################
# Listener — HTTPS :443 (only created when certificate_arn is provided)
##############################################################################

resource "aws_lb_listener" "https" {
  count = local.has_cert ? 1 : 0

  load_balancer_arn = aws_lb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = var.certificate_arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.upload_api.arn
  }
}
