#!/bin/bash
# Push NRDOT Docker images to registry

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REGISTRY=${REGISTRY:-docker.io/newrelic}
TAG=${TAG:-latest}
DRY_RUN=${DRY_RUN:-false}

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

# Components to push
COMPONENTS=(
    "base"
    "collector"
    "supervisor"
    "config-engine"
    "api-server"
    "privileged-helper"
    "ctl"
)

# Function to check if image exists
check_image() {
    local image=$1
    if docker image inspect "$image" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to push image
push_image() {
    local component=$1
    local image="${REGISTRY}/nrdot-${component}:${TAG}"
    
    if ! check_image "$image"; then
        print_error "Image not found: $image"
        return 1
    fi
    
    if [ "$DRY_RUN" = "true" ]; then
        print_info "[DRY RUN] Would push: $image"
    else
        print_info "Pushing $image..."
        if docker push "$image"; then
            print_success "Pushed $image"
        else
            print_error "Failed to push $image"
            return 1
        fi
    fi
}

# Function to tag image for multiple registries
tag_for_registry() {
    local component=$1
    local source_image="${REGISTRY}/nrdot-${component}:${TAG}"
    local target_registry=$2
    local target_tag=$3
    local target_image="${target_registry}/nrdot-${component}:${target_tag}"
    
    print_info "Tagging $source_image as $target_image"
    docker tag "$source_image" "$target_image"
}

# Main push process
main() {
    print_info "Starting NRDOT Docker push process"
    print_info "Registry: $REGISTRY"
    print_info "Tag: $TAG"
    print_info "Dry run: $DRY_RUN"
    
    # Check Docker login
    if [ "$DRY_RUN" != "true" ]; then
        if ! docker pull "${REGISTRY}/hello-world" >/dev/null 2>&1; then
            print_error "Not logged in to registry: $REGISTRY"
            print_info "Please run: docker login $REGISTRY"
            exit 1
        fi
    fi
    
    # Track push status
    local failed=0
    local pushed=0
    
    # Push each component
    for component in "${COMPONENTS[@]}"; do
        if push_image "$component"; then
            pushed=$((pushed + 1))
        else
            failed=$((failed + 1))
        fi
    done
    
    # Summary
    echo
    print_info "Push summary:"
    print_info "  Pushed: $pushed"
    print_info "  Failed: $failed"
    
    if [ $failed -eq 0 ]; then
        print_success "All images pushed successfully!"
    else
        print_error "$failed images failed to push"
        exit 1
    fi
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
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --component)
            # Push only specific component
            if push_image "$2"; then
                print_success "Component $2 pushed successfully"
            else
                print_error "Failed to push component $2"
                exit 1
            fi
            exit 0
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --registry REGISTRY  Docker registry (default: docker.io/newrelic)"
            echo "  --tag TAG           Image tag (default: latest)"
            echo "  --dry-run           Show what would be pushed without pushing"
            echo "  --component NAME    Push only specific component"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main push process
main