#!/bin/bash

# WGO Performance Testing Script
# Comprehensive performance testing and benchmarking for WGO

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PERF_TEST_DIR="$PROJECT_ROOT/test/performance"
RESULTS_DIR="$PROJECT_ROOT/performance-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Usage function
usage() {
    cat << EOF
WGO Performance Testing Script

Usage: $0 [OPTIONS] [TEST_TYPE]

TEST_TYPE:
    all             Run all performance tests (default)
    benchmarks      Run performance benchmarks only
    stress          Run stress tests only
    memory          Run memory analysis tests only
    concurrent      Run concurrent operation tests only
    large-dataset   Run large dataset tests only
    quick           Run quick benchmarks (development)
    
OPTIONS:
    -h, --help      Show this help message
    -v, --verbose   Enable verbose output
    -o, --output    Output directory (default: performance-results)
    -t, --timeout   Test timeout (default: 30m)
    -c, --count     Benchmark count (default: 3)
    --ci            CI mode (reduced test set for faster execution)
    --profile       Enable profiling (CPU and memory)
    --system-check  Check system requirements before running

EXAMPLES:
    $0                          # Run all tests
    $0 benchmarks               # Run only benchmarks
    $0 --ci quick               # Quick tests for CI
    $0 --profile stress         # Stress tests with profiling
    $0 --system-check all       # Check system then run all tests

EOF
}

# System requirements check
check_system_requirements() {
    log_info "Checking system requirements..."
    
    # Check CPU cores
    cpu_cores=$(nproc)
    log_info "CPU cores: $cpu_cores"
    
    # Check memory
    total_mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    total_mem_gb=$((total_mem_kb / 1024 / 1024))
    available_mem_kb=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
    available_mem_gb=$((available_mem_kb / 1024 / 1024))
    
    log_info "Total memory: ${total_mem_gb}GB"
    log_info "Available memory: ${available_mem_gb}GB"
    
    # Check disk space
    available_space=$(df -BG "$PROJECT_ROOT" | tail -1 | awk '{print $4}' | sed 's/G//')
    log_info "Available disk space: ${available_space}GB"
    
    # Check Go version
    go_version=$(go version | awk '{print $3}')
    log_info "Go version: $go_version"
    
    # Validate requirements
    warnings=0
    
    if [ "$cpu_cores" -lt 2 ]; then
        log_warning "Less than 2 CPU cores detected. Concurrent tests may not be reliable."
        ((warnings++))
    fi
    
    if [ "$available_mem_gb" -lt 4 ]; then
        log_warning "Less than 4GB available memory. Large dataset tests may fail."
        ((warnings++))
    fi
    
    if [ "$available_space" -lt 10 ]; then
        log_warning "Less than 10GB available disk space. Tests may fail due to space constraints."
        ((warnings++))
    fi
    
    if [ "$warnings" -eq 0 ]; then
        log_success "System requirements check passed"
    else
        log_warning "System requirements check completed with $warnings warnings"
    fi
    
    return 0
}

# Setup test environment
setup_environment() {
    log_info "Setting up performance test environment..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    mkdir -p "$RESULTS_DIR/profiles"
    mkdir -p "$RESULTS_DIR/logs"
    
    # Navigate to performance test directory
    cd "$PERF_TEST_DIR"
    
    # Build test binaries if needed
    if [ "$ENABLE_PROFILING" = "true" ]; then
        log_info "Building with profiling enabled..."
        go test -c -o performance.test .
    fi
    
    log_success "Environment setup complete"
}

