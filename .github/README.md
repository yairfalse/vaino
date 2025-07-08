# WGO CI/CD Pipeline

This directory contains GitHub Actions workflows for the WGO (What's Going On) project.

## Workflows

### 🚀 Main CI Pipeline (`ci.yml`)
The main CI pipeline runs on every push and pull request, providing comprehensive testing across multiple dimensions:

- **Unit Tests**: Core functionality testing with coverage reporting
- **Lint**: Code quality and style checking with golangci-lint
- **Security**: Security scanning with gosec and vulnerability checks
- **Build Matrix**: Cross-platform builds (Linux, macOS, Windows × amd64, arm64)
- **Collector Integration**: Tests for Terraform and Kubernetes collectors
- **Correlation Tests**: Dedicated testing for correlation and timeline features
- **System Tests**: Integration testing of complete workflows
- **E2E Tests**: End-to-end testing (conditional)

### 🧪 Correlation Feature Pipeline (`correlation-tests.yml`)
Specialized testing for correlation and timeline features, triggered by:
- Changes to correlation-related files
- Manual workflow dispatch
- Scheduled runs

**Test Coverage:**
- **Unit Tests**: All correlation patterns (scaling, deployments, config updates, security)
- **Performance Tests**: Benchmarks for 10-1000+ changes with memory usage tracking
- **System Tests**: Complete workflow integration testing
- **E2E Tests**: End-to-end correlation workflow validation

### ✅ Test Validation (`validate-tests.yml`)
Weekly validation to ensure test suite health:
- Test compilation validation
- Test discovery and statistics
- Smoke tests
- Benchmark availability checks

## Correlation Testing Features

### 🎯 Comprehensive Test Coverage
Our correlation features are tested at multiple levels:

1. **Unit Level**
   - Individual correlation pattern detection
   - Timeline formatting logic
   - Simple differ functionality
   - Confidence level assignment
   - Time window enforcement
   - False correlation prevention

2. **Performance Level**
   - Correlation engine benchmarks (10-1000+ changes)
   - Memory usage patterns
   - Timeline formatting performance
   - End-to-end workflow benchmarks
   - Performance requirement validation

3. **System Level**
   - Complete correlation workflows
   - Timeline visualization integration
   - JSON output format validation
   - Large-scale correlation accuracy
   - Multi-pattern correlation scenarios

4. **End-to-End Level**
   - CLI command integration
   - Real-world workflow simulation
   - Cross-platform compatibility

### 📊 Performance Requirements
Our CI enforces these performance requirements:
- **Small datasets** (50 changes): < 10ms
- **Medium datasets** (500 changes): < 100ms  
- **Large datasets** (2000 changes): < 1s
- **Memory growth**: Linear with input size
- **Correlation accuracy**: >90% for known patterns

### 🔍 Pattern Detection Validation
The CI validates detection of these correlation patterns:
- **Scaling Events**: Deployment replica changes + pod changes
- **Service Deployments**: New services + related resources
- **Configuration Updates**: ConfigMap/Secret changes + app restarts
- **Security Rotations**: Multiple secret updates in time window
- **Network Changes**: Service + ingress + policy correlations

## Usage

### Running Locally
```bash
# Run comprehensive test suite
./scripts/run-comprehensive-tests.sh

# Run specific test categories
go test -v ./internal/analyzer/... -run "Test.*Correlat"
go test -v ./test/performance/... -bench=BenchmarkCorrelation
go test -v ./test/system/... -run="TestCorrelation" -short
```

### Manual Workflow Triggers
You can manually trigger the correlation test pipeline:

1. Go to Actions → Correlation & Timeline Features
2. Click "Run workflow"
3. Select test level: `unit`, `performance`, `system`, `e2e`, or `all`

### CI Status Checks
All PRs must pass:
- ✅ Unit tests with >40% coverage
- ✅ Lint checks
- ✅ Security scans
- ✅ Cross-platform builds
- ✅ Correlation feature tests
- ✅ System integration tests

## Artifacts & Reports

### Coverage Reports
- Main coverage report: `coverage.out`
- Correlation-specific coverage: `correlation-coverage.out`
- Timeline formatter coverage: `timeline-coverage.out`

### Performance Reports
- Correlation benchmarks: `correlation-bench.txt`
- Memory usage benchmarks: `memory-bench.txt`
- Timeline formatting benchmarks: `timeline-bench.txt`

### Build Artifacts
- Cross-platform binaries: `wgo-{os}-{arch}`
- Test artifacts and logs
- Security scan results (SARIF format)

## Configuration

### Environment Variables
- `GO_VERSION`: Go version for builds (currently 1.21)

### Conditional Execution
- **E2E tests**: Only run on push or with `e2e-tests` label
- **Full performance tests**: Skipped in short mode
- **System tests**: Include timeouts for CI environments

## Monitoring & Alerting

### Scheduled Checks
- Weekly test validation (Mondays 10:00 UTC)
- Dependency vulnerability scanning
- Test suite health monitoring

### Failure Handling
- Test failures are tracked and reported
- Performance regressions are flagged
- Security issues block merges
- Build failures prevent releases

## Contributing

When adding new correlation features:

1. **Add unit tests** in `internal/analyzer/` or `internal/differ/`
2. **Add performance tests** in `test/performance/`
3. **Add system tests** in `test/system/` for integration scenarios
4. **Update CI** if new test patterns are needed
5. **Document** performance characteristics and requirements

The CI will automatically validate your correlation features across all test levels!