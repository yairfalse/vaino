# VAINO - Ancient Finnish Wisdom for Modern Infrastructure
# Divine Makefile for the creator god's build system

# Variables
BINARY_NAME=vaino
MAIN_PATH=./cmd/vaino
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
GOLINT=$(shell go env GOPATH)/bin/golangci-lint

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
		test-changed test-parallel perf-test perf-bench perf-stress perf-memory perf-concurrent \
		perf-large-dataset perf-quick perf-profile perf-report check-deps \
		agent-start agent-status agent-check pr-ready install release

# Default target
all: clean lint test build

# Help target
help:
	@echo "$(CYAN)VAINO Build & Test System$(RESET)"
	@echo "============================="
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
	@echo ""
	@echo "$(GREEN)⚡ Performance Testing:$(RESET)"
	@echo "  perf-test         Run comprehensive performance tests"
	@echo "  perf-bench        Run performance benchmarks"
	@echo "  perf-stress       Run stress tests"
	@echo "  perf-memory       Run memory analysis tests"
	@echo "  perf-concurrent   Run concurrent operation tests"
	@echo "  perf-large-dataset Run large dataset tests"
	@echo "  perf-quick        Run quick performance tests"
	@echo "  perf-profile      Run performance tests with profiling"
	@echo "  perf-report       Show latest performance report"
	@echo "  perf-ci           Run CI performance tests (reduced set)"
	@echo ""
	@echo "$(GREEN)🤖 Agent Management:$(RESET)"
	@echo "  agent-start       Start interactive agent creation"
	@echo "  agent-status      Show agent status and active work"
	@echo "  agent-check       Run quality checks (alias for pr-ready)"
	@echo "  pr-ready          Complete quality checks before PR"

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
	@echo "$(CYAN)Running unit tests...$(RESET)"
	@$(MAKE) test-unit

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
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./cmd/vaino/commands/...
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
	 $(GOTEST) ./cmd/vaino/commands/... -v -timeout $(TEST_TIMEOUT) & \
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

arch-check:
	@echo "$(CYAN)Checking architectural level violations...$(RESET)"
	@go run tools/archcheck/main.go

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
	docker build -t vaino:latest .

docker-test: docker-build
	@echo "Testing Docker image..."
	docker run --rm vaino:latest --help

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
ci-test: deps lint arch-check test-coverage

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

# Performance Testing targets
perf-test:
	@echo "$(CYAN)Running comprehensive performance tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...
	@echo "$(GREEN)✅ Performance tests completed$(RESET)"

perf-bench:
	@echo "$(CYAN)Running performance benchmarks...$(RESET)"
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...
	@echo "$(GREEN)✅ Performance benchmarks completed$(RESET)"

perf-stress:
	@echo "$(CYAN)Running stress tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -count=5 -run=^$$ ./...
	@echo "$(GREEN)✅ Stress tests completed$(RESET)"

perf-memory:
	@echo "$(CYAN)Running memory analysis tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -memprofile=mem.prof -run=^$$ ./...
	@echo "$(GREEN)✅ Memory tests completed$(RESET)"

perf-concurrent:
	@echo "$(CYAN)Running concurrent operation tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -cpu=1,2,4 -run=^$$ ./...
	@echo "$(GREEN)✅ Concurrent tests completed$(RESET)"

perf-large-dataset:
	@echo "$(CYAN)Running large dataset tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -benchtime=10s -run=^$$ ./...
	@echo "$(GREEN)✅ Large dataset tests completed$(RESET)"

perf-quick:
	@echo "$(CYAN)Running quick performance tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -benchtime=1s -run=^$$ ./...
	@echo "$(GREEN)✅ Quick performance tests completed$(RESET)"

perf-profile:
	@echo "$(CYAN)Running performance tests with profiling...$(RESET)"
	$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof -run=^$$ ./...
	@echo "$(GREEN)✅ Performance profiling completed$(RESET)"

perf-report:
	@echo "$(CYAN)Generating performance report...$(RESET)"
	@echo "$(YELLOW)Performance report functionality not implemented yet$(RESET)"

# CI Performance testing (reduced test set)
perf-ci:
	@echo "$(CYAN)Running CI performance tests...$(RESET)"
	$(GOTEST) -bench=. -benchmem -benchtime=1s -run=^$$ ./...
	@echo "$(GREEN)✅ CI performance tests completed$(RESET)"

# Dependency checking
check-deps:
	@echo "$(CYAN)Checking dependencies...$(RESET)"
	$(GOMOD) verify
	$(GOMOD) download
	@echo "$(GREEN)✅ Dependencies verified$(RESET)"

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

# Agent Management targets
agent-start:
	@echo "$(CYAN)Starting agent creation...$(RESET)"
	@./scripts/agent-branch.sh start
	@echo "$(GREEN)✅ Agent created successfully$(RESET)"

agent-status:
	@echo "$(CYAN)Checking agent status...$(RESET)"
	@./scripts/agent-branch.sh status

agent-check: pr-ready
	@echo "$(GREEN)✅ Agent quality checks passed$(RESET)"

pr-ready: fmt build
	@echo "$(CYAN)Running comprehensive quality checks...$(RESET)"
	@echo "$(YELLOW)Checking code formatting...$(RESET)"
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "$(RED)❌ Code is not formatted. Run 'make fmt' first.$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Code formatting OK$(RESET)"
	@echo "$(YELLOW)Checking for untracked files...$(RESET)"
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "$(YELLOW)⚠️  Untracked files present - ensure they're intentional$(RESET)"; \
		git status --porcelain; \
	fi
	@echo "$(YELLOW)Checking build...$(RESET)"
	@if [ ! -f "$(BUILD_DIR)/$(BINARY_NAME)" ]; then \
		echo "$(RED)❌ Build failed or binary not found$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Build OK$(RESET)"
	@echo "$(YELLOW)Checking agent registration...$(RESET)"
	@if [ -d ".agent-work" ]; then \
		echo "$(GREEN)✅ Agent system initialized$(RESET)"; \
	else \
		echo "$(YELLOW)⚠️  Agent system not initialized (run 'make agent-start' first)$(RESET)"; \
	fi
	@echo "$(GREEN)✅ All quality checks passed - ready for PR!$(RESET)"

# Simple installation for local development
install: build
	@echo "$(CYAN)Installing VAINO locally...$(RESET)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✅ VAINO installed to /usr/local/bin/$(RESET)"

# Create release binaries without CI
release:
	@echo "$(CYAN)Creating release binaries...$(RESET)"
	@./scripts/manual-release.sh $(VERSION)
	@echo "$(GREEN)✅ Release binaries created in dist/$(RESET)"