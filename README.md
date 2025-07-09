# WGO 🛡️

**What's Going On with your infrastructure?**

> Simple infrastructure scanning and drift detection across Terraform, AWS, and Kubernetes. Know what's actually running vs what you think is running.

---

## 🔥 The Problem

Infrastructure drift is the **#1 pain point** for DevOps and SRE teams in 2024:

- **33% of SRE time** is spent on toil caused by configuration drift
- **Manual changes** in cloud consoles break infrastructure-as-code workflows
- **Security vulnerabilities** emerge from unexpected configuration changes
- Teams managing **dozens of environments** struggle to maintain consistency
- **Alert fatigue** from reactive monitoring instead of proactive detection

> *"Configuration drift can impact system stability, performance, and security... lead to unexpected software behavior, compatibility issues, and security vulnerabilities"* - Industry Research 2024

## ✨ The Solution

WGO provides **simple infrastructure visibility** with:

🔍 **Clear Infrastructure Scanning** - See what's actually deployed across your environments  
🌐 **Multi-Provider Support** - Terraform, AWS, and Kubernetes in one tool  
⚡ **Fast Scanning** - Smart caching for quick infrastructure snapshots  
🛡️ **Drift Detection** - Compare actual state with your Terraform configurations  
📊 **Clean Reporting** - Multiple output formats that are easy to understand  
🎯 **No Complexity** - Simple commands, clear results  

---

## 🚀 Quick Start

### Installation

```bash
# Download latest release
curl -L https://github.com/yairfalse/wgo/releases/latest/download/wgo-linux-amd64.tar.gz | tar xz
sudo mv wgo /usr/local/bin/

# Or install with Go
go install github.com/yairfalse/wgo/cmd/wgo@latest

# Or build from source
git clone https://github.com/yairfalse/wgo.git
cd wgo && task build
```

### Essential Setup

```bash
# 1. Initialize WGO
wgo config init

# 2. Scan your infrastructure
wgo scan --provider terraform --provider aws

# 3. See what's going on
wgo status

# 4. Compare with Terraform state
wgo check --terraform-state ./terraform.tfstate
```

---

## 💡 Core Features

### 🔍 **Multi-Provider Scanning**

```bash
# Scan Terraform state
wgo scan --provider terraform --state-path ./terraform/*.tfstate

# Scan AWS resources  
wgo scan --provider aws --regions us-east-1,us-west-2 --services ec2,s3,rds

# Scan Kubernetes clusters
wgo scan --provider kubernetes --contexts prod,staging --namespaces default,kube-system

# Scan everything at once
wgo scan --provider terraform --provider aws --provider kubernetes
```

### 🧠 **Infrastructure Status**

```bash
# See what's going on across all providers
wgo status

# View last scan results
wgo status --latest

# Get detailed infrastructure summary
wgo status --detailed --format table

# Check specific provider status
wgo status --provider aws --regions us-east-1
```

### 🔍 **Drift Detection**

```bash
# Compare current AWS state with Terraform
wgo check --terraform-state ./terraform.tfstate --provider aws

# Compare Kubernetes with manifests
wgo check --k8s-manifests ./k8s/ --provider kubernetes

# Show detailed differences
wgo diff --terraform-state ./terraform.tfstate --format markdown
```

### ⚡ **Smart Caching**

```bash
# View cache statistics
wgo cache stats

# Clear cache
wgo cache clear

# Configure cache settings
wgo config set cache.default_ttl 2h
wgo config set cache.max_size 512MB
```

---

## 🏗️ Architecture

WGO follows a modular, extensible architecture:

```
┌─────────────────┐
│   CLI Layer     │  ← User interface (cobra commands)
├─────────────────┤
│  Collectors     │  ← Data gathering (Terraform, AWS, K8s) 
├─────────────────┤
│  Differ Engine  │  ← Compare states, detect drift
├─────────────────┤
│  Storage        │  ← Local JSON files, scan history
├─────────────────┤
│  Outputter      │  ← Format results (JSON, table, markdown)
└─────────────────┘
```

