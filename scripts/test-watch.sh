#!/bin/bash
# Test runner for watch mode comprehensive testing

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test categories
UNIT_TESTS=true
INTEGRATION_TESTS=true
E2E_TESTS=true
PERFORMANCE_TESTS=true
EDGE_TESTS=true
BENCHMARKS=true

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            INTEGRATION_TESTS=false
            E2E_TESTS=false
            PERFORMANCE_TESTS=false
            EDGE_TESTS=false
            BENCHMARKS=false
            shift
            ;;
        --skip-e2e)
            E2E_TESTS=false
            shift
            ;;
        --skip-benchmarks)
            BENCHMARKS=false
            shift
            ;;
        --quick)
            E2E_TESTS=false
            PERFORMANCE_TESTS=false
            BENCHMARKS=false
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--unit-only] [--skip-e2e] [--skip-benchmarks] [--quick]"
            exit 1
            ;;
    esac
done

# Helper functions
print_header() {
    echo -e "\n${YELLOW}========================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}========================================${NC}\n"
}

run_tests() {
    local test_name=$1
    local test_command=$2
    
    echo -e "${GREEN}Running $test_name...${NC}"
    if eval "$test_command"; then
        echo -e "${GREEN}✓ $test_name passed${NC}\n"
        return 0
    else
        echo -e "${RED}✗ $test_name failed${NC}\n"
        return 1
    fi
}

# Track failures
FAILED_TESTS=()

# Run unit tests
if [ "$UNIT_TESTS" = true ]; then
    print_header "UNIT TESTS"
    
    # Basic watcher tests
    run_tests "Watcher Unit Tests" \
        "go test -v ./internal/watcher/... -run '^Test[^E]'" || \
        FAILED_TESTS+=("Watcher Unit Tests")
fi

# Run edge case tests
if [ "$EDGE_TESTS" = true ]; then
    print_header "EDGE CASE TESTS"
    
    run_tests "Watcher Edge Cases" \
        "go test -v ./internal/watcher/... -run 'Edge'" || \
        FAILED_TESTS+=("Watcher Edge Cases")
fi

# Run integration tests
if [ "$INTEGRATION_TESTS" = true ]; then
    print_header "INTEGRATION TESTS"
    
    # Check if integration test directory exists
    if [ -d "tests/integration" ]; then
        run_tests "Watch Integration Tests" \
            "go test -v ./tests/integration/... -run 'Watch'" || \
            FAILED_TESTS+=("Watch Integration Tests")
    else
        echo -e "${YELLOW}Integration test directory not found, creating it...${NC}"
        mkdir -p tests/integration
    fi
fi

# Run E2E tests
if [ "$E2E_TESTS" = true ]; then
    print_header "END-TO-END TESTS"
    
    # Check if E2E test directory exists
    if [ -d "tests/e2e" ]; then
        run_tests "Watch E2E Tests" \
            "go test -v ./tests/e2e/... -run 'Watch' -timeout 5m" || \
            FAILED_TESTS+=("Watch E2E Tests")
    else
        echo -e "${YELLOW}E2E test directory not found, creating it...${NC}"
        mkdir -p tests/e2e
    fi
fi

# Run performance tests
if [ "$PERFORMANCE_TESTS" = true ]; then
    print_header "PERFORMANCE TESTS"
    
    # Check if performance test directory exists
    if [ -d "tests/performance" ]; then
        run_tests "Watch Performance Tests" \
            "go test -v ./tests/performance/... -run 'Watch' -timeout 10m" || \
            FAILED_TESTS+=("Watch Performance Tests")
    else
        echo -e "${YELLOW}Performance test directory not found, creating it...${NC}"
        mkdir -p tests/performance
    fi
fi

# Run benchmarks
if [ "$BENCHMARKS" = true ]; then
    print_header "BENCHMARKS"
    
    # Watcher benchmarks
    run_tests "Watcher Benchmarks" \
        "go test -bench=. -benchmem ./internal/watcher/... -run=^$" || \
        FAILED_TESTS+=("Watcher Benchmarks")
    
    # Performance benchmarks
    if [ -d "tests/performance" ]; then
        run_tests "Performance Benchmarks" \
            "go test -bench=. -benchmem ./tests/performance/... -run=^$" || \
            FAILED_TESTS+=("Performance Benchmarks")
    fi
fi

# Test coverage report
print_header "TEST COVERAGE"

echo "Generating coverage report..."
go test -coverprofile=coverage.out ./internal/watcher/... 2>/dev/null
go tool cover -html=coverage.out -o coverage.html

# Extract coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "Total coverage: ${GREEN}${COVERAGE}${NC}"

# Clean up
rm -f coverage.out

# Summary
print_header "TEST SUMMARY"

if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo -e "Coverage: ${GREEN}${COVERAGE}${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}- $test${NC}"
    done
    echo -e "\nCoverage: ${YELLOW}${COVERAGE}${NC}"
    exit 1
fi