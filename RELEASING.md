# VAINO Release Process Documentation

This document describes the complete release process for VAINO, including preparation, automation, and post-release steps.

## Overview

VAINO uses a fully automated release pipeline that:
- Builds binaries for multiple platforms
- Creates packages for various package managers
- Publishes Docker images
- Updates package repositories
- Validates releases across platforms

## Prerequisites

### Required Tools
- Go 1.23+
- Git
- GitHub CLI (`gh`)
- Docker (for local testing)

### Repository Setup
1. **GPG Key**: Run `./scripts/security/gpg-setup.sh` to set up package signing
2. **GitHub Secrets**: Add required secrets (see `github-secrets.md`)
3. **External Repositories**: Create Homebrew tap and Scoop bucket repositories

### GitHub Secrets Configuration

Add these secrets to your GitHub repository (Settings → Secrets and variables → Actions):

| Secret | Description | How to Get |
|--------|-------------|------------|
| `GPG_PRIVATE_KEY` | Private GPG key for signing | `gpg --armor --export-secret-keys packages@wgo.sh` |
| `GPG_PASSPHRASE` | GPG key passphrase | Set when creating GPG key |
| `HOMEBREW_TAP_GITHUB_TOKEN` | GitHub token for Homebrew tap | GitHub → Settings → Developer settings → Personal access tokens |
| `SCOOP_GITHUB_TOKEN` | GitHub token for Scoop bucket | Same as above |
| `DOCKER_USERNAME` | Docker Hub username | Your Docker Hub username |
| `DOCKER_PASSWORD` | Docker Hub password/token | Your Docker Hub password or access token |

## Release Types

### Major Release (x.0.0)
- Breaking changes
- New major features
- Significant architecture changes

### Minor Release (x.y.0)
- New features
- Enhancements
- Non-breaking changes

### Patch Release (x.y.z)
- Bug fixes
- Security updates
- Documentation updates

## Pre-Release Checklist

### 1. Code Preparation
- [ ] All features implemented and tested
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version bumped in relevant files
- [ ] All tests passing
- [ ] Code review completed

### 2. Testing
- [ ] Unit tests pass: `go test ./...`
- [ ] Integration tests pass
- [ ] Manual testing on target platforms
- [ ] Package building works locally

### 3. Documentation
- [ ] README.md updated
- [ ] API documentation current
- [ ] Installation instructions verified
- [ ] Examples working

## Release Process

### Step 1: Prepare Release Branch
```bash
# Create release branch
git checkout -b release/v1.2.3
git push -u origin release/v1.2.3
```

### Step 2: Update Version Information
```bash
# Update version in files that need it
# This is typically handled by the build process
```

### Step 3: Create Release Tag
```bash
# Create and push tag
git tag v1.2.3
git push origin v1.2.3
```

### Step 4: Automated Release
The GitHub Actions workflow (`.github/workflows/release.yml`) will automatically:

1. **Validate** the release
   - Run all tests
   - Check code quality
   - Verify tag format

2. **Build** binaries and packages
   - Multi-platform Go binaries
   - Linux packages (deb, rpm, snap)
   - Docker images

3. **Sign** packages
   - GPG signing for all packages
   - Checksum generation

4. **Publish** to distribution channels
   - GitHub Releases
   - Docker Hub
   - Homebrew tap
   - Scoop bucket

5. **Validate** release
   - Download and test binaries
   - Verify installation methods

### Step 5: Post-Release Tasks
1. **Update package repositories**
   - APT repository (if set up)
   - YUM repository (if set up)

2. **Announce release**
   - Twitter/X (automated)
   - Discord/Slack channels
   - Blog post (if applicable)

3. **Monitor for issues**
   - GitHub Issues
   - Package manager feedback
   - Installation reports

## Manual Release (Emergency)

If automated release fails, you can create a manual release:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Create release
goreleaser release --clean
```

## Package Repository Management

### APT Repository
```bash
# Set up APT repository
./packaging/apt/setup-repo.sh

# Build individual packages
./packaging/apt/build-deb.sh 1.2.3 amd64
```

### YUM Repository
```bash
# Set up YUM repository
./packaging/yum/setup-repo.sh

# Build individual packages
./packaging/yum/build-rpm.sh 1.2.3 x86_64
```

### Homebrew Tap
```bash
# Create tap repository
gh repo create yairfalse/homebrew-wgo --public
```

### Scoop Bucket
```bash
# Create bucket repository
gh repo create yairfalse/scoop-wgo --public
```

## Troubleshooting

### Common Issues

**Release workflow fails**
- Check GitHub Actions logs
- Verify all secrets are set
- Ensure GPG key is valid

**Package signing fails**
- Verify GPG key setup
- Check GPG_PRIVATE_KEY secret
- Ensure GPG_PASSPHRASE is correct

**Docker build fails**
- Check Dockerfile syntax
- Verify Docker credentials
- Test local Docker build

**Package installation fails**
- Test package locally
- Check dependencies
- Verify package metadata

### Debug Commands

```bash
# Test local build
go build -o wgo ./cmd/wgo

# Test package building
./packaging/apt/build-deb.sh test amd64
./packaging/yum/build-rpm.sh test x86_64

# Test Docker build
docker build -t wgo:test .

# Test installation script
bash scripts/setup/install.sh
```

## Release Validation

### Automated Validation
The release pipeline automatically validates:
- Binary functionality on all platforms
- Package installation
- Docker image execution
- Checksum verification

### Manual Validation
After release, manually verify:
- [ ] GitHub release page looks correct
- [ ] All binary downloads work
- [ ] Package manager installation works
- [ ] Docker image pulls and runs
- [ ] Documentation links work

## Rollback Process

If a release has critical issues:

1. **Immediate**: Remove from package managers
2. **Create hotfix**: Fix issues in hotfix branch
3. **Emergency release**: Create patch release
4. **Communicate**: Notify users about the issue

## Release Schedule

- **Major releases**: Quarterly
- **Minor releases**: Monthly
- **Patch releases**: As needed (bugs, security)

## Security Considerations

- All packages are signed with GPG
- Checksums provided for all downloads
- HTTPS used for all distribution channels
- Regular security audits
- Dependency updates

## Metrics and Monitoring

Track these metrics post-release:
- Download counts
- Installation success rates
- Issue reports
- Performance metrics

## Documentation Updates

After each release:
- [ ] Update installation instructions
- [ ] Update version numbers in docs
- [ ] Update changelog
- [ ] Update API documentation

## Contact

For questions about the release process:
- Create an issue in the GitHub repository
- Contact the maintainers
- Check the troubleshooting section

---

*This document is automatically updated with each release. Last updated: $(date)*