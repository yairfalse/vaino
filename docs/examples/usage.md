# WGO Configuration & Usage Examples

## Quick Start

### 1. Initialize WGO in your infrastructure repository
```bash
cd /path/to/your/infrastructure
wgo setup
```

### 2. For existing Terraform infrastructure
```bash
# Auto-detect and configure
wgo setup --providers terraform

# Manual configuration
wgo scan --provider terraform --path ./terraform
wgo scan --provider terraform --path ./environments/prod/terraform.tfstate
```

### 3. For AWS infrastructure
```bash
# Configure AWS regions
wgo setup --providers aws

# Scan specific regions
wgo scan --provider aws --region us-east-1,us-west-2
```

## Configuration Examples

### For Terraform-heavy Infrastructure
```yaml
# ~/.wgo/config.yaml
providers:
  terraform:
    enabled: true
    state_paths:
      - "./terraform/environments/*/terraform.tfstate"
      - "./modules/*/terraform.tfstate" 
      - "./terraform.tfstate"
    workspaces:
      - "production"
      - "staging"
      - "development"
    auto_discover: true

git:
  enabled: true
  track_commits: true
  baseline_on_tag: true
  baseline_branch: "main"
  ignore_branches:
    - "feature/*"
    - "hotfix/*"
```

### For Multi-Cloud Setup
```yaml
providers:
  terraform:
    enabled: true
    state_paths:
      - "./terraform"
      
  aws:
    enabled: true
    regions:
      - "us-east-1"
      - "us-west-2"
      - "eu-west-1" 
    profiles:
      - "production"
      - "staging"
      
  kubernetes:
    enabled: true
    contexts:
      - "prod-cluster"
      - "staging-cluster"
    namespaces:
      - "default"
      - "kube-system"
      - "production"
      - "staging"
```

## Git Integration Workflows

### 1. Automatic Baselines on Tags
```bash
# Tag a release - WGO will auto-create baseline
git tag v1.0.0
git push origin v1.0.0

# WGO automatically creates baseline from current state
wgo baseline list
```

### 2. Pre-commit Drift Detection
```bash
# Add to .git/hooks/pre-commit
#!/bin/bash
wgo scan --provider terraform
wgo check --baseline latest --fail-on-drift
```

### 3. CI/CD Integration
```bash
# In your CI pipeline
wgo scan --all
wgo check --baseline production-v1.0 --report drift-report.json
wgo analyze drift-report.json
```

## Common Use Cases

### 1. Daily Infrastructure Check
```bash
# Quick status overview
wgo status

# Detailed scan and comparison
wgo scan --all
wgo check --baseline latest --explain
```

### 2. Before Terraform Apply
```bash
# Capture current state
wgo scan --provider terraform

# Apply changes
terraform apply

# Check what actually changed
wgo scan --provider terraform
wgo check --baseline previous --detailed
```

### 3. Incident Response
```bash
# When something breaks, check recent changes
wgo status --since 2h
wgo check --baseline yesterday --explain

# Get AI analysis of what changed
wgo analyze --focus security --since 6h
```

### 4. Compliance Reporting
```bash
# Generate compliance report
wgo scan --all --output compliance-scan.json
wgo check --baseline certified-baseline --format json > compliance-report.json
wgo analyze compliance-report.json --focus security
```

## Environment-Specific Configuration

### Development
```bash
export WGO_TERRAFORM_STATE_PATH=./dev/terraform.tfstate
export WGO_DEBUG=true
wgo scan --provider terraform
```

### Production
```bash
export WGO_TERRAFORM_STATE_PATH=./prod/terraform.tfstate
export AWS_PROFILE=production
export KUBECONFIG=~/.kube/prod-config
wgo scan --all
```

### CI/CD
```bash
export WGO_VERBOSE=true
export ANTHROPIC_API_KEY=$CI_ANTHROPIC_KEY
wgo scan --all --output-file ci-scan-$BUILD_NUMBER.json
wgo check --baseline main-branch --fail-on-drift
```

## Advanced Configuration

### Custom Collectors
```yaml
collectors:
  terraform:
    timeout: "5m"
    retry_attempts: 3
    
  aws:
    timeout: "10m"
    rate_limit: "100/minute"
    
  kubernetes:
    timeout: "2m"
    include_secrets: false
```

### Output Customization
```yaml
output:
  formats:
    table:
      max_width: 120
      show_timestamps: true
    json:
      pretty: true
      include_metadata: true
```

### Integration with External Tools
```yaml
integrations:
  slack:
    webhook_url: "https://hooks.slack.com/..."
    channels:
      alerts: "#infrastructure-alerts"
      reports: "#infrastructure-reports"
      
  jira:
    server: "https://yourcompany.atlassian.net"
    project: "INFRA"
    
  datadog:
    api_key: "${DATADOG_API_KEY}"
    tags:
      - "team:platform"
      - "env:production"
```