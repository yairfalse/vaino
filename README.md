# VAINO

**Infrastructure drift detection and monitoring tool**

*Named after Väinö from Finnish mythology*

[![Build Status](https://github.com/yairfalse/vaino/workflows/CI/badge.svg)](https://github.com/yairfalse/vaino/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/yairfalse/vaino)](https://goreportcard.com/report/github.com/yairfalse/vaino)

VAINO helps you detect and monitor infrastructure drift across multiple cloud providers and Infrastructure as Code tools. Think of it as "git diff" for your infrastructure - track changes over time and quickly identify what's different between deployments.

## Features

- **Multi-Provider Support**: AWS, GCP, Kubernetes, Terraform
- **Drift Detection**: Compare infrastructure states over time
- **Unix-Style Output**: Clean, scriptable output for automation
- **Multiple Formats**: JSON, YAML, table, and markdown output
- **Real-time Monitoring**: Watch for changes as they happen
- **CI/CD Integration**: Perfect for automated infrastructure validation

## Quick Start

### Installation

```bash
# Universal installer
curl -sSL https://install.vaino.sh | bash

# Package managers
brew install yairfalse/vaino/vaino  # macOS
sudo apt install vaino              # Debian/Ubuntu
sudo dnf install vaino              # Red Hat/Fedora
```

### Basic Usage

```bash
# Scan your infrastructure
vaino scan

# Check for changes
vaino diff

# Get summary statistics
vaino diff --stat

# Monitor continuously
vaino watch
```

## Core Commands

### Scanning

```bash
vaino scan                        # Auto-discover and scan all providers
vaino scan --provider aws         # Scan AWS resources
vaino scan --provider kubernetes  # Scan Kubernetes cluster
vaino scan --provider terraform   # Scan Terraform state
```

### Drift Detection

```bash
vaino diff                        # Show changes since last scan
vaino diff --stat                 # Show change statistics
vaino diff --baseline production  # Compare against named baseline
vaino diff --quiet                # Silent mode (exit code only)
```

### Continuous Monitoring

```bash
vaino watch                       # Monitor for changes
vaino watch --interval 30s        # Custom check interval
vaino check                       # One-time drift check
```

## Supported Providers

| Provider | Resources | Notes |
|----------|-----------|-------|
| **AWS** | EC2, S3, RDS, Lambda, IAM | Requires AWS CLI configuration |
| **Kubernetes** | Pods, services, deployments | Uses current kubectl context |
| **Terraform** | State files, plans | Supports local and remote state |
| **GCP** | Compute, storage, networking | Requires gcloud authentication |

## Configuration

### Config File: `~/.vaino/config.yaml`

```yaml
providers:
  aws:
    regions: ["us-east-1", "us-west-2"]
    profile: "production"
  
  kubernetes:
    contexts: ["production", "staging"]
    
  terraform:
    state_paths: ["./infrastructure/"]

output:
  format: "table"
  no_color: false

storage:
  base_path: "~/.vaino/snapshots"
  retention_days: 30
```

### Environment Variables

```bash
# Provider authentication
export AWS_PROFILE=production
export KUBECONFIG=~/.kube/config

# VAINO settings
export VAINO_CONFIG=~/.vaino/config.yaml
export VAINO_VERBOSE=true
```

## Output Formats

VAINO supports multiple output formats for different use cases:

```bash
vaino diff --output table      # Human-readable table (default)
vaino diff --output json       # Machine-readable JSON
vaino diff --output yaml       # YAML format
vaino diff --output markdown   # Markdown for documentation
```

## Real-World Examples

### Daily Infrastructure Check

```bash
#!/bin/bash
# Daily infrastructure monitoring script

vaino scan
if vaino diff --quiet; then
    echo "✅ No infrastructure drift detected"
else
    echo "⚠️ Infrastructure changes detected:"
    vaino diff --stat
    vaino diff --output markdown > changes.md
fi
```

### CI/CD Integration

```yaml
# .github/workflows/infrastructure-check.yml
name: Infrastructure Drift Check

on:
  schedule:
    - cron: "0 8 * * *"  # Daily at 8 AM

jobs:
  check-drift:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install VAINO
        run: curl -sSL https://install.vaino.sh | bash
      
      - name: Scan Infrastructure
        run: vaino scan --output json > current-state.json
        
      - name: Check for Drift
        run: |
          if vaino diff --quiet; then
            echo "✅ No infrastructure drift"
          else
            echo "⚠️ Drift detected!"
            vaino diff --output markdown >> $GITHUB_STEP_SUMMARY
          fi
```

### Terraform Workflow

```bash
# Before applying changes
terraform plan -out=plan.tfplan
vaino scan --provider terraform

# Apply changes
terraform apply plan.tfplan

# Verify changes
vaino scan --provider terraform
vaino diff --provider terraform
```

## Security

VAINO is designed with security in mind:

- **Read-only access**: Never modifies your infrastructure
- **Credential respect**: Uses existing provider authentication
- **Secret filtering**: Automatically excludes sensitive data
- **Local storage**: Snapshots stored locally by default

## Documentation

- [Installation Guide](./docs/INSTALLATION.md) - Detailed installation instructions
- [Configuration Reference](./docs/configuration.md) - Complete configuration options
- [Provider Setup](./docs/providers/) - Provider-specific setup guides
- [Examples](./docs/examples/) - Real-world usage examples
- [Troubleshooting](./docs/troubleshooting.md) - Common issues and solutions

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

## Support

- **Issues**: [GitHub Issues](https://github.com/yairfalse/vaino/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yairfalse/vaino/discussions)
- **Documentation**: [docs/](./docs/)

## License

VAINO is released under the [MIT License](./LICENSE).

---

**VAINO** - Infrastructure drift detection made simple.
