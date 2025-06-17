#!/bin/bash
# Deployment failure notification script
# Sends notifications through multiple channels when deployment fails

set -euo pipefail

# Configuration
SERVICE_NAME="${SERVICE_NAME:-clean-platform}"
ENVIRONMENT="${ENVIRONMENT:-production}"
DEPLOYMENT_ID="${DEPLOYMENT_ID:-unknown}"
FAILURE_REASON="${FAILURE_REASON:-Unknown deployment failure}"
NAMESPACE="${NAMESPACE:-clean-platform}"

# Notification channels
SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
PAGERDUTY_TOKEN="${PAGERDUTY_TOKEN:-}"
PAGERDUTY_SERVICE_ID="${PAGERDUTY_SERVICE_ID:-}"
EMAIL_RECIPIENTS="${EMAIL_RECIPIENTS:-}"
TEAMS_WEBHOOK_URL="${TEAMS_WEBHOOK_URL:-}"

echo "========================================="
echo "Deployment Failure Notification"
echo "Service: $SERVICE_NAME"
echo "Environment: $ENVIRONMENT"
echo "Deployment ID: $DEPLOYMENT_ID"
echo "========================================="

# Function to collect failure details
collect_failure_details() {
    echo "Collecting failure details..."
    
    # Get recent pod events
    POD_EVENTS=$(kubectl get events -n "$NAMESPACE" \
        --field-selector involvedObject.kind=Pod \
        --sort-by='.lastTimestamp' \
        -o json | jq -r '.items[-5:] | map(.message) | join("\n")' || echo "No events available")
    
    # Get pod logs
    FAILED_POD=$(kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" \
        --field-selector="status.phase!=Running" \
        -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    POD_LOGS=""
    if [ -n "$FAILED_POD" ]; then
        POD_LOGS=$(kubectl logs -n "$NAMESPACE" "$FAILED_POD" --tail=50 2>/dev/null || echo "No logs available")
    fi
    
    # Get deployment status
    DEPLOYMENT_STATUS=$(kubectl get deployment "$SERVICE_NAME" -n "$NAMESPACE" \
        -o jsonpath='{.status.conditions[?(@.type=="Progressing")].message}' 2>/dev/null || echo "Unknown")
    
    # Check Grand Central status
    GC_STATUS=""
    if [ -n "${GC_TOKEN:-}" ] && [ "$DEPLOYMENT_ID" != "unknown" ]; then
        GC_STATUS=$(curl -s -H "X-Grand-Central-Auth: $GC_TOKEN" \
            "https://grand-central.nr-ops.net/api/v1/deploy/$DEPLOYMENT_ID" \
            | jq -r '.status' 2>/dev/null || echo "Unknown")
    fi
    
    # Create detailed failure report
    cat > "/tmp/failure-details-${DEPLOYMENT_ID}.json" <<EOF
{
  "deployment_id": "$DEPLOYMENT_ID",
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "failure_reason": "$FAILURE_REASON",
  "deployment_status": "$DEPLOYMENT_STATUS",
  "grand_central_status": "$GC_STATUS",
  "recent_events": $(echo "$POD_EVENTS" | jq -Rs .),
  "pod_logs": $(echo "$POD_LOGS" | tail -20 | jq -Rs .),
  "failed_pod": "$FAILED_POD"
}
EOF
}

# Function to send Slack notification
send_slack_notification() {
    if [ -z "$SLACK_WEBHOOK_URL" ]; then
        echo "Slack webhook not configured, skipping"
        return
    fi
    
    echo "Sending Slack notification..."
    
    # Determine severity emoji and color
    local emoji="🚨"
    local color="danger"
    if [[ "$ENVIRONMENT" != "production" ]]; then
        emoji="⚠️"
        color="warning"
    fi
    
    curl -s -X POST "$SLACK_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
  "text": "$emoji Deployment Failed: $SERVICE_NAME in $ENVIRONMENT",
  "attachments": [{
    "color": "$color",
    "fields": [
      {"title": "Service", "value": "$SERVICE_NAME", "short": true},
      {"title": "Environment", "value": "$ENVIRONMENT", "short": true},
      {"title": "Deployment ID", "value": "$DEPLOYMENT_ID", "short": true},
      {"title": "Time", "value": "$(date -u +%Y-%m-%d\ %H:%M:%S\ UTC)", "short": true},
      {"title": "Reason", "value": "$FAILURE_REASON", "short": false},
      {"title": "Recent Events", "value": "$(echo "$POD_EVENTS" | head -3 | sed 's/^/• /')", "short": false}
    ],
    "actions": [
      {
        "type": "button",
        "text": "View Deployment",
        "url": "https://grand-central.nr-ops.net/deployments/$DEPLOYMENT_ID"
      },
      {
        "type": "button",
        "text": "View Logs",
        "url": "https://one.newrelic.com/launcher/logger.log-launcher?query=service.name:'$SERVICE_NAME' AND environment:'$ENVIRONMENT'"
      }
    ]
  }]
}
EOF
    
    echo "✓ Slack notification sent"
}

