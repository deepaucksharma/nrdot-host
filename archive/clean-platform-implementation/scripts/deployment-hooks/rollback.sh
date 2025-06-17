#!/bin/bash
# Deployment rollback script
# Automatically triggered on deployment failure

set -euo pipefail

# Configuration
SERVICE_NAME="${SERVICE_NAME:-clean-platform}"
ENVIRONMENT="${ENVIRONMENT:-production}"
NAMESPACE="${NAMESPACE:-clean-platform}"
DEPLOYMENT_ID="${DEPLOYMENT_ID:-unknown}"
ROLLBACK_TIMEOUT="${ROLLBACK_TIMEOUT:-600}"

# Import Grand Central token
export GC_TOKEN="${GRAND_CENTRAL_TOKEN:-$(vault read -field=token secret/teams/platform-team/grand-central-token 2>/dev/null || echo "")}"

echo "========================================="
echo "Deployment Rollback"
echo "Service: $SERVICE_NAME"
echo "Environment: $ENVIRONMENT"
echo "Failed Deployment: $DEPLOYMENT_ID"
echo "========================================="

# Function to get previous stable version
get_previous_version() {
    echo "Getting previous stable version..."
    
    # Get deployment history from Grand Central
    if [ -n "$GC_TOKEN" ]; then
        history=$(curl -s \
            -H "X-Grand-Central-Auth: $GC_TOKEN" \
            "https://grand-central.nr-ops.net/api/v1/projects/${PROJECT_ORG}/${PROJECT_REPO}/deployments?environment=$ENVIRONMENT&limit=10")
        
        # Find last successful deployment
        previous_version=$(echo "$history" | jq -r '.deployments[] | select(.status == "completed") | .version' | head -1)
        
        if [ -n "$previous_version" ]; then
            echo "Previous stable version: $previous_version"
            echo "$previous_version"
            return
        fi
    fi
    
    # Fallback to Kubernetes rollout history
    previous_version=$(kubectl rollout history deployment/"$SERVICE_NAME" -n "$NAMESPACE" \
        | grep -v "REVISION" | tail -2 | head -1 | awk '{print $1}')
    
    if [ -n "$previous_version" ]; then
        echo "Previous Kubernetes revision: $previous_version"
        echo "$previous_version"
    else
        echo "ERROR: No previous version found"
        return 1
    fi
}

# Function to trigger Grand Central rollback
gc_rollback() {
    local deployment_id="$1"
    local reason="$2"
    
    echo "Triggering Grand Central rollback..."
    
    response=$(curl -s -X POST \
        -H "X-Grand-Central-Auth: $GC_TOKEN" \
        -H "Content-Type: application/json" \
        "https://grand-central.nr-ops.net/api/v1/deploy/${deployment_id}/rollback" \
        -d "{\"reason\": \"$reason\"}")
    
    rollback_id=$(echo "$response" | jq -r '.id')
    
    if [ -n "$rollback_id" ] && [ "$rollback_id" != "null" ]; then
        echo "Grand Central rollback initiated: $rollback_id"
        return 0
    else
        echo "ERROR: Grand Central rollback failed"
        echo "$response"
        return 1
    fi
}

# Function to perform Kubernetes rollback
k8s_rollback() {
    local revision="${1:-}"
    
    echo "Performing Kubernetes rollback..."
    
    if [ -n "$revision" ]; then
        kubectl rollout undo deployment/"$SERVICE_NAME" -n "$NAMESPACE" --to-revision="$revision"
    else
        kubectl rollout undo deployment/"$SERVICE_NAME" -n "$NAMESPACE"
    fi
    
    # Wait for rollout to complete
    echo "Waiting for rollback to complete..."
    if kubectl rollout status deployment/"$SERVICE_NAME" -n "$NAMESPACE" --timeout="${ROLLBACK_TIMEOUT}s"; then
        echo "✓ Kubernetes rollback completed"
        return 0
    else
        echo "ERROR: Kubernetes rollback failed"
        return 1
    fi
}

# Function to restore database if needed
restore_database() {
    echo "Checking if database restore is needed..."
    
    # Check if there were database migrations in the failed deployment
    if [ -f "/tmp/deployment-${DEPLOYMENT_ID}-migrations.log" ]; then
        echo "Database migrations were applied, considering restore..."
        
        # Get backup information
        backup_manifest="/tmp/backup-manifest-${SERVICE_NAME}-${ENVIRONMENT}.json"
        if [ -f "$backup_manifest" ]; then
            backup_location=$(jq -r '.backups.database.location' "$backup_manifest")
            
            if [ -n "$backup_location" ] && [ "$backup_location" != "null" ]; then
                echo "Restoring database from: $backup_location"
                
                # Trigger database restore
                "${SCRIPT_DIR}/../restore-database.sh" "$backup_location" || {
                    echo "WARNING: Database restore failed"
                    return 1
                }
                
                echo "✓ Database restored"
            fi
        fi
    else
        echo "No database changes detected, skipping restore"
    fi
    
    return 0
}

# Function to verify rollback success
verify_rollback() {
    echo "Verifying rollback success..."
    
    # Run health checks
    if "${SCRIPT_DIR}/health-check.sh"; then
        echo "✓ Service is healthy after rollback"
        return 0
    else
        echo "ERROR: Service is not healthy after rollback"
        return 1
    fi
}

# Function to clean up failed deployment artifacts
cleanup_failed_deployment() {
    echo "Cleaning up failed deployment artifacts..."
    
    # Remove temporary files
    rm -f "/tmp/deployment-${DEPLOYMENT_ID}-*" || true
    rm -f "/tmp/muting-rules-${SERVICE_NAME}-${ENVIRONMENT}.json" || true
    
    # Clean up failed pods
    kubectl delete pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" --field-selector="status.phase=Failed" || true
    
    # Clean up orphaned resources
    kubectl delete pvc -n "$NAMESPACE" -l "app=$SERVICE_NAME" --field-selector="status.phase=Released" || true
    
    echo "✓ Cleanup completed"
}

