#!/bin/bash
# Common functions for deployment hooks

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*" >&2
}

# Check if required environment variables are set
check_required_vars() {
    local required_vars=("$@")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_error "Missing required environment variables: ${missing_vars[*]}"
        return 1
    fi
    
    return 0
}

# Retry function with exponential backoff
retry_with_backoff() {
    local max_attempts="${1}"
    local delay="${2}"
    shift 2
    local command=("$@")
    local attempt=1
    
    until [[ ${attempt} -gt ${max_attempts} ]]; do
        if "${command[@]}"; then
            return 0
        fi
        
        log_warn "Command failed. Attempt ${attempt}/${max_attempts}"
        
        if [[ ${attempt} -lt ${max_attempts} ]]; then
            log_info "Retrying in ${delay} seconds..."
            sleep "${delay}"
            delay=$((delay * 2))
        fi
        
        attempt=$((attempt + 1))
    done
    
    log_error "Command failed after ${max_attempts} attempts"
    return 1
}

# Get Vault token for service authentication
get_vault_token() {
    local vault_addr="${VAULT_ADDR:-https://vault-prd1a.r10.us.nr-ops.net:8200}"
    local role="${VAULT_ROLE:-${GRAND_CENTRAL_PROJECT}-${GRAND_CENTRAL_ENVIRONMENT}}"
    
    # Check if we already have a token
    if [[ -n "${VAULT_TOKEN:-}" ]]; then
        return 0
    fi
    
    # Use approle auth if credentials are available
    if [[ -n "${VAULT_ROLE_ID:-}" ]] && [[ -n "${VAULT_SECRET_ID:-}" ]]; then
        local response=$(curl -s -X POST \
            "${vault_addr}/v1/auth/approle/login" \
            -d "{\"role_id\":\"${VAULT_ROLE_ID}\",\"secret_id\":\"${VAULT_SECRET_ID}\"}")
        
        VAULT_TOKEN=$(echo "${response}" | jq -r '.auth.client_token')
        export VAULT_TOKEN
        
        if [[ "${VAULT_TOKEN}" == "null" ]] || [[ -z "${VAULT_TOKEN}" ]]; then
            log_error "Failed to authenticate with Vault"
            return 1
        fi
        
        log_info "Successfully authenticated with Vault"
        return 0
    fi
    
    log_error "No Vault authentication method available"
    return 1
}

# Read secret from Vault
read_vault_secret() {
    local path="$1"
    local field="${2:-value}"
    
    if ! get_vault_token; then
        return 1
    fi
    
    local vault_addr="${VAULT_ADDR:-https://vault-prd1a.r10.us.nr-ops.net:8200}"
    local response=$(curl -s -X GET \
        -H "X-Vault-Token: ${VAULT_TOKEN}" \
        "${vault_addr}/v1/${path}")
    
    local value=$(echo "${response}" | jq -r ".data.${field}")
    
    if [[ "${value}" == "null" ]] || [[ -z "${value}" ]]; then
        log_error "Failed to read secret from Vault path: ${path}"
        return 1
    fi
    
    echo "${value}"
}

# Check Kubernetes deployment status
check_k8s_deployment() {
    local namespace="$1"
    local deployment="$2"
    local timeout="${3:-300}"
    
    log_info "Checking deployment status for ${deployment} in namespace ${namespace}"
    
    if ! kubectl rollout status deployment/"${deployment}" \
        -n "${namespace}" \
        --timeout="${timeout}s"; then
        log_error "Deployment ${deployment} failed to become ready"
        return 1
    fi
    
    log_success "Deployment ${deployment} is ready"
    return 0
}

# Send notification to Slack
send_slack_notification() {
    local webhook_url="$1"
    local message="$2"
    local color="${3:-#36a64f}"  # Default to green
    
    local payload=$(cat <<EOF
{
  "attachments": [
    {
      "color": "${color}",
      "text": "${message}",
      "footer": "Grand Central Deployment",
      "footer_icon": "https://platform.slack-edge.com/img/default_application_icon.png",
      "ts": $(date +%s)
    }
  ]
}
EOF
)
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "${payload}" \
        "${webhook_url}" > /dev/null
}

# Export functions for use in other scripts
export -f log_info log_success log_warn log_error
export -f check_required_vars retry_with_backoff
export -f get_vault_token read_vault_secret
export -f check_k8s_deployment send_slack_notification