variable "project" {
  type = string
}

variable "environment" {
  type = string
}

# Placeholder: Define VPC, subnets, security groups as needed
# For a Lambda + API Gateway pattern, a VPC is optional but recommended
# for private RDS/ElastiCache access.

output "vpc_id" {
  value = "" # replace with aws_vpc.main.id when implemented
}
