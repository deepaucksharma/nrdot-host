#!/bin/bash
set -euo pipefail

# Performance testing runner script
# Usage: ./run-performance-tests.sh [test-type] [environment]

TEST_TYPE="${1:-smoke}"
ENVIRONMENT="${2:-staging}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RESULTS_DIR="results/${TIMESTAMP}"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Test configurations
declare -A TEST_CONFIGS=(
    ["smoke"]="--users 10 --spawn-rate 2 --run-time 2m"
    ["load"]="--users 100 --spawn-rate 10 --run-time 10m"
    ["stress"]="--users 500 --spawn-rate 50 --run-time 5m"
    ["spike"]="--users 1000 --spawn-rate 100 --run-time 3m"
    ["soak"]="--users 200 --spawn-rate 5 --run-time 60m"
    ["breakpoint"]="--users 10 --spawn-rate 1 --step-load --step-users 10 --step-time 30s"
)

# Get API endpoint based on environment
get_api_endpoint() {
    case $ENVIRONMENT in
        "local")
            echo "http://localhost:8080"
            ;;
        "dev")
            echo "https://api-dev.platform.example.com"
            ;;
        "staging")
            echo "https://api-staging.platform.example.com"
            ;;
        "prod")
            echo "https://api.platform.example.com"
            ;;
        *)
            echo "Unknown environment: $ENVIRONMENT" >&2
            exit 1
            ;;
    esac
}

API_ENDPOINT=$(get_api_endpoint)

echo -e "${GREEN}Running $TEST_TYPE performance test against $ENVIRONMENT${NC}"
echo -e "API Endpoint: $API_ENDPOINT"
echo -e "Test Config: ${TEST_CONFIGS[$TEST_TYPE]}"

# Create results directory
mkdir -p "$RESULTS_DIR"

# Install dependencies
if ! command -v locust &> /dev/null; then
    echo -e "${YELLOW}Installing Locust...${NC}"
    pip install locust
fi

# Run pre-test health check
echo -e "${YELLOW}Running pre-test health check...${NC}"
if ! curl -sf "${API_ENDPOINT}/health" > /dev/null; then
    echo -e "${RED}API health check failed!${NC}"
    exit 1
fi

# Start monitoring if in non-local environment
if [[ "$ENVIRONMENT" != "local" ]]; then
    echo -e "${YELLOW}Setting up monitoring...${NC}"
    
    # Create Grafana annotation for test start
    GRAFANA_URL="https://grafana.platform.example.com"
    GRAFANA_API_KEY="${GRAFANA_API_KEY:-}"
    
    if [[ -n "$GRAFANA_API_KEY" ]]; then
        curl -X POST "$GRAFANA_URL/api/annotations" \
            -H "Authorization: Bearer $GRAFANA_API_KEY" \
            -H "Content-Type: application/json" \
            -d "{
                \"dashboardId\": 1,
                \"time\": $(date +%s)000,
                \"tags\": [\"performance-test\", \"$TEST_TYPE\"],
                \"text\": \"Started $TEST_TYPE test on $ENVIRONMENT\"
            }"
    fi
fi

# Run the test based on type
case $TEST_TYPE in
    "smoke")
        echo -e "${YELLOW}Running smoke test...${NC}"
        locust \
            --headless \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            ${TEST_CONFIGS[$TEST_TYPE]} \
            --html "$RESULTS_DIR/report.html" \
            --csv "$RESULTS_DIR/results" \
            --loglevel INFO \
            --only-summary
        ;;
        
    "load")
        echo -e "${YELLOW}Running load test...${NC}"
        locust \
            --headless \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            ${TEST_CONFIGS[$TEST_TYPE]} \
            --html "$RESULTS_DIR/report.html" \
            --csv "$RESULTS_DIR/results" \
            --loglevel INFO
        ;;
        
    "stress")
        echo -e "${YELLOW}Running stress test...${NC}"
        # Use StressUser class for stress testing
        locust \
            --headless \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            --class-picker \
            -u 500 -r 50 -t 5m \
            --html "$RESULTS_DIR/report.html" \
            --csv "$RESULTS_DIR/results" \
            --loglevel INFO
        ;;
        
    "spike")
        echo -e "${YELLOW}Running spike test...${NC}"
        # Gradually increase then suddenly spike
        locust \
            --headless \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            --class-picker \
            -u 100 -r 10 -t 1m \
            --html "$RESULTS_DIR/report_warmup.html" \
            --csv "$RESULTS_DIR/results_warmup" \
            --loglevel INFO
            
        # Spike
        locust \
            --headless \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            --class-picker \
            -u 1000 -r 500 -t 2m \
            --html "$RESULTS_DIR/report_spike.html" \
            --csv "$RESULTS_DIR/results_spike" \
            --loglevel INFO
        ;;
        
    "soak")
        echo -e "${YELLOW}Running soak test (this will take a while)...${NC}"
        locust \
            --headless \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            ${TEST_CONFIGS[$TEST_TYPE]} \
            --html "$RESULTS_DIR/report.html" \
            --csv "$RESULTS_DIR/results" \
            --loglevel INFO \
            --reset-stats
        ;;
        
    "breakpoint")
        echo -e "${YELLOW}Running breakpoint test...${NC}"
        # This requires interactive mode
        echo "Starting Locust web UI for breakpoint testing..."
        echo "Access at: http://localhost:8089"
        locust \
            --host "$API_ENDPOINT" \
            -f locustfile.py \
            --web-port 8089
        ;;
        
    *)
        echo -e "${RED}Unknown test type: $TEST_TYPE${NC}"
        echo "Available types: smoke, load, stress, spike, soak, breakpoint"
        exit 1
        ;;