# Run benchmarks
run_benchmarks() {
    log_info "Running performance benchmarks..."
    
    local bench_time=${BENCH_TIME:-10s}
    local bench_count=${BENCH_COUNT:-3}
    local timeout=${TEST_TIMEOUT:-30m}
    
    if [ "$CI_MODE" = "true" ]; then
        bench_time="3s"
        bench_count="1"
        timeout="10m"
    fi
    
    # Basic operation benchmarks
    log_info "Running basic operation benchmarks..."
    go test -run=^$ -bench=BenchmarkMegaFileProcessing -benchtime="$bench_time" -count="$bench_count" \
        -benchmem -timeout="$timeout" . 2>&1 | tee "$RESULTS_DIR/basic_benchmarks_$TIMESTAMP.txt"
    
    # Concurrent operation benchmarks
    log_info "Running concurrent operation benchmarks..."
    go test -run=^$ -bench=BenchmarkConcurrentOperations -benchtime="$bench_time" -count="$bench_count" \
        -benchmem -timeout="$timeout" . 2>&1 | tee "$RESULTS_DIR/concurrent_benchmarks_$TIMESTAMP.txt"
    
    # Memory intensive benchmarks
    log_info "Running memory intensive benchmarks..."
    go test -run=^$ -bench=BenchmarkMemoryIntensiveOperations -benchtime="$bench_time" -count="$bench_count" \
        -benchmem -timeout="$timeout" . 2>&1 | tee "$RESULTS_DIR/memory_benchmarks_$TIMESTAMP.txt"
    
    # End-to-end workflow benchmarks
    log_info "Running end-to-end workflow benchmarks..."
    go test -run=^$ -bench=BenchmarkEndToEndWorkflow -benchtime="$bench_time" -count="$bench_count" \
        -benchmem -timeout="$timeout" . 2>&1 | tee "$RESULTS_DIR/e2e_benchmarks_$TIMESTAMP.txt"
    
    if [ "$CI_MODE" != "true" ]; then
        # Watch mode benchmarks (skip in CI due to time)
        log_info "Running watch mode benchmarks..."
        go test -run=^$ -bench=BenchmarkWatchModePerformance -benchtime="$bench_time" -count="$bench_count" \
            -benchmem -timeout="$timeout" . 2>&1 | tee "$RESULTS_DIR/watch_benchmarks_$TIMESTAMP.txt"
    fi
    
    log_success "Benchmarks completed"
}

# Run stress tests
run_stress_tests() {
    log_info "Running stress tests..."
    
    local timeout=${TEST_TIMEOUT:-30m}
    
    if [ "$CI_MODE" = "true" ]; then
        timeout="10m"
    fi
    
    # Performance requirements tests
    log_info "Running performance requirements tests..."
    go test -run=TestPerformanceRequirements -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/requirements_test_$TIMESTAMP.txt"
    
    if [ "$CI_MODE" != "true" ]; then
        # Large dataset scaling tests (skip in CI)
        log_info "Running large dataset scaling tests..."
        go test -run=TestLargeDatasetScaling -timeout="$timeout" -v . 2>&1 | \
            tee "$RESULTS_DIR/scaling_test_$TIMESTAMP.txt"
        
        # System limits tests (skip in CI)
        log_info "Running system limits tests..."
        go test -run=TestSystemLimits -timeout="$timeout" -v . 2>&1 | \
            tee "$RESULTS_DIR/limits_test_$TIMESTAMP.txt"
    fi
    
    log_success "Stress tests completed"
}

# Run memory analysis tests
run_memory_tests() {
    log_info "Running memory analysis tests..."
    
    local timeout=${TEST_TIMEOUT:-30m}
    
    # Memory usage patterns
    log_info "Running memory usage pattern analysis..."
    go test -run=TestMemoryUsagePatterns -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/memory_patterns_$TIMESTAMP.txt"
    
    # Memory leak detection
    log_info "Running memory leak detection..."
    go test -run=TestMemoryLeakDetection -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/memory_leaks_$TIMESTAMP.txt"
    
    if [ "$CI_MODE" != "true" ]; then
        # Watch mode memory profiling (skip in CI)
        log_info "Running watch mode memory profile..."
        go test -run=TestMemoryProfileDuringWatchMode -timeout="$timeout" -v . 2>&1 | \
            tee "$RESULTS_DIR/watch_memory_$TIMESTAMP.txt"
    fi
    
    log_success "Memory tests completed"
}

