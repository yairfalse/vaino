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

# VAINO Development Guidelines

## 🎯 Mission
Build enterprise-grade infrastructure drift detection tool with multi-provider support and intelligent analysis.

## 🏗️ Architecture Rules (CRITICAL)

### Component Hierarchy (MANDATORY)
```
Core Layer:     pkg/types/           # Core data structures
Collectors:     internal/collectors/ # Provider implementations  
Commands:       cmd/vaino/commands/  # CLI command layer
Storage:        internal/storage/    # Persistence layer
Analysis:       internal/analysis/   # Drift detection & correlation
Errors:         internal/errors/     # Error handling system
```
**RULE:** Components can only import from same level or lower. NO circular dependencies.

### Module Structure (MANDATORY)
- Single go.mod at root
- Must build standalone: `go build ./...`
- Must test standalone: `go test ./...`
- Each provider must implement `EnhancedCollector` interface

## 🌿 Git Workflow & Branching Strategy

### Branch Strategy
- **main**: Production-ready code only
- **develop**: Integration branch for features
- **feature/**: New features (`feature/aws-ec2-collector`, `feature/claude-analysis`)
- **fix/**: Bug fixes (`fix/terraform-state-parsing`, `fix/gcp-auth-timeout`)
- **provider/**: New provider implementations (`provider/azure-support`)

### Commit Strategy
- **Small, atomic commits**: Each commit should represent one logical change
- **Descriptive messages**: Use conventional commit format
  ```
  type(scope): description
  
  Examples:
  feat(collectors): add AWS Lambda function scanning
  fix(storage): resolve snapshot corruption on concurrent writes
  docs(README): update provider authentication examples
  test(terraform): add unit tests for state file parsing
  provider(gcp): implement GKE cluster resource collection
  ```

### Workflow Rules
1. **Always work on dedicated branches** - Never commit directly to main/develop
2. **Pull Request only** - All code must go through PR review
3. **Branch from develop** for features, from main for hotfixes
4. **Keep branches short-lived** - Merge within 1-2 days when possible
5. **Rebase before merge** to maintain clean history

## ⚡ Agent Instructions (BRUTAL)

### Build Requirements
1. **MUST FORMAT:** `go fmt ./...` before any commit
2. **MUST COMPILE:** `go build ./...` must pass
3. **MUST TEST:** `go test ./...` must pass
4. **MUST LINT:** `golangci-lint run` must pass
5. **NO STUBS:** No "TODO", "not implemented", empty functions
6. **SHOW PROOF:** Paste build/test output or FAIL
7. **WORK ON BRANCHES:** Always create and work on dedicated feature/fix branches

### Quality Standards
- **80% test coverage minimum**
- **Implement complete EnhancedCollector interface** for new providers
- **Proper error handling** using internal/errors package
- **NO interface{} or map[string]interface{}** in public APIs
- **Context cancellation support** in all long-running operations
- **YOU work on a dedicated branch**

### Provider Implementation Requirements
All new providers MUST implement:
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

### Verification (MANDATORY)
```bash
# You MUST show this output:
go fmt ./...             # Format code first
go build ./...           # Must compile cleanly
go test ./...            # All tests must pass
golangci-lint run        # Linting must pass
go mod verify            # Dependencies verified
go mod tidy              # Clean unused dependencies
```

## 🔧 Current Priorities
- Multi-provider infrastructure drift detection
- Claude AI integration for intelligent analysis
- Real-time monitoring capabilities
- Performance optimization for large-scale infrastructure

### Success Metrics
- All providers building independently ✅
- Drift detection accuracy > 95% ✅
- AI analysis providing actionable insights ✅
- Sub-second response for diff operations ✅
- Enterprise-ready authentication support ✅

### Core Mission
- **Drift Detection:** Identify infrastructure changes across all providers
- **Intelligence:** AI-powered analysis of changes and recommendations
- **Unix Philosophy:** Simple, composable tools that work together
- **Zero Configuration:** Works out of the box with smart defaults

## 🚫 Failure Conditions

### Instant Task Reassignment If
- Code not formatted (go fmt failures)
- Build errors
- Test failures
- Linting errors
- Incomplete provider interface implementation
- Missing verification output
- Stub functions or TODOs
- **Work directly on main/develop without PR**
- **Commits without proper branch strategy**
- **Breaking existing provider compatibility**

### No Excuses For
- "Forgot to format" - Always run `go fmt ./...`
- "Complex existing code" - Follow VAINO patterns
- "Need to refactor first" - Implement incrementally
- "Just one small TODO" - Zero tolerance
- "Can't find interfaces" - Check pkg/types and internal/collectors
- "Provider too complex" - Use existing providers as templates
- **"Working directly on main" - Use branches**
- **"Big commit with everything" - Use small commits**

## 📋 Task Template
Every task must include:

```markdown
## Branch Information
- **Working Branch:** feature/task-name or fix/issue-name
- **Base Branch:** develop (or main for hotfixes)
- **Target Branch:** develop
- **Provider Affected:** (if applicable: aws/gcp/kubernetes/terraform)

## Verification Results

### Code Formatting:
```bash
$ go fmt ./...
[PASTE OUTPUT - should show no files changed]
```

### Build Test
```bash
$ go build ./...
[PASTE OUTPUT - should show successful compilation]
```

### Unit Tests
```bash
$ go test ./...
[PASTE OUTPUT - should show all tests passing]
```

### Linting
```bash
$ golangci-lint run
[PASTE OUTPUT - should show no issues]
```

### Provider Interface Compliance (if new provider)
```bash
$ go test ./internal/collectors/[provider]/ -v
[PASTE OUTPUT showing interface compliance tests pass]
```

### Commit History
```bash
$ git log --oneline -5
[PASTE OUTPUT showing small, atomic commits]
```

### Files Created/Modified
- file1.go (X lines) - [brief description]
- file2_test.go (Y lines) - [test coverage description]
Total: Z lines

## Architecture Compliance
✅ Code properly formatted
✅ Follows component hierarchy
✅ No circular dependencies
✅ Proper error handling with internal/errors
✅ Context cancellation support
✅ Provider interface compliance (if applicable)
✅ Working on dedicated branch
✅ Small, atomic commits
✅ Ready for PR review

## Testing Coverage
✅ Unit tests for new functionality
✅ Integration tests for provider changes
✅ Error path testing
✅ Edge case coverage
```

## 🎯 Bottom Line
**Format code. Build working code. Test thoroughly. Follow VAINO architecture. Use proper Git workflow. Small commits. Dedicated branches. Implement complete interfaces. No shortcuts.**

Infrastructure teams depend on VAINO for critical drift detection. Deliver enterprise-quality code or get reassigned.