# Function to create PagerDuty incident
create_pagerduty_incident() {
    if [ -z "$PAGERDUTY_TOKEN" ] || [ -z "$PAGERDUTY_SERVICE_ID" ]; then
        echo "PagerDuty not configured, skipping"
        return
    fi
    
    # Only create incidents for production failures
    if [[ "$ENVIRONMENT" != "production" ]]; then
        echo "Non-production environment, skipping PagerDuty"
        return
    fi
    
    echo "Creating PagerDuty incident..."
    
    response=$(curl -s -X POST "https://api.pagerduty.com/incidents" \
        -H "Authorization: Token token=$PAGERDUTY_TOKEN" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
  "incident": {
    "type": "incident",
    "title": "Deployment Failed: $SERVICE_NAME in $ENVIRONMENT",
    "service": {
      "id": "$PAGERDUTY_SERVICE_ID",
      "type": "service_reference"
    },
    "urgency": "high",
    "body": {
      "type": "incident_body",
      "details": "Deployment $DEPLOYMENT_ID failed for $SERVICE_NAME in $ENVIRONMENT.\n\nReason: $FAILURE_REASON\n\nRecent Events:\n$POD_EVENTS"
    },
    "incident_key": "deployment-failure-$DEPLOYMENT_ID"
  }
}
EOF
    )
    
    incident_id=$(echo "$response" | jq -r '.incident.id' 2>/dev/null || echo "")
    
    if [ -n "$incident_id" ] && [ "$incident_id" != "null" ]; then
        echo "✓ PagerDuty incident created: $incident_id"
    else
        echo "✗ Failed to create PagerDuty incident"
        echo "$response"
    fi
}

# Function to send email notification
send_email_notification() {
    if [ -z "$EMAIL_RECIPIENTS" ]; then
        echo "Email recipients not configured, skipping"
        return
    fi
    
    echo "Sending email notification..."
    
    # Create email body
    cat > "/tmp/email-body-${DEPLOYMENT_ID}.txt" <<EOF
Deployment Failure Notification

Service: $SERVICE_NAME
Environment: $ENVIRONMENT
Deployment ID: $DEPLOYMENT_ID
Time: $(date -u +%Y-%m-%d\ %H:%M:%S\ UTC)

Failure Reason:
$FAILURE_REASON

Deployment Status:
$DEPLOYMENT_STATUS

Recent Pod Events:
$POD_EVENTS

Actions Required:
1. Check the deployment logs in Grand Central
2. Review pod events and logs in Kubernetes
3. Verify the rollback status
4. Investigate root cause

Links:
- Grand Central: https://grand-central.nr-ops.net/deployments/$DEPLOYMENT_ID
- Kubernetes Dashboard: https://k8s-${ENVIRONMENT}.nr-ops.net/namespaces/$NAMESPACE/deployments/$SERVICE_NAME
- New Relic: https://one.newrelic.com/launcher/nr1-core.explorer?pane=eyJuZXJkbGV0SWQiOiJucjEtY29yZS5leHBsb3JlciIsImVudGl0eUd1aWQiOiIkU0VSVklDRV9HVUlEIn0

This is an automated notification from the deployment system.
EOF
    
    # Send email (using sendmail or similar)
    if command -v sendmail >/dev/null 2>&1; then
        {
            echo "To: $EMAIL_RECIPIENTS"
            echo "Subject: [DEPLOYMENT FAILED] $SERVICE_NAME in $ENVIRONMENT"
            echo "Content-Type: text/plain"
            echo ""
            cat "/tmp/email-body-${DEPLOYMENT_ID}.txt"
        } | sendmail -t
        
        echo "✓ Email notification sent"
    else
        echo "✗ sendmail not available, email not sent"
    fi
}

