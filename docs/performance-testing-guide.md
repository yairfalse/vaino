# WGO Performance Testing Guide

This guide provides comprehensive instructions for running, analyzing, and interpreting WGO performance tests.

## Quick Start

### Running Performance Tests

```bash
# Run all performance tests
make perf-test

# Run quick benchmarks (development)
make perf-quick

# Run specific test categories
make perf-bench          # Benchmarks only
make perf-stress         # Stress tests only
make perf-memory         # Memory analysis
make perf-concurrent     # Concurrent operations
make perf-large-dataset  # Large file tests
```

### Using the Performance Script Directly

```bash
# Comprehensive testing
./scripts/run-performance-tests.sh all

# CI-friendly quick tests
./scripts/run-performance-tests.sh quick --ci

# With profiling enabled
./scripts/run-performance-tests.sh benchmarks --profile

# System check before testing
./scripts/run-performance-tests.sh --system-check all
```

## Test Categories

### 1. Benchmarks (`perf-bench`)
**Purpose**: Measure raw performance of core operations
**Duration**: 5-15 minutes
**Resource Usage**: Moderate
**Best For**: Development, performance regression detection

**Includes**:
- Single file processing benchmarks
- Concurrent operation benchmarks  
- Memory intensive operation benchmarks
- End-to-end workflow benchmarks
- Watch mode performance benchmarks

### 2. Stress Tests (`perf-stress`)
**Purpose**: Test system behavior under extreme load
**Duration**: 10-30 minutes
**Resource Usage**: High
**Best For**: Release validation, system limits identification

**Includes**:
- Performance requirements validation
- Large dataset scaling tests
- System limits testing
- Resource exhaustion scenarios

### 3. Memory Analysis (`perf-memory`)
**Purpose**: Analyze memory usage patterns and detect leaks
**Duration**: 15-45 minutes
**Resource Usage**: High memory
**Best For**: Memory optimization, leak detection

**Includes**:
- Memory usage pattern analysis
- Memory leak detection tests
- Watch mode memory profiling
- Heap profiling with pprof integration

### 4. Concurrent Operations (`perf-concurrent`)
**Purpose**: Test parallel processing capabilities
**Duration**: 10-20 minutes
**Resource Usage**: High CPU
**Best For**: Concurrency optimization, scaling validation

**Includes**:
- Concurrent scanning tests
- Concurrent diff operations
- Concurrent watch mode
- Resource contention testing
- System limits under concurrency

### 5. Large Dataset Tests (`perf-large-dataset`)
**Purpose**: Validate performance with enterprise-scale data
**Duration**: 30-60 minutes
**Resource Usage**: Very high
**Best For**: Enterprise deployment validation

**Includes**:
- Large dataset scaling (1K-100K resources)
- Mega file parsing (100MB-500MB files)
- Multi-file processing
- Diff performance at scale

### 6. Quick Tests (`perf-quick`)
**Purpose**: Fast performance validation for development
**Duration**: 1-3 minutes
**Resource Usage**: Low
**Best For**: CI/CD pipelines, development workflow

## Understanding Test Results

### Benchmark Output Format

```
BenchmarkMegaFileProcessing/10MB_file-8         3    384572955 ns/op    45234567 B/op    123456 allocs/op
```

**Breakdown**:
- `BenchmarkMegaFileProcessing/10MB_file-8`: Test name and CPU cores
- `3`: Number of iterations run
- `384572955 ns/op`: Nanoseconds per operation (lower is better)
- `45234567 B/op`: Bytes allocated per operation (lower is better)
- `123456 allocs/op`: Allocations per operation (lower is better)

### Performance Metrics

#### Processing Rate
```
Processing rate: 1,558 resources/second
```
- **Good**: >1,000 resources/sec
- **Acceptable**: 500-1,000 resources/sec
- **Concerning**: <500 resources/sec

#### Memory Efficiency
```
Memory efficiency: 2.8MB per 1,000 resources
```
- **Excellent**: <3MB per 1,000 resources
- **Good**: 3-5MB per 1,000 resources
- **Concerning**: >5MB per 1,000 resources

#### Concurrent Scaling
```
Concurrency level 8: processed 40,000 resources in 7.4s
Throughput: 5,405 resources/sec
```
- **Linear scaling**: Throughput increases proportionally with workers
- **Sublinear scaling**: Throughput increases but with diminishing returns
- **Negative scaling**: Throughput decreases with more workers

## Performance Analysis

### Reading Performance Reports

Performance reports are generated in Markdown format and include:

1. **Test Environment**: Hardware specs, OS, Go version
2. **Test Configuration**: Timeouts, benchmark settings, CI mode
3. **Individual Test Results**: Raw output from each test category
4. **Summary**: Key findings and recommendations

### Key Performance Indicators (KPIs)

#### Primary KPIs
- **Processing Time**: <30 seconds for 10,000 resources
- **Memory Usage**: <100MB for 25,000 resources
- **Concurrent Scaling**: Linear up to 16 workers
- **Large File Support**: 200MB files processed successfully

#### Secondary KPIs
- **Error Rate**: <5% under normal load, <10% under stress
- **Watch Mode Latency**: <100ms change detection
- **Memory Leak Rate**: <50MB growth over 24 hours
- **Diff Performance**: >10,000 resources/second comparison rate

