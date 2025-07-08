# Performance Testing Makefile for WGO
# Provides comprehensive performance testing and benchmarking capabilities

.PHONY: all clean setup benchmarks stress-test memory-test concurrent-test large-dataset-test performance-report

# Configuration
GO := go
TEST_TIMEOUT := 30m
BENCH_TIME := 10s
BENCH_COUNT := 3
OUTPUT_DIR := results
PROFILE_DIR := profiles

# Default target
all: setup benchmarks performance-report

# Setup test environment
setup:
	@echo "Setting up performance testing environment..."
	@mkdir -p $(OUTPUT_DIR) $(PROFILE_DIR)
	@echo "Performance test environment ready"

# Clean up test artifacts
clean:
	@echo "Cleaning up performance test artifacts..."
	@rm -rf $(OUTPUT_DIR) $(PROFILE_DIR)
	@rm -f *.prof *.out *.test
	@echo "Cleanup complete"

# Run all performance benchmarks
benchmarks: setup
	@echo "Running comprehensive performance benchmarks..."
	@echo "Starting: $(shell date)"
	
	# Basic operation benchmarks
	@echo "\n=== Basic Operation Benchmarks ==="
	$(GO) test -run=^$$ -bench=BenchmarkMegaFileProcessing -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) \
		-benchmem -timeout=$(TEST_TIMEOUT) . | tee $(OUTPUT_DIR)/basic_benchmarks.txt
	
	# Concurrent operation benchmarks  
	@echo "\n=== Concurrent Operation Benchmarks ==="
	$(GO) test -run=^$$ -bench=BenchmarkConcurrentOperations -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) \
		-benchmem -timeout=$(TEST_TIMEOUT) . | tee $(OUTPUT_DIR)/concurrent_benchmarks.txt
	
	# Memory intensive benchmarks
	@echo "\n=== Memory Intensive Benchmarks ==="
	$(GO) test -run=^$$ -bench=BenchmarkMemoryIntensiveOperations -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) \
		-benchmem -timeout=$(TEST_TIMEOUT) . | tee $(OUTPUT_DIR)/memory_benchmarks.txt
	
	# End-to-end workflow benchmarks
	@echo "\n=== End-to-End Workflow Benchmarks ==="
	$(GO) test -run=^$$ -bench=BenchmarkEndToEndWorkflow -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) \
		-benchmem -timeout=$(TEST_TIMEOUT) . | tee $(OUTPUT_DIR)/e2e_benchmarks.txt
	
	# Watch mode benchmarks
	@echo "\n=== Watch Mode Benchmarks ==="
	$(GO) test -run=^$$ -bench=BenchmarkWatchModePerformance -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT) \
		-benchmem -timeout=$(TEST_TIMEOUT) . | tee $(OUTPUT_DIR)/watch_benchmarks.txt
	
	@echo "Benchmarks completed: $(shell date)"

# Run stress tests
stress-test: setup
	@echo "Running stress tests..."
	@echo "Starting: $(shell date)"
	
	# Performance requirements tests
	@echo "\n=== Performance Requirements Tests ==="
	$(GO) test -run=TestPerformanceRequirements -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/requirements_test.txt
	
	# Large dataset scaling tests
	@echo "\n=== Large Dataset Scaling Tests ==="
	$(GO) test -run=TestLargeDatasetScaling -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/scaling_test.txt
	
	# System limits tests
	@echo "\n=== System Limits Tests ==="
	$(GO) test -run=TestSystemLimits -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/limits_test.txt
	
	@echo "Stress tests completed: $(shell date)"

# Run memory analysis tests
memory-test: setup
	@echo "Running memory analysis tests..."
	@echo "Starting: $(shell date)"
	
	# Memory usage patterns
	@echo "\n=== Memory Usage Pattern Analysis ==="
	$(GO) test -run=TestMemoryUsagePatterns -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/memory_patterns.txt
	
	# Memory leak detection
	@echo "\n=== Memory Leak Detection ==="
	$(GO) test -run=TestMemoryLeakDetection -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/memory_leaks.txt
	
	# Watch mode memory profiling
	@echo "\n=== Watch Mode Memory Profile ==="
	$(GO) test -run=TestMemoryProfileDuringWatchMode -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/watch_memory.txt
	
	@echo "Memory tests completed: $(shell date)"

# Run concurrent operation tests
concurrent-test: setup
	@echo "Running concurrent operation tests..."
	@echo "Starting: $(shell date)"
	
	# Concurrent scanning
	@echo "\n=== Concurrent Scanning Tests ==="
	$(GO) test -run=TestConcurrentScanning -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/concurrent_scan.txt
	
	# Concurrent diff operations
	@echo "\n=== Concurrent Diff Tests ==="
	$(GO) test -run=TestConcurrentDiffOperations -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/concurrent_diff.txt
	
	# Concurrent watch mode
	@echo "\n=== Concurrent Watch Mode Tests ==="
	$(GO) test -run=TestConcurrentWatchModeOperations -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/concurrent_watch.txt
	
	# Resource contention tests
	@echo "\n=== Resource Contention Tests ==="
	$(GO) test -run=TestResourceContention -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/resource_contention.txt
	
	@echo "Concurrent tests completed: $(shell date)"

