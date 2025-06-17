.PHONY: all build test clean lint run deps

# Build binary
build:
	go build -o bin/config-engine cmd/config-engine/main.go

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Run linter
lint:
	golangci-lint run

# Run the config engine
run: build
	./bin/config-engine

# Update dependencies
deps:
	go mod download
	go mod tidy

# Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Format code
fmt:
	go fmt ./...

# Run all checks
check: fmt lint test

all: clean deps build test