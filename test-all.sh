#!/bin/bash
# Script to run all tests and generate coverage report

set -e

echo "Running all tests with coverage..."

# Initialize coverage file
echo "mode: set" > coverage.txt

# List of components to test
components=(
    "otel-processor-common"
    "nrdot-schema"
    "nrdot-template-lib"
    "nrdot-telemetry-client"
    "nrdot-config-engine"
    "nrdot-supervisor"
    "nrdot-ctl"
    "otel-processor-nrsecurity"
    "otel-processor-nrenrich"
    "otel-processor-nrtransform"
    "otel-processor-nrcap"
    "nrdot-api-server"
    "nrdot-privileged-helper"
)

# Run tests for each component
for component in "${components[@]}"; do
    echo "Testing $component..."
    if [ -d "$component" ]; then
        cd "$component"
        if go test -coverprofile=coverage.out -covermode=set ./... 2>/dev/null; then
            # Append coverage data
            if [ -f coverage.out ]; then
                tail -n +2 coverage.out >> ../coverage.txt
            fi
            echo "✓ $component tests passed"
        else
            echo "✗ $component tests failed"
        fi
        cd ..
    fi
done

echo "Generating coverage report..."
go tool cover -html=coverage.txt -o coverage.html

# Calculate total coverage
total_coverage=$(go tool cover -func=coverage.txt | grep total | awk '{print $3}')
echo "Total coverage: $total_coverage"

echo "Coverage report generated: coverage.html"