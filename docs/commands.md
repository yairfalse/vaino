# Commands Reference

Complete reference for all WGO commands with examples and use cases.

## Core Commands

### `wgo scan`

Scan and capture the current state of your infrastructure.

**Basic Usage:**
```bash
wgo scan [flags]
```

**Examples:**
```bash
# Auto-discover and scan all providers
wgo scan

# Scan specific provider
wgo scan --provider terraform
wgo scan --provider aws --region us-east-1
wgo scan --provider gcp --project my-project-123
wgo scan --provider kubernetes --namespace default

# Scan with custom output
wgo scan --output-file infrastructure-state.json
wgo scan --snapshot-name "pre-deployment-$(date +%Y%m%d)"

# Scan specific Terraform state files
wgo scan --provider terraform --state-file ./terraform.tfstate
wgo scan --provider terraform --path ./infrastructure

# Scan multiple AWS regions
wgo scan --provider aws --region us-east-1,us-west-2,eu-west-1

# Scan Kubernetes with specific context
wgo scan --provider kubernetes --context production --namespace default,monitoring

# Silent scan for automation
wgo scan --quiet --output-file scan-results.json
```

**Flags:**
- `--provider` - Infrastructure provider (terraform, aws, gcp, kubernetes)
- `--output-file` - Save snapshot to file
- `--snapshot-name` - Custom name for the snapshot
- `--region` - Regions to scan (comma-separated)
- `--path` - Path to Terraform files
- `--state-file` - Specific Terraform state files
- `--project` - GCP project ID
- `--context` - Kubernetes context
- `--namespace` - Kubernetes namespaces
- `--quiet` - Suppress output
- `--all` - Scan all configured providers

---

### `wgo diff`

Compare infrastructure states to detect changes.

**Basic Usage:**
```bash
wgo diff [flags] [snapshot1] [snapshot2]
```

**Examples:**
```bash
# Compare current state with last scan
wgo diff

# Compare two specific snapshots
wgo diff snapshot-1 snapshot-2

# Show only statistics
wgo diff --stat

# Export comparison to file
wgo diff --output diff-report.json --format json

# Compare with specific baseline
wgo diff --baseline production-baseline

# Show changes in different formats
wgo diff --format table
wgo diff --format json
wgo diff --format markdown

# Filter by severity
wgo diff --severity high
wgo diff --severity medium,high

# Filter by resource type
wgo diff --resource-type aws_instance,aws_s3_bucket

# Compare across time ranges
wgo diff --from "2024-01-01" --to "2024-01-15"

# Quiet mode for scripts (exit code indicates changes)
wgo diff --quiet
echo $? # 0 = no changes, 1 = changes detected
```

**Flags:**
- `--baseline` - Compare against named baseline
- `--format` - Output format (table, json, yaml, markdown)
- `--output` - Save comparison to file
- `--stat` - Show only statistics
- `--severity` - Filter by severity level
- `--resource-type` - Filter by resource types
- `--from` - Start date for comparison
- `--to` - End date for comparison
- `--quiet` - Silent mode (exit codes only)

---

### `wgo status`

Show WGO system status and provider connectivity.

**Basic Usage:**
```bash
wgo status [flags]
```

**Examples:**
```bash
# Check overall status
wgo status

# Check specific provider
wgo status --provider aws
wgo status --provider gcp
wgo status --provider kubernetes

# Detailed status with configuration
wgo status --verbose

# Check authentication for all providers
wgo status --check-auth

# Machine-readable status
wgo status --format json
```

**Output Example:**
```
WGO System Status
================
Version: v1.0.0
Config:  ~/.wgo/config.yaml

Provider Status:
┌─────────────┬─────────┬──────────────────────┬─────────────┐
│ Provider    │ Status  │ Configuration        │ Last Scan   │
├─────────────┼─────────┼──────────────────────┼─────────────┤
│ terraform   │ ✅ Ready │ 3 state files found  │ 2 hours ago │
│ aws         │ ✅ Ready │ us-east-1 (default)  │ Never       │
│ gcp         │ ❌ Error │ Project not set      │ Never       │
│ kubernetes  │ ⚠️ Warn  │ No context selected  │ Never       │
└─────────────┴─────────┴──────────────────────┴─────────────┘

Storage: ~/.wgo (12 snapshots, 45MB)
Cache:   Enabled (3 entries, 2MB)
```

---

### `wgo configure`

Interactive configuration wizard.

**Basic Usage:**
```bash
wgo configure [provider]
```

