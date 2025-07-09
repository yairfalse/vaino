# WGO 🔍 - "What's Going On?"

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
curl -sSL https://github.com/yairfalse/wgo/releases/latest/download/install.sh | bash
```

**Go Install:**
```bash
go install github.com/yairfalse/wgo/cmd/wgo@latest
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
| AWS        | ✅ Ready | EC2, S3, VPC, Lambda, RDS |
| Kubernetes | ✅ Ready | Workloads, Services, Config |

## Documentation

**📖 Complete Documentation:**
- [Getting Started Guide](docs/getting-started.md) - Your first scan in 5 minutes
- [Installation Guide](docs/installation.md) - All installation methods
- [Configuration Reference](docs/configuration.md) - Complete configuration options
- [Commands Reference](docs/commands.md) - All commands and options
- [Troubleshooting Guide](docs/troubleshooting.md) - Common issues and solutions
- [Best Practices](docs/best-practices.md) - Production deployment guidance

**🎯 Real-world Examples:**
- [Kubernetes Monitoring](docs/examples/kubernetes-monitoring.md)
- [Multi-Cloud Setup](docs/examples/multi-cloud-setup.md)
- [Terraform Drift Detection](docs/examples/terraform-drift.md)

**⚡ Performance & Advanced Features:**
- [Concurrent Scanning](docs/concurrent-scanning.md) - High-performance scanning
- [Performance Analysis](docs/performance/) - Benchmarks and optimization
- [Unix-style Output](docs/unix-style-output-examples.md) - Integration patterns

## Quick Configuration

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

## Common Commands

### Core Commands
- `wgo scan` - Discover and scan infrastructure
- `wgo baseline create` - Save current state as baseline
- `wgo check` - Compare current state vs baseline  
- `wgo diff` - Show detailed differences
- `wgo explain` - Get AI analysis of changes

### Providers
- `--provider terraform` - Scan Terraform state files
- `--provider gcp` - Scan Google Cloud resources
- `--provider aws` - Scan AWS resources
- `--provider kubernetes` - Scan Kubernetes cluster

### Output Formats  
- `--format table` - Human-readable table (default)
- `--format json` - Machine-readable JSON
- `--format markdown` - Documentation-friendly markdown

## Contributing

We welcome contributions! See our [Contributing Guide](CONTRIBUTING.md) for details.

**Areas we need help:**
- Additional provider implementations
- Performance optimizations
- Documentation improvements
- Testing and quality assurance

## Support

- 📖 **Documentation:** [Complete docs](docs/)
- 💬 **Discussions:** [GitHub Discussions](https://github.com/yairfalse/wgo/discussions)
- 🐛 **Issues:** [GitHub Issues](https://github.com/yairfalse/wgo/issues)
- 📧 **Email:** support@wgo.sh

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Made with ❤️ for DevOps and SRE teams who care about infrastructure reliability.**