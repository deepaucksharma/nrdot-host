# NRDOT-Host Master Makefile
# Orchestrates building, testing, and deployment across all repositories

.PHONY: all build test clean deploy

# Default target
all: build

# Build all components
build: build-core build-processors build-tools

build-core:
	@echo "Building Core Components..."
	$(MAKE) -C nrdot-ctl build
	$(MAKE) -C nrdot-config-engine build
	$(MAKE) -C nrdot-supervisor build
	$(MAKE) -C nrdot-telemetry-client build
	$(MAKE) -C nrdot-template-lib build

build-processors:
	@echo "Building OTel Processors..."
	$(MAKE) -C otel-processor-common build
	$(MAKE) -C otel-processor-nrsecurity build
	$(MAKE) -C otel-processor-nrenrich build
	$(MAKE) -C otel-processor-nrtransform build
	$(MAKE) -C otel-processor-nrcap build
	$(MAKE) -C nrdot-privileged-helper build

build-tools:
	@echo "Building Tools..."
	$(MAKE) -C nrdot-api-server build
	$(MAKE) -C nrdot-debug-tools build
	$(MAKE) -C nrdot-migrate build# Testing targets
test: test-unit test-integration test-security

test-unit:
	@echo "Running unit tests..."
	@for dir in */; do \
		if [ -f $$dir/Makefile ]; then \
			$(MAKE) -C $$dir test || exit 1; \
		fi \
	done

test-integration:
	@echo "Running integration tests..."
	$(MAKE) -C nrdot-test-harness test-integration

test-security:
	@echo "Running security validation..."
	$(MAKE) -C nrdot-compliance-validator test

# Benchmarking
benchmark:
	@echo "Running benchmarks..."
	$(MAKE) -C nrdot-benchmark-suite benchmark

# Package building
package: build
	@echo "Building packages..."
	$(MAKE) -C nrdot-packaging all

# Container images
docker: build
	@echo "Building container images..."
	$(MAKE) -C nrdot-container-images build# Deployment
deploy-guardian-fleet:
	@echo "Deploying Guardian Fleet..."
	$(MAKE) -C guardian-fleet-infra deploy

# Development setup
setup:
	@echo "Setting up development environment..."
	go mod download
	@for dir in */; do \
		if [ -f $$dir/go.mod ]; then \
			cd $$dir && go mod download && cd ..; \
		fi \
	done

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@for dir in */; do \
		if [ -f $$dir/Makefile ]; then \
			$(MAKE) -C $$dir clean; \
		fi \
	done

# Run local NRDOT instance
run-local: build
	@echo "Starting local NRDOT instance..."
	./nrdot-ctl/bin/nrdot-ctl start --config=./examples/local-config.yml

# Generate documentation
docs:
	@echo "Generating documentation..."
	@for dir in */; do \
		if [ -f $$dir/README.md ]; then \
			echo "Processing $$dir"; \
		fi \
	done

.DEFAULT_GOAL := all