**Examples:**
```bash
# General configuration wizard
wgo configure

# Provider-specific configuration
wgo configure aws
wgo configure gcp
wgo configure kubernetes
wgo configure terraform

# Show current configuration
wgo configure --show

# Validate configuration
wgo configure --validate
```

**Interactive Example:**
```
$ wgo configure gcp

🔧 GCP Configuration
==================

Current GCP project: (not set)
Enter GCP project ID: my-project-123

Current credentials: (not set)
Enter path to service account JSON: /path/to/service-account.json

Select regions to scan:
[✓] us-central1
[✓] us-east1
[ ] europe-west1
[ ] asia-southeast1

Configuration saved to ~/.wgo/config.yaml
Run 'wgo status --provider gcp' to verify setup.
```

---

## Analysis Commands

### `wgo explain`

AI-powered analysis of infrastructure changes.

**Examples:**
```bash
# Explain latest changes
wgo explain

# Explain specific comparison
wgo explain snapshot-1 snapshot-2

# Get recommendations
wgo explain --recommendations

# Explain in different formats
wgo explain --format markdown
wgo explain --format json
```

**Output Example:**
```
🤖 AI Analysis of Infrastructure Changes
======================================

Summary:
Your infrastructure has 3 significant changes that require attention.

Critical Issues:
• EC2 instance type changed from t3.medium to t3.large
  - Impact: 2x increase in compute costs (~$30/month)
  - Risk: High - may indicate capacity planning issues
  - Recommendation: Verify if increased capacity is needed

• S3 bucket versioning disabled
  - Impact: Data loss protection removed
  - Risk: High - accidental deletions not recoverable  
  - Recommendation: Re-enable versioning immediately

Medium Priority:
• RDS instance added
  - Impact: New database service
  - Risk: Medium - ensure backups configured
  - Recommendation: Verify backup settings and monitoring

Next Steps:
1. Review EC2 instance sizing requirements
2. Re-enable S3 bucket versioning
3. Configure RDS backup and monitoring
4. Update infrastructure documentation
```

---

### `wgo watch`

Real-time infrastructure monitoring.

**Examples:**
```bash
# Watch for changes with default interval
wgo watch

# Watch specific provider
wgo watch --provider terraform

# Custom check interval
wgo watch --interval 30s

# Watch with notifications
wgo watch --webhook https://hooks.slack.com/services/...

# Watch and auto-create baselines
wgo watch --auto-baseline

# Watch with correlation analysis
wgo watch --correlation
```

**Output Example:**
```
🔍 WGO Watch Mode - Real-time Infrastructure Monitoring
=====================================================
Provider: terraform | Interval: 60s | Started: 2024-01-15 14:30:00

14:30:00 ✅ Scan completed - No changes detected (4 resources)
14:31:00 ✅ Scan completed - No changes detected (4 resources)
14:32:00 ⚠️  Changes detected! 1 resource modified
         📊 aws_instance.web: instance_type t3.medium → t3.large
         🔗 Correlation detected: Recent Terraform apply (2 minutes ago)
14:33:00 ✅ Scan completed - No changes detected (4 resources)

Press Ctrl+C to stop monitoring...
```

---

## Baseline Management

### `wgo baseline`

Manage infrastructure baselines.

**Subcommands:**
- `create` - Create new baseline
- `list` - List all baselines
- `show` - Show baseline details
- `delete` - Delete baseline
- `compare` - Compare baselines

**Examples:**

#### Create Baseline
```bash
# Create from current scan
wgo baseline create --name production-v1.0

# Create from specific snapshot
wgo baseline create --name staging-deploy --from snapshot-123

# Create with metadata
wgo baseline create --name "release-v2.1" \
  --description "Production release v2.1" \
  --tags "version=2.1,environment=prod"

# Auto-name with timestamp
wgo baseline create --auto-name
```

#### List Baselines
```bash
# List all baselines
wgo baseline list

# List with details
wgo baseline list --verbose

# Filter by tags
wgo baseline list --tags "environment=prod"

# Sort by date
wgo baseline list --sort date
```

**Output:**
```
Infrastructure Baselines
========================
┌─────────────────────┬─────────────┬─────────────┬──────────────┬─────────────┐
│ Name                │ Provider    │ Resources   │ Created      │ Size        │
├─────────────────────┼─────────────┼─────────────┼──────────────┼─────────────┤
│ production-v1.0     │ terraform   │ 12          │ 2 days ago   │ 2.3 MB      │
│ staging-deploy      │ terraform   │ 8           │ 1 day ago    │ 1.8 MB      │
│ aws-baseline-jan    │ aws         │ 45          │ 1 week ago   │ 5.1 MB      │
└─────────────────────┴─────────────┴─────────────┴──────────────┴─────────────┘
```

