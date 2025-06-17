terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  
  backend "s3" {
    bucket = "nrdot-terraform-state"
    key    = "nrdot-host/terraform.tfstate"
    region = "us-east-1"
    
    dynamodb_table = "nrdot-terraform-locks"
    encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region
  
  default_tags {
    tags = {
      Project     = "NRDOT-HOST"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}

# Data sources
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# VPC and Networking
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"
  
  name = "${var.project_name}-vpc"
  cidr = var.vpc_cidr
  
  azs             = data.aws_availability_zones.available.names
  private_subnets = var.private_subnets
  public_subnets  = var.public_subnets
  
  enable_nat_gateway = true
  single_nat_gateway = var.environment != "production"
  enable_dns_hostnames = true
  enable_dns_support   = true
  
  # VPC Endpoints for AWS services
  enable_s3_endpoint       = true
  enable_dynamodb_endpoint = true
  
  vpc_endpoint_tags = {
    Name = "${var.project_name}-endpoint"
  }
  
  tags = {
    Name = "${var.project_name}-vpc"
  }
}

# Additional VPC Endpoints
resource "aws_vpc_endpoint" "kinesis" {
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.aws_region}.kinesis-streams"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.vpc_endpoints.id]
  
  private_dns_enabled = true
}

resource "aws_vpc_endpoint" "sqs" {
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.aws_region}.sqs"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.vpc_endpoints.id]
  
  private_dns_enabled = true
}

# Security Groups
resource "aws_security_group" "nrdot_host" {
  name_prefix = "${var.project_name}-host-"
  description = "Security group for NRDOT-HOST instances"
  vpc_id      = module.vpc.vpc_id
  
  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = module.vpc.private_subnets_cidr_blocks
    description = "HTTP API"
  }
  
  ingress {
    from_port   = 9090
    to_port     = 9090
    protocol    = "tcp"
    cidr_blocks = module.vpc.private_subnets_cidr_blocks
    description = "Metrics"
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound"
  }
  
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "vpc_endpoints" {
  name_prefix = "${var.project_name}-endpoints-"
  description = "Security group for VPC endpoints"
  vpc_id      = module.vpc.vpc_id
  
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = module.vpc.private_subnets_cidr_blocks
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# KMS Key for encryption
resource "aws_kms_key" "nrdot" {
  description             = "KMS key for NRDOT-HOST encryption"
  deletion_window_in_days = 10
  enable_key_rotation     = true
  
  tags = {
    Name = "${var.project_name}-kms"
  }
}

resource "aws_kms_alias" "nrdot" {
  name          = "alias/${var.project_name}"
  target_key_id = aws_kms_key.nrdot.key_id
}

# S3 Buckets
resource "aws_s3_bucket" "data_lake" {
  bucket = "${var.project_name}-data-lake-${data.aws_caller_identity.current.account_id}"
  
  tags = {
    Name = "${var.project_name}-data-lake"
  }
}

resource "aws_s3_bucket_versioning" "data_lake" {
  bucket = aws_s3_bucket.data_lake.id
  
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "data_lake" {
  bucket = aws_s3_bucket.data_lake.id
  
  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.nrdot.arn
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "data_lake" {
  bucket = aws_s3_bucket.data_lake.id
  
  rule {
    id     = "transition-to-ia"
    status = "Enabled"
    
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
    
    transition {
      days          = 90
      storage_class = "GLACIER"
    }
  }
}

resource "aws_s3_bucket" "schemas" {
  bucket = "${var.project_name}-schemas-${data.aws_caller_identity.current.account_id}"
  
  tags = {
    Name = "${var.project_name}-schemas"
  }
}

# Kinesis Data Stream
resource "aws_kinesis_stream" "events" {
  name             = "${var.project_name}-events"
  shard_count      = var.kinesis_shard_count
  retention_period = 168 # 7 days
  
  encryption_type = "KMS"
  kms_key_id      = aws_kms_key.nrdot.arn
  
  shard_level_metrics = [
    "IncomingBytes",
    "OutgoingBytes",
    "IncomingRecords",
    "OutgoingRecords",
  ]
  
  stream_mode_details {
    stream_mode = var.kinesis_on_demand ? "ON_DEMAND" : "PROVISIONED"
  }
  
  tags = {
    Name = "${var.project_name}-events-stream"
  }
}

# Kinesis Firehose
resource "aws_kinesis_firehose_delivery_stream" "output" {
  name        = "${var.project_name}-output"
  destination = "extended_s3"
  
  extended_s3_configuration {
    role_arn           = aws_iam_role.firehose.arn
    bucket_arn         = aws_s3_bucket.data_lake.arn
    prefix             = "firehose/year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/"
    error_output_prefix = "errors/"
    
    buffering_size     = 128
    buffering_interval = 300
    
    compression_format = "GZIP"
    
    data_format_conversion_configuration {
      output_format_configuration {
        serializer {
          parquet_ser_de {}
        }
      }
      
      schema_configuration {
        database_name = aws_glue_catalog_database.nrdot.name
        table_name    = aws_glue_catalog_table.events.name
      }
    }
    
    cloudwatch_logging_options {
      enabled         = true
      log_group_name  = aws_cloudwatch_log_group.firehose.name
      log_stream_name = aws_cloudwatch_log_stream.firehose.name
    }
  }
}

# SQS Queue
resource "aws_sqs_queue" "events" {
  name                       = "${var.project_name}-events"
  delay_seconds              = 0
  max_message_size           = 262144
  message_retention_seconds  = 1209600 # 14 days
  receive_wait_time_seconds  = 20
  visibility_timeout_seconds = 300
  
  kms_master_key_id = aws_kms_key.nrdot.id
  
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.events_dlq.arn
    maxReceiveCount     = 3
  })
  
  tags = {
    Name = "${var.project_name}-events-queue"
  }
}

resource "aws_sqs_queue" "events_dlq" {
  name = "${var.project_name}-events-dlq"
  
  kms_master_key_id = aws_kms_key.nrdot.id
  
  tags = {
    Name = "${var.project_name}-events-dlq"
  }
}

# DynamoDB Tables
resource "aws_dynamodb_table" "events" {
  name           = "${var.project_name}-events"
  billing_mode   = var.environment == "production" ? "PROVISIONED" : "PAY_PER_REQUEST"
  read_capacity  = var.environment == "production" ? 100 : null
  write_capacity = var.environment == "production" ? 100 : null
  hash_key       = "id"
  range_key      = "timestamp"
  
  attribute {
    name = "id"
    type = "S"
  }
  
  attribute {
    name = "timestamp"
    type = "N"
  }
  
  attribute {
    name = "event_type"
    type = "S"
  }
  
  global_secondary_index {
    name            = "event_type_index"
    hash_key        = "event_type"
    range_key       = "timestamp"
    projection_type = "ALL"
    read_capacity   = var.environment == "production" ? 50 : null
    write_capacity  = var.environment == "production" ? 50 : null
  }
  
  ttl {
    attribute_name = "expiry"
    enabled        = true
  }
  
  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.nrdot.arn
  }
  
  point_in_time_recovery {
    enabled = var.environment == "production"
  }
  
  tags = {
    Name = "${var.project_name}-events-table"
  }
}

