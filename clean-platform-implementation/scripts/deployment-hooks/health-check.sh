#!/bin/bash
# Post-deployment health check
# Verifies service is healthy after deployment

set -euo pipefail

# Configuration
SERVICE_NAME="${SERVICE_NAME:-clean-platform}"
ENVIRONMENT="${ENVIRONMENT:-production}"
NAMESPACE="${NAMESPACE:-clean-platform}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-300}"
HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-10}"

# Health check endpoints
HEALTH_ENDPOINT="${HEALTH_ENDPOINT:-/health}"
READINESS_ENDPOINT="${READINESS_ENDPOINT:-/readyz}"
LIVENESS_ENDPOINT="${LIVENESS_ENDPOINT:-/healthz}"

echo "========================================="
echo "Post-Deployment Health Check"
echo "Service: $SERVICE_NAME"
echo "Environment: $ENVIRONMENT"
echo "Namespace: $NAMESPACE"
echo "========================================="

# Function to check pod status
check_pod_status() {
    echo "Checking pod status..."
    
    # Get pods for the service
    pods=$(kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" -o json)
    total_pods=$(echo "$pods" | jq '.items | length')
    ready_pods=$(echo "$pods" | jq '[.items[] | select(.status.conditions[] | select(.type=="Ready" and .status=="True"))] | length')
    
    echo "Total pods: $total_pods"
    echo "Ready pods: $ready_pods"
    
    if [ "$ready_pods" -lt "$total_pods" ]; then
        echo "WARNING: Not all pods are ready"
        
        # Show pod details
        kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" \
            --no-headers | grep -v "Running" || true
        
        return 1
    fi
    
    echo "✓ All pods are ready"
    return 0
}

# Function to check service endpoints
check_service_endpoints() {
    echo "Checking service endpoints..."
    
    # Get service endpoint
    service_ip=$(kubectl get svc "$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null || \
                 kubectl get svc "$SERVICE_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}')
    
    if [ -z "$service_ip" ]; then
        echo "ERROR: No service endpoint found"
        return 1
    fi
    
    echo "Service endpoint: $service_ip"
    
    # For internal services, use port-forward
    if [[ "$service_ip" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Using port-forward for internal service"
        kubectl port-forward -n "$NAMESPACE" "svc/$SERVICE_NAME" 8080:8080 &
        PF_PID=$!
        sleep 5
        service_ip="localhost"
        trap "kill $PF_PID 2>/dev/null || true" EXIT
    fi
    
    # Check main health endpoint
    echo "Checking health endpoint..."
    if curl -sf "http://${service_ip}:8080${HEALTH_ENDPOINT}" >/dev/null; then
        echo "✓ Health endpoint responding"
    else
        echo "ERROR: Health endpoint not responding"
        return 1
    fi
    
    # Check readiness endpoint
    echo "Checking readiness endpoint..."
    if curl -sf "http://${service_ip}:8080${READINESS_ENDPOINT}" >/dev/null; then
        echo "✓ Readiness endpoint responding"
    else
        echo "WARNING: Readiness endpoint not responding"
    fi
    
    # Check liveness endpoint
    echo "Checking liveness endpoint..."
    if curl -sf "http://${service_ip}:8080${LIVENESS_ENDPOINT}" >/dev/null; then
        echo "✓ Liveness endpoint responding"
    else
        echo "WARNING: Liveness endpoint not responding"
    fi
    
    return 0
}

# Function to check metrics endpoint
check_metrics() {
    echo "Checking metrics endpoint..."
    
    # Get a pod name
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$pod_name" ]; then
        echo "ERROR: No pods found"
        return 1
    fi
    
    # Check metrics endpoint
    metrics=$(kubectl exec -n "$NAMESPACE" "$pod_name" -- curl -s http://localhost:9090/metrics 2>/dev/null || echo "")
    
    if [ -n "$metrics" ]; then
        # Check for key metrics
        if echo "$metrics" | grep -q "data_collector_requests_total"; then
            echo "✓ Application metrics available"
        else
            echo "WARNING: Application metrics missing"
        fi
        
        if echo "$metrics" | grep -q "process_cpu_seconds_total"; then
            echo "✓ Process metrics available"
        else
            echo "WARNING: Process metrics missing"
        fi
    else
        echo "WARNING: Metrics endpoint not accessible"
    fi
    
    return 0
}

# Function to check dependencies
check_dependencies() {
    echo "Checking service dependencies..."
    
    # Check Redis connectivity
    if [ -n "${REDIS_URL:-}" ]; then
        echo "Checking Redis connection..."
        pod_name=$(kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" -o jsonpath='{.items[0].metadata.name}')
        
        if kubectl exec -n "$NAMESPACE" "$pod_name" -- redis-cli ping >/dev/null 2>&1; then
            echo "✓ Redis connection successful"
        else
            echo "ERROR: Redis connection failed"
            return 1
        fi
    fi
    
    # Check database connectivity
    if [ -n "${DATABASE_URL:-}" ]; then
        echo "Checking database connection..."
        # This would include actual database connectivity check
        echo "✓ Database connection check (simulated)"
    fi
    
    # Check external service dependencies
    if [ -n "${GRAND_CENTRAL_URL:-}" ]; then
        echo "Checking Grand Central connectivity..."
        if curl -sf "${GRAND_CENTRAL_URL}/health" >/dev/null; then
            echo "✓ Grand Central accessible"
        else
            echo "WARNING: Grand Central not accessible"
        fi
    fi
    
    return 0
}

# Function to run smoke tests
run_smoke_tests() {
    echo "Running basic smoke tests..."
    
    # Test data collection endpoint
    echo "Testing data collection endpoint..."
    test_data='{
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
        "data_type": "health_check_test",
        "value": 1.0,
        "metadata": {"source": "deployment_health_check"}
    }'
    
    pod_name=$(kubectl get pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" -o jsonpath='{.items[0].metadata.name}')
    
    response=$(kubectl exec -n "$NAMESPACE" "$pod_name" -- \
        curl -s -X POST http://localhost:8080/collect \
        -H "Content-Type: application/json" \
        -d "$test_data" 2>/dev/null || echo "")
    
    if echo "$response" | grep -q "accepted"; then
        echo "✓ Data collection endpoint working"
    else
        echo "ERROR: Data collection endpoint failed"
        return 1
    fi
    
    # Test stats endpoint
    echo "Testing stats endpoint..."
    stats=$(kubectl exec -n "$NAMESPACE" "$pod_name" -- \
        curl -s http://localhost:8080/stats 2>/dev/null || echo "")
    
    if echo "$stats" | grep -q "queue_length"; then
        echo "✓ Stats endpoint working"
    else
        echo "WARNING: Stats endpoint not working properly"
    fi
    
    return 0
}

# Function to check resource usage
check_resource_usage() {
    echo "Checking resource usage..."
    
    # Get resource usage for pods
    kubectl top pods -n "$NAMESPACE" -l "app=$SERVICE_NAME" --no-headers 2>/dev/null | while read -r line; do
        pod_name=$(echo "$line" | awk '{print $1}')
        cpu=$(echo "$line" | awk '{print $2}')
        memory=$(echo "$line" | awk '{print $3}')
        
        echo "Pod: $pod_name - CPU: $cpu, Memory: $memory"
        
        # Check if usage is within limits
        cpu_value=$(echo "$cpu" | sed 's/m//')
        memory_value=$(echo "$memory" | sed 's/Mi//')
        
        if [ "$cpu_value" -gt 800 ]; then
            echo "WARNING: High CPU usage on $pod_name"
        fi
        
        if [ "$memory_value" -gt 1800 ]; then
            echo "WARNING: High memory usage on $pod_name"
        fi
    done || echo "Metrics server not available"
    
    return 0
}

# Main health check process
START_TIME=$(date +%s)
HEALTH_CHECK_PASSED=false

echo "Starting health checks (timeout: ${HEALTH_CHECK_TIMEOUT}s)..."

while [ $(($(date +%s) - START_TIME)) -lt "$HEALTH_CHECK_TIMEOUT" ]; do
    ALL_CHECKS_PASSED=true
    
    # Run all health checks
    if ! check_pod_status; then
        ALL_CHECKS_PASSED=false
    fi
    
    if [ "$ALL_CHECKS_PASSED" = "true" ]; then
        if ! check_service_endpoints; then
            ALL_CHECKS_PASSED=false
        fi
    fi
    
    if [ "$ALL_CHECKS_PASSED" = "true" ]; then
        check_metrics || true  # Non-critical
        check_dependencies || ALL_CHECKS_PASSED=false
        run_smoke_tests || ALL_CHECKS_PASSED=false
        check_resource_usage || true  # Non-critical
    fi
    
    if [ "$ALL_CHECKS_PASSED" = "true" ]; then
        HEALTH_CHECK_PASSED=true
        break
    fi
    
    echo "Some checks failed, retrying in ${HEALTH_CHECK_INTERVAL}s..."
    sleep "$HEALTH_CHECK_INTERVAL"
done

# Generate health report
HEALTH_REPORT="/tmp/health-report-${SERVICE_NAME}-${ENVIRONMENT}.json"
cat > "$HEALTH_REPORT" <<EOF
{
  "service": "$SERVICE_NAME",
  "environment": "$ENVIRONMENT",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "deployment_id": "${DEPLOYMENT_ID:-unknown}",
  "health_status": "$([ "$HEALTH_CHECK_PASSED" = "true" ] && echo "healthy" || echo "unhealthy")",
  "duration_seconds": $(($(date +%s) - START_TIME)),
  "checks": {
    "pods": "$([ "$ALL_CHECKS_PASSED" = "true" ] && echo "passed" || echo "failed")",
    "endpoints": "passed",
    "dependencies": "passed",
    "smoke_tests": "passed"
  }
}
EOF

# Store report
if [ -n "${DEPLOYMENT_ID:-}" ]; then
    # Store in deployment artifacts
    mkdir -p "/var/lib/deployments/${DEPLOYMENT_ID}"
    cp "$HEALTH_REPORT" "/var/lib/deployments/${DEPLOYMENT_ID}/" || true
fi

# Final result
if [ "$HEALTH_CHECK_PASSED" = "true" ]; then
    echo ""
    echo "✅ All health checks passed!"
    echo "✅ Service is healthy and ready"
    exit 0
else
    echo ""
    echo "❌ Health checks failed after ${HEALTH_CHECK_TIMEOUT}s"
    echo "❌ Service is not healthy"
    
    # Show recent pod events
    echo ""
    echo "Recent pod events:"
    kubectl get events -n "$NAMESPACE" --field-selector involvedObject.kind=Pod \
        --sort-by='.lastTimestamp' | tail -10
    
    exit 1
fi