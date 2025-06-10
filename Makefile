# Variables
BINARY_NAME := gwq
PACKAGE := github.com/d-kuro/gwq
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO_FILES := $(shell find . -name '*.go' -type f -not -path './vendor/*')

# Build variables
BUILD_DIR := build
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

.PHONY: all build clean test test-verbose test-coverage lint fmt vet install help

# Default target
all: clean build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) cmd/gwq/main.go

## build-all: Build for multiple platforms
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# macOS AMD64
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/gwq/main.go
	# macOS ARM64 (Apple Silicon)
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/gwq/main.go
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/gwq/main.go
	# Linux ARM64
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 cmd/gwq/main.go
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/gwq/main.go
	@echo "Build complete. Binaries are in $(BUILD_DIR)/"

## clean: Clean build files
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## test: Run tests
test:
	@echo "Running tests..."
	@go test ./...

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...

## test-coverage-report: Generate and open coverage report
test-coverage-report:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

## lint: Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: brew install golangci-lint"; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w $(GO_FILES); \
	else \
		echo "goimports not installed. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

## mod: Tidy and verify go modules
mod:
	@echo "Tidying modules..."
	@go mod tidy
	@echo "Verifying modules..."
	@go mod verify

## install: Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install -ldflags "$(LDFLAGS)" ./cmd/gwq

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