# Run concurrent operation tests
run_concurrent_tests() {
    log_info "Running concurrent operation tests..."
    
    local timeout=${TEST_TIMEOUT:-30m}
    
    # Concurrent scanning
    log_info "Running concurrent scanning tests..."
    go test -run=TestConcurrentScanning -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/concurrent_scan_$TIMESTAMP.txt"
    
    # Concurrent diff operations
    log_info "Running concurrent diff tests..."
    go test -run=TestConcurrentDiffOperations -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/concurrent_diff_$TIMESTAMP.txt"
    
    if [ "$CI_MODE" != "true" ]; then
        # Concurrent watch mode (skip in CI)
        log_info "Running concurrent watch mode tests..."
        go test -run=TestConcurrentWatchModeOperations -timeout="$timeout" -v . 2>&1 | \
            tee "$RESULTS_DIR/concurrent_watch_$TIMESTAMP.txt"
        
        # Resource contention tests (skip in CI)
        log_info "Running resource contention tests..."
        go test -run=TestResourceContention -timeout="$timeout" -v . 2>&1 | \
            tee "$RESULTS_DIR/resource_contention_$TIMESTAMP.txt"
    fi
    
    log_success "Concurrent tests completed"
}

# Run large dataset tests
run_large_dataset_tests() {
    log_info "Running large dataset tests..."
    
    local timeout=${TEST_TIMEOUT:-30m}
    
    if [ "$CI_MODE" = "true" ]; then
        log_warning "Skipping large dataset tests in CI mode"
        return 0
    fi
    
    # Mega file parsing
    log_info "Running mega file parsing tests..."
    go test -run=TestMegaFileParsing -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/mega_files_$TIMESTAMP.txt"
    
    # Multi-file processing
    log_info "Running multi-file processing tests..."
    go test -run=TestMultiFileProcessing -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/multi_files_$TIMESTAMP.txt"
    
    # Differ performance at scale
    log_info "Running differ performance at scale..."
    go test -run=TestDifferPerformanceAtScale -timeout="$timeout" -v . 2>&1 | \
        tee "$RESULTS_DIR/differ_scale_$TIMESTAMP.txt"
    
    log_success "Large dataset tests completed"
}

# Run quick benchmarks
run_quick_tests() {
    log_info "Running quick benchmarks..."
    
    go test -run=^$ -bench=. -benchtime=1s -count=1 -timeout=5m . 2>&1 | \
        tee "$RESULTS_DIR/quick_bench_$TIMESTAMP.txt"
    
    log_success "Quick benchmarks completed"
}

# Generate profiles
generate_profiles() {
    if [ "$ENABLE_PROFILING" != "true" ]; then
        return 0
    fi
    
    log_info "Generating performance profiles..."
    
    # CPU profiling
    log_info "Generating CPU profiles..."
    go test -run=TestCPUProfileDuringIntensiveOperations -timeout=15m -v . 2>&1 | \
        tee "$RESULTS_DIR/cpu_profile_$TIMESTAMP.txt"
    
    # Memory profiling
    log_info "Generating memory profiles..."
    go test -run=TestHeapProfileDuringMemoryIntensiveOps -timeout=15m -v . 2>&1 | \
        tee "$RESULTS_DIR/heap_profile_$TIMESTAMP.txt"
    
    # Move profiles to results directory
    find . -name "*.prof" -exec mv {} "$RESULTS_DIR/profiles/" \; 2>/dev/null || true
    
    log_success "Profiles generated"
}

