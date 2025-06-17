.PHONY: build test clean lint fmt

build:
	go build -o bin/nrdot-supervisor cmd/supervisor/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

coverage: test
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html

lint:
	golangci-lint run

fmt:
	go fmt ./...

run: build
	./bin/nrdot-supervisor

install:
	go install ./cmd/supervisor