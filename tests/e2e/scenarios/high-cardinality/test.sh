#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_NAME="high-cardinality"
RESULTS_DIR="../test-results/${TEST_NAME}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Starting High Cardinality E2E Test...${NC}"

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

# Function to get memory usage
get_memory_usage() {
    local container=$1
    docker stats --no-stream --format "{{.MemUsage}}" "$container" 2>/dev/null | awk '{print $1}'
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
check_health "Prometheus" "http://localhost:9090/-/ready" || exit 1
check_health "VictoriaMetrics" "http://localhost:8428/health" || exit 1

# Monitor initial state
echo -e "\n${YELLOW}Recording initial state...${NC}"
INITIAL_MEMORY=$(get_memory_usage "nrdot-cardinality")
echo -e "Initial NRDOT memory: ${GREEN}${INITIAL_MEMORY}${NC}"

# Let metric generator run for a while
echo -e "\n${YELLOW}Generating high cardinality metrics...${NC}"
echo "Waiting for metric generation (2 minutes)..."
sleep 120

# Check cardinality protection
echo -e "\n${YELLOW}Validating cardinality protection...${NC}"

# Get metrics from NRDOT
NRDOT_SERIES=$(curl -s "http://localhost:9090/api/v1/query?query=prometheus_tsdb_symbol_table_size_bytes{job=\"nrdot-metrics\"}" | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
echo -e "NRDOT series count: ${GREEN}${NRDOT_SERIES}${NC}"

# Check if cardinality limits are working
CARDINALITY_DROPS=$(curl -s http://localhost:8889/metrics | grep "otelcol_processor_cardinality_limit_dropped_total" | awk '{print $2}' || echo "0")
if [ "$CARDINALITY_DROPS" != "0" ]; then
    echo -e "✓ Cardinality limiter dropped ${GREEN}${CARDINALITY_DROPS}${NC} series"
else
    echo -e "⚠ ${YELLOW}No series dropped by cardinality limiter${NC}"
fi

# Check memory usage after load
FINAL_MEMORY=$(get_memory_usage "nrdot-cardinality")
echo -e "\n${YELLOW}Memory usage:${NC}"
echo -e "  Initial: ${INITIAL_MEMORY}"
echo -e "  Final: ${GREEN}${FINAL_MEMORY}${NC}"

# Parse memory values for comparison
INITIAL_MB=$(echo "$INITIAL_MEMORY" | sed 's/MiB//')
FINAL_MB=$(echo "$FINAL_MEMORY" | sed 's/MiB//')
MEMORY_INCREASE=$(echo "scale=2; $FINAL_MB - $INITIAL_MB" | bc)
echo -e "  Increase: ${GREEN}${MEMORY_INCREASE}MiB${NC}"

# Check if memory is within limits (should not exceed 512MB limit)
if (( $(echo "$FINAL_MB < 450" | bc -l) )); then
    echo -e "✓ Memory usage within limits"
else
    echo -e "✗ ${RED}Memory usage exceeds expected limits${NC}"
    FAILED=true
fi

# Check metric aggregation
echo -e "\n${YELLOW}Checking metric aggregation...${NC}"
HTTP_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_http_requests_total" | jq '.data.result | length')
if [ "$HTTP_METRICS" -gt 0 ] && [ "$HTTP_METRICS" -lt 1000 ]; then
    echo -e "✓ HTTP metrics aggregated to ${GREEN}$HTTP_METRICS${NC} series"
else
    echo -e "✗ ${RED}Metric aggregation may not be working properly${NC}"
    FAILED=true
fi

# Compare with VictoriaMetrics
echo -e "\n${YELLOW}Comparing with VictoriaMetrics...${NC}"
VM_SERIES=$(curl -s "http://localhost:8428/api/v1/query?query=vm_rows" | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
echo -e "VictoriaMetrics series count: ${GREEN}${VM_SERIES}${NC}"

# Check NRDOT internal metrics
echo -e "\n${YELLOW}NRDOT internal metrics:${NC}"
RECEIVED=$(curl -s http://localhost:8889/metrics | grep "otelcol_receiver_accepted_metric_points{receiver=\"otlp\"" | awk '{print $2}' || echo "0")
PROCESSED=$(curl -s http://localhost:8889/metrics | grep "otelcol_processor_processed_metric_points{processor=\"batch\"" | awk '{print $2}' || echo "0")
EXPORTED=$(curl -s http://localhost:8889/metrics | grep "otelcol_exporter_sent_metric_points{exporter=\"prometheus\"" | awk '{print $2}' || echo "0")

echo -e "  Received: ${GREEN}${RECEIVED}${NC} metric points"
echo -e "  Processed: ${GREEN}${PROCESSED}${NC} metric points"
echo -e "  Exported: ${GREEN}${EXPORTED}${NC} metric points"

# Check processor performance
echo -e "\n${YELLOW}Processor performance:${NC}"
BATCH_TIMEOUT=$(curl -s http://localhost:8889/metrics | grep "otelcol_processor_batch_timeout_trigger_send" | awk '{print $2}' || echo "0")
BATCH_SIZE=$(curl -s http://localhost:8889/metrics | grep "otelcol_processor_batch_batch_size_trigger_send" | awk '{print $2}' || echo "0")
echo -e "  Batch timeout triggers: ${GREEN}${BATCH_TIMEOUT}${NC}"
echo -e "  Batch size triggers: ${GREEN}${BATCH_SIZE}${NC}"

# Get profiling data if available
if curl -sf "http://localhost:6060/debug/pprof/" > /dev/null 2>&1; then
    echo -e "\n${YELLOW}Collecting profiling data...${NC}"
    curl -s "http://localhost:6060/debug/pprof/heap" > "$RESULTS_DIR/heap.pprof"
    curl -s "http://localhost:6060/debug/pprof/goroutine" > "$RESULTS_DIR/goroutine.pprof"
    echo -e "✓ Profiling data saved"
fi

# Generate report
echo -e "\n${YELLOW}Generating test report...${NC}"
cat > "$RESULTS_DIR/report.json" <<EOF
{
  "test": "$TEST_NAME",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "status": "${FAILED:-false}",
  "results": {
    "cardinality": {
      "nrdot_series": ${NRDOT_SERIES:-0},
      "dropped_series": ${CARDINALITY_DROPS:-0},
      "aggregated_series": $HTTP_METRICS
    },
    "memory": {
      "initial_mb": ${INITIAL_MB:-0},
      "final_mb": ${FINAL_MB:-0},
      "increase_mb": ${MEMORY_INCREASE:-0}
    },
    "metrics": {
      "received": ${RECEIVED:-0},
      "processed": ${PROCESSED:-0},
      "exported": ${EXPORTED:-0}
    },
    "comparison": {
      "victoriametrics_series": ${VM_SERIES:-0}
    }
  }
}
EOF

# Collect logs
echo "Collecting logs..."
docker-compose logs > "$RESULTS_DIR/docker-compose.log" 2>&1

# Save metrics file if exists
docker cp nrdot-cardinality:/tmp/high_cardinality_metrics.json "$RESULTS_DIR/metrics_sample.json" 2>/dev/null || true

# Cleanup if not in debug mode
if [ -z "$NRDOT_DEBUG" ]; then
    echo "Cleaning up..."
    docker-compose down -v
fi

if [ -z "$FAILED" ]; then
    echo -e "\n${GREEN}High Cardinality E2E Test PASSED${NC}"
    exit 0
else
    echo -e "\n${RED}High Cardinality E2E Test FAILED${NC}"
    exit 1
fi