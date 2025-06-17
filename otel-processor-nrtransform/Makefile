.PHONY: all build test clean lint fmt

all: build test

build:
	go build -v ./...

test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	go clean
	rm -f coverage.out coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -w .

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

tidy:
	go mod tidy

# Generate test coverage report
coverage: test
	go tool cover -func=coverage.out