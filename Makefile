# Makefile for gitlab-cli
# Go project with MCP server

.PHONY: help build test install clean coverage lint fmt vet

# Default target
.DEFAULT_GOAL := help

# Binary name
BINARY_NAME=gitlab-cli
BINARY_PATH=./bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Installation path
INSTALL_PATH=$(HOME)/.local/bin

# Version info (matches install.sh)
VERSION=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/user/gitlab-cli/internal/cli.Version=$(VERSION) -X github.com/user/gitlab-cli/internal/cli.BuildTime=$(BUILD_TIME)"

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  make build        - Build the gitlab-cli binary"
	@echo "  make test         - Run all tests"
	@echo "  make test-mcp     - Run MCP server tests only"
	@echo "  make test-cli     - Run CLI tests only"
	@echo "  make install      - Install binary to ~/.local/bin"
	@echo "  make clean        - Remove built binaries and test cache"
	@echo "  make coverage     - Run tests with coverage report"
	@echo "  make lint         - Run linters (fmt, vet)"
	@echo "  make fmt          - Format code"
	@echo "  make vet          - Run go vet"

## build: Build the gitlab-cli binary
build: fmt vet
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p bin
	$(GOMOD) download
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/gitlab-cli
	@echo "✓ Build complete: $(BINARY_PATH)"

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...
	@echo "✓ All tests passed"

## test-mcp: Run MCP server tests only
test-mcp:
	@echo "Running MCP tests..."
	$(GOTEST) -v ./internal/mcp/...
	@echo "✓ MCP tests passed"

## test-cli: Run CLI tests only
test-cli:
	@echo "Running CLI tests..."
	$(GOTEST) -v ./internal/cli/...
	@echo "✓ CLI tests passed"

## coverage: Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## install: Build and install binary to ~/.local/bin (matches install.sh)
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@mv $(BINARY_PATH) $(INSTALL_PATH)/$(BINARY_NAME)
	@chmod +x $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✓ Installed to $(INSTALL_PATH)/$(BINARY_NAME)"
	@echo "Run '$(BINARY_NAME) version' to verify installation"

## clean: Remove built binaries and test cache
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

## fmt: Format all Go files
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## lint: Run all linters
lint: fmt vet
	@echo "✓ Linting complete"

## deps: Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✓ Dependencies updated"
