# NRDOT-HOST Project Makefile
# Main build automation for all components

.PHONY: all build test clean install docker help

# Variables
GO := go
DOCKER := docker
DOCKER_COMPOSE := docker-compose
KUBECTL := kubectl
HELM := helm

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y%m%d-%H%M%S)
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)

# Component directories
COMPONENTS := processors/common \
              nrdot-schema \
              nrdot-template-lib \
              nrdot-telemetry-client \
              nrdot-privileged-helper \
              nrdot-api-server \
              nrdot-config-engine \
              nrdot-supervisor \
              processors/nrsecurity \
              processors/nrenrich \
              processors/nrtransform \
              processors/nrcap \
              nrdot-ctl \
              cmd/nrdot-host

# Output directory
BIN_DIR := build/bin
DIST_DIR := build/dist

## Default target - build all components
all: clean build

## Build all components
build: $(BIN_DIR)
	@echo "Building all NRDOT-HOST components..."
	@for component in $(COMPONENTS); do \
		echo "Building $$component..." ; \
		name=$$(basename $$component) ; \
		depth=$$(echo $$component | tr '/' '\n' | wc -l) ; \
		if [ $$depth -eq 1 ]; then \
			outpath="../$(BIN_DIR)/$$name" ; \
		else \
			outpath="../../$(BIN_DIR)/$$name" ; \
		fi ; \
		(cd $$component && $(GO) build -ldflags "$(LDFLAGS)" -o $$outpath ./...) || exit 1 ; \
	done
	@echo "Building OTel Collector with NRDOT processors..."
	@cd otelcol-builder && make build
	@echo "All components built successfully!"

## Run tests for all components
test:
	@echo "Running tests for all components..."
	@for component in $(COMPONENTS); do \
		echo "Testing $$component..." ; \
		(cd $$component && $(GO) test -v -race -coverprofile=coverage.out ./...) || exit 1 ; \
	done
	@echo "Running integration tests..."
	@cd tests/integration && make test
	@echo "All tests passed!"

## Run linting for all components
lint:
	@echo "Running linters..."
	@for component in $(COMPONENTS); do \
		echo "Linting $$component..." ; \
		(cd $$component && golangci-lint run) || exit 1 ; \
	done

## Generate code (if needed)
generate:
	@echo "Generating code..."
	@for component in $(COMPONENTS); do \
		if [ -f "$$component/generate.go" ]; then \
			echo "Generating for $$component..." ; \
			(cd $$component && $(GO) generate ./...) ; \
		fi \
	done

## Build Docker images
docker: docker-build

docker-build:
	@echo "Building Docker images..."
	@cd deployments/docker && make build-all TAG=$(VERSION)

docker-push:
	@echo "Pushing Docker images..."
	@cd deployments/docker && make push-all TAG=$(VERSION)

## Build unified Docker image (v2.0)
docker-unified:
	@echo "Building unified NRDOT-HOST Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME="$(shell date -u +%Y-%m-%dT%H:%M:%SZ)" \
		--build-arg GIT_COMMIT=$(COMMIT) \
		-t nrdot-host:$(VERSION) \
		-t nrdot-host:latest \
		-f deployments/docker/unified/Dockerfile .
	@echo "Unified image built: nrdot-host:$(VERSION)"

## Build all Docker images including unified
docker-all: docker-build docker-unified

