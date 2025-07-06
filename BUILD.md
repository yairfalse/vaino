# WGO Build Automation

This project uses [Task](https://taskfile.dev) for build automation and development workflows.

## Quick Start

### 1. Install Task (if not already installed)

```bash
# Option 1: Use our install script
bash scripts/install-task.sh

# Option 2: Install manually
# macOS
brew install go-task/tap/go-task

# Linux
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# Windows
choco install go-task
```

### 2. See Available Tasks

```bash
task
# or
task --list-all
```

### 3. Basic Development Workflow

```bash
# Download dependencies
task deps

# Build the binary
task build

# Run tests
task test

# Run linting
task lint

# Run the application
task run

# Show version info
task version
```

## Available Tasks

### Core Development
- **`task deps`** - Download dependencies and tidy modules
- **`task build`** - Build the CLI binary with version info
- **`task run`** - Run the built application
- **`task run:dev`** - Run the application in development mode (go run)
- **`task test`** - Run tests with coverage report
- **`task lint`** - Run golangci-lint

### Build Tasks
- **`task build:all`** - Cross-platform builds (linux, darwin, windows, amd64/arm64)
- **`task clean`** - Remove build artifacts and clear all caches
- **`task clean:build`** - Remove only build artifacts (keep caches)

### Installation
- **`task install`** - Install binary to system (/usr/local/bin)
- **`task install:local`** - Install binary to user's local bin (~/.local/bin)

### Quality Assurance
- **`task fmt`** - Format Go code with gofmt and goimports
- **`task vet`** - Run go vet
- **`task lint:fix`** - Run golangci-lint with auto-fix
- **`task sec`** - Run security checks with gosec
- **`task vuln`** - Run vulnerability checks with govulncheck
- **`task check`** - Run all quality checks (fmt, vet, lint, test, sec, vuln)

### Testing
- **`task test:unit`** - Run unit tests only
- **`task test:integration`** - Run integration tests

### CI/CD
- **`task ci`** - Run CI pipeline (used in GitHub Actions)
- **`task release`** - Prepare release build

### Docker
- **`task docker:build`** - Build Docker image
- **`task docker:run`** - Run Docker container

### Utility
- **`task version`** - Show version information
- **`task info`** - Show build configuration
- **`task help`** - Show detailed help

## Build Configuration

### Environment Variables
- **`CGO_ENABLED=0`** - Disable CGO for static binaries
- **`GO111MODULE=on`** - Enable Go modules

### Build Variables
The build system automatically injects version information:

```go
var (
    version   = "dev"        // Git tag or "dev"
    commit    = "unknown"    // Git commit hash
    buildTime = "unknown"    // Build timestamp
    builtBy   = "taskfile"   // Build system identifier
)
```

### LDFLAGS
The following linker flags are automatically applied:
- `-s -w` - Strip debug information for smaller binaries
- `-X main.version={{.VERSION}}` - Inject version from git
- `-X main.commit={{.COMMIT}}` - Inject commit hash
- `-X main.buildTime={{.BUILD_TIME}}` - Inject build timestamp
- `-X main.builtBy=taskfile` - Inject build system name

## Cross-Platform Builds

The `task build:all` command builds for multiple platforms:

- **Linux**: amd64, arm64
- **macOS**: amd64, arm64 (Intel & Apple Silicon)
- **Windows**: amd64, arm64

All binaries are created in the `dist/` directory with the naming convention:
- `wgo-linux-amd64`
- `wgo-darwin-arm64`
- `wgo-windows-amd64.exe`
- etc.

## Development Workflow

### Daily Development
```bash
# Start developing
task deps
task run:dev -- --help

# Run tests frequently
task test:unit

# Before committing
task check
```

### Preparing for Release
```bash
# Full release build
task release

# Check what was built
ls -la dist/
```

### Installing Locally
```bash
# Install to user's local bin (recommended)
task install:local

# Or install system-wide (requires sudo)
task install
```

## Directory Structure

```
dist/                    # Build output directory
├── wgo                 # Local platform binary
├── wgo-linux-amd64    # Linux binary
├── wgo-darwin-arm64   # macOS ARM binary
├── wgo-windows-amd64.exe # Windows binary
└── coverage/          # Test coverage reports
    ├── coverage.out   # Coverage data
    └── coverage.html  # HTML coverage report
```

## Integration with IDEs

### VS Code
Add to your `.vscode/tasks.json`:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "build",
            "type": "shell",
            "command": "task",
            "args": ["build"],
            "group": "build"
        },
        {
            "label": "test",
            "type": "shell",
            "command": "task",
            "args": ["test"],
            "group": "test"
        }
    ]
}
```

### GoLand/IntelliJ
Configure External Tools:
- Program: `task`
- Arguments: `build` (or other task name)
- Working directory: `$ProjectFileDir$`

## Troubleshooting

### Task not found
If you get "task: command not found":
1. Run `bash scripts/install-task.sh`
2. Or install manually from [taskfile.dev](https://taskfile.dev/installation/)

### Build failures
1. Check Go version: `go version` (requires Go 1.21+)
2. Clean and retry: `task clean && task build`
3. Update dependencies: `task deps`

### Permission errors during install
- Use `task install:local` instead of `task install`
- Or run `task install` with sudo

## Performance Tips

- Use `task build` instead of `go build` for optimized binaries with version info
- Run `task test:unit` for fast feedback during development
- Use `task run:dev` to skip the build step during development
- Keep `dist/` directory for incremental builds (don't run `task clean` too often)

## Additional Resources

- [Task Documentation](https://taskfile.dev/)
- [Go Build Documentation](https://golang.org/pkg/go/build/)
- [GoReleaser](https://goreleaser.com/) (for advanced release automation)