#### Show Baseline Details
```bash
# Show baseline summary
wgo baseline show production-v1.0

# Show with resource details
wgo baseline show production-v1.0 --verbose

# Export baseline
wgo baseline show production-v1.0 --format json > baseline.json
```

#### Compare Baselines
```bash
# Compare two baselines
wgo baseline compare baseline-1 baseline-2

# Compare baseline with current state
wgo baseline compare production-v1.0 current
```

---

## Utility Commands

### `wgo check-config`

Validate configuration and provider connectivity.

**Examples:**
```bash
# Check all configuration
wgo check-config

# Check specific provider
wgo check-config --provider aws

# Detailed validation
wgo check-config --verbose

# Check authentication only
wgo check-config --auth-only
```

**Output:**
```
Configuration Validation
========================

✅ Configuration file: ~/.wgo/config.yaml (valid)
✅ AWS configuration: Ready (us-east-1, default profile)
❌ GCP configuration: Error - project_id not set
⚠️  Kubernetes: Warning - no context specified

Connectivity Tests:
✅ AWS: Successfully connected to us-east-1
❌ GCP: Authentication failed
⚠️  Kubernetes: kubectl not found

Recommendations:
- Set GCP project_id in configuration
- Install kubectl for Kubernetes support
- Verify GCP credentials file path
```

---

### `wgo version`

Show version information.

**Examples:**
```bash
# Basic version
wgo version

# Detailed version info
wgo version --verbose

# JSON format
wgo version --format json
```

**Output:**
```
WGO (What's Going On) version v1.0.0
  commit: a1b2c3d
  built: 2024-01-15T10:30:00Z
  built by: goreleaser
  go version: go1.21.5

Platform: darwin/arm64
Config: ~/.wgo/config.yaml
```

---

### `wgo completion`

Generate shell completion scripts.

**Examples:**
```bash
# Generate bash completion
wgo completion bash

# Generate and install bash completion
wgo completion bash > /etc/bash_completion.d/wgo

# Generate zsh completion
wgo completion zsh > _wgo

# Generate fish completion
wgo completion fish > ~/.config/fish/completions/wgo.fish

# Generate PowerShell completion
wgo completion powershell
```

---

## Global Flags

These flags work with all commands:

```bash
--config string      # Config file path (default: ~/.wgo/config.yaml)
--debug              # Enable debug mode  
--log-level string   # Log level: debug, info, warn, error (default: info)
--no-color           # Disable colored output
--output string      # Output format: table, json, yaml, markdown (default: table)
--verbose            # Verbose output
--quiet              # Quiet mode (minimal output)
--help               # Show help
--version            # Show version
```

## Exit Codes

WGO uses standard Unix exit codes:

- `0` - Success (no changes detected for diff commands)
- `1` - Changes detected (for diff commands) or general errors
- `2` - Invalid command line arguments
- `3` - Configuration errors
- `4` - Authentication/permission errors
- `5` - Network/connectivity errors

**Example Usage in Scripts:**
```bash
#!/bin/bash
wgo diff --quiet
case $? in
  0) echo "✅ No infrastructure changes";;
  1) echo "⚠️ Infrastructure drift detected";;
  *) echo "❌ Error checking infrastructure";;
esac
```

## Advanced Usage

### Piping and Chaining

```bash
# Pipe scan results to jq
wgo scan --format json | jq '.resources[] | select(.type == "aws_instance")'

# Chain commands
wgo scan && wgo diff --quiet || echo "Changes detected!"

# Use in scripts
if wgo diff --quiet; then
  echo "No changes - deployment can proceed"
else
  echo "Drift detected - please review before deployment"
  exit 1
fi
```

### Integration Examples

```bash
# CI/CD Pipeline
wgo scan --provider terraform --quiet --output-file current-state.json
wgo baseline compare production-baseline current-state.json --format json > drift-report.json

# Monitoring Script
while true; do
  if ! wgo diff --quiet; then
    curl -X POST -H 'Content-type: application/json' \
      --data '{"text":"Infrastructure drift detected!"}' \
      $SLACK_WEBHOOK_URL
  fi
  sleep 300
done

# Backup Script
wgo scan --all --output-file "backup-$(date +%Y%m%d).json"
```

---

**Next:** [Examples →](examples/)