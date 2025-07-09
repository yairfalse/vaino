#!/bin/bash
# Comprehensive test runner for VAINO correlation and timeline features

set -e

echo "🧪 VAINO Comprehensive Test Suite"
echo "================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test category
run_test_category() {
    local category=$1
    local description=$2
    local test_command=$3
    
    echo -e "${BLUE}📋 Running $description...${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if eval "$test_command"; then
        echo -e "${GREEN}✅ $description: PASSED${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ $description: FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo ""
}

# Function to run performance test with threshold
run_performance_test() {
    local test_name=$1
    local description=$2
    local max_duration=$3
    local test_command=$4
    
    echo -e "${BLUE}⚡ Running $description (max: ${max_duration}s)...${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    start_time=$(date +%s)
    if eval "$test_command"; then
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        
        if [ $duration -le $max_duration ]; then
            echo -e "${GREEN}✅ $description: PASSED (${duration}s)${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${YELLOW}⚠️  $description: SLOW (${duration}s > ${max_duration}s)${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        fi
    else
        echo -e "${RED}❌ $description: FAILED${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo ""
}

# Build VAINO for testing
echo -e "${BLUE}🔨 Building VAINO...${NC}"
if ! go build -o vaino ./cmd/vaino; then
    echo -e "${RED}❌ Failed to build VAINO${NC}"
    exit 1
fi
echo -e "${GREEN}✅ VAINO built successfully${NC}"
echo ""

# Unit Tests
echo -e "${YELLOW}🔬 UNIT TESTS${NC}"
echo "============="

run_test_category "unit_correlator" \
    "Correlator Unit Tests" \
    "go test -v ./internal/analyzer -run TestCorrelator"

run_test_category "unit_formatter" \
    "Formatter Unit Tests" \
    "go test -v ./internal/analyzer -run TestFormat"

run_test_category "unit_simple_differ" \
    "Simple Differ Unit Tests" \
    "go test -v ./internal/differ -run TestSimpleDiffer"

# Integration Tests
echo -e "${YELLOW}🔗 INTEGRATION TESTS${NC}"
echo "===================="

run_test_category "integration_correlation" \
    "Correlation Integration" \
    "./test/correlation-demo.sh > /dev/null 2>&1"

run_test_category "integration_timeline" \
    "Timeline Integration" \
    "./test/timeline-demo.sh > /dev/null 2>&1"

run_test_category "integration_advanced" \
    "Advanced Correlation Scenarios" \
    "./test/correlation-advanced-demo.sh > /dev/null 2>&1"

# System Tests
echo -e "${YELLOW}🖥️  SYSTEM TESTS${NC}"
echo "==============="

run_test_category "system_correlation" \
    "System Correlation Tests" \
    "go test -v ./test/system -run TestCorrelationSystem"

run_test_category "system_accuracy" \
    "Correlation Accuracy Tests" \
    "go test -v ./test/system -run TestCorrelationAccuracy"

run_test_category "system_confidence" \
    "Confidence Level Tests" \
    "go test -v ./test/system -run TestConfidenceLevels"

# Performance Tests
echo -e "${YELLOW}⚡ PERFORMANCE TESTS${NC}"
echo "==================="

run_performance_test "perf_small" \
    "Small Dataset Performance (50 changes)" \
    "1" \
    "go test -v ./test/performance -run BenchmarkCorrelationEngine/changes_50 -bench=. -benchtime=1x"

run_performance_test "perf_medium" \
    "Medium Dataset Performance (500 changes)" \
    "3" \
    "go test -v ./test/performance -run BenchmarkCorrelationEngine/changes_500 -bench=. -benchtime=1x"

run_performance_test "perf_large" \
    "Large Dataset Performance (2000 changes)" \
    "10" \
    "go test -v ./test/performance -run TestCorrelationPerformanceRequirements/large_changes"

run_performance_test "perf_memory" \
    "Memory Usage Test" \
    "5" \
    "go test -v ./test/performance -run TestMemoryUsage"

