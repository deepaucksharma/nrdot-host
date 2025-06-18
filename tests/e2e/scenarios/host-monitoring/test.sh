#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_NAME="host-monitoring"
RESULTS_DIR="../test-results/${TEST_NAME}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Starting Host Monitoring E2E Test...${NC}"

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
check_health "Prometheus" "http://localhost:9090/-/ready" || exit 1
check_health "Node Exporter" "http://localhost:9100/metrics" || exit 1

# Generate load
echo -e "\n${YELLOW}Generating system load...${NC}"
bash load-generator/generate.sh

# Wait for metrics collection
echo "Waiting for metrics collection..."
sleep 30

# Validate host metrics
echo -e "\n${YELLOW}Validating host metrics...${NC}"

# Check CPU metrics
CPU_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_system_cpu_utilization" | jq '.data.result | length')
if [ "$CPU_METRICS" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$CPU_METRICS${NC} CPU metrics"
    CPU_VALUE=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_system_cpu_utilization" | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
    echo -e "  CPU utilization: ${GREEN}${CPU_VALUE}%${NC}"
else
    echo -e "✗ ${RED}No CPU metrics found${NC}"
    FAILED=true
fi

# Check memory metrics
MEMORY_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_system_memory_usage" | jq '.data.result | length')
if [ "$MEMORY_METRICS" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$MEMORY_METRICS${NC} memory metrics"
    MEM_VALUE=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_system_memory_usage" | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "0")
    echo -e "  Memory usage: ${GREEN}${MEM_VALUE} bytes${NC}"
else
    echo -e "✗ ${RED}No memory metrics found${NC}"
    FAILED=true
fi

# Check disk metrics
DISK_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_system_disk_io" | jq '.data.result | length')
if [ "$DISK_METRICS" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$DISK_METRICS${NC} disk I/O metrics"
else
    echo -e "✗ ${RED}No disk metrics found${NC}"
    FAILED=true
fi

# Check network metrics
NETWORK_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_system_network_io" | jq '.data.result | length')
if [ "$NETWORK_METRICS" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$NETWORK_METRICS${NC} network metrics"
else
    echo -e "✗ ${RED}No network metrics found${NC}"
    FAILED=true
fi

# Check container metrics
echo -e "\n${YELLOW}Validating container metrics...${NC}"
CONTAINER_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_container_cpu_usage_seconds_total" | jq '.data.result | length')
if [ "$CONTAINER_METRICS" -gt 0 ]; then
    echo -e "✓ Found ${GREEN}$CONTAINER_METRICS${NC} container CPU metrics"
    
    # Check specific containers
    STRESS_METRICS=$(curl -s "http://localhost:9090/api/v1/query?query=nrdot_container_cpu_usage_seconds_total{container_name=~\"stress.*\"}" | jq '.data.result | length')
    echo -e "  Stress container metrics: ${GREEN}$STRESS_METRICS${NC}"
else
    echo -e "✗ ${RED}No container metrics found${NC}"
    FAILED=true
fi

# Compare with node-exporter
echo -e "\n${YELLOW}Comparing with node-exporter...${NC}"
NODE_CPU=$(curl -s "http://localhost:9090/api/v1/query?query=node_cpu_seconds_total" | jq '.data.result | length')
if [ "$NODE_CPU" -gt 0 ] && [ "$CPU_METRICS" -gt 0 ]; then
    echo -e "✓ Both NRDOT and node-exporter collecting CPU metrics"
else
    echo -e "✗ ${RED}Metric collection mismatch${NC}"
fi

# Check NRDOT internal metrics
echo -e "\n${YELLOW}Checking NRDOT internal metrics...${NC}"
NRDOT_METRICS=$(curl -s http://localhost:8888/metrics | grep -c "^otelcol_" || true)
echo -e "✓ Found ${GREEN}$NRDOT_METRICS${NC} NRDOT internal metrics"

# Check memory usage
NRDOT_MEMORY=$(curl -s "http://localhost:8889/metrics" | grep "process_runtime_memstats_sys_bytes" | awk '{print $2}' || echo "0")
echo -e "  NRDOT memory usage: ${GREEN}$(echo "scale=2; $NRDOT_MEMORY / 1024 / 1024" | bc) MB${NC}"

# Generate report
echo -e "\n${YELLOW}Generating test report...${NC}"
cat > "$RESULTS_DIR/report.json" <<EOF
{
  "test": "$TEST_NAME",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "status": "${FAILED:-false}",
  "results": {
    "host_metrics": {
      "cpu": $CPU_METRICS,
      "memory": $MEMORY_METRICS,
      "disk": $DISK_METRICS,
      "network": $NETWORK_METRICS
    },
    "container_metrics": {
      "total": $CONTAINER_METRICS,
      "stress_containers": ${STRESS_METRICS:-0}
    },
    "comparison": {
      "node_exporter_cpu": $NODE_CPU
    },
    "nrdot": {
      "internal_metrics": $NRDOT_METRICS,
      "memory_mb": $(echo "scale=2; $NRDOT_MEMORY / 1024 / 1024" | bc)
    }
  }
}
EOF

# Collect logs
echo "Collecting logs..."
docker-compose logs > "$RESULTS_DIR/docker-compose.log" 2>&1

# Cleanup if not in debug mode
if [ -z "$NRDOT_DEBUG" ]; then
    echo "Cleaning up..."
    docker-compose down -v
fi

if [ -z "$FAILED" ]; then
    echo -e "\n${GREEN}Host Monitoring E2E Test PASSED${NC}"
    exit 0
else
    echo -e "\n${RED}Host Monitoring E2E Test FAILED${NC}"
    exit 1
fi