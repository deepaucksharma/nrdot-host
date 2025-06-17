#!/bin/bash
# Post-deployment hook to restore alerts after deployment
# Complements suppress-alerts.sh for complete alert lifecycle management

set -euo pipefail

# Configuration
SERVICE_NAME="${SERVICE_NAME:-clean-platform}"
ENVIRONMENT="${ENVIRONMENT:-production}"
NR_ACCOUNT_ID="${NR_ACCOUNT_ID:-}"
NR_API_KEY="${NR_API_KEY:-}"
MUTING_RULES_FILE="/tmp/muting-rules-${SERVICE_NAME}-${ENVIRONMENT}.json"

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common-functions.sh" 2>/dev/null || true

# Validation
if [ -z "$NR_ACCOUNT_ID" ] || [ -z "$NR_API_KEY" ]; then
    echo "ERROR: NEW_RELIC_ACCOUNT_ID and NEW_RELIC_API_KEY must be set"
    exit 1
fi

# Function to delete muting rule
delete_muting_rule() {
    local rule_id="$1"
    
    echo "Deleting muting rule: $rule_id"
    
    response=$(curl -s -X DELETE \
        "https://api.newrelic.com/graphql" \
        -H "Content-Type: application/json" \
        -H "API-Key: $NR_API_KEY" \
        -d @- <<EOF
{
  "query": "mutation(\$accountId: Int!, \$ruleId: ID!) {
    alertsMutingRuleDelete(accountId: \$accountId, id: \$ruleId) {
      id
    }
  }",
  "variables": {
    "accountId": $NR_ACCOUNT_ID,
    "ruleId": "$rule_id"
  }
}
EOF
    )
    
    # Check for errors
    if echo "$response" | jq -e '.errors' > /dev/null; then
        echo "WARNING: Failed to delete muting rule $rule_id"
        echo "$response" | jq '.errors'
        return 1
    fi
    
    echo "Successfully deleted muting rule: $rule_id"
    return 0
}

# Function to check deployment status
check_deployment_status() {
    echo "Checking deployment status..."
    
    # This would integrate with Grand Central API
    # For now, we'll check basic health
    local health_check_url="${HEALTH_CHECK_URL:-http://localhost:8080/health}"
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -sf "$health_check_url" > /dev/null; then
            echo "Deployment health check passed"
            return 0
        fi
        
        echo "Health check attempt $attempt/$max_attempts failed, retrying..."
        sleep 10
        ((attempt++))
    done
    
    echo "ERROR: Deployment health check failed after $max_attempts attempts"
    return 1
}

# Function to validate error rates before restoring
check_error_rates() {
    echo "Checking current error rates..."
    
    response=$(curl -s -X POST \
        "https://api.newrelic.com/graphql" \
        -H "Content-Type: application/json" \
        -H "API-Key: $NR_API_KEY" \
        -d @- <<EOF
{
  "query": "query(\$accountId: Int!, \$entityGuid: EntityGuid!) {
    actor {
      entity(guid: \$entityGuid) {
        nrdbQuery(nrql: \"SELECT percentage(count(*), WHERE error = true) FROM Transaction WHERE appName = '$SERVICE_NAME' SINCE 5 minutes ago\") {
          results
        }
      }
    }
  }",
  "variables": {
    "accountId": $NR_ACCOUNT_ID,
    "entityGuid": "${ENTITY_GUID:-}"
  }
}
EOF
    )
    
    error_rate=$(echo "$response" | jq -r '.data.actor.entity.nrdbQuery.results[0].percentage' 2>/dev/null || echo "0")
    
    # Check if error rate is acceptable (less than 5%)
    if (( $(echo "$error_rate > 5" | bc -l) )); then
        echo "WARNING: High error rate detected: ${error_rate}%"
        echo "Consider keeping alerts suppressed and investigating"
        return 1
    fi
    
    echo "Error rate is acceptable: ${error_rate}%"
    return 0
}

# Main restoration process
echo "========================================="
echo "Post-Deployment Alert Restoration"
echo "Service: $SERVICE_NAME"
echo "Environment: $ENVIRONMENT"
echo "========================================="

# Check if muting rules file exists
if [ ! -f "$MUTING_RULES_FILE" ]; then
    echo "No muting rules file found at: $MUTING_RULES_FILE"
    echo "Alerts may not have been suppressed or were already restored"
    exit 0
fi

# Load muting rule IDs
echo "Loading muting rules from: $MUTING_RULES_FILE"
MUTING_RULE_IDS=$(jq -r '.[]' "$MUTING_RULES_FILE" 2>/dev/null || echo "")

if [ -z "$MUTING_RULE_IDS" ]; then
    echo "No muting rules found to restore"
    exit 0
fi

# Check deployment status first
if ! check_deployment_status; then
    echo "ERROR: Deployment health check failed, keeping alerts suppressed"
    exit 1
fi

# Check error rates
if ! check_error_rates; then
    echo "WARNING: High error rates detected, consider manual review"
    # Continue anyway, but log the warning
fi

# Delete each muting rule
echo "Restoring alerts by removing muting rules..."
FAILED_DELETIONS=0

while IFS= read -r rule_id; do
    if [ -n "$rule_id" ]; then
        if ! delete_muting_rule "$rule_id"; then
            ((FAILED_DELETIONS++))
        fi
    fi
done <<< "$MUTING_RULE_IDS"

# Clean up muting rules file
rm -f "$MUTING_RULES_FILE"

# Create audit log entry
AUDIT_LOG="/var/log/alert-restoration.log"
if [ -w "$(dirname "$AUDIT_LOG")" ]; then
    cat >> "$AUDIT_LOG" <<EOF
$(date -u +%Y-%m-%dT%H:%M:%SZ) - Alert Restoration
Service: $SERVICE_NAME
Environment: $ENVIRONMENT
User: ${USER:-unknown}
Deployment ID: ${DEPLOYMENT_ID:-unknown}
Rules Restored: $(echo "$MUTING_RULE_IDS" | wc -l)
Failed Deletions: $FAILED_DELETIONS
EOF
fi

# Report status
if [ $FAILED_DELETIONS -gt 0 ]; then
    echo ""
    echo "WARNING: Failed to delete $FAILED_DELETIONS muting rules"
    echo "Manual cleanup may be required"
    exit 1
fi

echo ""
echo "✓ Successfully restored all alerts"
echo "✓ Monitoring is now active for $SERVICE_NAME in $ENVIRONMENT"

# Send notification
if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
    curl -s -X POST "$SLACK_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
  "text": "✅ Alerts restored for ${SERVICE_NAME} in ${ENVIRONMENT}",
  "attachments": [{
    "color": "good",
    "fields": [
      {"title": "Service", "value": "${SERVICE_NAME}", "short": true},
      {"title": "Environment", "value": "${ENVIRONMENT}", "short": true},
      {"title": "Deployment ID", "value": "${DEPLOYMENT_ID:-N/A}", "short": true},
      {"title": "Status", "value": "Monitoring Active", "short": true}
    ]
  }]
}
EOF
fi

exit 0