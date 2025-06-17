# E2E Tests Makefile

SCENARIOS := microservices kubernetes host-monitoring security-compliance high-cardinality
DOCKER_COMPOSE := docker-compose
TIMEOUT := 300
REPORT_DIR := reports

.PHONY: all test-all $(SCENARIOS) setup teardown clean report help

## Default target
all: test-all

## Run all E2E test scenarios
test-all: setup
	@echo "Running all E2E test scenarios..."
	@mkdir -p $(REPORT_DIR)
	@for scenario in $(SCENARIOS); do \
		echo "\n=== Running $$scenario scenario ===" ; \
		$(MAKE) test-$$scenario || exit 1 ; \
	done
	@$(MAKE) report
	@echo "\nAll scenarios completed successfully!"

## Setup test environment
setup:
	@echo "Setting up E2E test environment..."
	@./scripts/setup.sh
	@docker network create nrdot-test 2>/dev/null || true

## Teardown test environment
teardown:
	@echo "Tearing down E2E test environment..."
	@./scripts/teardown.sh
	@docker network rm nrdot-test 2>/dev/null || true

## Clean all test artifacts
clean: teardown
	@echo "Cleaning test artifacts..."
	@rm -rf $(REPORT_DIR)
	@find . -name "*.log" -delete
	@find . -name "*.tmp" -delete

## Generate test report
report:
	@echo "Generating test report..."
	@./scripts/generate-report.sh

## Test microservices scenario
test-microservices:
	@echo "Testing microservices monitoring..."
	@cd scenarios/microservices && \
		$(DOCKER_COMPOSE) up -d && \
		timeout $(TIMEOUT) ./tests/test.sh && \
		$(DOCKER_COMPOSE) down -v

## Test kubernetes scenario
test-kubernetes:
	@echo "Testing Kubernetes monitoring..."
	@cd scenarios/kubernetes && \
		./test.sh

## Test host monitoring scenario
test-host-monitoring:
	@echo "Testing host monitoring..."
	@cd scenarios/host-monitoring && \
		$(DOCKER_COMPOSE) up -d && \
		timeout $(TIMEOUT) ./tests/test.sh && \
		$(DOCKER_COMPOSE) down -v

## Test security compliance scenario
test-security-compliance:
	@echo "Testing security compliance..."
	@cd scenarios/security-compliance && \
		$(DOCKER_COMPOSE) up -d && \
		timeout $(TIMEOUT) ./tests/test.sh && \
		$(DOCKER_COMPOSE) down -v

## Test high cardinality scenario
test-high-cardinality:
	@echo "Testing high cardinality protection..."
	@cd scenarios/high-cardinality && \
		$(DOCKER_COMPOSE) up -d && \
		timeout $(TIMEOUT) ./tests/test.sh && \
		$(DOCKER_COMPOSE) down -v

## Run scenario with debug output
debug-%:
	@DEBUG=1 $(MAKE) test-$*

## View logs for a scenario
logs-%:
	@cd scenarios/$* && $(DOCKER_COMPOSE) logs -f

## Get status of running scenarios
status:
	@echo "Checking status of E2E test scenarios..."
	@for scenario in $(SCENARIOS); do \
		echo "\n=== $$scenario ===" ; \
		cd scenarios/$$scenario && $(DOCKER_COMPOSE) ps 2>/dev/null || echo "Not running" ; \
		cd ../.. ; \
	done

## Run quick smoke tests
smoke-test: setup
	@echo "Running smoke tests..."
	@cd scenarios/microservices && \
		$(DOCKER_COMPOSE) up -d nrdot-collector && \
		sleep 10 && \
		curl -f http://localhost:13133/health || exit 1 && \
		$(DOCKER_COMPOSE) down -v
	@echo "Smoke tests passed!"

## Run performance benchmarks
benchmark:
	@echo "Running performance benchmarks..."
	@cd scenarios/high-cardinality && \
		BENCHMARK=1 $(MAKE) test-high-cardinality

## Display help
help:
	@echo "NRDOT-HOST E2E Tests"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  test-all                 Run all E2E test scenarios"
	@echo "  test-microservices       Test microservices monitoring"
	@echo "  test-kubernetes          Test Kubernetes monitoring"  
	@echo "  test-host-monitoring     Test host monitoring"
	@echo "  test-security-compliance Test security compliance"
	@echo "  test-high-cardinality    Test cardinality protection"
	@echo "  setup                    Setup test environment"
	@echo "  teardown                 Teardown test environment"
	@echo "  clean                    Clean all test artifacts"
	@echo "  report                   Generate test report"
	@echo "  status                   Check scenario status"
	@echo "  smoke-test              Run quick smoke tests"
	@echo "  benchmark               Run performance benchmarks"
	@echo "  debug-<scenario>        Run scenario with debug output"
	@echo "  logs-<scenario>         View logs for scenario"
	@echo "  help                    Display this help message"