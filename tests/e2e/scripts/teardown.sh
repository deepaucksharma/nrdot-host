#!/bin/bash

echo "Tearing down E2E test environment..."

# Stop all test containers
for scenario in microservices host-monitoring security-compliance high-cardinality; do
    if [ -f "scenarios/$scenario/docker-compose.yaml" ]; then
        echo "Stopping $scenario scenario..."
        cd scenarios/$scenario && docker-compose down -v 2>/dev/null || true
        cd ../..
    fi
done

# Remove test network
docker network rm nrdot-test 2>/dev/null || true

# Clean up dangling volumes
docker volume prune -f 2>/dev/null || true

echo "E2E test environment teardown complete!"