# Main rollback process
ROLLBACK_REASON="${ROLLBACK_REASON:-Automatic rollback due to deployment failure}"
ROLLBACK_SUCCESS=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create rollback record
ROLLBACK_RECORD="/tmp/rollback-${DEPLOYMENT_ID}-$(date +%Y%m%d-%H%M%S).json"
cat > "$ROLLBACK_RECORD" <<EOF
{
  "deployment_id": "$DEPLOYMENT_ID",
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "reason": "$ROLLBACK_REASON",
  "triggered_by": "${USER:-system}",
  "rollback_type": "automatic"
}
EOF

# Step 1: Get previous version
PREVIOUS_VERSION=$(get_previous_version || echo "")

# Step 2: Try Grand Central rollback first
if [ -n "$GC_TOKEN" ] && [ "$DEPLOYMENT_ID" != "unknown" ]; then
    if gc_rollback "$DEPLOYMENT_ID" "$ROLLBACK_REASON"; then
        ROLLBACK_SUCCESS=true
    fi
fi

# Step 3: Fall back to Kubernetes rollback if needed
if [ "$ROLLBACK_SUCCESS" = "false" ]; then
    if k8s_rollback "$PREVIOUS_VERSION"; then
        ROLLBACK_SUCCESS=true
    fi
fi

# Step 4: Restore database if needed
if [ "$ROLLBACK_SUCCESS" = "true" ]; then
    restore_database || echo "WARNING: Database restore failed, manual intervention may be needed"
fi

# Step 5: Verify rollback
if [ "$ROLLBACK_SUCCESS" = "true" ]; then
    if verify_rollback; then
        echo "✓ Rollback verified successfully"
    else
        echo "WARNING: Rollback completed but verification failed"
    fi
fi

# Step 6: Restore alerts
if [ -f "${SCRIPT_DIR}/restore-alerts.sh" ]; then
    echo "Restoring alerts after rollback..."
    "${SCRIPT_DIR}/restore-alerts.sh" || true
fi

# Step 7: Clean up
cleanup_failed_deployment

# Step 8: Create incident if configured
if [ -n "${PAGERDUTY_TOKEN:-}" ] && [ "$ROLLBACK_SUCCESS" = "false" ]; then
    echo "Creating PagerDuty incident..."
    curl -s -X POST "https://api.pagerduty.com/incidents" \
        -H "Authorization: Token token=$PAGERDUTY_TOKEN" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
  "incident": {
    "type": "incident",
    "title": "Deployment rollback failed for $SERVICE_NAME in $ENVIRONMENT",
    "service": {
      "id": "${PAGERDUTY_SERVICE_ID:-P123456}",
      "type": "service_reference"
    },
    "urgency": "high",
    "body": {
      "type": "incident_body",
      "details": "Automatic rollback failed for deployment $DEPLOYMENT_ID. Manual intervention required."
    }
  }
}
EOF
fi

# Generate final report
ROLLBACK_REPORT="/tmp/rollback-report-${DEPLOYMENT_ID}.json"
cat > "$ROLLBACK_REPORT" <<EOF
{
  "deployment_id": "$DEPLOYMENT_ID",
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "rollback_status": "$([ "$ROLLBACK_SUCCESS" = "true" ] && echo "success" || echo "failed")",
  "previous_version": "${PREVIOUS_VERSION:-unknown}",
  "reason": "$ROLLBACK_REASON",
  "duration_seconds": $(($(date +%s) - $(stat -c %Y "$ROLLBACK_RECORD" 2>/dev/null || date +%s)))
}
EOF

# Store rollback report
vault write "secret/deployments/${SERVICE_NAME}/${ENVIRONMENT}/rollback-${DEPLOYMENT_ID}" \
    "@$ROLLBACK_REPORT" >/dev/null 2>&1 || true

# Send notification
if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
    slack_color="warning"
    slack_emoji="⚠️"
    slack_text="Deployment rollback initiated"
    
    if [ "$ROLLBACK_SUCCESS" = "true" ]; then
        slack_color="good"
        slack_emoji="✅"
        slack_text="Deployment rollback completed successfully"
    else
        slack_color="danger"
        slack_emoji="🚨"
        slack_text="Deployment rollback FAILED - Manual intervention required!"
    fi
    
    curl -s -X POST "$SLACK_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
  "text": "$slack_emoji $slack_text",
  "attachments": [{
    "color": "$slack_color",
    "fields": [
      {"title": "Service", "value": "$SERVICE_NAME", "short": true},
      {"title": "Environment", "value": "$ENVIRONMENT", "short": true},
      {"title": "Failed Deployment", "value": "$DEPLOYMENT_ID", "short": true},
      {"title": "Rolled Back To", "value": "${PREVIOUS_VERSION:-unknown}", "short": true},
      {"title": "Reason", "value": "$ROLLBACK_REASON", "short": false}
    ]
  }]
}
EOF
fi

# Exit with appropriate status
if [ "$ROLLBACK_SUCCESS" = "true" ]; then
    echo ""
    echo "✅ Rollback completed successfully"
    echo "✅ Service has been restored to previous version: ${PREVIOUS_VERSION:-unknown}"
    exit 0
else
    echo ""
    echo "🚨 ROLLBACK FAILED!"
    echo "🚨 Manual intervention required"
    echo "🚨 Check the logs and take immediate action"
    exit 1
fi