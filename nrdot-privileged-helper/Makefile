# nrdot-privileged-helper Makefile

.PHONY: all build test test-unit clean lint fmt deps install

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build parameters
BINARY_NAME=nrdot-privileged-helper
BINARY_PATH=cmd/privileged-helper
LDFLAGS=-ldflags "-s -w"
TESTFLAGS=-race -cover -coverprofile=coverage.out

# Installation parameters
INSTALL_PATH=/usr/local/bin
SETUID_USER=root
SETUID_GROUP=nrdot

all: clean deps lint test build

build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./$(BINARY_PATH)

test: test-unit

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) $(TESTFLAGS) ./...

clean:
	@echo "Cleaning..."
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -f coverage.html

lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

install: build
	@echo "Installing $(BINARY_NAME)..."
	@echo "Note: This requires sudo privileges"
	sudo cp $(BINARY_NAME) $(INSTALL_PATH)/
	sudo chown $(SETUID_USER):$(SETUID_GROUP) $(INSTALL_PATH)/$(BINARY_NAME)
	sudo chmod 4750 $(INSTALL_PATH)/$(BINARY_NAME)

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)

coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Component-specific targets used by parent Makefile
.PHONY: component-build component-test component-clean

component-build: build
component-test: test
component-clean: clean