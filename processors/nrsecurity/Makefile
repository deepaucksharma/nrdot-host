.PHONY: all build test clean lint fmt

all: lint test build

build:
	go build -v ./...

test:
	go test -v -race -cover ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping linting"; \
	fi

fmt:
	go fmt ./...

clean:
	go clean -testcache
	rm -rf coverage.out

coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench:
	go test -bench=. -benchmem ./...

deps:
	go mod download
	go mod tidy

.DEFAULT_GOAL := all