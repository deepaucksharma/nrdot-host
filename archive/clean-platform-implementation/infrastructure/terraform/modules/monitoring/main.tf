# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "application" {
  name              = "/aws/application/${var.environment}/${var.application_name}"
  retention_in_days = var.log_retention_days
  kms_key_id       = var.kms_key_arn

  tags = merge(
    var.tags,
    {
      Name = "${var.application_name}-${var.environment}-logs"
    }
  )
}

resource "aws_cloudwatch_log_group" "eks" {
  for_each = toset(var.eks_log_types)
  
  name              = "/aws/eks/${var.cluster_name}/cluster/${each.value}"
  retention_in_days = var.log_retention_days
  kms_key_id       = var.kms_key_arn

  tags = merge(
    var.tags,
    {
      Name = "${var.cluster_name}-${each.value}-logs"
    }
  )
}

# SNS Topics for Alerts
resource "aws_sns_topic" "alerts" {
  name              = "${var.application_name}-${var.environment}-alerts"
  kms_master_key_id = var.kms_key_arn

  tags = merge(
    var.tags,
    {
      Name = "${var.application_name}-${var.environment}-alerts"
    }
  )
}

resource "aws_sns_topic_subscription" "alerts_email" {
  for_each = toset(var.alert_email_addresses)

  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = each.value
}

# CloudWatch Dashboard
resource "aws_cloudwatch_dashboard" "main" {
  dashboard_name = "${var.application_name}-${var.environment}"

  dashboard_body = jsonencode({
    widgets = concat(
      # Application metrics
      [
        {
          type   = "metric"
          width  = 12
          height = 6
          x      = 0
          y      = 0
          properties = {
            metrics = [
              ["Platform", "Requests", { stat = "Sum", period = 300 }],
              [".", "Errors", { stat = "Sum", period = 300, yAxis = "right" }]
            ]
            view    = "timeSeries"
            stacked = false
            region  = var.aws_region
            title   = "Request Rate and Errors"
            period  = 300
          }
        },
        {
          type   = "metric"
          width  = 12
          height = 6
          x      = 12
          y      = 0
          properties = {
            metrics = [
              ["Platform", "RequestDuration", { stat = "Average", period = 60 }],
              [".", ".", { stat = "p99", period = 60 }]
            ]
            view    = "timeSeries"
            stacked = false
            region  = var.aws_region
            title   = "Request Duration"
            period  = 300
          }
        }
      ],
      # RDS metrics
      var.enable_rds_monitoring ? [
        {
          type   = "metric"
          width  = 12
          height = 6
          x      = 0
          y      = 6
          properties = {
            metrics = [
              ["AWS/RDS", "CPUUtilization", { "DBInstanceIdentifier" = var.rds_instance_id }],
              [".", "DatabaseConnections", { "DBInstanceIdentifier" = var.rds_instance_id, yAxis = "right" }]
            ]
            view    = "timeSeries"
            stacked = false
            region  = var.aws_region
            title   = "RDS Performance"
            period  = 300
          }
        },
        {
          type   = "metric"
          width  = 12
          height = 6
          x      = 12
          y      = 6
          properties = {
            metrics = [
              ["AWS/RDS", "FreeStorageSpace", { "DBInstanceIdentifier" = var.rds_instance_id }],
              [".", "ReadLatency", { "DBInstanceIdentifier" = var.rds_instance_id, yAxis = "right" }],
              [".", "WriteLatency", { "DBInstanceIdentifier" = var.rds_instance_id, yAxis = "right" }]
            ]
            view    = "timeSeries"
            stacked = false
            region  = var.aws_region
            title   = "RDS Storage and Latency"
            period  = 300
          }
        }
      ] : [],
      # EKS metrics
      var.enable_eks_monitoring ? [
        {
          type   = "metric"
          width  = 12
          height = 6
          x      = 0
          y      = 12
          properties = {
            metrics = [
              ["ContainerInsights", "cluster_node_count", { "ClusterName" = var.cluster_name }],
              [".", "cluster_failed_node_count", { "ClusterName" = var.cluster_name }]
            ]
            view    = "timeSeries"
            stacked = false
            region  = var.aws_region
            title   = "EKS Node Status"
            period  = 300
          }
        },
        {
          type   = "metric"
          width  = 12
          height = 6
          x      = 12
          y      = 12
          properties = {
            metrics = [
              ["ContainerInsights", "pod_cpu_utilization", { "ClusterName" = var.cluster_name }],
              [".", "pod_memory_utilization", { "ClusterName" = var.cluster_name }]
            ]
            view    = "timeSeries"
            stacked = false
            region  = var.aws_region
            title   = "EKS Resource Utilization"
            period  = 300
          }
        }
      ] : []
    )
  })
}

