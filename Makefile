.PHONY: lint lint-fix lint-install help

# Go parameters
GOLANGCI_LINT_VERSION := latest
GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo "$(HOME)/go/bin/golangci-lint")

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## lint: Run golangci-lint on the codebase
lint:
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run ./...

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@$(GOLANGCI_LINT) run --fix ./...

## lint-install: Install golangci-lint
lint-install:
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

## lint-verbose: Run golangci-lint with verbose output
lint-verbose:
	@echo "Running golangci-lint with verbose output..."
	@$(GOLANGCI_LINT) run -v ./...