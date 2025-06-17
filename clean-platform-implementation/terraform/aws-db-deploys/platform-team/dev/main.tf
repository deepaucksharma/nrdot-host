# Database configuration for platform-team development environment
# Generated using Tim Allen tool: https://tim-allen.vip.cf.nr-ops.net/

terraform {
  required_version = ">= 1.0"
  
  backend "s3" {
    # Backend config is injected by Grand Central
    # This ensures state is stored in the correct S3 bucket
  }
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    vault = {
      source  = "hashicorp/vault"
      version = "~> 3.0"
    }
  }
}

# Data source for current cell information
data "terraform_remote_state" "cell" {
  backend = "s3"
  config = {
    bucket = "nr-terraform-state-${var.cell_name}"
    key    = "cells/${var.cell_name}/terraform.tfstate"
    region = var.aws_region
  }
}

# Local variables
locals {
  team_name    = "platform-team"
  service_name = "clean-platform"
  environment  = "dev"
  cell_name    = var.cell_name
  
  # Database naming convention
  db_identifier = "${local.team_name}-${local.service_name}-${local.environment}"
  
  # Tags required by db-team
  common_tags = {
    Team         = local.team_name
    Service      = local.service_name
    Environment  = local.environment
    Cell         = local.cell_name
    ManagedBy    = "terraform"
    CostCenter   = var.cost_center
    DataClass    = "internal"
    Compliance   = "none"
  }
}

# Aurora PostgreSQL cluster
module "aurora_postgresql" {
  source = "git::https://source.datanerd.us/db-team/terraform-aurora-postgresql?ref=v2.0.0"
  
  # Cluster identification
  cluster_identifier = local.db_identifier
  
  # Engine configuration
  engine_version = "15.4"
  
  # Instance configuration
  instance_class = "db.t4g.medium"  # Burstable instance for dev
  instance_count = 1
  
  # Database configuration
  database_name   = "platform_db"
  master_username = "platform_admin"
  
  # Network configuration
  vpc_id     = data.terraform_remote_state.cell.outputs.vpc_id
  subnet_ids = data.terraform_remote_state.cell.outputs.database_subnet_ids
  
  # Security
  allowed_security_groups = [data.terraform_remote_state.cell.outputs.eks_worker_security_group_id]
  
  # Backup configuration
  backup_retention_period = 7
  backup_window          = "03:00-04:00"
  maintenance_window     = "sun:04:00-sun:05:00"
  
  # Performance insights
  performance_insights_enabled = true
  performance_insights_retention_period = 7
  
  # Monitoring
  enabled_cloudwatch_logs_exports = ["postgresql"]
  
  # Encryption
  storage_encrypted = true
  kms_key_id       = data.terraform_remote_state.cell.outputs.rds_kms_key_id
  
  # Parameter group settings
  db_cluster_parameter_group_family = "aurora-postgresql15"
  db_cluster_parameters = [
    {
      name  = "shared_preload_libraries"
      value = "pg_stat_statements,pg_hint_plan"
    },
    {
      name  = "log_statement"
      value = "all"
    }
  ]
  
  # Tags
  tags = local.common_tags
}

# Store credentials in Vault
resource "vault_generic_secret" "db_credentials" {
  path = "terraform/${local.team_name}/${local.environment}/${local.cell_name}/${local.service_name}/${local.service_name}-db-endpoint"
  
  data_json = jsonencode({
    endpoint        = module.aurora_postgresql.cluster_endpoint
    reader_endpoint = module.aurora_postgresql.cluster_reader_endpoint
    port           = module.aurora_postgresql.cluster_port
    database_name  = module.aurora_postgresql.database_name
    master_username = module.aurora_postgresql.master_username
    master_password = module.aurora_postgresql.master_password
  })
}

# ElastiCache Redis cluster
module "elasticache_redis" {
  source = "git::https://source.datanerd.us/db-team/terraform-elasticache-redis?ref=v1.5.0"
  
  # Cluster identification
  cluster_id = "${local.db_identifier}-redis"
  
  # Node configuration
  node_type       = "cache.t4g.micro"  # Burstable instance for dev
  number_of_nodes = 1
  
  # Engine configuration
  engine_version = "7.0"
  
  # Network configuration
  vpc_id     = data.terraform_remote_state.cell.outputs.vpc_id
  subnet_ids = data.terraform_remote_state.cell.outputs.cache_subnet_ids
  
  # Security
  allowed_security_groups = [data.terraform_remote_state.cell.outputs.eks_worker_security_group_id]
  
  # Backup configuration
  snapshot_retention_limit = 3
  snapshot_window         = "03:00-05:00"
  
  # Encryption
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token_enabled        = true
  
  # Parameter group
  parameter_group_family = "redis7"
  parameters = [
    {
      name  = "maxmemory-policy"
      value = "allkeys-lru"
    }
  ]
  
  # Tags
  tags = local.common_tags
}

# Store Redis credentials in Vault
resource "vault_generic_secret" "redis_credentials" {
  path = "terraform/${local.team_name}/${local.environment}/${local.cell_name}/${local.service_name}/${local.service_name}-redis-endpoint"
  
  data_json = jsonencode({
    primary_endpoint = module.elasticache_redis.primary_endpoint
    configuration_endpoint = module.elasticache_redis.configuration_endpoint
    auth_token = module.elasticache_redis.auth_token
    port = 6379
  })
}

# Outputs for Grand Central
output "database_endpoint" {
  value = module.aurora_postgresql.cluster_endpoint
  description = "Aurora PostgreSQL cluster endpoint"
}

output "database_reader_endpoint" {
  value = module.aurora_postgresql.cluster_reader_endpoint
  description = "Aurora PostgreSQL reader endpoint"
}

output "redis_endpoint" {
  value = module.elasticache_redis.primary_endpoint
  description = "ElastiCache Redis primary endpoint"
}

output "vault_paths" {
  value = {
    database = vault_generic_secret.db_credentials.path
    redis    = vault_generic_secret.redis_credentials.path
  }
  description = "Vault paths for credentials"
}