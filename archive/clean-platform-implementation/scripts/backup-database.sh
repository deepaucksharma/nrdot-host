#!/bin/bash
set -euo pipefail

# Database backup script with encryption and S3 upload
# Usage: ./backup-database.sh [environment]

ENVIRONMENT="${1:-prod}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_NAME="platform-${ENVIRONMENT}-${TIMESTAMP}"
BACKUP_DIR="/tmp/backups"
S3_BUCKET="platform-backups-${ENVIRONMENT}"
RETENTION_DAYS=30

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Starting database backup for environment: ${ENVIRONMENT}${NC}"

# Load environment configuration
if [[ -f ".env.${ENVIRONMENT}" ]]; then
    source ".env.${ENVIRONMENT}"
else
    echo -e "${RED}Environment file .env.${ENVIRONMENT} not found${NC}"
    exit 1
fi

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# Function to cleanup on exit
cleanup() {
    echo -e "${YELLOW}Cleaning up temporary files...${NC}"
    rm -f "${BACKUP_DIR}/${BACKUP_NAME}*"
}
trap cleanup EXIT

# Get database credentials from AWS Secrets Manager
get_db_credentials() {
    aws secretsmanager get-secret-value \
        --secret-id "platform/${ENVIRONMENT}/database" \
        --query 'SecretString' \
        --output text | jq -r '.password'
}

DB_PASSWORD=$(get_db_credentials)
DB_HOST="${DB_HOST:-platform-${ENVIRONMENT}.cluster-abc123.us-east-1.rds.amazonaws.com}"
DB_NAME="${DB_NAME:-platform}"
DB_USER="${DB_USER:-platform_admin}"

# Perform database backup
echo -e "${YELLOW}Creating database dump...${NC}"
PGPASSWORD="${DB_PASSWORD}" pg_dump \
    -h "${DB_HOST}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -Fc \
    -v \
    --no-password \
    -f "${BACKUP_DIR}/${BACKUP_NAME}.dump" 2>&1 | while read line; do
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] $line"
    done

# Check if backup was successful
if [[ ! -f "${BACKUP_DIR}/${BACKUP_NAME}.dump" ]]; then
    echo -e "${RED}Backup file not created!${NC}"
    exit 1
fi

BACKUP_SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_NAME}.dump" | cut -f1)
echo -e "${GREEN}Backup created successfully. Size: ${BACKUP_SIZE}${NC}"

# Create backup metadata
cat > "${BACKUP_DIR}/${BACKUP_NAME}.metadata.json" <<EOF
{
    "backup_name": "${BACKUP_NAME}",
    "environment": "${ENVIRONMENT}",
    "timestamp": "${TIMESTAMP}",
    "database_host": "${DB_HOST}",
    "database_name": "${DB_NAME}",
    "backup_size": "${BACKUP_SIZE}",
    "backup_tool": "pg_dump",
    "backup_format": "custom",
    "retention_days": ${RETENTION_DAYS}
}
EOF

# Compress backup
echo -e "${YELLOW}Compressing backup...${NC}"
gzip -9 "${BACKUP_DIR}/${BACKUP_NAME}.dump"

# Encrypt backup
echo -e "${YELLOW}Encrypting backup...${NC}"
# Get encryption key from AWS KMS
DATA_KEY=$(aws kms generate-data-key \
    --key-id "alias/platform-backup-key" \
    --key-spec AES_256 \
    --output json)

PLAINTEXT_KEY=$(echo "${DATA_KEY}" | jq -r '.Plaintext')
ENCRYPTED_KEY=$(echo "${DATA_KEY}" | jq -r '.CiphertextBlob')

# Encrypt file using openssl
openssl enc -aes-256-cbc \
    -in "${BACKUP_DIR}/${BACKUP_NAME}.dump.gz" \
    -out "${BACKUP_DIR}/${BACKUP_NAME}.dump.gz.enc" \
    -pass pass:"${PLAINTEXT_KEY}"

# Save encrypted key
echo "${ENCRYPTED_KEY}" > "${BACKUP_DIR}/${BACKUP_NAME}.key.enc"

# Upload to S3
echo -e "${YELLOW}Uploading to S3...${NC}"

# Upload encrypted backup
aws s3 cp \
    "${BACKUP_DIR}/${BACKUP_NAME}.dump.gz.enc" \
    "s3://${S3_BUCKET}/database/${BACKUP_NAME}.dump.gz.enc" \
    --storage-class STANDARD_IA \
    --server-side-encryption aws:kms \
    --sse-kms-key-id "alias/platform-backup-key"

