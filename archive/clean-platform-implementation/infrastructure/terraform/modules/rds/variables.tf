variable "identifier" {
  description = "The name of the RDS instance"
  type        = string
}

variable "engine" {
  description = "The database engine"
  type        = string
  default     = "postgres"
}

variable "engine_version" {
  description = "The engine version to use"
  type        = string
  default     = "15.3"
}

variable "family" {
  description = "The DB parameter group family"
  type        = string
  default     = "postgres15"
}

variable "instance_class" {
  description = "The instance type of the RDS instance"
  type        = string
  default     = "db.t3.medium"
}

variable "allocated_storage" {
  description = "The allocated storage in gigabytes"
  type        = number
  default     = 100
}

variable "max_allocated_storage" {
  description = "The upper limit for autoscaling storage in gigabytes"
  type        = number
  default     = 500
}

variable "storage_type" {
  description = "Storage type (gp2, gp3, io1)"
  type        = string
  default     = "gp3"
}

variable "database_name" {
  description = "The name of the database to create"
  type        = string
  default     = ""
}

variable "master_username" {
  description = "Username for the master DB user"
  type        = string
  default     = "postgres"
}

variable "master_password" {
  description = "Password for the master DB user"
  type        = string
  sensitive   = true
}

variable "port" {
  description = "The port on which the DB accepts connections"
  type        = number
  default     = 5432
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs"
  type        = list(string)
}

variable "allowed_cidr_blocks" {
  description = "List of CIDR blocks allowed to connect"
  type        = list(string)
  default     = []
}

variable "allowed_security_groups" {
  description = "List of security groups allowed to connect"
  type        = list(string)
  default     = []
}

variable "backup_retention_period" {
  description = "The days to retain backups for"
  type        = number
  default     = 7
}

variable "backup_window" {
  description = "The daily time range during which automated backups are created"
  type        = string
  default     = "03:00-04:00"
}

variable "maintenance_window" {
  description = "The window to perform maintenance in"
  type        = string
  default     = "sun:04:00-sun:05:00"
}

variable "multi_az" {
  description = "Specifies if the RDS instance is multi-AZ"
  type        = bool
  default     = true
}

variable "deletion_protection" {
  description = "If the DB instance should have deletion protection enabled"
  type        = bool
  default     = true
}

variable "skip_final_snapshot" {
  description = "Determines whether a final DB snapshot is created before deletion"
  type        = bool
  default     = false
}

variable "enabled_cloudwatch_logs_exports" {
  description = "List of log types to enable for exporting to CloudWatch logs"
  type        = list(string)
  default     = ["postgresql"]
}

variable "performance_insights_enabled" {
  description = "Specifies whether Performance Insights are enabled"
  type        = bool
  default     = true
}

variable "performance_insights_retention_period" {
  description = "The amount of time in days to retain Performance Insights data"
  type        = number
  default     = 7
}

variable "apply_immediately" {
  description = "Specifies whether any database modifications are applied immediately"
  type        = bool
  default     = false
}

variable "parameters" {
  description = "A list of DB parameters to apply"
  type = list(object({
    name  = string
    value = string
  }))
  default = []
}

variable "kms_key_id" {
  description = "The ARN for the KMS encryption key"
  type        = string
  default     = ""
}

variable "create_replica" {
  description = "Whether to create a read replica"
  type        = bool
  default     = false
}

variable "replica_count" {
  description = "Number of read replicas to create"
  type        = number
  default     = 1
}

variable "source_db_identifier" {
  description = "Source database identifier for read replica"
  type        = string
  default     = ""
}

variable "replica_instance_class" {
  description = "Instance class for read replicas (defaults to main instance class)"
  type        = string
  default     = ""
}

variable "cpu_threshold" {
  description = "CPU utilization threshold for alarms"
  type        = number
  default     = 80
}

variable "storage_threshold" {
  description = "Free storage threshold in GB for alarms"
  type        = number
  default     = 10
}

variable "connections_threshold" {
  description = "Database connections threshold for alarms"
  type        = number
  default     = 80
}

variable "alarm_actions" {
  description = "List of actions to execute when this alarm transitions to ALARM state"
  type        = list(string)
  default     = []
}

variable "tags" {
  description = "A map of tags to add to all resources"
  type        = map(string)
  default     = {}
}