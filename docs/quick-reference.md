# VAINO Quick Reference

Essential commands and patterns for daily use.

## 🚀 Installation

```bash
# One-line install
curl -sSL https://install.vaino.sh | bash

# Homebrew (macOS)
brew install yairfalse/vaino/vaino

# Go install
go install github.com/yairfalse/vaino/cmd/vaino@latest
```

## ⚡ Most Common Commands

```bash
# Scan current infrastructure
vaino scan

# See what changed
vaino diff

# Check system status
vaino status

# Get help
vaino --help
```

## 📊 Scanning Infrastructure

### Basic Scans
```bash
vaino scan                                    # Auto-detect everything
vaino scan --provider terraform              # Scan Terraform only
vaino scan --provider aws --region us-east-1 # Scan AWS in specific region
vaino scan --provider gcp --project my-proj  # Scan GCP project
vaino scan --provider kubernetes             # Scan Kubernetes cluster
```

### Advanced Scans
```bash
vaino scan --all                              # Scan all providers
vaino scan --output-file state.json          # Save to file
vaino scan --snapshot-name "pre-deploy"      # Custom snapshot name
vaino scan --quiet                           # Silent mode for scripts
```

## 🔍 Detecting Changes

### Basic Diff
```bash
vaino diff                          # Compare with last scan
vaino diff snapshot1 snapshot2      # Compare specific snapshots
vaino diff --stat                   # Show summary only
vaino diff --quiet                  # Silent (exit code indicates changes)
```

### Filtered Diff
```bash
vaino diff --severity high          # Only high-severity changes
vaino diff --format json           # JSON output
vaino diff --baseline prod-v1       # Compare against baseline
```

## 📋 Baseline Management

```bash
vaino baseline create --name "prod-v1.0"     # Create baseline
vaino baseline list                          # List all baselines
vaino baseline show prod-v1.0                # Show baseline details
vaino baseline delete old-baseline           # Delete baseline
```

## 🛠️ Configuration

```bash
vaino configure                     # Interactive setup
vaino configure aws                 # Configure AWS provider
vaino check-config                  # Validate configuration
vaino status --provider aws         # Check AWS connectivity
```

## 📁 Provider-Specific Examples

### Terraform
```bash
vaino scan --provider terraform --path ./infrastructure
vaino scan --provider terraform --state-file terraform.tfstate
```

### AWS
```bash
vaino scan --provider aws --region us-east-1,us-west-2
vaino scan --provider aws --profile production
```

### GCP
```bash
vaino scan --provider gcp --project my-project-123
vaino scan --provider gcp --region us-central1,us-east1
```

### Kubernetes
```bash
vaino scan --provider kubernetes --context production
vaino scan --provider kubernetes --namespace default,monitoring
```

## 🔄 Automation & CI/CD

### Basic Automation
```bash
# Check for changes (exit code indicates drift)
vaino diff --quiet && echo "No changes" || echo "Drift detected"

# Daily monitoring
vaino scan --snapshot-name "daily-$(date +%Y%m%d)"
```

### CI/CD Pipeline
```bash
# Scan and save results
vaino scan --provider terraform --output-file scan-results.json

# Compare against baseline
vaino diff --baseline production-baseline --format json > diff-report.json

# Fail build if high-severity drift
vaino diff --severity high --quiet || exit 1
```

## 📊 Output Formats

```bash
vaino scan --format table           # Human-readable table (default)
vaino scan --format json            # Machine-readable JSON
vaino scan --format yaml            # YAML format
vaino scan --format markdown        # Markdown table
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
vaino scan --quiet
if ! vaino diff --quiet; then
  echo "⚠️ Infrastructure drift detected!"
  vaino diff --format markdown | mail -s "Drift Alert" team@company.com
fi
```

### Pre-Deployment Check
```bash
#!/bin/bash
echo "Checking infrastructure before deployment..."
vaino scan --snapshot-name "pre-deploy-$(date +%Y%m%d-%H%M)"
vaino diff --baseline production-baseline --severity high --quiet
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
vaino scan --all --output-file "backup-$(date +%Y%m%d).json"

# Create baseline for current state
vaino baseline create --name "backup-$(date +%Y%m%d)" \
  --description "Automated backup of infrastructure state"
```

### Multi-Environment Monitoring
```bash
#!/bin/bash
for env in prod staging dev; do
  echo "Checking $env environment..."
  vaino scan --provider terraform --path "./environments/$env" \
    --snapshot-name "$env-$(date +%Y%m%d)"
  
  vaino diff --baseline "$env-baseline" --quiet || \
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
vaino completion bash > /etc/bash_completion.d/wgo
vaino completion zsh > /usr/local/share/zsh/site-functions/_wgo
vaino completion fish > ~/.config/fish/completions/wgo.fish
```

## 🆘 Troubleshooting

### Common Issues
```bash
# Check configuration
vaino check-config

# Verify authentication
vaino status --provider aws
vaino status --provider gcp

# Debug mode
vaino --debug scan

# Verbose output
vaino --verbose diff
```

### Reset Configuration
```bash
# Remove all configuration
rm -rf ~/.wgo

# Reconfigure
vaino configure
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
   vaino scan --snapshot-name "before-major-change"
   ```

2. **Combine with jq for filtering**:
   ```bash
   vaino scan --format json | jq '.resources[] | select(.type == "aws_instance")'
   ```

3. **Set up aliases**:
   ```bash
   alias wgo-prod='vaino --config ~/.wgo/prod-config.yaml'
   alias wgo-staging='vaino --config ~/.wgo/staging-config.yaml'
   ```

4. **Use baselines for environments**:
   ```bash
   vaino baseline create --name "prod-baseline"
   vaino baseline create --name "staging-baseline"
   ```

5. **Monitor with watch**:
   ```bash
   vaino watch --interval 30s --provider terraform
   ```

---

**📚 More Info:**
- [Full Documentation](getting-started.md)
- [Installation Guide](installation.md)
- [Configuration Reference](configuration.md)
- [Command Reference](commands.md)