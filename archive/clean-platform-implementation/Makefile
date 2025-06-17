# Platform Makefile

.PHONY: help build test deploy-dev deploy-staging deploy-prod clean

DOCKER_REGISTRY ?= your-registry.com
VERSION ?= $(shell git rev-parse --short HEAD)
SERVICES := api-gateway data-collector data-processor

help:
	@echo "Available targets:"
	@echo "  build          - Build all services"
	@echo "  test           - Run all tests"
	@echo "  deploy-dev     - Deploy to development"
	@echo "  deploy-staging - Deploy to staging"
	@echo "  deploy-prod    - Deploy to production"
	@echo "  clean          - Clean build artifacts"

build:
	@echo "Building services..."
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		docker build -t $(DOCKER_REGISTRY)/$$service:$(VERSION) services/$$service; \
	done

test:
	@echo "Running tests..."
	@python -m pytest tests/ -v
	@echo "Running linters..."
	@flake8 services/
	@echo "Running security scan..."
	@trivy image $(DOCKER_REGISTRY)/api-gateway:$(VERSION)

push:
	@echo "Pushing images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service..."; \
		docker push $(DOCKER_REGISTRY)/$$service:$(VERSION); \
	done

deploy-dev: build push
	@echo "Deploying to development..."
	@kubectl apply -k infrastructure/kubernetes/overlays/dev
	@kubectl set image deployment/api-gateway api-gateway=$(DOCKER_REGISTRY)/api-gateway:$(VERSION) -n platform-dev
	@kubectl set image deployment/data-collector data-collector=$(DOCKER_REGISTRY)/data-collector:$(VERSION) -n platform-dev
	@kubectl set image deployment/data-processor data-processor=$(DOCKER_REGISTRY)/data-processor:$(VERSION) -n platform-dev

deploy-staging:
	@echo "Deploying to staging..."
	@kubectl apply -k infrastructure/kubernetes/overlays/staging
	@kubectl set image deployment/api-gateway api-gateway=$(DOCKER_REGISTRY)/api-gateway:$(VERSION) -n platform-staging
	@kubectl set image deployment/data-collector data-collector=$(DOCKER_REGISTRY)/data-collector:$(VERSION) -n platform-staging
	@kubectl set image deployment/data-processor data-processor=$(DOCKER_REGISTRY)/data-processor:$(VERSION) -n platform-staging

deploy-prod:
	@echo "Deploying to production..."
	@read -p "Are you sure you want to deploy to production? [y/N] " confirm && \
	if [ "$$confirm" = "y" ]; then \
		kubectl apply -k infrastructure/kubernetes/overlays/prod; \
		kubectl set image deployment/api-gateway api-gateway=$(DOCKER_REGISTRY)/api-gateway:$(VERSION) -n platform-prod; \
		kubectl set image deployment/data-collector data-collector=$(DOCKER_REGISTRY)/data-collector:$(VERSION) -n platform-prod; \
		kubectl set image deployment/data-processor data-processor=$(DOCKER_REGISTRY)/data-processor:$(VERSION) -n platform-prod; \
	fi

clean:
	@echo "Cleaning build artifacts..."
	@docker rmi $(DOCKER_REGISTRY)/api-gateway:$(VERSION) || true
	@docker rmi $(DOCKER_REGISTRY)/data-collector:$(VERSION) || true
	@docker rmi $(DOCKER_REGISTRY)/data-processor:$(VERSION) || true

# Infrastructure targets
infra-init:
	@cd infrastructure/terraform/environments/$(ENV) && terraform init

infra-plan:
	@cd infrastructure/terraform/environments/$(ENV) && terraform plan

infra-apply:
	@cd infrastructure/terraform/environments/$(ENV) && terraform apply

infra-destroy:
	@cd infrastructure/terraform/environments/$(ENV) && terraform destroy

# Local development
local-up:
	@docker-compose -f docker-compose.local.yml up -d

local-down:
	@docker-compose -f docker-compose.local.yml down

local-logs:
	@docker-compose -f docker-compose.local.yml logs -f

# Database migrations
db-migrate:
	@python scripts/db-migrate.py

db-rollback:
	@python scripts/db-migrate.py --rollback

# Monitoring
port-forward-grafana:
	@kubectl port-forward -n monitoring svc/grafana 3000:80

port-forward-prometheus:
	@kubectl port-forward -n monitoring svc/prometheus 9090:9090