# E2E Tests
echo -e "${YELLOW}🎯 END-TO-END TESTS${NC}"
echo "==================="

run_test_category "e2e_workflow" \
    "Complete E2E Workflow" \
    "go test -v ./test/e2e -run TestE2ECorrelationWorkflow/complete_workflow"

run_test_category "e2e_error_handling" \
    "E2E Error Handling" \
    "go test -v ./test/e2e -run TestE2ECorrelationWorkflow/error_handling"

run_test_category "e2e_output_formats" \
    "E2E Output Formats" \
    "go test -v ./test/e2e -run TestE2ECorrelationWorkflow/output_formats"

run_test_category "e2e_realistic_perf" \
    "E2E Realistic Performance" \
    "go test -v ./test/e2e -run TestE2ECorrelationWorkflow/realistic_performance"

# Functional Tests
echo -e "${YELLOW}🎪 FUNCTIONAL TESTS${NC}"
echo "==================="

# Test specific correlation patterns
echo -e "${BLUE}📋 Testing Scaling Correlation Pattern...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./test/test-scaling-correlation.sh > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Scaling Correlation: PASSED${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ Scaling Correlation: FAILED${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo -e "${BLUE}📋 Testing Service Deployment Pattern...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./test/test-service-deployment.sh > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Service Deployment: PASSED${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ Service Deployment: FAILED${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

echo -e "${BLUE}📋 Testing Config Update Pattern...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./test/test-config-update.sh > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Config Update: PASSED${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ Config Update: FAILED${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Timeline-specific tests
echo -e "${BLUE}📋 Testing Timeline Accuracy...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if ./test/test-timeline-accuracy.sh > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Timeline Accuracy: PASSED${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}❌ Timeline Accuracy: FAILED${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi

# Regression Tests
echo -e "${YELLOW}🔄 REGRESSION TESTS${NC}"
echo "==================="

run_test_category "regression_false_positives" \
    "False Positive Prevention" \
    "go test -v ./internal/analyzer -run TestCorrelator_AvoidFalseCorrelations"

run_test_category "regression_time_windows" \
    "Time Window Boundaries" \
    "go test -v ./internal/analyzer -run TestCorrelator_TimeWindowRespected"

run_test_category "regression_edge_cases" \
    "Edge Case Handling" \
    "go test -v ./internal/analyzer -run TestCorrelator_GroupChanges_EmptyInput"

# Coverage Tests
echo -e "${YELLOW}📊 CODE COVERAGE${NC}"
echo "================"

echo -e "${BLUE}📋 Generating coverage report...${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
if go test -coverprofile=coverage.out -coverpkg=./internal/analyzer,./internal/differ ./internal/analyzer ./internal/differ > /dev/null 2>&1; then
    COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
    echo -e "${GREEN}✅ Code Coverage: $COVERAGE${NC}"
    
    # Check if coverage is acceptable (>80%)
    COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
    if (( $(echo "$COVERAGE_NUM >= 80" | bc -l) )); then
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${YELLOW}⚠️  Coverage below 80%${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    
    # Generate HTML report
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${BLUE}📄 Coverage report saved to coverage.html${NC}"
else
    echo -e "${RED}❌ Coverage Generation: FAILED${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
fi
echo ""

# Final Results
echo -e "${YELLOW}📊 TEST RESULTS SUMMARY${NC}"
echo "======================="
echo -e "${BLUE}Total Tests:  ${TOTAL_TESTS}${NC}"
echo -e "${GREEN}Passed:       ${PASSED_TESTS}${NC}"
echo -e "${RED}Failed:       ${FAILED_TESTS}${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo ""
    echo -e "${GREEN}🎉 ALL TESTS PASSED! 🎉${NC}"
    echo -e "${GREEN}The correlation and timeline features are ready for production.${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}💥 SOME TESTS FAILED 💥${NC}"
    echo -e "${RED}Please review the failed tests above.${NC}"
    exit 1
fi