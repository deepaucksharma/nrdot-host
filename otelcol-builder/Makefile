.PHONY: install-builder build clean test docker-build docker-run

# Variables
BUILDER_VERSION := 0.96.0
BINARY_NAME := nrdot-collector
DOCKER_IMAGE := nrdot/otel-collector
DOCKER_TAG := latest

# Install otelcol-builder if not already installed
install-builder:
	@if ! command -v builder &> /dev/null; then \
		echo "Installing OpenTelemetry Collector Builder..."; \
		go install go.opentelemetry.io/collector/cmd/builder@v$(BUILDER_VERSION); \
	else \
		echo "OpenTelemetry Collector Builder already installed"; \
	fi

# Build the custom collector
build: install-builder
	@echo "Building NRDOT Custom Collector..."
	builder --config=otelcol-builder.yaml
	@echo "Build complete: ./$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf _build
	@echo "Clean complete"

# Test the built collector with test configuration
test: build
	@echo "Testing NRDOT Custom Collector..."
	./$(BINARY_NAME) validate --config=test-config.yaml
	@echo "Starting collector for 10 seconds..."
	timeout 10s ./$(BINARY_NAME) --config=test-config.yaml || true
	@echo "Test complete"

# Build Docker image
docker-build: build
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f docker/Dockerfile .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Run Docker container
docker-run:
	@echo "Running NRDOT Collector in Docker..."
	cd docker && docker-compose up -d
	@echo "Container started. View logs with: docker-compose -f docker/docker-compose.yaml logs -f"

# Stop Docker container
docker-stop:
	@echo "Stopping NRDOT Collector..."
	cd docker && docker-compose down
	@echo "Container stopped"

# Development workflow
dev: build test
	@echo "Development build and test complete"

# Production build with optimizations
prod: clean
	@echo "Building production binary..."
	builder --config=otelcol-builder.yaml --ldflags="-s -w"
	@echo "Production build complete"

# Update dependencies
update-deps:
	@echo "Updating NRDOT processor dependencies..."
	cd ../nrdot-ctl && go mod tidy
	@echo "Dependencies updated"

# Show version info
version: build
	./$(BINARY_NAME) --version

# Help
help:
	@echo "NRDOT OpenTelemetry Collector Builder"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Build the custom collector"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make test          - Test the built collector"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run collector in Docker"
	@echo "  make docker-stop   - Stop Docker container"
	@echo "  make dev           - Development build and test"
	@echo "  make prod          - Production optimized build"
	@echo "  make version       - Show collector version"
	@echo "  make help          - Show this help message"