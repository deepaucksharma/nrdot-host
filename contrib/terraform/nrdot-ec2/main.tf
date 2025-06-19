# Terraform module for deploying NRDOT-HOST on AWS EC2

terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "new_relic_license_key" {
  description = "New Relic license key"
  type        = string
  sensitive   = true
}

variable "new_relic_api_key" {
  description = "New Relic API key for remote configuration"
  type        = string
  sensitive   = true
  default     = ""
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.medium"
}

variable "key_name" {
  description = "EC2 key pair name"
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID for EC2 instance"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for security group"
  type        = string
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "nrdot_version" {
  description = "NRDOT-HOST version to install"
  type        = string
  default     = "latest"
}

# Security group for NRDOT-HOST
resource "aws_security_group" "nrdot_host" {
  name_prefix = "nrdot-host-"
  description = "Security group for NRDOT-HOST monitoring"
  vpc_id      = var.vpc_id

  # Outbound HTTPS for New Relic
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS to New Relic"
  }

  # Outbound DNS
  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "DNS resolution"
  }

  # Optional: API access (restricted to VPC)
  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.selected.cidr_block]
    description = "NRDOT API (internal only)"
  }

  tags = {
    Name        = "nrdot-host-sg"
    Environment = var.environment
  }
}

# IAM role for EC2 instance
resource "aws_iam_role" "nrdot_host" {
  name_prefix = "nrdot-host-"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })

  tags = {
    Name        = "nrdot-host-role"
    Environment = var.environment
  }
}

# IAM policy for CloudWatch and EC2 metadata
resource "aws_iam_role_policy" "nrdot_host" {
  name_prefix = "nrdot-host-"
  role        = aws_iam_role.nrdot_host.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
          "ec2:DescribeTags",
          "cloudwatch:GetMetricStatistics",
          "cloudwatch:ListMetrics"
        ]
        Resource = "*"
      }
    ]
  })
}

# Instance profile
resource "aws_iam_instance_profile" "nrdot_host" {
  name_prefix = "nrdot-host-"
  role        = aws_iam_role.nrdot_host.name
}

# Data source for VPC
data "aws_vpc" "selected" {
  id = var.vpc_id
}

# Data source for latest Amazon Linux 2023 AMI
data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# User data script for NRDOT-HOST installation
locals {
  user_data = templatefile("${path.module}/user-data.sh", {
    new_relic_license_key = var.new_relic_license_key
    new_relic_api_key    = var.new_relic_api_key
    environment          = var.environment
    nrdot_version       = var.nrdot_version
    instance_id         = "INSTANCE_ID_PLACEHOLDER"
  })
}

# EC2 instance with NRDOT-HOST
resource "aws_instance" "nrdot_host" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = var.instance_type
  key_name              = var.key_name
  subnet_id             = var.subnet_id
  vpc_security_group_ids = [aws_security_group.nrdot_host.id]
  iam_instance_profile   = aws_iam_instance_profile.nrdot_host.name

  user_data = base64encode(local.user_data)

  root_block_device {
    volume_type = "gp3"
    volume_size = 30
    encrypted   = true
  }

  metadata_options {
    http_tokens = "required"  # Require IMDSv2
  }

  tags = {
    Name        = "nrdot-host-${var.environment}"
    Environment = var.environment
    Monitoring  = "nrdot-host"
  }
}

# Outputs
output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.nrdot_host.id
}

output "private_ip" {
  description = "Private IP address"
  value       = aws_instance.nrdot_host.private_ip
}

output "security_group_id" {
  description = "Security group ID"
  value       = aws_security_group.nrdot_host.id
}