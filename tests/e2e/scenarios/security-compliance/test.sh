#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_NAME="security-compliance"
RESULTS_DIR="../test-results/${TEST_NAME}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Starting Security Compliance E2E Test...${NC}"

# Create results directory
mkdir -p "$RESULTS_DIR"

# Function to check service health
check_health() {
    local service=$1
    local url=$2
    local max_attempts=30
    local attempt=0

    echo -n "Checking $service health..."
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "$url" > /dev/null 2>&1; then
            echo -e " ${GREEN}OK${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    echo -e " ${RED}FAILED${NC}"
    return 1
}

# Start services
echo "Starting services..."
cd "$SCRIPT_DIR"
docker-compose up -d

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 15

# Check service health
check_health "NRDOT" "http://localhost:13133" || exit 1
check_health "Vulnerable App" "http://localhost:5000/health" || exit 1
check_health "Elasticsearch" "http://localhost:9200" || exit 1
check_health "Jaeger" "http://localhost:16686" || exit 1

# Generate traffic with secrets
echo -e "\n${YELLOW}Generating traffic with sensitive data...${NC}"

# Home endpoint
curl -s http://localhost:5000/ > /dev/null

# Login attempts
curl -s -X POST http://localhost:5000/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"secret123"}' > /dev/null

curl -s -X POST http://localhost:5000/login \
    -H "Content-Type: application/json" \
    -d '{"username":"hacker","password":"wrong_password"}' > /dev/null

# Payment processing
curl -s -X POST http://localhost:5000/api/payment \
    -H "Content-Type: application/json" \
    -d '{"card_number":"4111111111111111","amount":99.99}' > /dev/null

# AWS endpoint
curl -s http://localhost:5000/api/aws > /dev/null

# GitHub endpoint
curl -s http://localhost:5000/api/github > /dev/null

# Generate logs with secrets
curl -s http://localhost:5000/generate-logs > /dev/null

# Wait for processing
echo "Waiting for telemetry processing..."
sleep 30

# Validate secret redaction in traces
echo -e "\n${YELLOW}Validating secret redaction in traces...${NC}"

# Get traces from Jaeger
TRACES=$(curl -s "http://localhost:16686/api/traces?service=vulnerable-app&limit=10")
TRACE_COUNT=$(echo "$TRACES" | jq '.data | length')