resource "aws_dynamodb_table" "state" {
  name           = "${var.project_name}-state"
  billing_mode   = "PAY_PER_REQUEST"
  hash_key       = "key"
  
  attribute {
    name = "key"
    type = "S"
  }
  
  ttl {
    attribute_name = "expiry"
    enabled        = true
  }
  
  server_side_encryption {
    enabled     = true
    kms_key_arn = aws_kms_key.nrdot.arn
  }
  
  tags = {
    Name = "${var.project_name}-state-table"
  }
}

# ElastiCache Redis Cluster
resource "aws_elasticache_subnet_group" "nrdot" {
  name       = "${var.project_name}-cache"
  subnet_ids = module.vpc.private_subnets
}

resource "aws_elasticache_replication_group" "nrdot" {
  replication_group_id = "${var.project_name}-cache"
  description          = "Redis cluster for NRDOT-HOST"
  
  engine               = "redis"
  engine_version       = "7.0"
  parameter_group_name = "default.redis7.cluster.on"
  port                 = 6379
  
  node_type            = var.elasticache_node_type
  num_cache_clusters   = var.environment == "production" ? 3 : 1
  
  subnet_group_name = aws_elasticache_subnet_group.nrdot.name
  security_group_ids = [aws_security_group.elasticache.id]
  
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token_enabled         = true
  auth_token                 = random_password.elasticache_auth.result
  
  automatic_failover_enabled = var.environment == "production"
  multi_az_enabled           = var.environment == "production"
  
  snapshot_retention_limit = var.environment == "production" ? 7 : 0
  snapshot_window          = "03:00-05:00"
  maintenance_window       = "sun:05:00-sun:07:00"
  
  log_delivery_configuration {
    destination      = aws_cloudwatch_log_group.elasticache.name
    destination_type = "cloudwatch-logs"
    log_format       = "json"
    log_type         = "slow-log"
  }
  
  tags = {
    Name = "${var.project_name}-cache"
  }
}

