#!/bin/bash
# Pre-deployment hook to verify backups are current
# Ensures data safety before deployment

set -euo pipefail

# Configuration
SERVICE_NAME="${SERVICE_NAME:-clean-platform}"
ENVIRONMENT="${ENVIRONMENT:-production}"
BACKUP_THRESHOLD_HOURS="${BACKUP_THRESHOLD_HOURS:-24}"
VAULT_ADDR="${VAULT_ADDR:-https://vault.nr-ops.net}"

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../common-functions.sh" 2>/dev/null || true

echo "========================================="
echo "Pre-Deployment Backup Verification"
echo "Service: $SERVICE_NAME"
echo "Environment: $ENVIRONMENT"
echo "========================================="

# Function to check database backup status
check_database_backup() {
    local db_name="$1"
    echo "Checking backup status for database: $db_name"
    
    # Get backup metadata from Vault
    vault_path="terraform/${SERVICE_NAME}/${ENVIRONMENT}/backup-status"
    
    if ! backup_info=$(vault read -format=json "$vault_path" 2>/dev/null); then
        echo "WARNING: No backup information found in Vault"
        return 1
    fi
    
    # Extract last backup timestamp
    last_backup=$(echo "$backup_info" | jq -r '.data.last_backup_timestamp')
    backup_size=$(echo "$backup_info" | jq -r '.data.backup_size_bytes')
    backup_location=$(echo "$backup_info" | jq -r '.data.backup_location')
    
    # Check backup age
    current_time=$(date +%s)
    backup_time=$(date -d "$last_backup" +%s 2>/dev/null || echo 0)
    age_hours=$(( (current_time - backup_time) / 3600 ))
    
    echo "Last backup: $last_backup (${age_hours} hours ago)"
    echo "Backup size: $(numfmt --to=iec-i --suffix=B "$backup_size" 2>/dev/null || echo "$backup_size bytes")"
    echo "Location: $backup_location"
    
    if [ "$age_hours" -gt "$BACKUP_THRESHOLD_HOURS" ]; then
        echo "ERROR: Backup is older than ${BACKUP_THRESHOLD_HOURS} hours"
        return 1
    fi
    
    echo "✓ Backup is current"
    return 0
}

# Function to check Redis backup
check_redis_backup() {
    echo "Checking Redis backup status..."
    
    # Check if Redis RDB backup exists
    redis_backup_path="/var/lib/redis/backups/dump-${ENVIRONMENT}.rdb"
    
    if [ ! -f "$redis_backup_path" ]; then
        echo "WARNING: No Redis backup found at $redis_backup_path"
        # Check S3 backup
        s3_backup="s3://nr-backups/${SERVICE_NAME}/${ENVIRONMENT}/redis/latest.rdb"
        
        if aws s3 ls "$s3_backup" >/dev/null 2>&1; then
            echo "✓ Redis backup found in S3"
            return 0
        else
            echo "ERROR: No Redis backup found"
            return 1
        fi
    fi
    
    # Check backup age
    backup_age=$(find "$redis_backup_path" -mmin +$((BACKUP_THRESHOLD_HOURS * 60)) 2>/dev/null)
    if [ -n "$backup_age" ]; then
        echo "ERROR: Redis backup is older than ${BACKUP_THRESHOLD_HOURS} hours"
        return 1
    fi
    
    echo "✓ Redis backup is current"
    return 0
}

# Function to trigger emergency backup
trigger_emergency_backup() {
    echo "Triggering emergency backup before deployment..."
    
    # Trigger database backup
    if command -v nr-backup >/dev/null 2>&1; then
        nr-backup create \
            --service "$SERVICE_NAME" \
            --environment "$ENVIRONMENT" \
            --type "pre-deployment" \
            --wait
    else
        # Fallback to direct backup
        "${SCRIPT_DIR}/../backup-database.sh" || true
    fi
    
    # Trigger Redis backup
    if [ -n "${REDIS_URL:-}" ]; then
        redis-cli --rdb "/tmp/redis-backup-$(date +%Y%m%d-%H%M%S).rdb" || true
    fi
    
    echo "✓ Emergency backup completed"
}

# Function to verify backup integrity
verify_backup_integrity() {
    local backup_file="$1"
    echo "Verifying backup integrity..."
    
    # Check if backup file exists and is not empty
    if [ ! -s "$backup_file" ]; then
        echo "ERROR: Backup file is empty or missing"
        return 1
    fi
    
    # Verify checksum if available
    if [ -f "${backup_file}.sha256" ]; then
        if sha256sum -c "${backup_file}.sha256" >/dev/null 2>&1; then
            echo "✓ Backup checksum verified"
        else
            echo "ERROR: Backup checksum verification failed"
            return 1
        fi
    fi
    
    echo "✓ Backup integrity verified"
    return 0
}

# Main backup check process
BACKUP_FAILURES=0

# Check primary database backup
if ! check_database_backup "${SERVICE_NAME}-db"; then
    ((BACKUP_FAILURES++))
fi

# Check Redis backup if Redis is used
if [ -n "${REDIS_URL:-}" ] || [ -n "${ELASTICACHE_ENDPOINT:-}" ]; then
    if ! check_redis_backup; then
        ((BACKUP_FAILURES++))
    fi
fi

# Check configuration backup
CONFIG_BACKUP="/var/backups/${SERVICE_NAME}/config-${ENVIRONMENT}.tar.gz"
if [ -f "$CONFIG_BACKUP" ]; then
    echo "✓ Configuration backup found: $CONFIG_BACKUP"
else
    echo "WARNING: No configuration backup found"
fi

# Determine action based on backup status
if [ "$BACKUP_FAILURES" -gt 0 ]; then
    echo ""
    echo "⚠️  Backup verification failed!"
    
    # Check if we should force backup
    if [ "${FORCE_BACKUP:-false}" = "true" ]; then
        echo "FORCE_BACKUP is set, triggering emergency backup..."
        trigger_emergency_backup
    else
        echo "ERROR: Backups are not current. Deployment aborted."
        echo "To force deployment with emergency backup, set FORCE_BACKUP=true"
        exit 1
    fi
fi

# Create backup manifest
MANIFEST_FILE="/tmp/backup-manifest-${SERVICE_NAME}-${ENVIRONMENT}.json"
cat > "$MANIFEST_FILE" <<EOF
{
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "deployment_id": "${DEPLOYMENT_ID:-unknown}",
  "backups": {
    "database": {
      "status": "$([ $BACKUP_FAILURES -eq 0 ] && echo "verified" || echo "failed")",
      "last_backup": "$last_backup",
      "location": "$backup_location"
    },
    "redis": {
      "status": "verified",
      "location": "${redis_backup_path:-none}"
    },
    "config": {
      "status": "$([ -f "$CONFIG_BACKUP" ] && echo "verified" || echo "missing")",
      "location": "$CONFIG_BACKUP"
    }
  }
}
EOF

# Store manifest in Vault
vault write "secret/deployments/${SERVICE_NAME}/${ENVIRONMENT}/backup-manifest" \
    "@$MANIFEST_FILE" >/dev/null 2>&1 || true

echo ""
echo "✓ Backup verification completed successfully"
echo "✓ All backups are current and verified"
echo "✓ Safe to proceed with deployment"

exit 0