# Contributing to VAINO

Thank you for your interest in contributing to VAINO! This document provides guidelines for contributing to the project.

## Branch Strategy

We use a Git Flow-inspired branching model:

- **`main`** - Production-ready code only. All code here should be stable and tested.
- **`develop`** - Integration branch for features. This is where features are merged before release.
- **`feature/`** - New features (e.g., `feature/aws-ec2-collector`, `feature/claude-analysis`)
- **`fix/`** - Bug fixes (e.g., `fix/terraform-state-parsing`, `fix/gcp-auth-timeout`)
- **`provider/`** - New provider implementations (e.g., `provider/azure-support`)

### Branch Naming Examples

```bash
feature/add-azure-provider
feature/webhook-slack-integration
fix/kubernetes-auth-error
fix/memory-leak-large-snapshots
provider/digitalocean-support
```

## Commit Strategy

### Small, Atomic Commits

Each commit should represent one logical change. This makes it easier to:
- Review changes
- Revert if necessary
- Understand the project history
- Cherry-pick specific fixes

### Commit Message Format

We follow the Conventional Commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

#### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning (white-space, formatting)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Changes to build process or auxiliary tools

#### Scope (optional)

The scope should be the name of the affected component:

- `aws`, `gcp`, `kubernetes`, `terraform` - for providers
- `cli` - for command-line interface changes
- `storage`, `differ`, `watcher` - for core components
- `config`, `auth` - for configuration and authentication

#### Examples

```bash
feat(aws): add support for RDS cluster snapshots

fix(terraform): handle empty state files gracefully

docs: update installation instructions for Windows

refactor(differ): extract comparison logic to separate module

test(kubernetes): add integration tests for pod listing

chore: update Go version to 1.21
```

### Commit Message Best Practices

1. **Subject line**
   - Use imperative mood ("Add" not "Added" or "Adds")
   - Don't capitalize first letter after type
   - No period at the end
   - Keep under 50 characters

2. **Body** (optional)
   - Wrap at 72 characters
   - Explain *what* and *why*, not *how*
   - Include motivation for the change
   - Contrast behavior with previous behavior

3. **Footer** (optional)
   - Reference GitHub issues: `Fixes #123`
   - Note breaking changes: `BREAKING CHANGE: <description>`
   - Co-authors: `Co-authored-by: Name <email>`

### Full Example

```
feat(aws): add support for EKS cluster discovery

Previously, EKS clusters were not included in AWS scans. This adds
support for discovering and collecting EKS cluster configurations,
including:
- Cluster metadata (version, endpoint, status)
- Node groups and their configurations
- Associated IAM roles and policies

The implementation uses the AWS EKS API and follows the same
pattern as other AWS resource collectors.

Fixes #234
```

## Pull Request Process

1. Create your feature branch from `develop`
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the commit guidelines

3. Run tests and ensure they pass
   ```bash
   make test
   make lint
   ```

4. Push your branch and create a Pull Request to `develop`
   ```bash
   git push origin feature/your-feature-name
   ```

5. Ensure your PR:
   - Has a clear title and description
   - References any related issues
   - Includes tests for new functionality
   - Updates documentation as needed
   - Passes all CI checks

## Development Setup

1. Fork and clone the repository
   ```bash
   git clone https://github.com/YOUR_USERNAME/vaino.git
   cd vaino
   ```

2. Install dependencies
   ```bash
   go mod download
   ```

3. Build the project
   ```bash
   make build
   ```

4. Run tests
   ```bash
   make test
   ```

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Add comments for exported functions
- Keep functions small and focused

## Testing

- Write unit tests for new functionality
- Include integration tests for provider implementations
- Aim for >80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

## Documentation

- Update README.md for user-facing changes
- Add/update command documentation in docs/
- Include examples for new features
- Document configuration options
- Keep CLAUDE.md updated with architectural changes

## Questions?

Feel free to:
- Open an issue for discussion
- Ask in GitHub Discussions
- Review existing PRs for examples

Thank you for contributing to VAINO!