resource "aws_security_group" "elasticache" {
  name_prefix = "${var.project_name}-elasticache-"
  description = "Security group for ElastiCache"
  vpc_id      = module.vpc.vpc_id
  
  ingress {
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.nrdot_host.id]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "random_password" "elasticache_auth" {
  length  = 32
  special = true
}

resource "aws_secretsmanager_secret" "elasticache_auth" {
  name = "${var.project_name}/elasticache/auth-token"
}

resource "aws_secretsmanager_secret_version" "elasticache_auth" {
  secret_id     = aws_secretsmanager_secret.elasticache_auth.id
  secret_string = random_password.elasticache_auth.result
}

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "nrdot" {
  name              = "/aws/ecs/${var.project_name}"
  retention_in_days = 30
  
  kms_key_id = aws_kms_key.nrdot.arn
}

resource "aws_cloudwatch_log_group" "firehose" {
  name              = "/aws/kinesisfirehose/${var.project_name}"
  retention_in_days = 7
}

resource "aws_cloudwatch_log_stream" "firehose" {
  name           = "S3Delivery"
  log_group_name = aws_cloudwatch_log_group.firehose.name
}

resource "aws_cloudwatch_log_group" "elasticache" {
  name              = "/aws/elasticache/${var.project_name}"
  retention_in_days = 7
}

# Glue Catalog for Data Lake
resource "aws_glue_catalog_database" "nrdot" {
  name = "${var.project_name}_db"
  
  description = "NRDOT data lake database"
}

resource "aws_glue_catalog_table" "events" {
  name          = "events"
  database_name = aws_glue_catalog_database.nrdot.name
  
  table_type = "EXTERNAL_TABLE"
  
  parameters = {
    "projection.enabled"            = "true"
    "projection.year.type"          = "integer"
    "projection.year.range"         = "2024,2030"
    "projection.month.type"         = "integer"
    "projection.month.range"        = "1,12"
    "projection.month.digits"       = "2"
    "projection.day.type"           = "integer"
    "projection.day.range"          = "1,31"
    "projection.day.digits"         = "2"
    "storage.location.template"     = "s3://${aws_s3_bucket.data_lake.id}/firehose/year=$${year}/month=$${month}/day=$${day}/"
  }
  
  storage_descriptor {
    location      = "s3://${aws_s3_bucket.data_lake.id}/firehose/"
    input_format  = "org.apache.hadoop.hive.ql.io.parquet.MapredParquetInputFormat"
    output_format = "org.apache.hadoop.hive.ql.io.parquet.MapredParquetOutputFormat"
    
    ser_de_info {
      serialization_library = "org.apache.hadoop.hive.ql.io.parquet.serde.ParquetHiveSerDe"
    }
    
    columns {
      name = "id"
      type = "string"
    }
    
    columns {
      name = "timestamp"
      type = "timestamp"
    }
    
    columns {
      name = "event_type"
      type = "string"
    }
    
    columns {
      name = "data"
      type = "string"
    }
  }
  
  partition_keys {
    name = "year"
    type = "int"
  }
  
  partition_keys {
    name = "month"
    type = "int"
  }
  
  partition_keys {
    name = "day"
    type = "int"
  }
}

# EventBridge
resource "aws_cloudwatch_event_bus" "nrdot" {
  name = var.project_name
}

resource "aws_cloudwatch_event_rule" "nrdot_events" {
  name           = "${var.project_name}-events-rule"
  description    = "Capture NRDOT events"
  event_bus_name = aws_cloudwatch_event_bus.nrdot.name
  
  event_pattern = jsonencode({
    source = ["nrdot.host"]
  })
}

# Outputs
output "vpc_id" {
  value = module.vpc.vpc_id
}

output "private_subnets" {
  value = module.vpc.private_subnets
}

output "kinesis_stream_name" {
  value = aws_kinesis_stream.events.name
}

output "sqs_queue_url" {
  value = aws_sqs_queue.events.url
}

output "s3_data_lake_bucket" {
  value = aws_s3_bucket.data_lake.id
}

output "elasticache_endpoint" {
  value = aws_elasticache_replication_group.nrdot.primary_endpoint_address
}

output "kms_key_id" {
  value = aws_kms_key.nrdot.id
}