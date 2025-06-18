# NRDOT-HOST Docker Build System

# Variables
REGISTRY ?= docker.io/newrelic
TAG ?= latest
PLATFORMS ?= linux/amd64,linux/arm64
DOCKER_BUILDKIT ?= 1

# Component list
COMPONENTS = base collector supervisor config-engine api-server privileged-helper ctl

# Build context paths
BASE_PATH = ../
COLLECTOR_PATH = ../otelcol-builder
SUPERVISOR_PATH = ../nrdot-supervisor
CONFIG_ENGINE_PATH = ../nrdot-config-engine
API_SERVER_PATH = ../nrdot-api-server
PRIVILEGED_HELPER_PATH = ../nrdot-privileged-helper
CTL_PATH = ../nrdot-ctl

# Image names
BASE_IMAGE = $(REGISTRY)/nrdot-base:$(TAG)
COLLECTOR_IMAGE = $(REGISTRY)/nrdot-collector:$(TAG)
SUPERVISOR_IMAGE = $(REGISTRY)/nrdot-supervisor:$(TAG)
CONFIG_ENGINE_IMAGE = $(REGISTRY)/nrdot-config-engine:$(TAG)
API_SERVER_IMAGE = $(REGISTRY)/nrdot-api-server:$(TAG)
PRIVILEGED_HELPER_IMAGE = $(REGISTRY)/nrdot-privileged-helper:$(TAG)
CTL_IMAGE = $(REGISTRY)/nrdot-ctl:$(TAG)

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: all build-all push-all clean help

## Default target
all: build-all

## Build all images
build-all: build-base build-collector build-supervisor build-config-engine build-api-server build-privileged-helper build-ctl
	@echo "$(GREEN)✓ All images built successfully$(NC)"

## Build base image
build-base:
	@echo "$(BLUE)Building base image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--cache-from $(BASE_IMAGE) \
		-t $(BASE_IMAGE) \
		-f base/Dockerfile.base \
		$(BASE_PATH)
	@echo "$(GREEN)✓ Base image built: $(BASE_IMAGE)$(NC)"

## Build collector image
build-collector: build-base
	@echo "$(BLUE)Building collector image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--cache-from $(COLLECTOR_IMAGE) \
		-t $(COLLECTOR_IMAGE) \
		-f collector/Dockerfile \
		$(COLLECTOR_PATH)
	@echo "$(GREEN)✓ Collector image built: $(COLLECTOR_IMAGE)$(NC)"

## Build supervisor image
build-supervisor: build-base
	@echo "$(BLUE)Building supervisor image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--cache-from $(SUPERVISOR_IMAGE) \
		-t $(SUPERVISOR_IMAGE) \
		-f supervisor/Dockerfile \
		$(SUPERVISOR_PATH)
	@echo "$(GREEN)✓ Supervisor image built: $(SUPERVISOR_IMAGE)$(NC)"

## Build config-engine image
build-config-engine: build-base
	@echo "$(BLUE)Building config-engine image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--cache-from $(CONFIG_ENGINE_IMAGE) \
		-t $(CONFIG_ENGINE_IMAGE) \
		-f config-engine/Dockerfile \
		$(CONFIG_ENGINE_PATH)
	@echo "$(GREEN)✓ Config engine image built: $(CONFIG_ENGINE_IMAGE)$(NC)"

## Build api-server image
build-api-server: build-base
	@echo "$(BLUE)Building api-server image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--cache-from $(API_SERVER_IMAGE) \
		-t $(API_SERVER_IMAGE) \
		-f api-server/Dockerfile \
		$(API_SERVER_PATH)
	@echo "$(GREEN)✓ API server image built: $(API_SERVER_IMAGE)$(NC)"

## Build privileged-helper image
build-privileged-helper: build-base
	@echo "$(BLUE)Building privileged-helper image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--cache-from $(PRIVILEGED_HELPER_IMAGE) \
		-t $(PRIVILEGED_HELPER_IMAGE) \
		-f privileged-helper/Dockerfile \
		$(PRIVILEGED_HELPER_PATH)
	@echo "$(GREEN)✓ Privileged helper image built: $(PRIVILEGED_HELPER_IMAGE)$(NC)"

## Build ctl image
build-ctl: build-base
	@echo "$(BLUE)Building ctl image...$(NC)"
	@docker build \
		--build-arg BUILDKIT_INLINE_CACHE=1 \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--cache-from $(CTL_IMAGE) \
		-t $(CTL_IMAGE) \
		-f ctl/Dockerfile \
		$(CTL_PATH)
	@echo "$(GREEN)✓ CTL image built: $(CTL_IMAGE)$(NC)"

## Build images for multiple architectures
build-multiarch:
	@echo "$(BLUE)Building multi-architecture images...$(NC)"
	@for component in $(COMPONENTS); do \
		echo "Building $$component for $(PLATFORMS)..."; \
		docker buildx build \
			--platform $(PLATFORMS) \
			--build-arg BASE_IMAGE=$(BASE_IMAGE) \
			-t $(REGISTRY)/nrdot-$$component:$(TAG) \
			-f $$component/Dockerfile* \
			--push \
			$$(echo $$component | sed 's/base/../;s/collector/../otelcol-builder/;s/supervisor/../nrdot-supervisor/;s/config-engine/../nrdot-config-engine/;s/api-server/../nrdot-api-server/;s/privileged-helper/../nrdot-privileged-helper/;s/ctl/../nrdot-ctl/'); \
	done
	@echo "$(GREEN)✓ Multi-architecture images built and pushed$(NC)"