esac

# Analyze results
echo -e "${YELLOW}Analyzing results...${NC}"

# Parse CSV results
if [[ -f "$RESULTS_DIR/results_stats.csv" ]]; then
    echo -e "\n${GREEN}Test Results Summary:${NC}"
    
    # Calculate key metrics
    python3 - <<EOF
import csv
import json

with open('$RESULTS_DIR/results_stats.csv', 'r') as f:
    reader = csv.DictReader(f)
    stats = list(reader)
    
    total_requests = sum(int(s['Request Count']) for s in stats if s['Type'] == '')
    total_failures = sum(int(s['Failure Count']) for s in stats if s['Type'] == '')
    
    # Find aggregate row
    aggregate = next((s for s in stats if s['Name'] == 'Aggregated'), None)
    
    if aggregate:
        print(f"Total Requests: {aggregate['Request Count']}")
        print(f"Total Failures: {aggregate['Failure Count']}")
        print(f"Median Response Time: {aggregate['50%']} ms")
        print(f"90th Percentile: {aggregate['90%']} ms")
        print(f"95th Percentile: {aggregate['95%']} ms")
        print(f"99th Percentile: {aggregate['99%']} ms")
        print(f"RPS: {aggregate['Requests/s']}")
        
        # Check SLA compliance
        p95 = float(aggregate['95%'])
        failure_rate = (int(aggregate['Failure Count']) / int(aggregate['Request Count'])) * 100 if int(aggregate['Request Count']) > 0 else 0
        
        print("\nSLA Compliance:")
        print(f"✓ P95 < 500ms: {'PASS' if p95 < 500 else 'FAIL'} ({p95:.2f} ms)")
        print(f"✓ Error Rate < 1%: {'PASS' if failure_rate < 1 else 'FAIL'} ({failure_rate:.2f}%)")
        
        # Save summary
        summary = {
            'test_type': '$TEST_TYPE',
            'environment': '$ENVIRONMENT',
            'timestamp': '$TIMESTAMP',
            'total_requests': int(aggregate['Request Count']),
            'total_failures': int(aggregate['Failure Count']),
            'median_response_time': float(aggregate['50%']),
            'p90_response_time': float(aggregate['90%']),
            'p95_response_time': float(aggregate['95%']),
            'p99_response_time': float(aggregate['99%']),
            'rps': float(aggregate['Requests/s']),
            'error_rate': failure_rate,
            'sla_p95_pass': p95 < 500,
            'sla_error_rate_pass': failure_rate < 1
        }
        
        with open('$RESULTS_DIR/summary.json', 'w') as f:
            json.dump(summary, f, indent=2)
EOF
fi

# Generate performance report
cat > "$RESULTS_DIR/performance_report.md" <<EOF
# Performance Test Report

**Test Type**: $TEST_TYPE
**Environment**: $ENVIRONMENT
**Timestamp**: $TIMESTAMP
**API Endpoint**: $API_ENDPOINT

## Configuration
\`\`\`
${TEST_CONFIGS[$TEST_TYPE]}
\`\`\`

## Results
See attached HTML report and CSV files for detailed results.

## Recommendations
Based on the test results:
1. Monitor response times during peak hours
2. Consider scaling if P95 > 500ms consistently
3. Investigate any endpoints with high error rates

## Files
- \`report.html\`: Detailed HTML report with graphs
- \`results_stats.csv\`: Response time statistics
- \`results_failures.csv\`: Failure details
- \`summary.json\`: Test summary in JSON format
EOF

# Post-test actions
if [[ "$ENVIRONMENT" != "local" ]]; then
    # Create Grafana annotation for test end
    if [[ -n "$GRAFANA_API_KEY" ]]; then
        curl -X POST "$GRAFANA_URL/api/annotations" \
            -H "Authorization: Bearer $GRAFANA_API_KEY" \
            -H "Content-Type: application/json" \
            -d "{
                \"dashboardId\": 1,
                \"time\": $(date +%s)000,
                \"tags\": [\"performance-test\", \"$TEST_TYPE\"],
                \"text\": \"Completed $TEST_TYPE test on $ENVIRONMENT\"
            }"
    fi
    
    # Upload results to S3
    if command -v aws &> /dev/null; then
        echo -e "${YELLOW}Uploading results to S3...${NC}"
        aws s3 cp "$RESULTS_DIR" "s3://platform-performance-tests/$ENVIRONMENT/$TIMESTAMP/" --recursive
    fi
fi

echo -e "\n${GREEN}Performance test completed!${NC}"
echo -e "Results saved to: $RESULTS_DIR"
echo -e "View HTML report: $RESULTS_DIR/report.html"

# Return exit code based on SLA compliance
if [[ -f "$RESULTS_DIR/summary.json" ]]; then
    SLA_PASS=$(python3 -c "import json; s=json.load(open('$RESULTS_DIR/summary.json')); print(int(s['sla_p95_pass'] and s['sla_error_rate_pass']))")
    if [[ "$SLA_PASS" == "1" ]]; then
        echo -e "${GREEN}✓ All SLAs passed${NC}"
        exit 0
    else
        echo -e "${RED}✗ SLA violations detected${NC}"
        exit 1
    fi
fi