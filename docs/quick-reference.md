# VAINO Quick Reference

Essential commands and patterns for daily use.

## 🚀 Installation

```bash
# One-line install
curl -sSL https://install.wgo.sh | bash

# Homebrew (macOS)
brew install yairfalse/wgo/wgo

# Go install
go install github.com/yairfalse/wgo/cmd/wgo@latest
```

## ⚡ Most Common Commands

```bash
# Scan current infrastructure
wgo scan

# See what changed
wgo diff

# Check system status
wgo status

# Get help
wgo --help
```

## 📊 Scanning Infrastructure

### Basic Scans
```bash
wgo scan                                    # Auto-detect everything
wgo scan --provider terraform              # Scan Terraform only
wgo scan --provider aws --region us-east-1 # Scan AWS in specific region
wgo scan --provider gcp --project my-proj  # Scan GCP project
wgo scan --provider kubernetes             # Scan Kubernetes cluster
```

### Advanced Scans
```bash
wgo scan --all                              # Scan all providers
wgo scan --output-file state.json          # Save to file
wgo scan --snapshot-name "pre-deploy"      # Custom snapshot name
wgo scan --quiet                           # Silent mode for scripts
```

## 🔍 Detecting Changes

### Basic Diff
```bash
wgo diff                          # Compare with last scan
wgo diff snapshot1 snapshot2      # Compare specific snapshots
wgo diff --stat                   # Show summary only
wgo diff --quiet                  # Silent (exit code indicates changes)
```

### Filtered Diff
```bash
wgo diff --severity high          # Only high-severity changes
wgo diff --format json           # JSON output
wgo diff --baseline prod-v1       # Compare against baseline
```

## 📋 Baseline Management

```bash
wgo baseline create --name "prod-v1.0"     # Create baseline
wgo baseline list                          # List all baselines
wgo baseline show prod-v1.0                # Show baseline details
wgo baseline delete old-baseline           # Delete baseline
```

## 🛠️ Configuration

```bash
wgo configure                     # Interactive setup
wgo configure aws                 # Configure AWS provider
wgo check-config                  # Validate configuration
wgo status --provider aws         # Check AWS connectivity
```

## 📁 Provider-Specific Examples

### Terraform
```bash
wgo scan --provider terraform --path ./infrastructure
wgo scan --provider terraform --state-file terraform.tfstate
```

### AWS
```bash
wgo scan --provider aws --region us-east-1,us-west-2
wgo scan --provider aws --profile production
```

### GCP
```bash
wgo scan --provider gcp --project my-project-123
wgo scan --provider gcp --region us-central1,us-east1
```

### Kubernetes
```bash
wgo scan --provider kubernetes --context production
wgo scan --provider kubernetes --namespace default,monitoring
```

## 🔄 Automation & CI/CD

### Basic Automation
```bash
# Check for changes (exit code indicates drift)
wgo diff --quiet && echo "No changes" || echo "Drift detected"

# Daily monitoring
wgo scan --snapshot-name "daily-$(date +%Y%m%d)"
```

### CI/CD Pipeline
```bash
# Scan and save results
wgo scan --provider terraform --output-file scan-results.json

# Compare against baseline
wgo diff --baseline production-baseline --format json > diff-report.json

# Fail build if high-severity drift
wgo diff --severity high --quiet || exit 1
```

## 📊 Output Formats

```bash
wgo scan --format table           # Human-readable table (default)
wgo scan --format json            # Machine-readable JSON
wgo scan --format yaml            # YAML format
wgo scan --format markdown        # Markdown table
```

## 🎯 Exit Codes

- `0` = Success (no changes for diff)
- `1` = Changes detected or general error
- `2` = Invalid arguments
- `3` = Configuration error
- `4` = Authentication error

## 🔧 Common Patterns

### Daily Drift Check
```bash
#!/bin/bash
wgo scan --quiet
if ! wgo diff --quiet; then
  echo "⚠️ Infrastructure drift detected!"
  wgo diff --format markdown | mail -s "Drift Alert" team@company.com
fi
```

