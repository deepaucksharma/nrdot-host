# otel-processor-common Makefile

.PHONY: all build test test-unit test-integration clean lint fmt deps bench

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-s -w"
TESTFLAGS=-race -cover -coverprofile=coverage.out

all: clean deps lint test build

build:
	@echo "Building otel-processor-common..."
	$(GOBUILD) $(LDFLAGS) ./...

test: test-unit

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) $(TESTFLAGS) ./...

test-integration:
	@echo "Running integration tests..."
	$(GOTEST) $(TESTFLAGS) -tags=integration ./...

clean:
	@echo "Cleaning..."
	$(GOCMD) clean
	rm -f coverage.out
	rm -f coverage.html

lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Component-specific targets used by parent Makefile
.PHONY: component-build component-test component-clean

component-build: build
component-test: test
component-clean: clean