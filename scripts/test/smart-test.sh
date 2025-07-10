#!/bin/bash

# Smart Test Script - Only run tests for changed components
# This script detects which files have changed and runs only the relevant tests

set -e

# Colors
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

# Configuration
TIMEOUT="10m"
VERBOSE=${VERBOSE:-false}

log() {
    echo -e "${CYAN}[SMART-TEST]${RESET} $1"
}

success() {
    echo -e "${GREEN}✅ $1${RESET}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${RESET}"
}

error() {
    echo -e "${RED}❌ $1${RESET}"
}

# Get changed files
get_changed_files() {
    # Try different methods to detect changes
    local changed_files=""
    
    # Method 1: Git diff against main/master
    if git rev-parse --verify main >/dev/null 2>&1; then
        changed_files=$(git diff --name-only main...HEAD 2>/dev/null || true)
    elif git rev-parse --verify master >/dev/null 2>&1; then
        changed_files=$(git diff --name-only master...HEAD 2>/dev/null || true)
    fi
    
    # Method 2: If no main/master, use staged + unstaged files
    if [[ -z "$changed_files" ]]; then
        changed_files=$(git diff --name-only HEAD 2>/dev/null || true)
        local staged=$(git diff --name-only --cached 2>/dev/null || true)
        changed_files="$changed_files $staged"
    fi
    
    # Method 3: If still nothing, use recent commits
    if [[ -z "$changed_files" ]]; then
        changed_files=$(git diff --name-only HEAD~1 2>/dev/null || true)
    fi
    
    echo "$changed_files" | grep -E '\.(go|mod|sum)$' | sort -u || true
}

# Analyze which components need testing
analyze_changes() {
    local files="$1"
    local components=""
    
    for file in $files; do
        case "$file" in
            internal/collectors/terraform/*)
                components="$components terraform"
                ;;
            internal/collectors/gcp/*)
                components="$components gcp"
                ;;
            internal/collectors/aws/*)
                components="$components aws"
                ;;
            internal/collectors/kubernetes/*)
                components="$components kubernetes"
                ;;
            internal/collectors/*)
                components="$components collectors"
                ;;
            cmd/vaino/commands/*)
                components="$components commands"
                ;;
            pkg/config/*)
                components="$components config"
                ;;
            internal/differ/*)
                components="$components differ"
                ;;
            internal/output/*)
                components="$components output"
                ;;
            go.mod|go.sum)
                components="$components deps"
                ;;
            *.go)
                # Generic Go file - might affect multiple components
                components="$components generic"
                ;;
        esac
    done
    
    echo "$components" | tr ' ' '\n' | sort -u | tr '\n' ' '
}

# Run tests for specific component
run_component_test() {
    local component="$1"
    local test_cmd=""
    
    case "$component" in
        terraform)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/collectors/terraform/..."
            ;;
        gcp)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/collectors/gcp/..."
            ;;
        aws)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/collectors/aws/..."
            ;;
        kubernetes)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/collectors/kubernetes/..."
            ;;
        collectors)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/collectors/..."
            ;;
        commands)
            test_cmd="go test -v -timeout $TIMEOUT ./cmd/vaino/commands/..."
            ;;
        config)
            test_cmd="go test -v -timeout $TIMEOUT ./pkg/config/..."
            ;;
        differ)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/differ/..."
            ;;
        output)
            test_cmd="go test -v -timeout $TIMEOUT ./internal/output/..."
            ;;
        deps)
            log "Dependency changes detected - running module verification"
            go mod tidy
            go mod verify
            return $?
            ;;
        generic)
            log "Generic Go changes - running core tests"
            test_cmd="go test -v -timeout $TIMEOUT ./internal/... ./pkg/..."
            ;;
        *)
            warning "Unknown component: $component"
            return 0
            ;;
    esac
    
    if [[ -n "$test_cmd" ]]; then
        log "Running tests for $component: $test_cmd"
        if $test_cmd; then
            success "$component tests passed"
            return 0
        else
            error "$component tests failed"
            return 1
        fi
    fi
}

# Main execution
main() {
    log "Starting smart test analysis..."
    
    # Get changed files
    local changed_files
    changed_files=$(get_changed_files)
    
    if [[ -z "$changed_files" ]]; then
        log "No changed files detected - running basic unit tests"
        go test -short ./...
        success "Basic tests completed"
        return 0
    fi
    
    log "Changed files detected:"
    for file in $changed_files; do
        echo "  📄 $file"
    done
    echo
    
    # Analyze which components are affected
    local components
    components=$(analyze_changes "$changed_files")
    
    if [[ -z "$components" ]]; then
        log "No testable components affected - skipping tests"
        return 0
    fi
    
    log "Components to test: $components"
    echo
    
    # Run tests for each affected component
    local failed_components=""
    local test_count=0
    
    for component in $components; do
        ((test_count++))
        if ! run_component_test "$component"; then
            failed_components="$failed_components $component"
        fi
        echo
    done
    
    # Summary
    if [[ -n "$failed_components" ]]; then
        error "Tests failed for components:$failed_components"
        echo
        echo "To run all tests: make test-all"
        echo "To run specific component: make test-<component>"
        return 1
    else
        success "All $test_count component test(s) passed!"
        echo
        echo "💡 Smart testing saved time by only testing affected components"
        echo "   To run full test suite: make test-all"
        return 0
    fi
}

# Handle arguments
case "${1:-}" in
    --help|-h)
        echo "Smart Test Script"
        echo "=================="
        echo "Automatically detects changed files and runs only relevant tests"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help"
        echo "  --verbose, -v  Enable verbose output"
        echo "  --all         Force run all tests"
        echo ""
        echo "Environment variables:"
        echo "  VERBOSE=true   Enable verbose output"
        exit 0
        ;;
    --verbose|-v)
        VERBOSE=true
        ;;
    --all)
        log "Force running all tests..."
        exec go test -v -timeout $TIMEOUT ./...
        ;;
esac

# Run main function
main "$@"