if [ "$TRACE_COUNT" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$TRACE_COUNT${NC} traces"
    
    # Check for redacted secrets in traces
    TRACE_ID=$(echo "$TRACES" | jq -r '.data[0].traceID')
    TRACE_DATA=$(curl -s "http://localhost:16686/api/traces/$TRACE_ID")
    
    # Check various secret patterns
    SECRETS_FOUND=0
    
    # Check for unredacted passwords
    if echo "$TRACE_DATA" | grep -q "super_secret_password_123"; then
        echo -e "✗ ${RED}Found unredacted password in traces${NC}"
        SECRETS_FOUND=$((SECRETS_FOUND + 1))
    else
        echo -e "✓ Passwords properly redacted"
    fi
    
    # Check for unredacted API keys
    if echo "$TRACE_DATA" | grep -q "sk-1234567890abcdef"; then
        echo -e "✗ ${RED}Found unredacted API key in traces${NC}"
        SECRETS_FOUND=$((SECRETS_FOUND + 1))
    else
        echo -e "✓ API keys properly redacted"
    fi
    
    # Check for unredacted AWS credentials
    if echo "$TRACE_DATA" | grep -q "AKIAIOSFODNN7EXAMPLE"; then
        echo -e "✗ ${RED}Found unredacted AWS access key in traces${NC}"
        SECRETS_FOUND=$((SECRETS_FOUND + 1))
    else
        echo -e "✓ AWS credentials properly redacted"
    fi
    
    # Check for redaction markers
    REDACTED_COUNT=$(echo "$TRACE_DATA" | grep -o "REDACTED" | wc -l)
    echo -e "✓ Found ${GREEN}$REDACTED_COUNT${NC} redaction markers"
    
else
    echo -e "✗ ${RED}No traces found${NC}"
    FAILED=true
fi

# Validate secret redaction in logs
echo -e "\n${YELLOW}Validating secret redaction in logs...${NC}"

# Wait a bit for logs to be indexed
sleep 10

# Check Elasticsearch for logs
ES_LOGS=$(curl -s -X GET "http://localhost:9200/security-logs/_search?size=100" \
    -H "Content-Type: application/json" \
    -d '{
        "query": {
            "match_all": {}
        }
    }')

LOG_COUNT=$(echo "$ES_LOGS" | jq '.hits.total.value')
if [ "$LOG_COUNT" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$LOG_COUNT${NC} log entries"
    
    # Check for unredacted secrets in logs
    LOG_SECRETS=0
    
    if echo "$ES_LOGS" | grep -q "super_secret_password_123"; then
        echo -e "✗ ${RED}Found unredacted password in logs${NC}"
        LOG_SECRETS=$((LOG_SECRETS + 1))
    fi
    
    if echo "$ES_LOGS" | grep -q "ghp_1234567890abcdefghijklmnopqrstuvwxyz"; then
        echo -e "✗ ${RED}Found unredacted GitHub token in logs${NC}"
        LOG_SECRETS=$((LOG_SECRETS + 1))
    fi
    
    if [ "$LOG_SECRETS" -eq 0 ]; then
        echo -e "✓ All secrets properly redacted in logs"
    fi
else
    echo -e "⚠ ${YELLOW}No logs found in Elasticsearch${NC}"
fi

# Check security metrics
echo -e "\n${YELLOW}Checking security metrics...${NC}"
SECURITY_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_security_spans_total" | jq '.data.result | length')
if [ "$SECURITY_METRICS" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$SECURITY_METRICS${NC} security metrics"
fi

# Check NRDOT internal metrics
echo -e "\n${YELLOW}Checking NRDOT performance...${NC}"
NRDOT_METRICS=$(curl -s http://localhost:8888/metrics | grep -c "^otelcol_" || true)
echo -e "✓ Found ${GREEN}$NRDOT_METRICS${NC} NRDOT internal metrics"

# Check processor metrics
REDACTION_PROCESSED=$(curl -s http://localhost:8889/metrics | grep "otelcol_processor_redaction_processed_total" | awk '{print $2}' || echo "0")
echo -e "✓ Redaction processor handled ${GREEN}${REDACTION_PROCESSED}${NC} items"

# Generate report
echo -e "\n${YELLOW}Generating test report...${NC}"
cat > "$RESULTS_DIR/report.json" <<EOF
{
  "test": "$TEST_NAME",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "status": "${FAILED:-false}",
  "results": {
    "traces": {
      "count": $TRACE_COUNT,
      "secrets_found": ${SECRETS_FOUND:-0},
      "redaction_markers": ${REDACTED_COUNT:-0}
    },
    "logs": {
      "count": ${LOG_COUNT:-0},
      "secrets_found": ${LOG_SECRETS:-0}
    },
    "metrics": {
      "security_metrics": $SECURITY_METRICS
    },
    "nrdot": {
      "internal_metrics": $NRDOT_METRICS,
      "redaction_processed": ${REDACTION_PROCESSED:-0}
    }
  }
}
EOF

# Collect logs
echo "Collecting logs..."
docker-compose logs > "$RESULTS_DIR/docker-compose.log" 2>&1

# Save security alerts if any
if [ -f "/tmp/security_alerts.json" ]; then
    docker cp nrdot-security:/tmp/security_alerts.json "$RESULTS_DIR/security_alerts.json" 2>/dev/null || true
fi

# Cleanup if not in debug mode
if [ -z "$NRDOT_DEBUG" ]; then
    echo "Cleaning up..."
    docker-compose down -v
fi

if [ -z "$FAILED" ] && [ "${SECRETS_FOUND:-0}" -eq 0 ] && [ "${LOG_SECRETS:-0}" -eq 0 ]; then
    echo -e "\n${GREEN}Security Compliance E2E Test PASSED${NC}"
    exit 0
else
    echo -e "\n${RED}Security Compliance E2E Test FAILED${NC}"
    exit 1
fi