# WGO - Infrastructure Drift Detection Tool
# Makefile for building, testing, and development

# Variables
BINARY_NAME=wgo
MAIN_PATH=./cmd/wgo
BUILD_DIR=./build
COVERAGE_DIR=./coverage
TEST_TIMEOUT=10m

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty) -X main.commit=$(shell git rev-parse HEAD) -X main.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

.PHONY: all build clean test test-unit test-integration test-e2e test-coverage lint fmt help

# Default target
all: clean lint test build

# Help target
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  test-e2e       - Run end-to-end tests only"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-race      - Run tests with race detection"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  install        - Install binary to GOBIN"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-test    - Test Docker image"

# Build targets
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GOGET) -u $(MAIN_PATH)

# Test targets
test: test-unit test-integration test-e2e

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -timeout $(TEST_TIMEOUT) ./internal/... ./pkg/... ./cmd/...

test-integration:
	@echo "Running integration tests..."
	@$(MAKE) build
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./test/integration/...

test-e2e:
	@echo "Running end-to-end tests..."
	@$(MAKE) build
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./test/e2e/...

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out

test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -v -race -timeout $(TEST_TIMEOUT) ./...

bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...

# Code quality targets
lint:
	@echo "Running linters..."
	$(GOLINT) run --timeout=5m

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) mod tidy

# Dependency targets
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t wgo:latest .

docker-test: docker-build
	@echo "Testing Docker image..."
	docker run --rm wgo:latest --help

# Development targets
dev-setup:
	@echo "Setting up development environment..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

dev-test:
	@echo "Running quick development tests..."
	$(GOTEST) -short ./...

# Clean target
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f $(BINARY_NAME)

# Continuous Integration targets
ci-test: deps lint test-coverage

ci-build: clean ci-test build-all

# Release targets
release-check:
	@echo "Checking release readiness..."
	@git diff --exit-code || (echo "Working directory is dirty"; exit 1)
	@git diff --cached --exit-code || (echo "Index is dirty"; exit 1)
	@$(MAKE) ci-test

# Documentation targets
docs-serve:
	@echo "Serving documentation..."
	@command -v mkdocs >/dev/null 2>&1 || { echo "mkdocs not installed"; exit 1; }
	mkdocs serve

# Maintenance targets
mod-tidy:
	$(GOMOD) tidy

security-scan:
	@echo "Running security scan..."
	gosec ./...

# Performance profiling
profile-cpu:
	@echo "Running CPU profiling..."
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	$(GOCMD) tool pprof cpu.prof

profile-mem:
	@echo "Running memory profiling..."
	$(GOTEST) -memprofile=mem.prof -bench=. ./...
	$(GOCMD) tool pprof mem.prof

# Quick validation before commit
pre-commit: fmt lint test-unit
	@echo "Pre-commit checks passed!"

# Full validation
validate: clean deps lint test-coverage build-all
	@echo "Full validation completed!"