## Install components locally
install: build
	@echo "Installing NRDOT-HOST components..."
	@mkdir -p /usr/local/bin
	@for binary in $(BIN_DIR)/*; do \
		echo "Installing $$(basename $$binary)..." ; \
		sudo install -m 755 $$binary /usr/local/bin/ ; \
	done
	@echo "Installation complete!"

## Deploy to Kubernetes
deploy-k8s:
	@echo "Deploying to Kubernetes..."
	@cd kubernetes/helm/nrdot && $(HELM) upgrade --install nrdot . -f values.yaml

## Run development environment
dev:
	@echo "Starting development environment..."
	@cd docker && $(DOCKER_COMPOSE) up -d
	@echo "Development environment running!"
	@echo "API Server: http://localhost:8089"
	@echo "Collector: http://localhost:4317"
	@echo "Prometheus: http://localhost:9090"

## Stop development environment
dev-stop:
	@echo "Stopping development environment..."
	@cd docker && $(DOCKER_COMPOSE) down

## Run E2E tests
e2e-test:
	@echo "Running E2E tests..."
	@cd e2e-tests && make test-all

## Run specific component
run-%:
	@echo "Running $*..."
	@$(BIN_DIR)/$* $(ARGS)

## Build specific component
build-%:
	@echo "Building $*..."
	@cd $* && $(GO) build -ldflags "$(LDFLAGS)" -o ../$(BIN_DIR)/$* ./...

## Test specific component
test-%:
	@echo "Testing $*..."
	@cd $* && $(GO) test -v -race ./...

## Create release artifacts
release: clean
	@echo "Creating release artifacts..."
	@mkdir -p $(DIST_DIR)
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			echo "Building for $$os/$$arch..." ; \
			for component in $(COMPONENTS); do \
				GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" \
					-o $(DIST_DIR)/$$component-$$os-$$arch$$( [ $$os = "windows" ] && echo ".exe" ) \
					./$$component/... || exit 1 ; \
			done ; \
		done ; \
	done
	@echo "Creating tarballs..."
	@cd $(DIST_DIR) && for file in *; do tar czf $$file.tar.gz $$file; done
	@echo "Release artifacts created in $(DIST_DIR)/"

## Run security scan
security-scan:
	@echo "Running security scans..."
	@for component in $(COMPONENTS); do \
		echo "Scanning $$component..." ; \
		(cd $$component && gosec -fmt json -out security-report.json ./...) ; \
	done
	@cd deployments/docker && ./security-scan.sh

## Generate documentation
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "Documentation server running at http://localhost:6060"

## Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@for component in $(COMPONENTS); do \
		if ls $$component/*_test.go 2>/dev/null | grep -q bench; then \
			echo "Benchmarking $$component..." ; \
			(cd $$component && $(GO) test -bench=. -benchmem ./...) ; \
		fi \
	done

## Update dependencies
update-deps:
	@echo "Updating dependencies..."
	@for component in $(COMPONENTS); do \
		echo "Updating $$component dependencies..." ; \
		(cd $$component && $(GO) get -u ./... && $(GO) mod tidy) ; \
	done

## Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR) $(DIST_DIR)
	@for component in $(COMPONENTS); do \
		cd $$component && $(GO) clean -cache -testcache ; \
		cd .. ; \
	done
	@find . -name "*.out" -type f -delete
	@find . -name "*.log" -type f -delete
	@echo "Cleanup complete!"

## Setup development environment
setup:
	@echo "Setting up development environment..."
	@./scripts/setup-dev.sh
	@echo "Installing tools..."
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest
	@$(GO) install golang.org/x/tools/cmd/godoc@latest
	@echo "Setup complete!"

## Verify installation
verify:
	@echo "Verifying NRDOT-HOST installation..."
	@for binary in nrdot-ctl nrdot-collector nrdot-supervisor; do \
		if command -v $$binary >/dev/null 2>&1; then \
			echo "✓ $$binary installed" ; \
		else \
			echo "✗ $$binary not found" ; \
		fi \
	done

## Show component status
status:
	@echo "NRDOT-HOST Component Status"
	@echo "=========================="
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo ""
	@echo "Components:"
	@for component in $(COMPONENTS); do \
		if [ -f "$(BIN_DIR)/$$component" ]; then \
			echo "✓ $$component (built)" ; \
		else \
			echo "✗ $$component (not built)" ; \
		fi \
	done

## Create directories
$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

$(DIST_DIR):
	@mkdir -p $(DIST_DIR)

## Display help
help:
	@echo "NRDOT-HOST Makefile"
	@echo "=================="
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Main targets:"
	@echo "  all              Build all components (default)"
	@echo "  build            Build all components"
	@echo "  test             Run all tests"
	@echo "  docker           Build Docker images"
	@echo "  install          Install components locally"
	@echo "  clean            Clean build artifacts"
	@echo ""
	@echo "Development targets:"
	@echo "  dev              Start development environment"
	@echo "  dev-stop         Stop development environment"
	@echo "  lint             Run linters"
	@echo "  setup            Setup development environment"
	@echo ""
	@echo "Testing targets:"
	@echo "  test             Run unit tests"
	@echo "  e2e-test         Run E2E tests"
	@echo "  benchmark        Run benchmarks"
	@echo "  security-scan    Run security scans"
	@echo ""
	@echo "Component targets:"
	@echo "  build-<name>     Build specific component"
	@echo "  test-<name>      Test specific component"
	@echo "  run-<name>       Run specific component"
	@echo ""
	@echo "Release targets:"
	@echo "  release          Create release artifacts"
	@echo "  docker-push      Push Docker images"
	@echo "  deploy-k8s       Deploy to Kubernetes"
	@echo ""
	@echo "Other targets:"
	@echo "  docs             Generate documentation"
	@echo "  update-deps      Update dependencies"
	@echo "  verify           Verify installation"
	@echo "  status           Show component status"
	@echo "  help             Display this help"