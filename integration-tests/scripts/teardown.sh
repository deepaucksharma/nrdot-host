#!/bin/bash
set -euo pipefail

echo "Tearing down integration test environment..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to safely remove containers
remove_containers() {
    local containers=$(docker ps -a --filter "network=nrdot-test-network" -q)
    if [[ -n "$containers" ]]; then
        echo "Removing test containers..."
        docker rm -f $containers 2>/dev/null || true
    fi
}

# Function to remove test network
remove_network() {
    if docker network ls | grep -q "nrdot-test-network"; then
        echo "Removing test network..."
        docker network rm nrdot-test-network 2>/dev/null || {
            echo -e "${YELLOW}Warning: Could not remove network (may still be in use)${NC}"
        }
    fi
}

# Function to clean up test outputs
cleanup_outputs() {
    if [[ -d "test-output" ]] && [[ -z "${PRESERVE_OUTPUTS:-}" ]]; then
        echo "Cleaning up test outputs..."
        rm -rf test-output
    elif [[ -n "${PRESERVE_OUTPUTS:-}" ]]; then
        echo -e "${YELLOW}Preserving test outputs (PRESERVE_OUTPUTS is set)${NC}"
    fi
}

# Function to clean up Docker resources
cleanup_docker() {
    echo "Cleaning up Docker resources..."
    
    # Remove dangling images related to our tests
    docker image prune -f --filter "label=test=integration" 2>/dev/null || true
    
    # Clean up volumes
    docker volume prune -f 2>/dev/null || true
}

# Main cleanup process
echo -e "${YELLOW}Starting cleanup...${NC}"

# Stop and remove containers
remove_containers

# Remove network
remove_network

# Clean up test outputs
cleanup_outputs

# Clean up Docker resources
if [[ -z "${SKIP_DOCKER_CLEANUP:-}" ]]; then
    cleanup_docker
fi

# Remove setup marker
rm -f .setup-complete

# Final cleanup check
if [[ -n "${THOROUGH_CLEANUP:-}" ]]; then
    echo "Performing thorough cleanup..."
    
    # Remove all containers with our label
    docker ps -a --filter "label=integration-test=nrdot" -q | xargs -r docker rm -f
    
    # Remove all networks with our prefix
    docker network ls --filter "name=nrdot-" -q | xargs -r docker network rm
    
    # Clean build cache
    docker builder prune -f
fi

echo -e "${GREEN}Teardown complete!${NC}"

# Show remaining Docker resources
if [[ -n "${SHOW_REMAINING:-}" ]]; then
    echo ""
    echo "Remaining Docker resources:"
    echo "Containers:"
    docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Networks}}"
    echo ""
    echo "Networks:"
    docker network ls --format "table {{.Name}}\t{{.Driver}}"
fi