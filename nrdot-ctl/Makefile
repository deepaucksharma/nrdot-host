.PHONY: all build test clean install fmt vet lint completion

BINARY_NAME=nrdot-ctl
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/newrelic/nrdot-host/nrdot-ctl/cmd.Version=${VERSION} -X github.com/newrelic/nrdot-host/nrdot-ctl/cmd.BuildTime=${BUILD_TIME}"

all: fmt vet test build

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

test:
	go test -v ./...

clean:
	go clean
	rm -f ${BINARY_NAME}
	rm -f completion/*

install: build
	sudo cp ${BINARY_NAME} /usr/local/bin/

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

completion:
	mkdir -p completion
	./${BINARY_NAME} completion bash > completion/${BINARY_NAME}.bash
	./${BINARY_NAME} completion zsh > completion/${BINARY_NAME}.zsh
	./${BINARY_NAME} completion fish > completion/${BINARY_NAME}.fish

run: build
	./${BINARY_NAME}

# Development helpers
dev-status: build
	./${BINARY_NAME} status

dev-config-validate: build
	./${BINARY_NAME} config validate -f example-config.yaml

dev-help: build
	./${BINARY_NAME} --help