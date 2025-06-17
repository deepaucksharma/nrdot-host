variable "environment" {
  description = "Environment name"
  type        = string
  default     = "staging"
}

variable "service_name" {
  description = "Service name"
  type        = string
  default     = "clean-platform"
}

variable "team_name" {
  description = "Team name"
  type        = string
  default     = "platform-team"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "vpc_id" {
  description = "VPC ID for the environment"
  type        = string
  # This will be provided by Grand Central
}

variable "subnet_ids" {
  description = "Subnet IDs for database deployment"
  type        = list(string)
  # This will be provided by Grand Central
}

# Database configuration
variable "db_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "db_allocated_storage" {
  description = "Allocated storage in GB"
  type        = number
  default     = 100
}

variable "db_engine_version" {
  description = "Database engine version"
  type        = string
  default     = "8.0.35"
}

variable "db_backup_retention_period" {
  description = "Backup retention period in days"
  type        = number
  default     = 7
}

variable "db_multi_az" {
  description = "Enable Multi-AZ deployment"
  type        = bool
  default     = true
}

# ElastiCache configuration
variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.micro"
}

variable "redis_num_cache_nodes" {
  description = "Number of cache nodes"
  type        = number
  default     = 2
}

variable "redis_parameter_group_family" {
  description = "Redis parameter group family"
  type        = string
  default     = "redis7"
}

# Monitoring configuration
variable "enable_enhanced_monitoring" {
  description = "Enable enhanced monitoring"
  type        = bool
  default     = true
}

variable "monitoring_interval" {
  description = "Enhanced monitoring interval"
  type        = number
  default     = 60
}

# Tags
variable "common_tags" {
  description = "Common tags to apply to all resources"
  type        = map(string)
  default = {
    ManagedBy   = "terraform"
    Repository  = "clean-platform-implementation"
    CostCenter  = "platform-team"
  }
}