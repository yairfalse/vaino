# WGO - Infrastructure Drift Detection Tool
# Modular Makefile for targeted testing and building

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

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
RESET := \033[0m

.PHONY: all build clean test test-all test-unit test-integration test-e2e test-coverage lint fmt help \
		test-collectors test-terraform test-gcp test-aws test-kubernetes test-commands test-config \
		test-changed test-parallel

# Default target
all: clean lint test build

# Help target
help:
	@echo "$(CYAN)WGO Build & Test System$(RESET)"
	@echo "========================"
	@echo ""
	@echo "$(GREEN)🔨 Build Targets:$(RESET)"
	@echo "  build              Build the binary"
	@echo "  build-all          Build for all platforms"
	@echo "  install            Install binary to GOBIN"
	@echo "  clean              Clean build artifacts"
	@echo ""
	@echo "$(GREEN)🧪 Test Targets:$(RESET)"
	@echo "  test               Run tests for changed components only"
	@echo "  test-all           Run all tests (full suite)"
	@echo "  test-unit          Run unit tests only"
	@echo "  test-integration   Run integration tests only"
	@echo "  test-e2e           Run end-to-end tests only"
	@echo "  test-parallel      Run tests in parallel"
	@echo ""
	@echo "$(GREEN)📦 Component Tests:$(RESET)"
	@echo "  test-collectors    Test all collectors"
	@echo "  test-terraform     Test Terraform collector only"
	@echo "  test-gcp          Test GCP collector only" 
	@echo "  test-aws          Test AWS collector only"
	@echo "  test-kubernetes   Test Kubernetes collector only"
	@echo "  test-commands     Test CLI commands"
	@echo "  test-config       Test configuration system"
	@echo ""
	@echo "$(GREEN)🔍 Quality Targets:$(RESET)"
	@echo "  lint              Run linters"
	@echo "  fmt               Format code"
	@echo "  test-coverage     Run tests with coverage report"
	@echo "  deps              Download dependencies"

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

# Smart Test - Run tests for changed components only (default)
test:
	@echo "$(CYAN)Running smart tests (changed components only)...$(RESET)"
	@./scripts/smart-test.sh || $(MAKE) test-unit

# Test All - Run complete test suite
test-all: test-unit test-integration test-e2e

# Unit Tests - Fast unit tests only
test-unit:
	@echo "$(CYAN)Running unit tests...$(RESET)"
	$(GOTEST) -v -race -timeout $(TEST_TIMEOUT) ./internal/... ./pkg/... ./cmd/...
	@echo "$(GREEN)✅ Unit tests completed$(RESET)"

test-integration:
	@echo "$(CYAN)Running integration tests...$(RESET)"
	@$(MAKE) build
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./test/integration/...
	@echo "$(GREEN)✅ Integration tests completed$(RESET)"

test-e2e:
	@echo "$(CYAN)Running end-to-end tests...$(RESET)"
	@$(MAKE) build
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./test/e2e/...
	@echo "$(GREEN)✅ E2E tests completed$(RESET)"

# Component-specific test targets
test-collectors:
	@echo "$(CYAN)Testing all collectors...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/collectors/...
	@echo "$(GREEN)✅ Collector tests completed$(RESET)"

test-terraform:
	@echo "$(CYAN)Testing Terraform collector...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/collectors/terraform/...
	@echo "$(GREEN)✅ Terraform tests completed$(RESET)"

test-gcp:
	@echo "$(CYAN)Testing GCP collector...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/collectors/gcp/...
	@echo "$(GREEN)✅ GCP tests completed$(RESET)"

test-aws:
	@echo "$(CYAN)Testing AWS collector...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/collectors/aws/...
	@echo "$(GREEN)✅ AWS tests completed$(RESET)"

test-kubernetes:
	@echo "$(CYAN)Testing Kubernetes collector...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/collectors/kubernetes/...
	@echo "$(GREEN)✅ Kubernetes tests completed$(RESET)"

test-commands:
	@echo "$(CYAN)Testing CLI commands...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./cmd/wgo/commands/...
	@echo "$(GREEN)✅ Command tests completed$(RESET)"

test-config:
	@echo "$(CYAN)Testing configuration system...$(RESET)"
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./pkg/config/...
	@echo "$(GREEN)✅ Configuration tests completed$(RESET)"

# Parallel test execution for CI
test-parallel:
	@echo "$(CYAN)Running tests in parallel...$(RESET)"
	@$(GOTEST) ./internal/collectors/terraform/... -v -timeout $(TEST_TIMEOUT) & \
	 $(GOTEST) ./internal/collectors/gcp/... -v -timeout $(TEST_TIMEOUT) & \
	 $(GOTEST) ./internal/collectors/aws/... -v -timeout $(TEST_TIMEOUT) & \
	 $(GOTEST) ./internal/collectors/kubernetes/... -v -timeout $(TEST_TIMEOUT) & \
	 $(GOTEST) ./cmd/wgo/commands/... -v -timeout $(TEST_TIMEOUT) & \
	 $(GOTEST) ./pkg/config/... -v -timeout $(TEST_TIMEOUT) & \
	 wait
	@echo "$(GREEN)✅ Parallel tests completed$(RESET)"

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