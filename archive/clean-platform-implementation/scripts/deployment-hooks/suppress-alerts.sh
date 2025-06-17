#!/bin/bash
# Pre-deployment hook: Suppress alerts during deployment
# This prevents false alerts during rolling updates

set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

# Configuration
ALERT_SUPPRESSION_DURATION="${ALERT_SUPPRESSION_DURATION:-30}"
SERVICE_NAME="${GRAND_CENTRAL_PROJECT:-clean-platform}"
ENVIRONMENT="${GRAND_CENTRAL_ENVIRONMENT:-unknown}"
CELL="${GRAND_CENTRAL_CELL:-unknown}"

log_info "Starting alert suppression for deployment"
log_info "Service: ${SERVICE_NAME}, Environment: ${ENVIRONMENT}, Cell: ${CELL}"

# Create alert suppression via New Relic API
create_alert_suppression() {
    local policy_name="$1"
    local duration="$2"
    
    # Get New Relic API key from environment
    if [[ -z "${NEW_RELIC_API_KEY:-}" ]]; then
        log_error "NEW_RELIC_API_KEY not set"
        return 1
    fi
    
    # Create muting rule
    local response=$(curl -s -X POST "https://api.newrelic.com/graphql" \
        -H "Content-Type: application/json" \
        -H "API-Key: ${NEW_RELIC_API_KEY}" \
        -d @- <<EOF
{
  "query": "mutation {
    alertsMutingRuleCreate(
      accountId: ${NEW_RELIC_ACCOUNT_ID}
      rule: {
        name: \"Deployment: ${SERVICE_NAME} - ${ENVIRONMENT}\"
        description: \"Auto-suppression for deployment via Grand Central\"
        enabled: true
        condition: {
          operator: AND
          conditions: [
            {
              attribute: \"policyName\"
              operator: EQUALS
              values: [\"${policy_name}\"]
            },
            {
              attribute: \"tags.service\"
              operator: EQUALS
              values: [\"${SERVICE_NAME}\"]
            },
            {
              attribute: \"tags.environment\"
              operator: EQUALS
              values: [\"${ENVIRONMENT}\"]
            }
          ]
        }
        schedule: {
          startTime: \"$(date -u +%Y-%m-%dT%H:%M:%S)\"
          endTime: \"$(date -u -d "+${duration} minutes" +%Y-%m-%dT%H:%M:%S)\"
          timeZone: \"UTC\"
        }
      }
    ) {
      id
      name
    }
  }"
}
EOF
)
    
    # Check response
    if echo "${response}" | grep -q "errors"; then
        log_error "Failed to create alert suppression: ${response}"
        return 1
    fi
    
    local rule_id=$(echo "${response}" | jq -r '.data.alertsMutingRuleCreate.id')
    log_info "Created muting rule: ${rule_id}"
    
    # Store rule ID for cleanup
    echo "${rule_id}" > "/tmp/muting_rule_${SERVICE_NAME}_${ENVIRONMENT}.id"
}

# Suppress alerts for all relevant policies
suppress_alerts() {
    local policies=(
        "${SERVICE_NAME}-availability"
        "${SERVICE_NAME}-performance"
        "${SERVICE_NAME}-errors"
        "platform-team-slo"
    )
    
    for policy in "${policies[@]}"; do
        log_info "Suppressing alerts for policy: ${policy}"
        if ! create_alert_suppression "${policy}" "${ALERT_SUPPRESSION_DURATION}"; then
            log_warn "Failed to suppress alerts for policy: ${policy}"
        fi
    done
}

# Main execution
main() {
    log_info "Alert suppression duration: ${ALERT_SUPPRESSION_DURATION} minutes"
    
    # Check if we should skip alert suppression
    if [[ "${SKIP_ALERT_SUPPRESSION:-false}" == "true" ]]; then
        log_info "Skipping alert suppression (SKIP_ALERT_SUPPRESSION=true)"
        exit 0
    fi
    
    # Suppress alerts
    if ! suppress_alerts; then
        log_error "Failed to suppress some alerts"
        # Don't fail deployment if alert suppression fails
        log_warn "Continuing deployment despite alert suppression failures"
    fi
    
    log_info "Alert suppression completed successfully"
}

# Run main function
main "$@"