### Key Components

- **Collectors**: Gather infrastructure data from multiple providers
- **Differ Engine**: Compare current state with Terraform configurations  
- **Storage**: Scan history and caching
- **Outputter**: Clean, readable output formats

---

## ⚙️ Configuration

Create `~/.wgo/config.yaml`:

```yaml
cache:
  enabled: true
  default_ttl: "1h"
  max_size: "256MB"
  cleanup_interval: "5m"

collectors:
  terraform:
    state_paths:
      - "./terraform/*.tfstate"
      - "s3://my-bucket/terraform.tfstate"
  
  aws:
    regions: ["us-east-1", "us-west-2"]
    profile: "default"
    services: ["ec2", "s3", "rds", "lambda", "iam"]
  
  kubernetes:
    contexts: ["prod", "staging"]
    namespaces: ["default", "kube-system", "istio-system"]
    resources: ["deployments", "services", "configmaps", "secrets"]

storage:
  base_path: "~/.wgo"
  max_history: 30
  compress_scans: true

output:
  default_format: "table"
  color: true
  verbose: false
```

---

## 📖 Usage Examples

### Daily Operations

```bash
# Morning infrastructure check - "What's going on?"
wgo status --all-providers

# Quick security scan
wgo scan --provider aws --services iam,security-groups --format table

# Weekly comprehensive scan
wgo scan --all-providers --output weekly-scan-$(date +%Y%m%d).json
```

### CI/CD Integration

```bash
# Pre-deployment infrastructure check
wgo scan --provider terraform --fail-on-issues

# Post-deployment drift detection
wgo check --terraform-state ./terraform.tfstate --provider aws

# Automated reporting
wgo scan --all-providers --format html > reports/infrastructure-$(date +%Y%m%d).html
```

### Troubleshooting

```bash
# When things break - quickly see what's different
wgo check --terraform-state ./terraform.tfstate --format table

# Compare environments
wgo scan --provider aws --profile prod > prod.json
wgo scan --provider aws --profile staging > staging.json
wgo diff --file1 prod.json --file2 staging.json
```

---

## 🛠️ Development

### Prerequisites