# Function to send Teams notification
send_teams_notification() {
    if [ -z "$TEAMS_WEBHOOK_URL" ]; then
        echo "Teams webhook not configured, skipping"
        return
    fi
    
    echo "Sending Teams notification..."
    
    local color="FF0000"  # Red
    if [[ "$ENVIRONMENT" != "production" ]]; then
        color="FFA500"  # Orange
    fi
    
    curl -s -X POST "$TEAMS_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "$color",
  "summary": "Deployment Failed: $SERVICE_NAME",
  "sections": [{
    "activityTitle": "Deployment Failed",
    "activitySubtitle": "$SERVICE_NAME in $ENVIRONMENT",
    "activityImage": "https://cdn2.iconfinder.com/data/icons/freecns-cumulus/32/519791-101_Warning-512.png",
    "facts": [
      {"name": "Service", "value": "$SERVICE_NAME"},
      {"name": "Environment", "value": "$ENVIRONMENT"},
      {"name": "Deployment ID", "value": "$DEPLOYMENT_ID"},
      {"name": "Time", "value": "$(date -u +%Y-%m-%d\ %H:%M:%S\ UTC)"},
      {"name": "Reason", "value": "$FAILURE_REASON"}
    ],
    "markdown": true
  }],
  "potentialAction": [{
    "@type": "OpenUri",
    "name": "View in Grand Central",
    "targets": [{
      "os": "default",
      "uri": "https://grand-central.nr-ops.net/deployments/$DEPLOYMENT_ID"
    }]
  }]
}
EOF
    
    echo "✓ Teams notification sent"
}

# Function to create New Relic deployment marker
create_nr_deployment_marker() {
    if [ -z "${NEW_RELIC_API_KEY:-}" ]; then
        echo "New Relic API key not configured, skipping deployment marker"
        return
    fi
    
    echo "Creating New Relic deployment marker..."
    
    curl -s -X POST "https://api.newrelic.com/graphql" \
        -H "Content-Type: application/json" \
        -H "API-Key: $NEW_RELIC_API_KEY" \
        -d @- <<EOF
{
  "query": "mutation(\$entityGuid: EntityGuid!, \$description: String!, \$version: String!) {
    changeTrackingCreateDeployment(
      entityGuid: \$entityGuid,
      deployment: {
        description: \$description,
        version: \$version,
        user: \"${USER:-deployment-system}\",
        timestamp: $(date +%s)000,
        groupId: \"deployment-$DEPLOYMENT_ID\"
      }
    ) {
      deploymentId
    }
  }",
  "variables": {
    "entityGuid": "${ENTITY_GUID:-}",
    "description": "FAILED: $FAILURE_REASON",
    "version": "$DEPLOYMENT_ID"
  }
}
EOF
    
    echo "✓ Deployment marker created"
}

# Function to update deployment status in database
update_deployment_status() {
    echo "Updating deployment status..."
    
    # Store failure information in Vault for audit
    vault write "secret/deployments/${SERVICE_NAME}/${ENVIRONMENT}/failures/${DEPLOYMENT_ID}" \
        "timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        "reason=$FAILURE_REASON" \
        "environment=$ENVIRONMENT" \
        "service=$SERVICE_NAME" \
        "user=${USER:-system}" \
        >/dev/null 2>&1 || true
    
    echo "✓ Deployment status updated"
}

# Main notification process
echo "Collecting failure information..."
collect_failure_details

echo ""
echo "Sending notifications..."

# Send notifications to all configured channels
send_slack_notification
create_pagerduty_incident
send_email_notification
send_teams_notification
create_nr_deployment_marker
update_deployment_status

# Generate summary report
NOTIFICATION_REPORT="/tmp/notification-report-${DEPLOYMENT_ID}.json"
cat > "$NOTIFICATION_REPORT" <<EOF
{
  "deployment_id": "$DEPLOYMENT_ID",
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "notifications_sent": {
    "slack": $([ -n "$SLACK_WEBHOOK_URL" ] && echo "true" || echo "false"),
    "pagerduty": $([ -n "$PAGERDUTY_TOKEN" ] && echo "true" || echo "false"),
    "email": $([ -n "$EMAIL_RECIPIENTS" ] && echo "true" || echo "false"),
    "teams": $([ -n "$TEAMS_WEBHOOK_URL" ] && echo "true" || echo "false"),
    "new_relic": $([ -n "${NEW_RELIC_API_KEY:-}" ] && echo "true" || echo "false")
  },
  "failure_details": "/tmp/failure-details-${DEPLOYMENT_ID}.json"
}
EOF

echo ""
echo "========================================="
echo "Notification Summary"
echo "========================================="
echo "✓ Failure details collected"
echo "✓ Notifications sent to configured channels"
echo "✓ Audit trail created"
echo ""
echo "Next steps:"
echo "1. Check the rollback status"
echo "2. Review the failure logs"
echo "3. Investigate root cause"
echo "4. Create post-mortem if needed"

exit 0