# Run large dataset tests
large-dataset-test: setup
	@echo "Running large dataset tests..."
	@echo "Starting: $(shell date)"
	
	# Mega file parsing
	@echo "\n=== Mega File Parsing Tests ==="
	$(GO) test -run=TestMegaFileParsing -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/mega_files.txt
	
	# Multi-file processing
	@echo "\n=== Multi-File Processing Tests ==="
	$(GO) test -run=TestMultiFileProcessing -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/multi_files.txt
	
	# Differ performance at scale
	@echo "\n=== Differ Performance at Scale ==="
	$(GO) test -run=TestDifferPerformanceAtScale -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/differ_scale.txt
	
	@echo "Large dataset tests completed: $(shell date)"

# Generate CPU profiles
cpu-profile: setup
	@echo "Generating CPU profiles..."
	$(GO) test -run=TestCPUProfileDuringIntensiveOperations -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/cpu_profile.txt
	@mv *.prof $(PROFILE_DIR)/ 2>/dev/null || true
	@echo "CPU profiles generated in $(PROFILE_DIR)/"

# Generate memory profiles  
memory-profile: setup
	@echo "Generating memory profiles..."
	$(GO) test -run=TestHeapProfileDuringMemoryIntensiveOps -timeout=$(TEST_TIMEOUT) -v . | tee $(OUTPUT_DIR)/heap_profile.txt
	@mv *.prof $(PROFILE_DIR)/ 2>/dev/null || true
	@echo "Memory profiles generated in $(PROFILE_DIR)/"

# Generate comprehensive performance report
performance-report: setup
	@echo "Generating comprehensive performance report..."
	@echo "# WGO Performance Test Report" > $(OUTPUT_DIR)/performance_report.md
	@echo "Generated: $(shell date)" >> $(OUTPUT_DIR)/performance_report.md
	@echo "" >> $(OUTPUT_DIR)/performance_report.md
	
	@echo "## Test Environment" >> $(OUTPUT_DIR)/performance_report.md
	@echo "\`\`\`" >> $(OUTPUT_DIR)/performance_report.md
	@echo "OS: $(shell uname -s) $(shell uname -r)" >> $(OUTPUT_DIR)/performance_report.md
	@echo "CPU: $(shell nproc) cores" >> $(OUTPUT_DIR)/performance_report.md
	@echo "Memory: $(shell free -h | grep Mem | awk '{print $$2}')" >> $(OUTPUT_DIR)/performance_report.md
	@echo "Go Version: $(shell $(GO) version)" >> $(OUTPUT_DIR)/performance_report.md
	@echo "\`\`\`" >> $(OUTPUT_DIR)/performance_report.md
	@echo "" >> $(OUTPUT_DIR)/performance_report.md
	
	# Include benchmark results if they exist
	@if [ -f $(OUTPUT_DIR)/basic_benchmarks.txt ]; then \
		echo "## Basic Operation Benchmarks" >> $(OUTPUT_DIR)/performance_report.md; \
		echo "\`\`\`" >> $(OUTPUT_DIR)/performance_report.md; \
		cat $(OUTPUT_DIR)/basic_benchmarks.txt >> $(OUTPUT_DIR)/performance_report.md; \
		echo "\`\`\`" >> $(OUTPUT_DIR)/performance_report.md; \
		echo "" >> $(OUTPUT_DIR)/performance_report.md; \
	fi
	
	@if [ -f $(OUTPUT_DIR)/concurrent_benchmarks.txt ]; then \
		echo "## Concurrent Operation Benchmarks" >> $(OUTPUT_DIR)/performance_report.md; \
		echo "\`\`\`" >> $(OUTPUT_DIR)/performance_report.md; \
		cat $(OUTPUT_DIR)/concurrent_benchmarks.txt >> $(OUTPUT_DIR)/performance_report.md; \
		echo "\`\`\`" >> $(OUTPUT_DIR)/performance_report.md; \
		echo "" >> $(OUTPUT_DIR)/performance_report.md; \
	fi
	
	@echo "## Summary" >> $(OUTPUT_DIR)/performance_report.md
	@echo "Performance testing completed successfully." >> $(OUTPUT_DIR)/performance_report.md
	@echo "Check individual test files for detailed results." >> $(OUTPUT_DIR)/performance_report.md
	
	@echo "Performance report generated: $(OUTPUT_DIR)/performance_report.md"