# Generate comprehensive report
generate_report() {
    log_info "Generating comprehensive performance report..."
    
    local report_file="$RESULTS_DIR/performance_report_$TIMESTAMP.md"
    
    cat > "$report_file" << EOF
# WGO Performance Test Report

**Generated:** $(date)
**Timestamp:** $TIMESTAMP

## Test Environment

\`\`\`
OS: $(uname -s) $(uname -r)
CPU: $(nproc) cores
Memory: $(free -h | grep Mem | awk '{print $2}')
Available Memory: $(free -h | grep Mem | awk '{print $7}')
Disk Space: $(df -h "$PROJECT_ROOT" | tail -1 | awk '{print $4}')
Go Version: $(go version)
\`\`\`

## Test Configuration

\`\`\`
Test Timeout: ${TEST_TIMEOUT:-30m}
Benchmark Time: ${BENCH_TIME:-10s}
Benchmark Count: ${BENCH_COUNT:-3}
CI Mode: ${CI_MODE:-false}
Profiling Enabled: ${ENABLE_PROFILING:-false}
\`\`\`

EOF

    # Include benchmark results if they exist
    for result_file in "$RESULTS_DIR"/*_"$TIMESTAMP".txt; do
        if [ -f "$result_file" ]; then
            local test_name=$(basename "$result_file" | sed "s/_$TIMESTAMP.txt//")
            echo "## $test_name Results" >> "$report_file"
            echo '```' >> "$report_file"
            cat "$result_file" >> "$report_file"
            echo '```' >> "$report_file"
            echo "" >> "$report_file"
        fi
    done
    
    cat >> "$report_file" << EOF

## Summary

Performance testing completed successfully.

- **Results Directory:** $RESULTS_DIR
- **Timestamp:** $TIMESTAMP
- **Log Files:** Check individual test files for detailed results
- **Profiles:** Available in $RESULTS_DIR/profiles/ (if profiling was enabled)

## Analysis Commands

To analyze profiles:
\`\`\`bash
# CPU profile analysis
go tool pprof $RESULTS_DIR/profiles/cpu-profile.prof

# Memory profile analysis
go tool pprof $RESULTS_DIR/profiles/heap-profile.prof
\`\`\`

EOF
    
    log_success "Performance report generated: $report_file"
}

# Cleanup function
cleanup() {
    log_info "Performing cleanup..."
    
    # Remove temporary files
    find "$PERF_TEST_DIR" -name "*.test" -delete 2>/dev/null || true
    find "$PERF_TEST_DIR" -name "*.prof" -delete 2>/dev/null || true
    
    log_info "Cleanup completed"
}

# Main execution function
main() {
    local test_type="all"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -v|--verbose)
                set -x
                VERBOSE=true
                shift
                ;;
            -o|--output)
                RESULTS_DIR="$2"
                shift 2
                ;;
            -t|--timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            -c|--count)
                BENCH_COUNT="$2"
                shift 2
                ;;
            --ci)
                CI_MODE=true
                shift
                ;;
            --profile)
                ENABLE_PROFILING=true
                shift
                ;;
            --system-check)
                SYSTEM_CHECK=true
                shift
                ;;
            all|benchmarks|stress|memory|concurrent|large-dataset|quick)
                test_type="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # Set defaults
    CI_MODE=${CI_MODE:-false}
    ENABLE_PROFILING=${ENABLE_PROFILING:-false}
    SYSTEM_CHECK=${SYSTEM_CHECK:-false}
    VERBOSE=${VERBOSE:-false}
    
    log_info "Starting WGO performance testing..."
    log_info "Test type: $test_type"
    log_info "Results directory: $RESULTS_DIR"
    
    # System check if requested
    if [ "$SYSTEM_CHECK" = "true" ]; then
        check_system_requirements
    fi
    
    # Setup environment
    setup_environment
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    # Run tests based on type
    case $test_type in
        all)
            run_benchmarks
            run_stress_tests
            run_memory_tests
            run_concurrent_tests
            if [ "$CI_MODE" != "true" ]; then
                run_large_dataset_tests
            fi
            generate_profiles
            ;;
        benchmarks)
            run_benchmarks
            ;;
        stress)
            run_stress_tests
            ;;
        memory)
            run_memory_tests
            generate_profiles
            ;;
        concurrent)
            run_concurrent_tests
            ;;
        large-dataset)
            run_large_dataset_tests
            ;;
        quick)
            run_quick_tests
            ;;
        *)
            log_error "Unknown test type: $test_type"
            exit 1
            ;;
    esac
    
    # Generate comprehensive report
    generate_report
    
    log_success "Performance testing completed successfully!"
    log_info "Results available in: $RESULTS_DIR"
}

# Run main function
main "$@"