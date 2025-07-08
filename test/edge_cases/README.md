# Edge Cases & Error Scenarios Test Suite

This directory contains comprehensive tests for failure scenarios and edge cases that might not be covered by standard CI testing.

## Test Categories

### 1. Network Failures (`network_failures_test.go`)
Tests various network-related failure scenarios:
- **Timeouts**: Quick timeouts, slow responses, extremely slow responses
- **Connectivity Issues**: DNS resolution failures, connection refused, invalid URLs
- **Rate Limiting**: API rate limit handling with Retry-After headers
- **Proxy Issues**: Invalid proxies, malformed proxy URLs
- **Partial Failures**: Some requests succeed, others fail intermittently
- **Concurrent Failures**: Multiple simultaneous network issues

### 2. Authentication Edge Cases (`auth_edge_cases_test.go`)
Tests authentication-related failure scenarios:
- **Expired Credentials**: AWS session tokens, GCP service account keys, Kubernetes tokens
- **Insufficient Permissions**: Valid credentials but insufficient permissions for operations
- **Missing Credentials**: Completely absent credential files or environment variables
- **Credential Rotation**: Scenarios during active credential rotation
- **Malformed Credentials**: Corrupted JSON files, invalid YAML structures, empty files
- **Concurrent Auth**: Authentication under concurrent load

### 3. File System Issues (`filesystem_issues_test.go`)
Tests file system-related failure scenarios:
- **Corrupted Files**: Malformed JSON/YAML, binary data in config files, null bytes
- **Permission Errors**: Read/write permission denied, directory access issues
- **Disk Space Issues**: Large file operations, insufficient disk space
- **Race Conditions**: Concurrent file operations, create/delete races
- **Symlink Issues**: Broken symlinks, circular references, mount point problems

### 4. Configuration Edge Cases (`config_edge_cases_test.go`)
Tests configuration parsing and validation failures:
- **Invalid YAML**: Syntax errors, inconsistent indentation, tab/space mixing
- **Missing Fields**: Required configuration fields absent
- **Type Mismatches**: Wrong field types, arrays where objects expected
- **Special Characters**: Unicode, emojis, control characters in configurations
- **Configuration Overrides**: Conflicting merges, circular references
- **Large Configurations**: Extremely nested structures, very long values

### 5. Real-World Scenarios (`real_world_scenarios_test.go`)
Tests bizarre edge cases that might occur in production:
- **Cloud Provider Maintenance**: Service unavailability, intermittent errors
- **Massive Infrastructure**: Large-scale scans with thousands of resources
- **Concurrent Collector Failures**: Multiple collectors failing simultaneously
- **Dynamic Resources**: Infrastructure changing during scans
- **Resource Pressure**: Memory/CPU exhaustion scenarios
- **Corrupted API Responses**: Malformed JSON, mixed encoding, unexpected formats
- **Multi-Region Failures**: Some regions accessible, others failing
- **Weird Scenarios**: File replacement during read, deep directory structures, Unicode edge cases

## Key Testing Principles

### Graceful Degradation
WGO should fail gracefully and provide actionable error messages even in the worst scenarios:
- Clear error messages with specific failure reasons
- Suggested remediation steps
- Professional Unix-style output without emojis
- Appropriate exit codes for different error types

### Error Classification
Tests verify that errors are properly classified using the WGO error framework:
- `ErrorTypeNetwork`: Network connectivity issues
- `ErrorTypeAuthentication`: Authentication failures
- `ErrorTypePermission`: Insufficient permissions
- `ErrorTypeConfiguration`: Configuration problems
- `ErrorTypeValidation`: Data validation errors

### Real-World Relevance
All edge cases are based on scenarios that could realistically occur:
- Cloud provider maintenance windows
- Credential rotation procedures
- Network infrastructure issues
- File system failures
- Large-scale infrastructure environments

## Running the Tests

### Run All Edge Case Tests
```bash
go test ./test/edge_cases/... -v
```

### Run Specific Test Categories
```bash
# Network failures only
go test ./test/edge_cases/network_failures_test.go -v

# Authentication edge cases
go test ./test/edge_cases/auth_edge_cases_test.go -v

# File system issues
go test ./test/edge_cases/filesystem_issues_test.go -v

# Configuration edge cases
go test ./test/edge_cases/config_edge_cases_test.go -v

# Real-world scenarios
go test ./test/edge_cases/real_world_scenarios_test.go -v
```

### Run with Short Mode (Skip Long-Running Tests)
```bash
go test ./test/edge_cases/... -v -short
```

## Expected Results

### Successful Failure Handling
Many tests are designed to verify that WGO fails **gracefully** rather than succeeding:
- Authentication errors should be caught and reported clearly
- Network timeouts should be handled with appropriate error messages
- Configuration errors should provide specific remediation steps
- File system issues should not crash the application

### Error Message Quality
Tests verify that error messages are:
- **Actionable**: Tell users what to do to fix the problem
- **Professional**: No emojis or casual language
- **Specific**: Identify the exact cause of the failure
- **Environment-Aware**: Provide context-specific solutions

### Performance Under Stress
Tests ensure WGO remains stable under adverse conditions:
- Large JSON parsing doesn't cause memory exhaustion
- Concurrent operations don't cause race conditions
- Network failures don't cause indefinite hangs
- File system issues are handled without data corruption

## Integration with CI/CD

These edge case tests complement the regular CI testing but are designed to:
- **Catch scenarios not covered by unit tests**
- **Verify error handling under realistic stress**
- **Ensure graceful degradation in production environments**
- **Test the professional error framework**

## Contributing New Edge Cases

When adding new edge case tests:
1. **Base on real scenarios**: Use actual production failure modes
2. **Test error handling**: Verify that failures are handled gracefully
3. **Check error messages**: Ensure they're actionable and professional
4. **Include documentation**: Explain what scenario you're testing
5. **Use appropriate timeouts**: Don't make tests unnecessarily slow

## Recovery Procedures

For each edge case category, see the [operational documentation](../../docs/troubleshooting.md) for:
- Detection procedures
- Root cause analysis
- Recovery steps
- Prevention measures

This test suite ensures that WGO remains robust and user-friendly even when everything goes wrong.