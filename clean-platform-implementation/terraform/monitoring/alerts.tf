# New Relic alerting configuration for clean-platform
# Deployed via Grand Central Terraform mechanism

terraform {
  required_providers {
    newrelic = {
      source  = "newrelic/newrelic"
      version = "~> 3.0"
    }
  }
}

# Alert policy for the platform
resource "newrelic_alert_policy" "clean_platform" {
  name                = "clean-platform-${var.environment}"
  incident_preference = "PER_CONDITION_AND_TARGET"
}

# Service availability alert
resource "newrelic_nrql_alert_condition" "service_availability" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "Service Availability"
  description = "Alert when service availability drops below SLO"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT percentage(count(*), WHERE response.status < 500) 
      FROM Transaction 
      WHERE appName = 'clean-platform-${var.environment}'
      FACET service
    EOT
  }

  critical {
    operator              = "below"
    threshold             = 99.9
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "below"
    threshold             = 99.95
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  fill_option        = "none"
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# Response time alert (P99)
resource "newrelic_nrql_alert_condition" "response_time_p99" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "Response Time P99"
  description = "Alert when P99 response time exceeds threshold"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT percentile(duration, 99) 
      FROM Transaction 
      WHERE appName = 'clean-platform-${var.environment}'
      FACET service
    EOT
  }

  critical {
    operator              = "above"
    threshold             = 1000  # 1 second
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "above"
    threshold             = 500   # 500ms
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  fill_option        = "none"
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# Error rate alert
resource "newrelic_nrql_alert_condition" "error_rate" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "Error Rate"
  description = "Alert when error rate exceeds threshold"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT percentage(count(*), WHERE error IS true) 
      FROM Transaction 
      WHERE appName = 'clean-platform-${var.environment}'
      FACET service
    EOT
  }

  critical {
    operator              = "above"
    threshold             = 5
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "above"
    threshold             = 2
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  fill_option        = "none"
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# CPU utilization alert
resource "newrelic_nrql_alert_condition" "cpu_utilization" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "CPU Utilization"
  description = "Alert when CPU utilization is too high"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT average(cpuPercent) 
      FROM K8sContainerSample 
      WHERE clusterName = '${var.cluster_name}' 
      AND namespace = 'clean-platform-${var.environment}'
      FACET podName
    EOT
  }

  critical {
    operator              = "above"
    threshold             = 80
    threshold_duration    = 600
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "above"
    threshold             = 70
    threshold_duration    = 600
    threshold_occurrences = "ALL"
  }

  fill_option        = "none"
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# Memory utilization alert
resource "newrelic_nrql_alert_condition" "memory_utilization" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "Memory Utilization"
  description = "Alert when memory utilization is too high"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT average(memoryWorkingSetUtilization) 
      FROM K8sContainerSample 
      WHERE clusterName = '${var.cluster_name}' 
      AND namespace = 'clean-platform-${var.environment}'
      FACET podName
    EOT
  }

  critical {
    operator              = "above"
    threshold             = 85
    threshold_duration    = 600
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "above"
    threshold             = 75
    threshold_duration    = 600
    threshold_occurrences = "ALL"
  }

  fill_option        = "none"
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# Database connection pool alert
resource "newrelic_nrql_alert_condition" "database_connections" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "Database Connection Pool"
  description = "Alert when database connections are exhausted"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT average(provider.databaseConnections.used) / average(provider.databaseConnections.max) * 100
      FROM DatastoreSample 
      WHERE provider = 'PostgresDatabase'
      AND label.service = 'clean-platform-${var.environment}'
    EOT
  }

  critical {
    operator              = "above"
    threshold             = 90
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "above"
    threshold             = 80
    threshold_duration    = 300
    threshold_occurrences = "ALL"
  }

  fill_option        = "static"
  fill_value         = 0
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# Queue depth alert (Redis)
resource "newrelic_nrql_alert_condition" "queue_depth" {
  policy_id = newrelic_alert_policy.clean_platform.id

  name        = "Queue Depth"
  description = "Alert when Redis queue depth is too high"
  enabled     = true

  nrql {
    query = <<-EOT
      SELECT average(provider.keyspace.keys) 
      FROM DatastoreSample 
      WHERE provider = 'RedisDatabase'
      AND label.service = 'clean-platform-${var.environment}'
      AND label.queue = 'data_queue'
    EOT
  }

  critical {
    operator              = "above"
    threshold             = 10000
    threshold_duration    = 600
    threshold_occurrences = "ALL"
  }

  warning {
    operator              = "above"
    threshold             = 5000
    threshold_duration    = 600
    threshold_occurrences = "ALL"
  }

  fill_option        = "static"
  fill_value         = 0
  aggregation_window = 60
  aggregation_method = "event_flow"
  aggregation_delay  = 120
}

# SLO for service availability
resource "newrelic_service_level" "availability_slo" {
  guid        = var.entity_guid
  name        = "Availability SLO"
  description = "99.9% availability over 28 days"

  events {
    account_id = var.account_id
    valid_events {
      from  = "Transaction"
      where = "appName = 'clean-platform-${var.environment}'"
    }
    good_events {
      from  = "Transaction"
      where = "appName = 'clean-platform-${var.environment}' AND response.status < 500"
    }
  }

  objective {
    target = 99.9
    time_window {
      rolling {
        count = 28
        unit  = "DAY"
      }
    }
  }
}

# Notification channels
resource "newrelic_notification_channel" "pagerduty" {
  name = "Platform Team PagerDuty"
  type = "PagerDuty"

  destination_id = var.pagerduty_destination_id
  product        = "IINT"

  property {
    key   = "summary"
    value = "{{#if issueTitle}}{{issueTitle}}{{else}}{{annotations.title.[0]}}{{/if}}"
  }

  property {
    key   = "service"
    value = var.pagerduty_service_id
  }
}

resource "newrelic_notification_channel" "slack" {
  name = "Platform Team Slack"
  type = "SLACK"

  destination_id = var.slack_destination_id
  product        = "IINT"

  property {
    key   = "channelId"
    value = var.slack_channel_id
  }
}

# Workflow for alert notifications
resource "newrelic_workflow" "platform_alerts" {
  name                  = "Platform Team Alert Workflow"
  muting_rules_handling = "NOTIFY_ALL_ISSUES"

  issues_filter {
    name = "Platform Policy Filter"
    type = "FILTER"

    predicate {
      attribute = "labels.policyIds"
      operator  = "CONTAINS"
      values    = [newrelic_alert_policy.clean_platform.id]
    }
  }

  destination {
    channel_id = newrelic_notification_channel.slack.id
  }

  destination {
    channel_id = newrelic_notification_channel.pagerduty.id
    
    # Only page for critical alerts
    notification_triggers = ["CRITICAL"]
  }
}

# Outputs for reference
output "alert_policy_id" {
  value = newrelic_alert_policy.clean_platform.id
  description = "ID of the alert policy"
}

output "slo_id" {
  value = newrelic_service_level.availability_slo.id
  description = "ID of the availability SLO"
}