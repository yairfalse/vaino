# VAINO - Infrastructure Drift Detection

VAINO (What's Going On) is a command-line tool for detecting drift in your infrastructure by comparing snapshots over time. Think of it as "git diff for infrastructure."

## What It Does

VAINO scans your infrastructure, takes snapshots of the current state, and compares them to previous snapshots to detect changes. It supports multiple infrastructure providers and gives you Unix-style output that's easy to parse and integrate with other tools.

### Core Features

- **Multi-Provider Support**: Scan AWS, GCP, Kubernetes, and Terraform
- **Snapshot Comparison**: Track changes between any two points in time
- **Timeline Analysis**: View infrastructure changes over time with correlation
- **Unix-Style Output**: Clean, parseable output formats (table, JSON, YAML, markdown)
- **Zero Configuration**: Works out of the box with smart defaults

## Supported Providers

### AWS
Collects resources from multiple services:
- EC2 instances, security groups, load balancers
- ECS clusters and services, EKS clusters
- Lambda functions, S3 buckets
- RDS databases, DynamoDB tables
- IAM roles and policies, VPC components
- CloudFormation stacks, CloudWatch alarms

### Google Cloud Platform (GCP)
- Compute instances and cloud SQL
- GKE clusters and container services
- IAM bindings and service accounts

### Kubernetes
- Pods, services, deployments, configmaps
- Ingresses, secrets, persistent volumes
- Network policies, service accounts

### Terraform
- Parses Terraform state files (local and remote)
- Supports all resource types that Terraform manages

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│    Collectors   │────│   Snapshots      │────│   Comparison    │
│ (AWS/GCP/K8s/TF)│    │  (~/.vaino/)     │    │   & Analysis    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                       │
         ▼                        ▼                       ▼
   ┌──────────┐          ┌─────────────────┐     ┌─────────────────┐
   │ Scanning │          │ JSON Storage    │     │ Diff Detection  │
   │ Commands │          │ History Mgmt    │     │ Timeline View   │
   └──────────┘          └─────────────────┘     └─────────────────┘
```

## Quick Start

### Installation

```bash
# Download latest release
curl -sSL https://install.wgo.sh | bash

# Or with Go
go install github.com/yairfalse/vaino/cmd/vaino@latest
```

### Basic Usage

```bash
# Take a snapshot of your infrastructure
vaino scan

# Compare current state to last scan
vaino diff

# View the timeline of changes
vaino timeline

# Check system status and configuration  
vaino status
```

### Provider-Specific Scanning

```bash
# Scan specific providers
vaino scan --providers aws,kubernetes
vaino scan --providers terraform --terraform-dir ./infrastructure

# AWS with specific regions
vaino scan --providers aws --aws-regions us-east-1,eu-west-1

# GCP with specific project
vaino scan --providers gcp --gcp-project my-project-id
```

## Configuration

VAINO works with zero configuration, but you can customize it:

```bash
# Generate default config file
vaino configure

# Edit the configuration
# ~/.vaino/config.yaml
```

Example configuration:
```yaml
providers:
  aws:
    enabled: true
    regions: ["us-east-1", "eu-west-1"]
    profile: default
  
  gcp:
    enabled: true
    project_id: my-project
    
  kubernetes:
    enabled: true
    context: my-cluster
    
  terraform:
    enabled: true
    state_files:
      - ./terraform/terraform.tfstate
      - s3://my-bucket/terraform.tfstate

output:
  format: table
  no_color: false

storage:
  history_limit: 50
```

## Example Output

### Diff Output
```bash
$ vaino diff
╭────────────────────────────────────────╮
│ Infrastructure Changes (2 changes)     │
├────────────────────────────────────────┤
│ Provider: AWS (us-east-1)              │
│                                        │
│ ● EC2 Instance: i-1234567890abcdef0    │
│   │ + Tags.Environment: "production"   │
│   │ ~ State: "running" → "stopped"     │
│                                        │
│ ● S3 Bucket: my-application-logs       │
│   │ + Created                          │
│   │ + Versioning: enabled              │
╰────────────────────────────────────────╯
```

### Timeline Output
```bash
$ vaino timeline --days 7
2024-08-10 14:30  ● 3 changes  (AWS: 2, K8s: 1)
2024-08-09 09:15  ● 1 change   (AWS: 1)
2024-08-08 16:45  ● 5 changes  (K8s: 3, TF: 2)
```

## Development

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Format code (required before commits)
make fmt

# Run linting
make lint
```

### Requirements

- Go 1.21+
- Access to the infrastructure providers you want to scan
- Appropriate credentials configured (AWS CLI, gcloud, kubectl, etc.)

### Project Structure

```
cmd/vaino/           # CLI commands and main entry point
internal/collectors/ # Provider-specific collection logic  
internal/storage/    # Snapshot storage and management
internal/differ/     # Diff computation and analysis
pkg/types/          # Core data structures
docs/               # Documentation
scripts/            # Build and utility scripts
test/               # Test suites and fixtures
```

## Authentication

VAINO uses standard provider authentication methods:

- **AWS**: AWS CLI credentials, IAM roles, or environment variables
- **GCP**: gcloud CLI, service account keys, or environment variables  
- **Kubernetes**: kubectl configuration files
- **Terraform**: Direct access to state files (local or remote)

## Current Status

VAINO is actively developed and used for infrastructure drift detection. It provides:

- ✅ Multi-provider resource collection
- ✅ Snapshot-based change tracking  
- ✅ Timeline correlation and analysis
- ✅ Multiple output formats
- ✅ Configuration management

### Known Limitations

- Large infrastructures may take time to scan (optimizations in progress)
- Some provider APIs have rate limits
- Remote Terraform state requires appropriate access credentials
- AI analysis features are experimental

## Contributing

See [docs/development/contributing.md](docs/development/contributing.md) for development guidelines.

## License

Apache 2.0 - see LICENSE file for details.