# Quick benchmark (faster for development)
quick-bench: setup
	@echo "Running quick benchmarks (shorter duration)..."
	$(GO) test -run=^$$ -bench=. -benchtime=1s -count=1 -timeout=5m . | tee $(OUTPUT_DIR)/quick_bench.txt

# Profile analysis helpers
analyze-cpu:
	@if [ -f $(PROFILE_DIR)/cpu-profile.prof ]; then \
		echo "Analyzing CPU profile..."; \
		$(GO) tool pprof $(PROFILE_DIR)/cpu-profile.prof; \
	else \
		echo "No CPU profile found. Run 'make cpu-profile' first."; \
	fi

analyze-memory:
	@if [ -f $(PROFILE_DIR)/heap-profile.prof ]; then \
		echo "Analyzing memory profile..."; \
		$(GO) tool pprof $(PROFILE_DIR)/heap-profile.prof; \
	else \
		echo "No memory profile found. Run 'make memory-profile' first."; \
	fi

# Continuous performance monitoring
monitor:
	@echo "Starting continuous performance monitoring..."
	@while true; do \
		echo "Running performance check: $(shell date)"; \
		$(GO) test -run=TestPerformanceRequirements -timeout=5m -v . > $(OUTPUT_DIR)/monitor_$(shell date +%Y%m%d_%H%M%S).txt 2>&1; \
		sleep 3600; \
	done

# Performance regression test (compare with baseline)
regression-test: setup
	@echo "Running performance regression tests..."
	@if [ ! -f $(OUTPUT_DIR)/baseline_benchmarks.txt ]; then \
		echo "No baseline found. Creating baseline..."; \
		$(GO) test -run=^$$ -bench=. -benchtime=5s -count=1 . > $(OUTPUT_DIR)/baseline_benchmarks.txt; \
		echo "Baseline created. Run again to compare."; \
	else \
		echo "Running current benchmarks..."; \
		$(GO) test -run=^$$ -bench=. -benchtime=5s -count=1 . > $(OUTPUT_DIR)/current_benchmarks.txt; \
		echo "Comparing with baseline..."; \
		echo "# Performance Regression Analysis" > $(OUTPUT_DIR)/regression_analysis.md; \
		echo "Baseline: $(OUTPUT_DIR)/baseline_benchmarks.txt" >> $(OUTPUT_DIR)/regression_analysis.md; \
		echo "Current:  $(OUTPUT_DIR)/current_benchmarks.txt" >> $(OUTPUT_DIR)/regression_analysis.md; \
		echo "Analysis: Manual comparison required" >> $(OUTPUT_DIR)/regression_analysis.md; \
	fi

# Validate system requirements
check-system:
	@echo "Checking system requirements for performance testing..."
	@echo "CPU cores: $(shell nproc)"
	@echo "Available memory: $(shell free -h | grep Mem | awk '{print $$7}')"
	@echo "Available disk space: $(shell df -h . | tail -1 | awk '{print $$4}')"
	@echo "Go version: $(shell $(GO) version)"
	@if [ $(shell nproc) -lt 2 ]; then \
		echo "WARNING: Less than 2 CPU cores detected. Some concurrent tests may not be reliable."; \
	fi
	@if [ $(shell free -m | grep Mem | awk '{print $$2}') -lt 4096 ]; then \
		echo "WARNING: Less than 4GB RAM detected. Large dataset tests may fail."; \
	fi

# Help target
help:
	@echo "WGO Performance Testing Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Run setup, benchmarks, and generate report"
	@echo "  setup            - Create test environment"
	@echo "  clean            - Clean up test artifacts"
	@echo "  benchmarks       - Run all performance benchmarks"
	@echo "  stress-test      - Run stress tests"
	@echo "  memory-test      - Run memory analysis tests"
	@echo "  concurrent-test  - Run concurrent operation tests"
	@echo "  large-dataset-test - Run large dataset tests"
	@echo "  cpu-profile      - Generate CPU profiles"
	@echo "  memory-profile   - Generate memory profiles"
	@echo "  performance-report - Generate comprehensive report"
	@echo "  quick-bench      - Run quick benchmarks (development)"
	@echo "  analyze-cpu      - Analyze CPU profile with pprof"
	@echo "  analyze-memory   - Analyze memory profile with pprof"
	@echo "  monitor          - Continuous performance monitoring"
	@echo "  regression-test  - Performance regression testing"
	@echo "  check-system     - Validate system requirements"
	@echo "  help             - Show this help message"
	@echo ""
	@echo "Configuration:"
	@echo "  TEST_TIMEOUT=$(TEST_TIMEOUT)"
	@echo "  BENCH_TIME=$(BENCH_TIME)" 
	@echo "  BENCH_COUNT=$(BENCH_COUNT)"
	@echo "  OUTPUT_DIR=$(OUTPUT_DIR)"
	@echo "  PROFILE_DIR=$(PROFILE_DIR)"