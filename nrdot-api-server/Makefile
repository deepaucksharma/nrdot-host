# nrdot-api-server Makefile

.PHONY: all build test test-unit clean lint fmt deps run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build parameters
BINARY_NAME=nrdot-api-server
BINARY_PATH=cmd/api-server
LDFLAGS=-ldflags "-s -w"
TESTFLAGS=-race -cover -coverprofile=coverage.out

# Server parameters
DEFAULT_PORT=8089
DEFAULT_HOST=127.0.0.1

all: clean deps lint test build

build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./$(BINARY_PATH)

test: test-unit

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) $(TESTFLAGS) ./...

clean:
	@echo "Cleaning..."
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
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

run: build
	@echo "Starting API server on $(DEFAULT_HOST):$(DEFAULT_PORT)..."
	./$(BINARY_NAME) -host $(DEFAULT_HOST) -port $(DEFAULT_PORT)

coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Component-specific targets used by parent Makefile
.PHONY: component-build component-test component-clean

component-build: build
component-test: test
component-clean: clean