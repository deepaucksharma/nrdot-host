# Makefile for nrdot-host unified binary

BINARY_NAME := nrdot-host
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS := -ldflags "\
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE) \
	-s -w"

# Directories
BUILD_DIR := ../../bin
DIST_DIR := ../../dist

.PHONY: all build clean test install

all: build

build:
	@echo "Building $(BINARY_NAME) $(VERSION)"
	@mkdir -p $(BUILD_DIR)
	GO_VERSION=$(GO_VERSION) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	# Darwin ARM64 (M1)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .

test:
	go test -v ./...

clean:
	@rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@rm -rf $(DIST_DIR)

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

# Development helpers
run-all: build
	$(BUILD_DIR)/$(BINARY_NAME) --mode=all --log-level=debug

run-agent: build
	$(BUILD_DIR)/$(BINARY_NAME) --mode=agent --log-level=debug

version: build
	$(BUILD_DIR)/$(BINARY_NAME) --mode=version