- Go 1.21+
- [Task](https://taskfile.dev/) (recommended)
- [golangci-lint](https://golangci-lint.run/)
- Valid cloud provider credentials

### Build & Test

```bash
# Install dependencies
task deps

# Run tests
task test

# Build for current platform
task build

# Build for all platforms
task build:all

# Run linting
task lint

# Generate documentation
task docs
```

### Project Structure

```
wgo/
├── cmd/wgo/              # CLI application entry point
├── internal/             # Private application code
│   ├── collectors/       # Infrastructure collectors
│   │   ├── terraform/    # Terraform state parsing
│   │   ├── aws/         # AWS SDK integration
│   │   ├── kubernetes/  # Kubernetes client-go
│   │   └── interface.go # Collector interface
│   ├── differ/          # Drift detection engine
│   ├── storage/         # Data persistence
│   ├── cache/           # Caching layer
│   └── output/          # Result formatting
├── pkg/types/           # Public data types
├── test/                # Test data and integration tests
├── examples/            # Usage examples
├── docs/                # Documentation
└── scripts/             # Build and deployment scripts
```

---

## 🌟 Provider Support

### ✅ Terraform
- Local `.tfstate` files
- Remote state backends (S3, Azure, GCS)
- Multiple workspaces
- All resource types
- State locking detection

### ✅ AWS
- **Compute**: EC2 instances, Lambda functions
- **Storage**: S3 buckets, EBS volumes
- **Database**: RDS instances, DynamoDB tables
- **Security**: IAM roles, Security Groups
- **Networking**: VPCs, Load Balancers
- Multi-region support

### ✅ Kubernetes
- **Workloads**: Deployments, StatefulSets, DaemonSets, Jobs
- **Networking**: Services, Ingresses, NetworkPolicies
- **Storage**: PVs, PVCs, StorageClasses
- **Configuration**: ConfigMaps, Secrets, ServiceAccounts
- **Security**: RBAC, SecurityContexts
- **Custom Resources**: CRDs and instances
- Multi-cluster support

### 🔄 Coming Soon
- **Google Cloud Platform**: Compute Engine, Cloud Storage, GKE
- **Azure**: Virtual Machines, Storage Accounts, AKS
- **GitOps**: ArgoCD, Flux applications
- **Monitoring**: Prometheus, Grafana configurations

---

## 🚨 Troubleshooting

### Common Issues

**Claude API Key Not Found**
```bash
# WGO works without AI - this is for future versions
# Current version focuses on simple scanning and drift detection
```

**AWS Credentials**
```bash
aws configure
# Or use IAM roles, environment variables
```

**Kubernetes Access**
```bash
kubectl config current-context
kubectl config get-contexts
```

**Debug Mode**
```bash
wgo scan --verbose --debug
wgo check --verbose --debug --log-level debug
```

### Performance Tuning

```bash
# Increase cache size for large infrastructures
wgo config set cache.max_size 1GB

# Adjust TTL for frequently changing resources
wgo config set cache.aws_ttl 5m
wgo config set cache.k8s_ttl 2m

# Parallel scanning for faster execution
wgo config set collectors.parallel_limit 10
```

---

## 🤝 Contributing

We welcome contributions! Here's how to get started:

1. **Fork** the repository
2. **Create** your feature branch (`git checkout -b feature/amazing-feature`)
3. **Make** your changes and add tests
4. **Run** linting and tests (`task lint test`)
5. **Commit** your changes (`git commit -m 'Add amazing feature'`)
6. **Push** to the branch (`git push origin feature/amazing-feature`)
7. **Open** a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation for user-facing changes
- Use conventional commit messages
- Ensure all CI checks pass

### Areas for Contribution

- 🔌 **New Providers**: GCP, Azure, GitOps tools
- 📊 **Output Formats**: Custom templates, integrations
- ⚡ **Performance**: Caching strategies, parallel processing
- 🛡️ **Security**: Compliance checks, vulnerability detection
- 🧠 **Future AI**: Claude integration for analysis

---

## 📈 Roadmap

### Q1 2025
- ✅ Core Terraform, AWS, Kubernetes support
- ✅ Smart caching and performance optimization
- 🔄 Advanced drift detection
- 🔄 Security scanning features

### Q2 2025
- 📋 Claude AI analysis integration
- 📋 Google Cloud Platform support
- 📋 Azure integration
- 📋 GitOps drift detection (ArgoCD, Flux)
- 📋 Compliance reporting (SOC2, PCI, HIPAA)

### Q3 2025
- 📋 Real-time monitoring (eBPF-based)
- 📋 Predictive analysis
- 📋 Auto-remediation suggestions
- 📋 Cost impact analysis
- 📋 Enterprise features (RBAC, audit logs)
- ✅ Smart caching and performance optimization
- 🔄 Advanced security analysis
- 🔄 Historical trend analysis

### Q2 2025
- 📋 Google Cloud Platform support
- 📋 Azure integration
- 📋 GitOps drift detection (ArgoCD, Flux)
- 📋 Compliance reporting (SOC2, PCI, HIPAA)
- 📋 Slack/Teams notifications

### Q3 2025
- 📋 Real-time monitoring (eBPF-based)
- 📋 Predictive drift analysis
- 📋 Auto-remediation suggestions
- 📋 Cost impact analysis
- 📋 Enterprise features (RBAC, audit logs)

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [Polar Signals](https://www.polarsignals.com/) for eBPF inspiration
- The Go community for excellent tooling
- DevOps practitioners sharing their pain points and feedback

---

<div align="center">

**What's Going On with your infrastructure?** 🔍

[Get Started](https://github.com/yairfalse/wgo/releases) • [Documentation](https://github.com/yairfalse/wgo/docs) • [Contributing](https://github.com/yairfalse/wgo/CONTRIBUTING.md) • [Discord](https://discord.gg/wgo)

</div>
