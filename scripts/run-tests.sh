#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
FAILED_TESTS=()

echo -e "${GREEN}WGO Test Runner${NC}"
echo "=================="

# Function to run a test suite
run_test_suite() {
    local name=$1
    local cmd=$2
    
    echo -e "\n${YELLOW}Running $name...${NC}"
    if eval "$cmd"; then
        echo -e "${GREEN}✓ $name passed${NC}"
    else
        echo -e "${RED}✗ $name failed${NC}"
        FAILED_TESTS+=("$name")
    fi
}

# Parse arguments
TEST_TYPE=${1:-all}
VERBOSE=${2:-}

# Build the binary first
echo -e "\n${YELLOW}Building WGO binary...${NC}"
go build -o wgo ./cmd/wgo

# Run different test suites based on argument
case $TEST_TYPE in
    unit)
        run_test_suite "Unit Tests" "go test -v -race ./internal/... ./pkg/... ./cmd/..."
        ;;
    system)
        run_test_suite "System Tests" "go test -v -timeout=10m ./test/system/..."
        ;;
    e2e)
        export PATH=$PATH:$(pwd)
        run_test_suite "E2E Tests" "go test -v -timeout=15m ./test/e2e/..."
        ;;
    integration)
        run_test_suite "Integration Tests" "go test -v ./test/integration/..."
        ;;
    provider)
        run_test_suite "Terraform Tests" "go test -v ./internal/collectors/terraform/..."
        run_test_suite "Kubernetes Tests" "go test -v ./internal/collectors/kubernetes/..."
        run_test_suite "GCP Tests" "go test -v ./internal/collectors/gcp/..."
        ;;
    bench)
        run_test_suite "Benchmarks" "go test -bench=. -benchmem -run=^$ ./internal/..."
        ;;
    coverage)
        echo -e "\n${YELLOW}Generating coverage report...${NC}"
        go test -coverprofile=coverage.out -covermode=atomic ./...
        go tool cover -html=coverage.out -o coverage.html
        go tool cover -func=coverage.out
        echo -e "${GREEN}Coverage report generated: coverage.html${NC}"
        ;;
    quick)
        # Quick tests for development
        run_test_suite "Quick Unit Tests" "go test -short ./internal/... ./pkg/..."
        run_test_suite "CLI Tests" "./wgo --help && ./wgo version"
        ;;
    all)
        # Run all test suites
        run_test_suite "Unit Tests" "go test -v -race ./internal/... ./pkg/... ./cmd/..."
        run_test_suite "System Tests" "go test -v -timeout=10m ./test/system/..."
        run_test_suite "Integration Tests" "go test -v ./test/integration/..."
        
        export PATH=$PATH:$(pwd)
        run_test_suite "E2E Tests" "go test -v -timeout=15m ./test/e2e/..."
        
        run_test_suite "Terraform Provider Tests" "go test -v ./internal/collectors/terraform/..."
        run_test_suite "Kubernetes Provider Tests" "go test -v ./internal/collectors/kubernetes/..."
        run_test_suite "GCP Provider Tests" "go test -v ./internal/collectors/gcp/..."
        ;;
    *)
        echo "Usage: $0 [test-type] [verbose]"
        echo "Test types:"
        echo "  unit       - Run unit tests"
        echo "  system     - Run system integration tests"
        echo "  e2e        - Run end-to-end tests"
        echo "  integration- Run integration tests"
        echo "  provider   - Run provider-specific tests"
        echo "  bench      - Run benchmarks"
        echo "  coverage   - Generate coverage report"
        echo "  quick      - Run quick tests for development"
        echo "  all        - Run all tests (default)"
        exit 1
        ;;
esac

# Summary
echo -e "\n=================="
if [ ${#FAILED_TESTS[@]} -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Failed tests:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}✗ $test${NC}"
    done
    exit 1
fi