# NRDOT-HOST End-to-End Test Scenarios

This directory contains comprehensive end-to-end test scenarios that demonstrate real-world usage of NRDOT-HOST.

## Overview

These E2E tests validate the complete NRDOT-HOST system in realistic environments, ensuring all components work correctly together.

## Test Scenarios

### 1. Microservices Monitoring
- **Purpose**: Validate distributed tracing and metrics collection across microservices
- **Components**: Frontend, Backend, Database services
- **Validates**: 
  - Trace propagation
  - Service dependency mapping
  - Request latency tracking
  - Error rate monitoring

### 2. Kubernetes Monitoring
- **Purpose**: Test Kubernetes-native deployment and monitoring
- **Components**: K8s deployments, services, configmaps
- **Validates**:
  - K8s metadata enrichment
  - Pod/container metrics
  - Service discovery
  - ConfigMap-based configuration

### 3. Host Monitoring
- **Purpose**: System-level metrics collection and analysis
- **Components**: Host metrics receiver, load generator
- **Validates**:
  - CPU, memory, disk, network metrics
  - Process monitoring
  - Resource utilization alerts
  - Performance under load

### 4. Security Compliance
- **Purpose**: Validate secret redaction and security features
- **Components**: Vulnerable app with exposed secrets
- **Validates**:
  - API key redaction
  - Password masking
  - PII protection
  - Compliance reporting

### 5. High Cardinality Protection
- **Purpose**: Test cardinality limiting under stress
- **Components**: Metric generator creating high-cardinality data
- **Validates**:
  - Cardinality explosion prevention
  - Label aggregation strategies
  - Performance under high cardinality
  - Memory usage control

## Running Tests

### Run All Scenarios
```bash
make test-all
```

### Run Individual Scenarios
```bash
make test-microservices
make test-kubernetes
make test-host-monitoring
make test-security
make test-cardinality
```

### Generate Test Report
```bash
make report
```

## Directory Structure

```
e2e-tests/
├── scenarios/           # Test scenarios
│   ├── microservices/   # Distributed system test
│   ├── kubernetes/      # K8s deployment test
│   ├── host-monitoring/ # System metrics test
│   ├── security-compliance/ # Security test
│   └── high-cardinality/    # Cardinality test
├── scripts/            # Test automation scripts
└── docker-compose.yaml # Full stack setup
```

## Requirements

- Docker and Docker Compose
- Kubernetes cluster (for K8s scenario)
- Make
- Bash 4.0+
- curl, jq for validation

## Test Validation

Each scenario validates:
1. **Functionality**: Components work as designed
2. **Performance**: Meets performance requirements
3. **Resource Usage**: CPU/Memory within limits
4. **Data Accuracy**: Telemetry data is correct
5. **Integration**: Components integrate properly

## Continuous Integration

These tests are designed to run in CI/CD pipelines:
- GitHub Actions workflow provided
- Supports parallel execution
- Generates JUnit XML reports
- Uploads test artifacts

## Troubleshooting

### Common Issues

1. **Docker daemon not running**
   ```bash
   sudo systemctl start docker
   ```

2. **Port conflicts**
   - Check for services using required ports
   - Modify docker-compose.yaml port mappings

3. **Resource limits**
   - Ensure sufficient CPU/Memory
   - Adjust resource limits in compose files

### Debug Mode

Run tests with debug output:
```bash
DEBUG=1 make test-all
```

View container logs:
```bash
docker-compose -f scenarios/microservices/docker-compose.yaml logs
```