### Pre-Deployment Check
```bash
#!/bin/bash
echo "Checking infrastructure before deployment..."
wgo scan --snapshot-name "pre-deploy-$(date +%Y%m%d-%H%M)"
wgo diff --baseline production-baseline --severity high --quiet
if [ $? -eq 1 ]; then
  echo "❌ High-severity drift detected. Please review before deploying."
  exit 1
fi
echo "✅ No critical drift detected. Safe to deploy."
```

### Backup Current State
```bash
#!/bin/bash
# Create dated backup
wgo scan --all --output-file "backup-$(date +%Y%m%d).json"

# Create baseline for current state
wgo baseline create --name "backup-$(date +%Y%m%d)" \
  --description "Automated backup of infrastructure state"
```

### Multi-Environment Monitoring
```bash
#!/bin/bash
for env in prod staging dev; do
  echo "Checking $env environment..."
  wgo scan --provider terraform --path "./environments/$env" \
    --snapshot-name "$env-$(date +%Y%m%d)"
  
  wgo diff --baseline "$env-baseline" --quiet || \
    echo "⚠️ Drift in $env environment"
done
```

## 🐳 Docker Usage

```bash
# Basic scan
docker run --rm -v $(pwd):/workspace yairfalse/wgo:latest scan

# With AWS credentials
docker run --rm \
  -v $(pwd):/workspace \
  -v ~/.aws:/home/wgo/.aws:ro \
  yairfalse/wgo:latest scan --provider aws

# Save results
docker run --rm -v $(pwd):/workspace yairfalse/wgo:latest \
  scan --output-file scan-results.json
```

## 📱 Shell Completions

```bash
# Install completions
wgo completion bash > /etc/bash_completion.d/wgo
wgo completion zsh > /usr/local/share/zsh/site-functions/_wgo
wgo completion fish > ~/.config/fish/completions/wgo.fish
```

## 🆘 Troubleshooting

### Common Issues
```bash
# Check configuration
wgo check-config

# Verify authentication
wgo status --provider aws
wgo status --provider gcp

# Debug mode
wgo --debug scan

# Verbose output
wgo --verbose diff
```

### Reset Configuration
```bash
# Remove all configuration
rm -rf ~/.wgo

# Reconfigure
wgo configure
```

## 🔗 Environment Variables

```bash
# AWS
export AWS_PROFILE=production
export AWS_REGION=us-east-1

# GCP
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
export GOOGLE_CLOUD_PROJECT=my-project

# Kubernetes
export KUBECONFIG=~/.kube/config

# VAINO
export VAINO_CONFIG=/path/to/config.yaml
export VAINO_LOG_LEVEL=debug
```

## 📖 File Locations

```bash
~/.wgo/config.yaml              # Main configuration
~/.wgo/snapshots/               # Stored snapshots
~/.wgo/baselines/               # Baseline files
~/.wgo/cache/                   # Cache directory
~/.wgo/logs/                    # Log files
```

## 💡 Pro Tips

1. **Use snapshots for rollback points**:
   ```bash
   wgo scan --snapshot-name "before-major-change"
   ```

2. **Combine with jq for filtering**:
   ```bash
   wgo scan --format json | jq '.resources[] | select(.type == "aws_instance")'
   ```

3. **Set up aliases**:
   ```bash
   alias wgo-prod='wgo --config ~/.wgo/prod-config.yaml'
   alias wgo-staging='wgo --config ~/.wgo/staging-config.yaml'
   ```

4. **Use baselines for environments**:
   ```bash
   wgo baseline create --name "prod-baseline"
   wgo baseline create --name "staging-baseline"
   ```

5. **Monitor with watch**:
   ```bash
   wgo watch --interval 30s --provider terraform
   ```

---

**📚 More Info:**
- [Full Documentation](getting-started.md)
- [Installation Guide](installation.md)
- [Configuration Reference](configuration.md)
- [Command Reference](commands.md)