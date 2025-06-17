# nrdot-schema Makefile

.PHONY: all build test test-unit clean lint fmt deps validate-schemas

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-s -w"
TESTFLAGS=-race -cover -coverprofile=coverage.out

all: clean deps lint test build

build:
	@echo "Building nrdot-schema..."
	$(GOBUILD) $(LDFLAGS) ./...

test: test-unit validate-schemas

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) $(TESTFLAGS) ./...

validate-schemas:
	@echo "Validating schema files..."
	@for schema in schemas/*.json; do \
		echo "Validating $$schema..."; \
		$(GOCMD) run ./cmd/validate-schema/main.go $$schema || exit 1; \
	done

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

coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Component-specific targets used by parent Makefile
.PHONY: component-build component-test component-clean

component-build: build
component-test: test
component-clean: clean