# Upload encrypted key
aws s3 cp \
    "${BACKUP_DIR}/${BACKUP_NAME}.key.enc" \
    "s3://${S3_BUCKET}/database/${BACKUP_NAME}.key.enc"

# Upload metadata
aws s3 cp \
    "${BACKUP_DIR}/${BACKUP_NAME}.metadata.json" \
    "s3://${S3_BUCKET}/database/${BACKUP_NAME}.metadata.json"

# Set lifecycle policy for retention
aws s3api put-object-tagging \
    --bucket "${S3_BUCKET}" \
    --key "database/${BACKUP_NAME}.dump.gz.enc" \
    --tagging "TagSet=[{Key=RetentionDays,Value=${RETENTION_DAYS}},{Key=BackupType,Value=scheduled}]"

# Verify upload
echo -e "${YELLOW}Verifying backup upload...${NC}"
aws s3 ls "s3://${S3_BUCKET}/database/${BACKUP_NAME}*" --human-readable

# Clean up old backups
echo -e "${YELLOW}Cleaning up old backups...${NC}"
CUTOFF_DATE=$(date -d "${RETENTION_DAYS} days ago" +%Y-%m-%d)

aws s3api list-objects-v2 \
    --bucket "${S3_BUCKET}" \
    --prefix "database/" \
    --query "Contents[?LastModified<='${CUTOFF_DATE}'].Key" \
    --output text | tr '\t' '\n' | while read key; do
        if [[ -n "${key}" ]]; then
            echo "Deleting old backup: ${key}"
            aws s3 rm "s3://${S3_BUCKET}/${key}"
        fi
    done

# Send notification
send_notification() {
    local status=$1
    local message=$2
    
    # Send to SNS topic
    aws sns publish \
        --topic-arn "arn:aws:sns:us-east-1:123456789012:platform-backup-notifications" \
        --subject "Database Backup ${status} - ${ENVIRONMENT}" \
        --message "${message}"
    
    # Send to Slack webhook (if configured)
    if [[ -n "${SLACK_WEBHOOK_URL:-}" ]]; then
        curl -X POST "${SLACK_WEBHOOK_URL}" \
            -H 'Content-Type: application/json' \
            -d "{
                \"text\": \"Database Backup ${status}\",
                \"attachments\": [{
                    \"color\": \"$([ \"${status}\" = \"Success\" ] && echo \"good\" || echo \"danger\")\",
                    \"fields\": [
                        {\"title\": \"Environment\", \"value\": \"${ENVIRONMENT}\", \"short\": true},
                        {\"title\": \"Backup Name\", \"value\": \"${BACKUP_NAME}\", \"short\": true},
                        {\"title\": \"Size\", \"value\": \"${BACKUP_SIZE}\", \"short\": true},
                        {\"title\": \"Message\", \"value\": \"${message}\", \"short\": false}
                    ]
                }]
            }"
    fi
}

# Record backup in database
record_backup() {
    kubectl run -it --rm record-backup --image=postgres:15 --restart=Never -- psql \
        -h "${DB_HOST}" \
        -U "${DB_USER}" \
        -d "${DB_NAME}" \
        -c "INSERT INTO backup_history (backup_name, backup_type, environment, size_bytes, location, status, created_at) 
            VALUES ('${BACKUP_NAME}', 'scheduled', '${ENVIRONMENT}', 
                    $(stat -c%s "${BACKUP_DIR}/${BACKUP_NAME}.dump.gz" 2>/dev/null || echo 0), 
                    's3://${S3_BUCKET}/database/${BACKUP_NAME}.dump.gz.enc', 
                    'completed', NOW());"
}

# Send success notification
send_notification "Success" "Database backup completed successfully. Backup: ${BACKUP_NAME}, Size: ${BACKUP_SIZE}"

echo -e "${GREEN}Database backup completed successfully!${NC}"
echo -e "Backup name: ${BACKUP_NAME}"
echo -e "Location: s3://${S3_BUCKET}/database/"
echo -e "Retention: ${RETENTION_DAYS} days"

# Record metrics
aws cloudwatch put-metric-data \
    --namespace "Platform/Backup" \
    --metric-name "BackupSize" \
    --value $(stat -c%s "${BACKUP_DIR}/${BACKUP_NAME}.dump.gz" 2>/dev/null || echo 0) \
    --unit Bytes \
    --dimensions Environment="${ENVIRONMENT}",BackupType=database

aws cloudwatch put-metric-data \
    --namespace "Platform/Backup" \
    --metric-name "BackupDuration" \
    --value $SECONDS \
    --unit Seconds \
    --dimensions Environment="${ENVIRONMENT}",BackupType=database

exit 0