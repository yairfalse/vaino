# Performance Testing CI/CD Integration Guide

This document explains how VAINO's performance testing framework is integrated with CI/CD pipelines and when each type of test runs.

## 🚀 Performance Testing Strategy

### Automated Triggers

#### 1. **Pull Request Performance Checks**
- **Trigger**: PRs affecting performance-critical components
- **Test Level**: Quick performance validation
- **Duration**: 1-3 minutes
- **Purpose**: Catch performance regressions early

```yaml
# Triggers on changes to:
- internal/collectors/**
- internal/differ/**
- internal/analyzer/**
- internal/storage/**
- pkg/types/**
- cmd/wgo/**
```

#### 2. **Main Branch Performance Validation**
- **Trigger**: Merges to main branch
- **Test Level**: Comprehensive benchmarks
- **Duration**: 5-15 minutes
- **Purpose**: Establish performance baselines

#### 3. **Release Performance Certification**
- **Trigger**: Version tags (v*)
- **Test Level**: Full comprehensive testing
- **Duration**: 15-45 minutes
- **Purpose**: Validate release performance

#### 4. **Nightly Performance Monitoring**
- **Trigger**: Daily at 2 AM UTC
- **Test Level**: Comprehensive with profiling
- **Duration**: 30-60 minutes
- **Purpose**: Continuous performance monitoring

## 🎯 CI/CD Workflow Integration

### Selective Testing Enhancement

When performance-critical code changes are detected, the selective testing workflow automatically includes a performance regression check:

```yaml
# .github/workflows/selective-tests.yml
performance-regression-check:
  if: needs.detect-changes.outputs.performance == 'true'
  # Runs quick performance validation
```

**Example Trigger Scenarios:**
- Modifying collector logic → Performance check runs
- Updating diff algorithms → Performance check runs
- Changing data structures → Performance check runs
- Documentation changes → Performance check skipped

### Comprehensive Testing Integration

Main branch commits trigger performance validation as part of comprehensive testing:

```yaml
# .github/workflows/comprehensive-tests.yml
performance-validation:
  if: github.ref == 'refs/heads/main'
  # Runs full benchmarks and establishes baselines
```

### Dedicated Performance CI

The main performance testing workflow handles different scenarios:

```yaml
# .github/workflows/performance-ci.yml
- PR Performance: Quick validation for regression detection
- Main Branch: Comprehensive benchmarks with baseline updates
- Releases: Full validation with profiling and certification
- Nightly: Continuous monitoring with health checks
```

## 📊 Performance Test Types by Scenario

### Development Workflow

#### **Feature Development**
```bash
# Local development
make perf-quick        # Quick validation during development
make perf-bench        # Before creating PR
```

#### **Pull Request Review**
- **Automatic**: Performance regression check if touching critical components
- **Manual**: Reviewers can trigger comprehensive tests via workflow dispatch

#### **Pre-Merge Validation**
```bash
# CI runs automatically
- Quick performance check (if performance components changed)
- Full test suite validation
- Performance regression analysis
```

### Release Workflow

#### **Release Candidate Testing**
```bash
# Triggered on release branches
- Comprehensive performance validation
- Stress testing under various loads
- Memory usage analysis
- Concurrent operation validation
```

#### **Release Certification**
```bash
# Triggered on version tags
- Full performance test suite
- Performance profiling with pprof
- Scaling limits validation
- Release performance report generation
```

### Monitoring & Maintenance

#### **Continuous Monitoring**
```bash
# Nightly automated testing
- Performance health checks
- Baseline drift detection
- System compatibility validation
- Performance trend analysis
```

#### **Performance Investigation**
```bash
# Manual workflow dispatch options
- quick: Fast validation (1-3 min)
- benchmarks: Standard benchmarks (5-15 min)
- stress: Stress testing (15-30 min)
- comprehensive: Full validation (30-60 min)
```

## 🔍 Performance Results & Analysis

### Automated Result Analysis

#### **GitHub Actions Summaries**
Each performance run generates a detailed summary in the workflow:

```markdown
## 📊 Performance Benchmark Results
### 🖥️ Test Environment
- OS: Linux 5.15.0
- CPU Cores: 8
- Memory: 32GB
- Go Version: go1.23

### 🚀 Performance Highlights
File Processing Performance:
BenchmarkMegaFileProcessing/10MB_file-8    34   34096336 ns/op
BenchmarkMegaFileProcessing/50MB_file-8     7  162702125 ns/op
BenchmarkMegaFileProcessing/100MB_file-8    4  321437875 ns/op
```

