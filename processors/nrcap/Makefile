.PHONY: build test lint clean

build:
	go build ./...

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	go clean -testcache
	rm -rf bin/

coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench:
	go test -bench=. -benchmem ./...