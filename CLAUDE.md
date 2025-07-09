# VAINO Project Context for Claude

This document provides context about the VAINO project to help Claude understand the codebase and assist effectively.

## Project Overview

VAINO (What's Going On) is a comprehensive infrastructure drift detection tool that acts as "git diff for infrastructure". It helps DevOps teams track changes in their infrastructure over time by:

- Scanning infrastructure state from multiple providers (Terraform, AWS, GCP, Kubernetes)
- Creating snapshots of the current state
- Comparing snapshots to detect drift
- Providing Unix-style output for easy integration with existing tools

## Key Design Principles

1. **Simplicity First**: VAINO should be as easy to use as `git diff`
2. **Zero Configuration**: Works out of the box with smart defaults
3. **Multi-Provider**: Supports multiple infrastructure providers through a plugin architecture
4. **Unix Philosophy**: Does one thing well, plays nicely with other tools
5. **Accessibility**: Clear error messages, helpful documentation, easy installation

## Architecture

### Core Components

- **Collectors**: Provider-specific implementations that gather infrastructure state
  - Located in `internal/collectors/`
  - Each implements the `EnhancedCollector` interface
  - Current providers: Terraform, AWS, GCP, Kubernetes

- **Commands**: CLI commands implemented with Cobra
  - Located in `cmd/wgo/commands/`
  - Main commands: scan, diff, watch, status, configure

- **Types**: Core data structures
  - Located in `pkg/types/`
  - Key types: Snapshot, Resource, Diff

- **Storage**: Handles persistence of snapshots
  - Located in `internal/storage/`
  - Stores snapshots in `~/.wgo/`

### Provider Architecture

Each provider implements the `EnhancedCollector` interface:

```go
type EnhancedCollector interface {
    Name() string
    Collect(ctx context.Context, config CollectorConfig) (*types.Snapshot, error)
    Status() string
    Validate(config CollectorConfig) error
    AutoDiscover() (CollectorConfig, error)
    SupportedRegions() []string
}
```

## Testing Strategy

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test provider interactions with mocked APIs
- **System Tests**: Test full workflows
- **E2E Tests**: Test complete user scenarios

Run tests with: `make test` or `go test ./...`

## Error Handling

VAINO uses a custom error system (`internal/errors`) that provides:
- Categorized errors (Provider, Configuration, Network, etc.)
- User-friendly error messages
- Actionable solutions
- Help command references

## Development Workflow

1. **Building**: `make build` or `go build ./cmd/wgo`
2. **Testing**: `make test` or specific test files
3. **Linting**: `make lint` (runs golangci-lint)
4. **Type Checking**: Built into Go compilation

## Installation Methods

1. **Universal Script**: `curl -sSL https://install.wgo.sh | bash`
2. **Homebrew**: `brew install yairfalse/wgo/wgo`
3. **Package Managers**: APT, YUM, Chocolatey, Scoop
4. **Docker**: `docker run yairfalse/wgo:latest`
5. **From Source**: `go install github.com/yairfalse/wgo/cmd/wgo@latest`

## Common Tasks

### Adding a New Provider

1. Create directory: `internal/collectors/newprovider/`
2. Implement `EnhancedCollector` interface
3. Add normalizer for resource conversion
4. Register in `cmd/wgo/commands/scan.go`
5. Add tests

### Modifying Commands

1. Commands are in `cmd/wgo/commands/`
2. Use Cobra for command structure
3. Follow existing patterns for flags and output
4. Update help text and examples

### Working with Snapshots

- Snapshots are JSON files stored in `~/.wgo/history/`
- Latest scan stored as `~/.wgo/last-scan-{provider}.json`
- Use `internal/storage` package for operations

## Code Style

- Follow Go idioms and conventions
- Use meaningful variable names
- Add comments for complex logic
- Keep functions small and focused
- Handle errors explicitly
- Use the custom error system for user-facing errors

## CI/CD

- GitHub Actions workflow in `.github/workflows/`
- Runs on push and PR
- Tests all platforms and Go versions
- Releases handled by GoReleaser

## Important Files

- `cmd/wgo/main.go`: Entry point
- `cmd/wgo/commands/root.go`: Root command setup
- `internal/collectors/registry.go`: Collector registration
- `pkg/types/snapshot.go`: Core data types
- `.goreleaser.yml`: Multi-platform build configuration
- `scripts/install.sh`: Universal installation script

## Debugging Tips

1. Use `-v` or `--verbose` flag for detailed output
2. Check `~/.wgo/logs/` for debug logs (if enabled)
3. Use `VAINO_DEBUG=1` environment variable
4. Provider-specific debugging:
   - AWS: Check AWS credentials and regions
   - GCP: Verify project ID and authentication
   - Kubernetes: Check kubeconfig and contexts
   - Terraform: Ensure state files are accessible

## Common Issues

1. **"Provider not found"**: Provider not registered in scan.go
2. **Authentication errors**: Check provider credentials
3. **Permission errors**: Ensure proper IAM/RBAC permissions
4. **No resources found**: Check provider configuration and regions

## Future Enhancements

- Policy engine for drift rules
- Web UI for visualization
- Metrics and alerting integrations
- More providers (Azure, DigitalOcean, etc.)
- State comparison with "expected" configurations