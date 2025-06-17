#!/bin/bash
set -euo pipefail

echo "Setting up integration test environment..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
check_prerequisite() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        exit 1
    fi
}

echo "Checking prerequisites..."
check_prerequisite docker
check_prerequisite go

# Check Docker daemon
if ! docker info &> /dev/null; then
    echo -e "${RED}Error: Docker daemon is not running${NC}"
    exit 1
fi

# Build collector image if needed
if [[ -z "${SKIP_BUILD:-}" ]]; then
    echo -e "${YELLOW}Building NRDOT-HOST collector image...${NC}"
    cd ../..
    make docker-build || {
        echo -e "${RED}Failed to build collector image${NC}"
        exit 1
    }
    cd integration-tests
else
    echo -e "${YELLOW}Skipping collector build (SKIP_BUILD is set)${NC}"
fi

# Create test network
echo "Creating test network..."
docker network create nrdot-test-network 2>/dev/null || {
    echo "Network already exists, continuing..."
}

# Pull required images
echo "Pulling required Docker images..."
docker pull prom/prometheus:latest
docker pull testcontainers/ryuk:0.5.1

# Create directories for test outputs
echo "Creating test output directories..."
mkdir -p test-output/{logs,metrics,traces}

# Download Go dependencies
echo "Downloading Go dependencies..."
go mod download

# Run go mod tidy to ensure consistency
go mod tidy

# Set up environment variables
export TEST_NETWORK=nrdot-test-network
export COLLECTOR_IMAGE=${COLLECTOR_IMAGE:-nrdot-host:latest}
export TEST_OUTPUT_DIR=$(pwd)/test-output

# Create a marker file to indicate setup is complete
touch .setup-complete

echo -e "${GREEN}Setup complete!${NC}"
echo ""
echo "Environment variables set:"
echo "  TEST_NETWORK=$TEST_NETWORK"
echo "  COLLECTOR_IMAGE=$COLLECTOR_IMAGE"
echo "  TEST_OUTPUT_DIR=$TEST_OUTPUT_DIR"
echo ""
echo "You can now run tests with: make test"