### Performance Troubleshooting

#### Common Performance Issues

**Slow Processing Times**
```bash
# Symptoms
Processing time: >5 minutes for <10K resources
Memory usage: High swap activity

# Investigation
make perf-profile  # Generate CPU/memory profiles
go tool pprof profiles/cpu-profile.prof

# Solutions
- Enable streaming parser for large files
- Increase memory limits
- Reduce concurrent operations
- Use local state file copies
```

**High Memory Usage**
```bash
# Symptoms  
Memory usage: >2GB for normal operations
Test failures: OutOfMemory errors

# Investigation
make perf-memory  # Run memory analysis
go tool pprof profiles/heap-profile.prof

# Solutions
- Check for memory leaks
- Reduce batch sizes
- Enable garbage collection tuning
- Limit concurrent operations
```

**Poor Concurrent Performance**
```bash
# Symptoms
Concurrent throughput: Lower than single-threaded
High CPU usage: >90% with low throughput

# Investigation
make perf-concurrent  # Run concurrency tests

# Solutions
- Optimize worker count (2x CPU cores)
- Check for resource contention
- Verify I/O bottlenecks
- Review goroutine usage
```

## CI/CD Integration

### GitHub Actions Integration

Add performance testing to your CI workflow:

```yaml
# .github/workflows/performance.yml
name: Performance Tests

on:
  pull_request:
    branches: [ main ]
  schedule:
    - cron: '0 2 * * *'  # Nightly

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.21'
    
    - name: Run performance tests
      run: make perf-ci
    
    - name: Upload results
      uses: actions/upload-artifact@v4
      with:
        name: performance-results
        path: performance-results/
```

### Performance Regression Detection

```bash
# Create baseline
make perf-bench > baseline-results.txt

# After changes, compare
make perf-bench > current-results.txt
./scripts/compare-performance.sh baseline-results.txt current-results.txt
```

## Advanced Usage

### Custom Performance Testing

Create custom performance tests by extending existing benchmarks:

```go
// test/performance/custom_benchmarks_test.go
func BenchmarkCustomWorkflow(b *testing.B) {
    // Your custom benchmark implementation
}
```

### Profiling Integration

```bash
# Generate CPU profile
make perf-profile

# Analyze CPU profile
go tool pprof profiles/cpu-profile.prof
(pprof) top 10
(pprof) web

# Generate memory profile
go tool pprof profiles/heap-profile.prof
(pprof) top 10
(pprof) list FunctionName
```

### Continuous Performance Monitoring

```bash
# Run continuous monitoring (runs every hour)
make monitor

# Custom monitoring interval
while true; do
  make perf-quick
  sleep 1800  # 30 minutes
done
```

## Best Practices

### Development Workflow

1. **Before Making Changes**: Run `make perf-quick` to establish baseline
2. **During Development**: Use `make perf-quick` for rapid feedback
3. **Before PR**: Run `make perf-bench` to validate performance
4. **Release Testing**: Run `make perf-test` for comprehensive validation

### Performance Optimization Workflow

1. **Identify Bottlenecks**: Use `make perf-profile` to find hotspots
2. **Optimize Code**: Focus on top CPU/memory consumers
3. **Validate Improvements**: Compare before/after benchmarks
4. **Stress Test**: Run `make perf-stress` to validate under load

### System Requirements

#### Minimum for Testing
- **CPU**: 2 cores, 2.4GHz
- **RAM**: 4GB available
- **Storage**: 10GB free space
- **OS**: Linux, macOS, or Windows

#### Recommended for Full Testing
- **CPU**: 4+ cores, 3.0GHz
- **RAM**: 8GB+ available
- **Storage**: 20GB+ free space (SSD recommended)
- **Network**: High-speed connection for remote state files

## Troubleshooting

### Common Test Failures

**OutOfMemory Errors**
```bash
# Reduce test scope
./scripts/run-performance-tests.sh quick --ci

# Increase available memory
export GOMAXPROCS=2  # Limit Go processes
ulimit -m 8388608    # Limit memory to 8GB
```

**Timeout Errors**
```bash
# Increase timeout
./scripts/run-performance-tests.sh benchmarks --timeout 60m

# Reduce test scope
./scripts/run-performance-tests.sh --ci quick
```

**Permission Errors**
```bash
# Ensure script is executable
chmod +x scripts/run-performance-tests.sh

# Check file permissions
ls -la scripts/run-performance-tests.sh
```

### Getting Help

1. **Check System Requirements**: Run `./scripts/run-performance-tests.sh --system-check`
2. **Review Test Logs**: Check `performance-results/` directory
3. **Analyze Profiles**: Use `go tool pprof` for detailed analysis
4. **Consult Documentation**: See `docs/performance-analysis.md` for detailed metrics

## Conclusion

WGO's performance testing framework provides comprehensive tools for validating performance across various scales and usage patterns. Regular performance testing helps ensure WGO maintains excellent performance characteristics as the codebase evolves.

For optimal results:
- Run appropriate test categories for your use case
- Monitor key performance indicators
- Use profiling for detailed optimization
- Integrate performance testing into your development workflow

The performance testing framework scales from quick development checks to comprehensive enterprise validation, ensuring WGO performs well in all deployment scenarios.