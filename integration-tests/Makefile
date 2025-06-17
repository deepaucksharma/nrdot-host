.PHONY: all test test-basic test-security test-enrichment test-transform test-cardinality test-config
.PHONY: setup teardown clean build benchmark lint coverage

# Variables
COLLECTOR_IMAGE ?= nrdot-host:latest
TEST_TIMEOUT ?= 10m
COVERAGE_FILE ?= coverage.out
BENCHMARK_TIME ?= 30s

# Default target
all: setup test teardown

# Build the collector image for testing
build:
	@echo "Building NRDOT-HOST collector image..."
	cd .. && make docker-build

# Setup test environment
setup:
	@echo "Setting up test environment..."
	./scripts/setup.sh

# Teardown test environment
teardown:
	@echo "Tearing down test environment..."
	./scripts/teardown.sh

# Run all tests
test: test-basic test-security test-enrichment test-transform test-cardinality test-config

# Run specific test scenarios
test-basic:
	@echo "Running basic functionality tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./scenarios -run TestBasic

test-security:
	@echo "Running security tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./scenarios -run TestSecurity

test-enrichment:
	@echo "Running enrichment tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./scenarios -run TestEnrichment

test-transform:
	@echo "Running transformation tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./scenarios -run TestTransform

test-cardinality:
	@echo "Running cardinality tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./scenarios -run TestCardinality

test-config:
	@echo "Running configuration tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./scenarios -run TestConfig

# Run benchmarks
benchmark:
	@echo "Running performance benchmarks..."
	go test -v -bench=. -benchtime=$(BENCHMARK_TIME) -run=^$ ./scenarios

# Run with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -v -race -timeout $(TEST_TIMEOUT) ./...

# Run with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -f $(COVERAGE_FILE) coverage.html
	go clean -testcache

# Memory leak detection
test-memory:
	@echo "Running memory leak tests..."
	go test -v -timeout $(TEST_TIMEOUT) -memprofile=mem.prof ./scenarios -run TestMemory
	go tool pprof -text mem.prof

# Integration test with external systems
test-e2e: build setup
	@echo "Running end-to-end tests..."
	go test -v -timeout $(TEST_TIMEOUT) -tags=e2e ./scenarios
	$(MAKE) teardown

# Parallel test execution
test-parallel:
	@echo "Running tests in parallel..."
	go test -v -parallel 4 -timeout $(TEST_TIMEOUT) ./scenarios

# Test with verbose logging
test-verbose:
	@echo "Running tests with verbose logging..."
	TEST_LOG_LEVEL=debug go test -v -timeout $(TEST_TIMEOUT) ./scenarios

# Help
help:
	@echo "Available targets:"
	@echo "  make build          - Build collector image"
	@echo "  make setup          - Setup test environment"
	@echo "  make test           - Run all tests"
	@echo "  make test-basic     - Run basic functionality tests"
	@echo "  make test-security  - Run security tests"
	@echo "  make test-enrichment- Run enrichment tests"
	@echo "  make test-transform - Run transformation tests"
	@echo "  make test-cardinality- Run cardinality tests"
	@echo "  make test-config    - Run configuration tests"
	@echo "  make benchmark      - Run performance benchmarks"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make coverage       - Generate coverage report"
	@echo "  make test-memory    - Run memory leak tests"
	@echo "  make test-e2e       - Run end-to-end tests"
	@echo "  make test-parallel  - Run tests in parallel"
	@echo "  make teardown       - Teardown test environment"
	@echo "  make clean          - Clean build artifacts"