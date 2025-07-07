# WGO 🔍 - "Whats going on?"

**AI-powered infrastructure drift detection made simple**

WGO automatically detects when your infrastructure drifts from its intended state across Terraform, AWS, GCP, and Kubernetes. Get instant visibility into what changed, why it matters, and how to fix it.

```bash
# Scan your infrastructure
wgo scan

# Create a baseline
wgo baseline create --name prod

# Check for drift  
wgo check --baseline prod
```

## Why WGO?

**Infrastructure drift happens.** Manual changes, failed deployments, configuration updates outside of IaC - your infrastructure slowly drifts from what you think it should be.

WGO helps you:
- 🔍 **Detect drift automatically** across multiple providers
- 📊 **Understand what changed** with clear before/after comparisons  
- ⚡ **Fix issues fast** with actionable recommendations
- 🛡️ **Prevent security gaps** from configuration drift
- 📈 **Track changes over time** with baseline management

## Quick Start

### 1. Install WGO

**macOS/Linux (Homebrew):**
```bash
brew install wgo
```

**Direct Download:**
```bash
curl -sSL https://github.com/yourusername/wgo/releases/latest/download/install.sh | bash
```

**Go Install:**
```bash
go install github.com/yourusername/wgo/cmd/wgo@latest
```

### 2. Scan Your Infrastructure

**Auto-detect and scan everything:**
```bash
wgo scan
```

**Scan specific providers:**
```bash
wgo scan --provider terraform
wgo scan --provider gcp --project my-project
wgo scan --provider aws --regions us-east-1,us-west-2
```

### 3. Create Baselines & Check for Drift

```bash
# Save current state as baseline
wgo baseline create --name production

# Check for any changes
wgo check --baseline production

# See detailed differences
wgo diff --baseline production --format table
```

## Example Output

```bash
$ wgo scan --provider terraform
🔍 Infrastructure Scan
=====================
✅ Found 3 Terraform state file(s)
📊 Resources found: 12
📈 Resource breakdown:
  • aws_instance: 6
  • aws_s3_bucket: 2  
  • aws_vpc: 1
  • aws_security_group: 1
  • aws_rds_instance: 1
  • aws_lambda_function: 1

$ wgo check --baseline production
⚠️  Infrastructure Drift Detected
==================================
📊 Drift Summary: 3 changes detected
🔴 Critical: 1 change
🟡 Medium: 2 changes

Changes:
┌─────────────────────┬──────────┬──────────┬─────────────────┐
│ Resource            │ Change   │ Severity │ Impact          │
├─────────────────────┼──────────┼──────────┼─────────────────┤
│ aws_instance.web    │ Modified │ HIGH     │ Size: t3.medium │
│                     │          │          │ → t3.large      │
├─────────────────────┼──────────┼──────────┼─────────────────┤
│ aws_s3_bucket.logs  │ Modified │ MEDIUM   │ Public access   │
│                     │          │          │ enabled         │
└─────────────────────┴──────────┴──────────┴─────────────────┘

🎯 Run 'wgo explain' for AI-powered analysis and recommendations
```

## Supported Providers

| Provider   | Status | Resources |
|------------|--------|-----------|
| Terraform  | ✅ Ready | All state file resources |  
| GCP        | ✅ Ready | Compute, Storage, Networking, IAM |
| AWS        | 🔄 Coming Soon | EC2, S3, VPC, Lambda, RDS |
| Kubernetes | 🔄 Coming Soon | Workloads, Services, Config |

## Configuration

WGO works out of the box with zero configuration. For advanced usage:

**~/.wgo/config.yaml:**
```yaml
# GCP Configuration
gcp:
  project_id: "my-project-123"
  regions: ["us-central1", "us-east1"]

# Storage Location  
storage:
  base_path: "~/.wgo"

# Output Preferences
output:
  format: "table"  # table, json, markdown
  color: true
```

## Common Workflows

### Daily Drift Monitoring
```bash
# Morning infrastructure health check
wgo scan
wgo check --baseline yesterday
```

### Team Collaboration
```bash
# Share baselines across team
wgo baseline create --name "release-v1.2.0"
wgo baseline list
wgo check --baseline "release-v1.2.0" --format json > drift-report.json
```

### CI/CD Integration
```bash
# In your deployment pipeline
wgo scan --provider terraform
wgo baseline create --name "pre-deploy-$(date +%Y%m%d)"
# ... deploy changes ...
wgo check --baseline "pre-deploy-$(date +%Y%m%d)" --format json
```

## Commands Reference

### Core Commands
- `wgo scan` - Discover and scan infrastructure
- `wgo baseline create` - Save current state as baseline
- `wgo check` - Compare current state vs baseline  
- `wgo diff` - Show detailed differences
- `wgo explain` - Get AI analysis of changes

### Providers
- `--provider terraform` - Scan Terraform state files
- `--provider gcp` - Scan Google Cloud resources
- `--provider aws` - Scan AWS resources (coming soon)
- `--provider kubernetes` - Scan Kubernetes cluster (coming soon)

### Output Formats  
- `--format table` - Human-readable table (default)
- `--format json` - Machine-readable JSON
- `--format markdown` - Documentation-friendly markdown

### Baseline Management
- `wgo baseline list` - Show all saved baselines
- `wgo baseline delete --name <name>` - Remove baseline
- `wgo baseline show --name <name>` - View baseline details

## Installation Options

### Package Managers

**Homebrew (macOS/Linux):**
```bash
brew install wgo
```

**Chocolatey (Windows):**
```bash
choco install wgo
```

**Scoop (Windows):**
```bash
scoop bucket add wgo https://github.com/yourusername/scoop-wgo
scoop install wgo
```

### Direct Downloads

Download binaries from [GitHub Releases](https://github.com/yourusername/wgo/releases)

**Linux/macOS:**
```bash
curl -sSL https://github.com/yourusername/wgo/releases/latest/download/install.sh | bash
```

**Manual Installation:**
1. Download binary for your platform
2. Extract and place in PATH
3. Run `wgo --help` to verify

### Docker
```bash
docker run --rm -v $(pwd):/workspace wgo/wgo:latest scan
```

## Authentication

### GCP
```bash
# Install gcloud CLI
brew install google-cloud-sdk

# Authenticate
gcloud auth login
gcloud auth application-default login
gcloud config set project your-project-id
```

### AWS (Coming Soon)
```bash
# Configure AWS CLI
aws configure
# or use IAM roles, environment variables
```

### Kubernetes (Coming Soon)
```bash
# Uses your current kubectl context
kubectl config current-context
```

## Contributing

We welcome contributions! See our [Contributing Guide](CONTRIBUTING.md) for details.

**Areas we need help:**
- AWS provider implementation
- Kubernetes provider enhancement  
- Additional output formats
- Performance optimizations
- Documentation improvements

## Support

- 📖 **Documentation:** [docs.wgo.sh](https://docs.wgo.sh)
- 💬 **Discussions:** [GitHub Discussions](https://github.com/yourusername/wgo/discussions)
- 🐛 **Issues:** [GitHub Issues](https://github.com/yourusername/wgo/issues)
- 📧 **Email:** support@wgo.sh

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Made with ❤️ for DevOps and SRE teams who care about infrastructure reliability.**