## Push all images to registry
push-all:
	@echo "$(BLUE)Pushing all images to $(REGISTRY)...$(NC)"
	@docker push $(BASE_IMAGE)
	@docker push $(COLLECTOR_IMAGE)
	@docker push $(SUPERVISOR_IMAGE)
	@docker push $(CONFIG_ENGINE_IMAGE)
	@docker push $(API_SERVER_IMAGE)
	@docker push $(PRIVILEGED_HELPER_IMAGE)
	@docker push $(CTL_IMAGE)
	@echo "$(GREEN)✓ All images pushed successfully$(NC)"

## Tag images for release
tag-release:
	@echo "$(BLUE)Tagging images for release $(TAG)...$(NC)"
	@for component in $(COMPONENTS); do \
		docker tag $(REGISTRY)/nrdot-$$component:latest $(REGISTRY)/nrdot-$$component:$(TAG); \
	done
	@echo "$(GREEN)✓ Images tagged for release $(TAG)$(NC)"

## Run security scans on all images
scan-all:
	@echo "$(BLUE)Running security scans...$(NC)"
	@for component in $(COMPONENTS); do \
		echo "Scanning $$component..."; \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
			aquasec/trivy image --severity HIGH,CRITICAL \
			$(REGISTRY)/nrdot-$$component:$(TAG) || true; \
	done
	@echo "$(GREEN)✓ Security scans completed$(NC)"

## Generate security scan report
scan-report:
	@echo "$(BLUE)Generating security scan report...$(NC)"
	@mkdir -p reports
	@for component in $(COMPONENTS); do \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
			aquasec/trivy image --format json --output reports/$$component-scan.json \
			$(REGISTRY)/nrdot-$$component:$(TAG) || true; \
	done
	@echo "$(GREEN)✓ Security reports generated in reports/$(NC)"

## Show image sizes
sizes:
	@echo "$(BLUE)Image sizes:$(NC)"
	@docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep nrdot

## Clean up Docker resources
clean:
	@echo "$(BLUE)Cleaning up Docker resources...$(NC)"
	@docker-compose down -v || true
	@docker-compose -f docker-compose.prod.yaml down -v || true
	@docker system prune -f
	@echo "$(GREEN)✓ Cleanup completed$(NC)"

## Run development environment
dev:
	@echo "$(BLUE)Starting development environment...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)✓ Development environment started$(NC)"
	@echo "Services available at:"
	@echo "  - API Server: http://localhost:8080"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Jaeger: http://localhost:16686"
	@echo "  - Grafana: http://localhost:3000"

## Run production environment
prod:
	@echo "$(BLUE)Starting production environment...$(NC)"
	@docker-compose -f docker-compose.prod.yaml up -d
	@echo "$(GREEN)✓ Production environment started$(NC)"

## Stop all environments
stop:
	@echo "$(BLUE)Stopping all environments...$(NC)"
	@docker-compose down || true
	@docker-compose -f docker-compose.prod.yaml down || true
	@echo "$(GREEN)✓ All environments stopped$(NC)"

## Show logs
logs:
	@docker-compose logs -f

## Run tests in containers
test:
	@echo "$(BLUE)Running tests in containers...$(NC)"
	@docker-compose -f docker-compose.test.yaml up --abort-on-container-exit
	@echo "$(GREEN)✓ Tests completed$(NC)"

## Display help
help:
	@echo "NRDOT-HOST Docker Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build-all          - Build all images"
	@echo "  build-base         - Build base image"
	@echo "  build-collector    - Build collector image"
	@echo "  build-supervisor   - Build supervisor image"
	@echo "  build-config-engine - Build config engine image"
	@echo "  build-api-server   - Build API server image"
	@echo "  build-privileged-helper - Build privileged helper image"
	@echo "  build-ctl          - Build CTL image"
	@echo "  build-multiarch    - Build for multiple architectures"
	@echo ""
	@echo "Registry targets:"
	@echo "  push-all           - Push all images to registry"
	@echo "  tag-release        - Tag images for release"
	@echo ""
	@echo "Security targets:"
	@echo "  scan-all           - Run security scans"
	@echo "  scan-report        - Generate scan reports"
	@echo ""
	@echo "Environment targets:"
	@echo "  dev                - Start development environment"
	@echo "  prod               - Start production environment"
	@echo "  stop               - Stop all environments"
	@echo "  logs               - Show container logs"
	@echo ""
	@echo "Utility targets:"
	@echo "  sizes              - Show image sizes"
	@echo "  clean              - Clean up resources"
	@echo "  test               - Run tests in containers"
	@echo "  help               - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  REGISTRY=$(REGISTRY)"
	@echo "  TAG=$(TAG)"
	@echo "  PLATFORMS=$(PLATFORMS)"