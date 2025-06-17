#!/bin/bash

# NRDOT OpenTelemetry Collector Builder Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi
    print_success "Go is installed: $(go version)"
    
    # Check Docker (optional)
    if command -v docker &> /dev/null; then
        print_success "Docker is installed: $(docker --version)"
    else
        print_info "Docker not found. Docker builds will not be available."
    fi
    
    # Check if NRDOT processors exist
    if [ ! -d "../nrdot-ctl" ]; then
        print_error "NRDOT processors directory not found at ../nrdot-ctl"
        exit 1
    fi
    print_success "NRDOT processors directory found"
}

# Main build function
build_collector() {
    print_info "Building NRDOT Custom Collector..."
    
    # Install builder if needed
    if ! command -v builder &> /dev/null; then
        print_info "Installing OpenTelemetry Collector Builder..."
        go install go.opentelemetry.io/collector/cmd/builder@v0.96.0
    fi
    
    # Run builder
    builder --config=otelcol-builder.yaml
    
    if [ -f "./nrdot-collector" ]; then
        print_success "Build completed successfully!"
        print_info "Binary location: ./nrdot-collector"
        print_info "Binary size: $(du -h ./nrdot-collector | cut -f1)"
    else
        print_error "Build failed - binary not found"
        exit 1
    fi
}

# Test function
test_collector() {
    print_info "Testing NRDOT Custom Collector..."
    
    if [ ! -f "./nrdot-collector" ]; then
        print_error "Collector binary not found. Run build first."
        exit 1
    fi
    
    # Validate configuration
    print_info "Validating test configuration..."
    if ./nrdot-collector validate --config=test-config.yaml; then
        print_success "Configuration is valid"
    else
        print_error "Configuration validation failed"
        exit 1
    fi
    
    # Quick runtime test
    print_info "Starting collector for quick test (10 seconds)..."
    timeout 10s ./nrdot-collector --config=test-config.yaml &> test.log || true
    
    if grep -q "Everything is ready" test.log 2>/dev/null || grep -q "service started" test.log 2>/dev/null; then
        print_success "Collector started successfully"
    else
        print_error "Collector may have issues starting. Check test.log for details."
    fi
    rm -f test.log
}

# Docker build function
docker_build() {
    print_info "Building Docker image..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    
    # Ensure binary is built
    if [ ! -f "./nrdot-collector" ]; then
        build_collector
    fi
    
    # Build Docker image
    docker build -t nrdot/otel-collector:latest -f docker/Dockerfile .
    
    if [ $? -eq 0 ]; then
        print_success "Docker image built: nrdot/otel-collector:latest"
        print_info "Image size: $(docker images nrdot/otel-collector:latest --format '{{.Size}}')"
    else
        print_error "Docker build failed"
        exit 1
    fi
}

# Show usage
usage() {
    echo "NRDOT OpenTelemetry Collector Builder"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  check     - Check prerequisites"
    echo "  build     - Build the collector"
    echo "  test      - Test the collector"
    echo "  docker    - Build Docker image"
    echo "  all       - Run all steps"
    echo "  help      - Show this help"
    echo ""
}

# Main script
case "${1:-all}" in
    check)
        check_prerequisites
        ;;
    build)
        check_prerequisites
        build_collector
        ;;
    test)
        check_prerequisites
        test_collector
        ;;
    docker)
        check_prerequisites
        docker_build
        ;;
    all)
        check_prerequisites
        build_collector
        test_collector
        if command -v docker &> /dev/null; then
            docker_build
        fi
        print_success "All steps completed!"
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        print_error "Unknown command: $1"
        usage
        exit 1
        ;;
esac