#### **Performance Regression Detection**
- **Baseline Comparison**: Automatic comparison with previous main branch results
- **Threshold Alerts**: Configurable thresholds for performance degradation
- **Trend Analysis**: Long-term performance trend tracking

### Artifact Storage

#### **Result Retention**
- **PR Results**: 7 days (regression detection)
- **Main Branch**: 30 days (baseline maintenance)
- **Release Results**: 365 days (long-term tracking)
- **Nightly Results**: 90 days (trend analysis)

#### **Profile Storage**
- **CPU Profiles**: Available for detailed analysis
- **Memory Profiles**: Heap analysis for optimization
- **Performance Reports**: Markdown summaries for documentation

## 🛠️ Configuration & Customization

### Environment Variables

```yaml
env:
  GO_VERSION: '1.23'
  PERFORMANCE_TIMEOUT: '30m'
  BENCHMARK_COUNT: '3'
  BENCHMARK_TIME: '10s'
```

### Test Level Configuration

```yaml
workflow_dispatch:
  inputs:
    test_level:
      type: choice
      options:
      - quick          # 1-3 minutes
      - benchmarks     # 5-15 minutes  
      - stress         # 15-30 minutes
      - comprehensive  # 30-60 minutes
```

### Performance Thresholds

```yaml
# Configurable performance requirements
performance_requirements:
  file_processing:
    max_time_10k_resources: "30s"
    max_memory_per_1k_resources: "5MB"
  
  concurrent_operations:
    min_linear_scaling_workers: 8
    max_error_rate_percent: 5
  
  diff_operations:
    min_resources_per_second: 10000
    max_memory_growth_mb: 100
```

## 🚨 Performance Alerts & Monitoring

### Automated Alerting

#### **Regression Detection**
- **Threshold Breach**: >20% performance degradation
- **Memory Leak**: >100MB sustained growth
- **Error Rate**: >5% failure rate in tests

#### **Notification Channels**
```yaml
# Future integrations
- Slack: Critical performance issues
- Email: Weekly performance reports  
- Dashboard: Real-time performance metrics
```

### Manual Monitoring

#### **Performance Dashboards**
- **GitHub Actions**: Workflow summaries and trends
- **Artifacts**: Detailed reports and profiles
- **Release Notes**: Performance certification status

#### **Investigation Tools**
```bash
# Download and analyze profiles
gh run download <run-id> --name performance-profiles
go tool pprof cpu-profile.prof
go tool pprof heap-profile.prof
```

## 📈 Performance Optimization Workflow

### Development Optimization

1. **Identify Bottlenecks**: Use profiling results from CI
2. **Local Testing**: Run `make perf-profile` for detailed analysis  
3. **Iterative Improvement**: Test changes with `make perf-quick`
4. **Validation**: PR triggers regression check automatically

### Release Optimization

1. **Baseline Analysis**: Review nightly performance trends
2. **Targeted Optimization**: Focus on degraded metrics
3. **Validation Testing**: Use workflow dispatch for comprehensive testing
4. **Release Certification**: Automatic validation on release tags

## 🎯 Best Practices

### Development

- **Run Local Tests**: Use `make perf-quick` before committing
- **Monitor CI Results**: Check performance summaries in PRs
- **Profile When Needed**: Use `make perf-profile` for optimization
- **Understand Triggers**: Know which changes trigger performance tests

### Release Management

- **Monitor Trends**: Review nightly performance monitoring
- **Validate Releases**: Ensure performance certification passes
- **Document Changes**: Include performance impact in release notes
- **Plan Optimization**: Schedule performance improvements based on trends

### CI/CD Management

- **Threshold Tuning**: Adjust performance requirements as needed
- **Resource Monitoring**: Ensure CI resources are adequate
- **Result Analysis**: Regularly review performance artifacts
- **Baseline Maintenance**: Keep performance baselines current

## 🚀 Future Enhancements

### Planned Improvements

- **Performance Dashboard**: Real-time performance metrics visualization
- **Advanced Alerting**: Integration with monitoring systems
- **Regression Analysis**: ML-based performance trend analysis
- **Optimization Suggestions**: Automated performance improvement recommendations

### Integration Opportunities

- **Monitoring Systems**: Prometheus/Grafana integration
- **Alerting Platforms**: PagerDuty/Slack integration  
- **Performance Tracking**: Long-term performance database
- **Benchmarking Service**: External performance validation service

This comprehensive performance testing integration ensures VAINO maintains excellent performance characteristics while providing early detection of regressions and comprehensive validation for releases.