#!/bin/bash

set -e

echo "Setting up E2E test environment..."

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed. Aborting." >&2; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose is required but not installed. Aborting." >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required but not installed. Aborting." >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl is required but not installed. Aborting." >&2; exit 1; }

# Build NRDOT collector if not exists
if ! docker images | grep -q "nrdot-collector"; then
    echo "Building NRDOT collector image..."
    cd ../../otelcol-builder && make docker-build
fi

# Create necessary directories
mkdir -p reports/{microservices,kubernetes,host-monitoring,security-compliance,high-cardinality}

# Set up test network
docker network create nrdot-test 2>/dev/null || true

echo "E2E test environment setup complete!"