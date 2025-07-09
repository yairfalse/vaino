# WGO Testing Strategy & Organization

## Overview

WGO uses a modular testing approach to avoid running all tests when only specific components change. This strategy provides faster feedback cycles and more efficient development workflows.

## Test Categories

### 🎯 Smart Testing (Default)
- **Command**: `make test`
- **Purpose**: Automatically detects changed files and runs only relevant tests
- **Speed**: ⚡ Fast (30 seconds - 2 minutes)
- **When to use**: Daily development, quick validation

### 🔬 Component Testing
Target specific components when you know what you're working on:

| Component | Command | Description |
|-----------|---------|-------------|
| Terraform | `make test-terraform` | Terraform collector only |
| GCP | `make test-gcp` | GCP collector only |
| AWS | `make test-aws` | AWS collector only |
| Kubernetes | `make test-kubernetes` | Kubernetes collector only |
| Commands | `make test-commands` | CLI commands only |
| Config | `make test-config` | Configuration system only |
| All Collectors | `make test-collectors` | All collectors together |

### 🧪 Full Testing
- **Command**: `make test-all`
- **Purpose**: Complete test suite including integration and E2E tests
- **Speed**: 🐌 Slow (5-15 minutes)
- **When to use**: Before PRs, releases, comprehensive validation

### ⚡ Parallel Testing
- **Command**: `make test-parallel`
- **Purpose**: Run component tests in parallel for CI environments
- **Speed**: 🚀 Fast (2-5 minutes)
- **When to use**: CI/CD pipelines, automated testing

## Test Organization

### Directory Structure
```
├── internal/
│   ├── collectors/
│   │   ├── terraform/
│   │   │   ├── collector_test.go      # Unit tests
│   │   │   ├── performance_test.go    # Performance tests
│   │   │   └── benchmark_test.go      # Benchmarks
│   │   ├── aws/
│   │   │   ├── collector_test.go
│   │   │   ├── client_test.go
│   │   │   └── *_test.go              # Service-specific tests
│   │   └── .../
│   └── .../
├── test/
│   ├── integration/                   # Integration tests
│   ├── e2e/                          # End-to-end tests
│   └── fixtures/                     # Test data
└── cmd/wgo/commands/
    └── *_test.go                     # Command tests
```

### Test Types

#### 🟢 Unit Tests
- **Location**: Alongside source files (`*_test.go`)
- **Scope**: Single functions, methods, small components
- **Speed**: Very fast (< 100ms per test)
- **Mocking**: Heavy use of mocks and stubs

#### 🟡 Integration Tests
- **Location**: `test/integration/`
- **Scope**: Component interactions, external dependencies
- **Speed**: Medium (1-10 seconds per test)
- **Dependencies**: May require external services

#### 🔴 End-to-End Tests
- **Location**: `test/e2e/`
- **Scope**: Full user workflows, complete system
- **Speed**: Slow (10+ seconds per test)
- **Dependencies**: Real or realistic environments

## Smart Testing Logic

The smart testing system uses git to detect changes and maps them to test suites:

### File Change → Test Mapping

| Changed Files | Tests Run | Reasoning |
|---------------|-----------|-----------|
| `internal/collectors/terraform/*` | Terraform collector tests | Direct component change |
| `internal/collectors/aws/*` | AWS collector tests | Direct component change |
| `cmd/wgo/commands/*` | Command tests | CLI functionality change |
| `pkg/config/*` | Config tests | Configuration system change |
| `go.mod`, `go.sum` | Dependency verification | Dependency changes |
| Generic `*.go` files | Core tests | Potential wide impact |

### Smart Test Algorithm
1. **Detect Changes**: Compare against main/master branch
2. **Analyze Impact**: Map files to components
3. **Select Tests**: Choose minimal test set
4. **Execute**: Run only necessary tests
5. **Report**: Show time saved vs full test suite

## Development Workflows

### 🚀 Daily Development
```bash
# Quick validation after changes
make test

# Test specific component you're working on
make test-aws

# Quick build check
make build
```

### 🔧 Before Committing
```bash
# Format and run affected tests
make fmt test

# Or use the pre-commit target
make pre-commit
```

### 🚢 Before Pull Request
```bash
# Full validation
make test-all

# Generate coverage report
make test-coverage

# Lint everything
make lint
```

### 🏗️ CI/CD Pipeline
```bash
# Parallel execution for speed
make test-parallel

# Full coverage for release builds
make test-all test-coverage
```

## Performance Guidelines

### Test Performance Targets
- **Unit tests**: < 100ms each
- **Integration tests**: < 10s each
- **E2E tests**: < 60s each
- **Full test suite**: < 15 minutes

### Optimization Strategies
1. **Parallel Execution**: Use `make test-parallel`
2. **Smart Selection**: Use `make test` for development
3. **Test Caching**: Go automatically caches unchanged tests
4. **Mock Heavy**: Use mocks for external dependencies
5. **Short Flag**: Use `-short` flag to skip slow tests

## Test Configuration

### Environment Variables
```bash
# Enable verbose test output
VERBOSE=true make test

# Set custom timeout
TEST_TIMEOUT=5m make test-terraform

# Skip integration tests
SKIP_INTEGRATION=true make test
```

### Test Tags
```go
// +build integration
// Integration test - only runs with integration flag

// +build !short  
// Slow test - skipped with -short flag
```

## Troubleshooting

### Common Issues

#### "No tests to run"
```bash
# Ensure you're in the right directory
cd /path/to/wgo

# Check for changed files
git status

# Force run all tests
make test-all
```

#### "Tests failing after merge"
```bash
# Update dependencies
make deps

# Clean and rebuild
make clean build

# Run full test suite
make test-all
```

#### "Smart test not detecting changes"
```bash
# Check git status
git status

# Run with verbose output
VERBOSE=true make test

# Fallback to manual testing
make test-terraform  # or specific component
```

## Best Practices

### ✅ Do
- Use `make test` for daily development
- Run component-specific tests when working on specific areas
- Use `make test-all` before submitting PRs
- Write fast unit tests with good mocking
- Keep integration tests focused and independent

### ❌ Don't
- Always run full test suite during development
- Write slow unit tests without the `-short` flag
- Skip testing when making "small" changes
- Write tests that depend on external state
- Ignore failing tests

## Metrics & Monitoring

The testing system tracks:
- **Test Execution Time**: Time saved by smart testing
- **Test Coverage**: Code coverage by component
- **Test Reliability**: Flaky test detection
- **Performance Trends**: Test speed over time

Use `make test-coverage` to generate detailed coverage reports.

## Contributing

When adding new components:
1. Create component-specific test targets in `Makefile`
2. Update the smart test script mapping
3. Follow the established test organization
4. Add appropriate test categories (unit/integration/e2e)
5. Update this documentation

## Summary

The modular testing approach provides:
- ⚡ **Faster Feedback**: Only test what changed
- 🎯 **Focused Testing**: Target specific components
- 🔄 **Parallel Execution**: Speed up CI/CD
- 📊 **Better Organization**: Clear test categories
- 🚀 **Developer Productivity**: Less waiting, more coding

Choose the right testing strategy for your workflow and enjoy faster, more efficient development!