#!/bin/bash
# Build all NRDOT Docker images

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REGISTRY=${REGISTRY:-docker.io/newrelic}
TAG=${TAG:-latest}
BUILD_ARGS=${BUILD_ARGS:-}
PUSH=${PUSH:-false}

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to build an image
build_image() {
    local name=$1
    local dockerfile=$2
    local context=$3
    local build_args=$4
    
    print_info "Building $name..."
    
    if docker build \
        ${BUILD_ARGS} \
        ${build_args} \
        -t "${REGISTRY}/nrdot-${name}:${TAG}" \
        -f "${dockerfile}" \
        "${context}"; then
        print_success "$name built successfully"
        return 0
    else
        print_error "Failed to build $name"
        return 1
    fi
}

# Function to push an image
push_image() {
    local name=$1
    
    if [ "$PUSH" = "true" ]; then
        print_info "Pushing $name to registry..."
        if docker push "${REGISTRY}/nrdot-${name}:${TAG}"; then
            print_success "$name pushed successfully"
        else
            print_error "Failed to push $name"
            return 1
        fi
    fi
}

# Main build process
main() {
    print_info "Starting NRDOT Docker build process"
    print_info "Registry: $REGISTRY"
    print_info "Tag: $TAG"
    print_info "Push: $PUSH"
    
    # Track build status
    local failed=0
    
    # Build base image first
    if build_image "base" "base/Dockerfile.base" ".." "--target base"; then
        push_image "base"
    else
        failed=$((failed + 1))
    fi
    
    # Build other components
    local components=(
        "collector:collector/Dockerfile:../otelcol-builder"
        "supervisor:supervisor/Dockerfile:.."
        "config-engine:config-engine/Dockerfile:.."
        "api-server:api-server/Dockerfile:.."
        "privileged-helper:privileged-helper/Dockerfile:.."
        "ctl:ctl/Dockerfile:.."
    )
    
    for component in "${components[@]}"; do
        IFS=':' read -r name dockerfile context <<< "$component"
        
        if build_image "$name" "$dockerfile" "$context" "--build-arg BASE_IMAGE=${REGISTRY}/nrdot-base:${TAG}"; then
            push_image "$name"
        else
            failed=$((failed + 1))
        fi
    done
    
    # Summary
    echo
    if [ $failed -eq 0 ]; then
        print_success "All images built successfully!"
    else
        print_error "$failed images failed to build"
        exit 1
    fi
    
    # Show image sizes
    print_info "Image sizes:"
    docker images "${REGISTRY}/nrdot-*:${TAG}" --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --build-arg)
            BUILD_ARGS="$BUILD_ARGS --build-arg $2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --registry REGISTRY  Docker registry (default: docker.io/newrelic)"
            echo "  --tag TAG           Image tag (default: latest)"
            echo "  --push              Push images to registry"
            echo "  --build-arg ARG     Additional build arguments"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main build process
main