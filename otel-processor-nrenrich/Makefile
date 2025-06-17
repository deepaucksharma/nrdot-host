.PHONY: all build test clean lint fmt

all: build test

build:
	go build -v ./...

test:
	go test -v -race -cover ./...

clean:
	go clean
	rm -f coverage.out

lint:
	golangci-lint run

fmt:
	go fmt ./...
	goimports -w .

tidy:
	go mod tidy

coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out