# Application Alarms
resource "aws_cloudwatch_metric_alarm" "high_error_rate" {
  alarm_name          = "${var.application_name}-${var.environment}-high-error-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  threshold           = var.error_rate_threshold
  alarm_description   = "This metric monitors application error rate"
  alarm_actions       = [aws_sns_topic.alerts.arn]

  metric_query {
    id          = "error_rate"
    expression  = "errors / requests * 100"
    label       = "Error Rate"
    return_data = true
  }

  metric_query {
    id = "errors"
    metric {
      metric_name = "Errors"
      namespace   = "Platform"
      period      = "300"
      stat        = "Sum"
    }
  }

  metric_query {
    id = "requests"
    metric {
      metric_name = "Requests"
      namespace   = "Platform"
      period      = "300"
      stat        = "Sum"
    }
  }

  tags = var.tags
}

resource "aws_cloudwatch_metric_alarm" "high_latency" {
  alarm_name          = "${var.application_name}-${var.environment}-high-latency"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "3"
  metric_name         = "RequestDuration"
  namespace           = "Platform"
  period              = "300"
  statistic           = "Average"
  threshold           = var.latency_threshold
  alarm_description   = "This metric monitors application latency"
  alarm_actions       = [aws_sns_topic.alerts.arn]

  tags = var.tags
}

# Log Metric Filters
resource "aws_cloudwatch_log_metric_filter" "error_logs" {
  name           = "${var.application_name}-${var.environment}-error-logs"
  log_group_name = aws_cloudwatch_log_group.application.name
  pattern        = "[timestamp, request_id, level=ERROR, ...]"

  metric_transformation {
    name      = "ErrorLogs"
    namespace = "Platform/${var.environment}"
    value     = "1"
  }
}

resource "aws_cloudwatch_log_metric_filter" "slow_requests" {
  name           = "${var.application_name}-${var.environment}-slow-requests"
  log_group_name = aws_cloudwatch_log_group.application.name
  pattern        = "[timestamp, request_id, level, ... duration > ${var.slow_request_threshold}]"

  metric_transformation {
    name      = "SlowRequests"
    namespace = "Platform/${var.environment}"
    value     = "1"
  }
}

# EventBridge Rules for Automated Actions
resource "aws_cloudwatch_event_rule" "eks_node_failure" {
  count = var.enable_eks_monitoring ? 1 : 0

  name        = "${var.cluster_name}-node-failure"
  description = "Trigger when EKS node fails"

  event_pattern = jsonencode({
    source      = ["aws.eks"]
    detail-type = ["EKS Node Failure"]
    detail = {
      clusterName = [var.cluster_name]
    }
  })

  tags = var.tags
}

resource "aws_cloudwatch_event_target" "sns_node_failure" {
  count = var.enable_eks_monitoring ? 1 : 0

  rule      = aws_cloudwatch_event_rule.eks_node_failure[0].name
  target_id = "SendToSNS"
  arn       = aws_sns_topic.alerts.arn
}

# IAM Role for CloudWatch Logs
resource "aws_iam_role" "cloudwatch_logs" {
  name = "${var.application_name}-${var.environment}-cloudwatch-logs"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "eks.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy" "cloudwatch_logs" {
  name = "${var.application_name}-${var.environment}-cloudwatch-logs"
  role = aws_iam_role.cloudwatch_logs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ]
        Resource = ["arn:aws:logs:*:*:*"]
      }
    ]
  })
}