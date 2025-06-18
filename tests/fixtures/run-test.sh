#!/bin/bash
# NRDOT-HOST End-to-End Test Script

set -e

echo "==================================="
echo "NRDOT-HOST End-to-End Test"
echo "==================================="
echo

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check Docker
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    log_error "docker-compose is not installed"
    exit 1
fi

# Start services
log_info "Starting test environment..."
docker-compose up -d

# Wait for services to be ready
log_info "Waiting for services to start..."
sleep 10

# Check service health
log_info "Checking service health..."

# Check OTel Collector
if curl -f http://localhost:13133/health > /dev/null 2>&1; then
    log_info "✓ OpenTelemetry Collector is healthy"
else
    log_error "✗ OpenTelemetry Collector health check failed"
fi

# Check Prometheus
if curl -f http://localhost:9090/-/healthy > /dev/null 2>&1; then
    log_info "✓ Prometheus is healthy"
else
    log_error "✗ Prometheus health check failed"
fi

# Check Grafana
if curl -f http://localhost:3000/api/health > /dev/null 2>&1; then
    log_info "✓ Grafana is healthy"
else
    log_error "✗ Grafana health check failed"
fi

# Check if metrics are flowing
sleep 5
log_info "Checking metric flow..."

# Query Prometheus for node exporter metrics
METRICS=$(curl -s http://localhost:9090/api/v1/query?query=up | jq -r '.data.result | length')
if [ "$METRICS" -gt 0 ]; then
    log_info "✓ Metrics are being collected ($METRICS series found)"
else
    log_error "✗ No metrics found in Prometheus"
fi

# Check OTel Collector metrics
OTEL_METRICS=$(curl -s http://localhost:8889/metrics | grep -c "^otelcol_" || true)
if [ "$OTEL_METRICS" -gt 0 ]; then
    log_info "✓ OTel Collector is processing data ($OTEL_METRICS internal metrics)"
else
    log_warn "⚠ No OTel Collector internal metrics found"
fi

# Display access information
echo
log_info "Test environment is running!"
echo
echo "Access points:"
echo "  - Grafana:        http://localhost:3000 (admin/admin)"
echo "  - Prometheus:     http://localhost:9090"
echo "  - OTel Collector: http://localhost:55679/debug/tracez"
echo "  - Health Check:   http://localhost:13133/health"
echo
echo "Useful commands:"
echo "  - View logs:      docker-compose logs -f otel-collector"
echo "  - View metrics:   curl http://localhost:8889/metrics"
echo "  - Stop test:      docker-compose down"
echo

# Keep running and show logs
log_info "Showing collector logs (Ctrl+C to stop)..."
docker-compose logs -f otel-collector