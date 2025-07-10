# Configuration Reference

VAINO works out of the box with zero configuration, but can be customized for advanced workflows and team environments.

## Configuration File

### Default Location
```bash
~/.wgo/config.yaml
```

### Custom Location
```bash
wgo --config /path/to/config.yaml scan
```

### Interactive Configuration
```bash
wgo configure
# Launches interactive configuration wizard
```

## Configuration Structure

### Complete Example
```yaml
# ~/.wgo/config.yaml

# Provider Configurations
providers:
  terraform:
    state_paths:
      - "./terraform"
      - "./infrastructure"
    exclude_paths:
      - "**/.terraform"
      - "**/terraform.tfstate.backup"
    remote_state:
      enabled: true
      backends: ["s3", "gcs", "azurerm"]
  
  aws:
    regions: ["us-east-1", "us-west-2", "eu-west-1"]
    profile: "default"
    assume_role:
      role_arn: "arn:aws:iam::123456789012:role/VAINORole"
      session_name: "wgo-session"
    exclude_services: ["cloudtrail", "cloudwatch"]
  
  gcp:
    project_id: "my-project-123"
    regions: ["us-central1", "us-east1", "europe-west1"]
    credentials_file: "/path/to/service-account.json"
    exclude_services: ["logging", "monitoring"]
  
  kubernetes:
    contexts: ["prod", "staging"]
    namespaces: ["default", "kube-system", "monitoring"]
    kubeconfig: "~/.kube/config"
    exclude_namespaces: ["kube-public"]

# Storage Configuration
storage:
  base_path: "~/.wgo"
  retention_days: 30
  compression: true
  encryption:
    enabled: false
    key_file: "/path/to/encryption.key"

# Output Configuration
output:
  format: "table"  # table, json, yaml, markdown
  pretty: true
  colors: true
  pager: true
  max_width: 120
  truncate: true

# Scanning Configuration
scan:
  parallel: true
  timeout: "300s"
  retry_attempts: 3
  cache_enabled: true
  cache_ttl: "1h"

# Drift Detection Configuration
drift:
  sensitivity: "medium"  # low, medium, high
  ignore_patterns:
    - "*.created_at"
    - "*.last_modified"
    - "*.etag"
  severity_thresholds:
    high: ["instance_type", "security_groups", "iam_roles"]
    medium: ["tags", "environment_variables"]
    low: ["descriptions", "names"]

# Baseline Configuration
baselines:
  auto_create: false
  naming_pattern: "baseline-{provider}-{date}"
  max_baselines: 10

# Webhook Configuration
webhooks:
  enabled: false
  drift_detected:
    url: "https://hooks.slack.com/services/..."
    method: "POST"
    headers:
      Content-Type: "application/json"
  scan_completed:
    url: "https://monitoring.example.com/webhook"
    method: "POST"

# Logging Configuration
logging:
  level: "info"  # debug, info, warn, error
  file: "~/.wgo/wgo.log"
  rotation:
    max_size: "100MB"
    max_files: 5
```

## Provider-Specific Configuration

### Terraform

```yaml
providers:
  terraform:
    # State file locations
    state_paths:
      - "./terraform"
      - "./infrastructure/prod"
      - "./infrastructure/staging"
    
    # Files/directories to exclude
    exclude_paths:
      - "**/.terraform"
      - "**/terraform.tfstate.backup"
      - "**/crash.log"
    
    # Remote state configuration
    remote_state:
      enabled: true
      backends:
        - "s3"
        - "gcs"
        - "azurerm"
        - "consul"
        - "etcd"
      
      # Backend-specific settings
      s3:
        bucket: "my-terraform-state"
        region: "us-east-1"
        profile: "terraform"
      
      gcs:
        bucket: "my-terraform-state"
        project: "my-project-123"
    
    # Parsing options
    parsing:
      strict_mode: false
      validate_schema: true
      max_file_size: "100MB"
```

### AWS

```yaml
providers:
  aws:
    # Default region
    region: "us-east-1"
    
    # Multiple regions to scan
    regions:
      - "us-east-1"
      - "us-west-2"
      - "eu-west-1"
      - "ap-southeast-1"
    
    # AWS Profile
    profile: "default"
    
    # Role assumption
    assume_role:
      role_arn: "arn:aws:iam::123456789012:role/VAINORole"
      session_name: "wgo-session"
      external_id: "unique-external-id"
    
    # Services to include/exclude
    include_services:
      - "ec2"
      - "s3"
      - "rds"
      - "lambda"
      - "iam"
      - "vpc"
    
    exclude_services:
      - "cloudtrail"
      - "cloudwatch"
      - "logs"
    
    # Resource filtering
    filters:
      ec2:
        instance_states: ["running", "stopped"]
        tags:
          Environment: ["prod", "staging"]
      
      s3:
        exclude_system_buckets: true
        min_size: "1MB"
    
    # Rate limiting
    rate_limit:
      requests_per_second: 10
      burst: 20
```

### GCP

```yaml
providers:
  gcp:
    # Project configuration
    project_id: "my-project-123"
    
    # Multiple projects
    projects:
      - "prod-project-123"
      - "staging-project-456"
      - "dev-project-789"
    
    # Regions to scan
    regions:
      - "us-central1"
      - "us-east1"
      - "europe-west1"
      - "asia-southeast1"
    
    # Authentication
    credentials_file: "/path/to/service-account.json"
    
    # Or use environment variable:
    # GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
    
    # Services to include/exclude
    include_services:
      - "compute"
      - "storage"
      - "sql"
      - "container"
      - "iam"
    
    exclude_services:
      - "logging"
      - "monitoring"
      - "cloudfunctions"
    
    # Resource filtering
    filters:
      compute:
        instance_states: ["RUNNING", "STOPPED"]
        zones: ["us-central1-a", "us-central1-b"]
      
      storage:
        exclude_system_buckets: true
        storage_classes: ["STANDARD", "NEARLINE"]
```

### Kubernetes

```yaml
providers:
  kubernetes:
    # Kubeconfig file
    kubeconfig: "~/.kube/config"
    
    # Contexts to scan
    contexts:
      - "prod-cluster"
      - "staging-cluster"
      - "dev-cluster"
    
    # Namespaces to include
    namespaces:
      - "default"
      - "kube-system"
      - "monitoring"
      - "istio-system"
    
    # Namespaces to exclude
    exclude_namespaces:
      - "kube-public"
      - "kube-node-lease"
    
    # Resource types to include
    include_resources:
      - "deployments"
      - "services"
      - "configmaps"
      - "secrets"
      - "ingresses"
      - "persistentvolumes"
      - "persistentvolumeclaims"
    
    # Resource types to exclude
    exclude_resources:
      - "events"
      - "pods"
      - "replicasets"
    
    # Filtering
    filters:
      labels:
        app: ["web", "api", "database"]
        environment: ["prod", "staging"]
      
      annotations:
        exclude_patterns:
          - "kubectl.kubernetes.io/*"
          - "deployment.kubernetes.io/*"
```

## Output Configuration

### Format Options

```yaml
output:
  # Output format
  format: "table"  # table, json, yaml, markdown, csv
  
  # Pretty printing
  pretty: true
  
  # Colors (auto-detected for terminals)
  colors: true
  
  # Use pager for long output
  pager: true
  
  # Table formatting
  table:
    max_width: 120
    truncate: true
    border: true
    headers: true
  
  # JSON formatting
  json:
    indent: 2
    compact: false
  
  # Markdown formatting
  markdown:
    github_flavor: true
    table_style: "pipe"
```

### File Output

```yaml
output:
  # Default output file patterns
  file_patterns:
    scan: "scan-{provider}-{date}.json"
    diff: "diff-{from}-{to}-{date}.json"
    baseline: "baseline-{name}-{date}.json"
  
  # Output directory
  directory: "./wgo-outputs"
  
  # File permissions
  file_mode: 0644
  
  # Compression
  compression:
    enabled: true
    format: "gzip"  # gzip, bzip2, xz
```

## Environment Variables

VAINO respects standard environment variables:

### AWS
```bash
export AWS_PROFILE=production
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...
```

### GCP
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
export GOOGLE_CLOUD_PROJECT=my-project-123
```

### Kubernetes
```bash
export KUBECONFIG=~/.kube/config
export KUBE_CONTEXT=production
```

### VAINO-Specific
```bash
export VAINO_CONFIG=/path/to/config.yaml
export VAINO_LOG_LEVEL=debug
export VAINO_NO_COLOR=true
export VAINO_CACHE_DIR=/tmp/wgo-cache
```

## Configuration Inheritance

Configuration is loaded in this order (last wins):

1. **Built-in defaults**
2. **System config**: `/etc/wgo/config.yaml`
3. **User config**: `~/.wgo/config.yaml`
4. **Project config**: `./wgo.yaml` or `./.wgo/config.yaml`
5. **Environment variables**
6. **Command-line flags**

### Example Inheritance

```yaml
# ~/.wgo/config.yaml (user defaults)
providers:
  aws:
    region: "us-east-1"
    profile: "default"

# ./wgo.yaml (project-specific)
providers:
  aws:
    profile: "production"  # Overrides user default
    # region: still "us-east-1" from user config
```

## Configuration Validation

### Check Configuration
```bash
# Validate configuration file
wgo check-config

# Validate specific provider
wgo check-config --provider aws

# Verbose validation
wgo check-config --verbose
```

### Configuration Errors

VAINO provides clear error messages for configuration issues:

```bash
$ wgo check-config
Error: Invalid GCP configuration
Cause: project_id is required but not specified
Environment: Development workstation detected

Solutions:
  Set GOOGLE_CLOUD_PROJECT environment variable
  Add project_id to ~/.wgo/config.yaml
  Use --project flag when running commands

Verify: echo $GOOGLE_CLOUD_PROJECT
Help: wgo configure gcp
```

## Team Configuration

### Shared Configuration

```yaml
# .wgo/team-config.yaml
# Commit this to your repository

providers:
  terraform:
    state_paths:
      - "./terraform"
    
  aws:
    regions: ["us-east-1", "us-west-2"]
    # Use profiles, not hardcoded credentials
    
drift:
  ignore_patterns:
    - "*.last_modified"
    - "*.created_at"
    - "tags.LastUpdated"

output:
  format: "json"  # For CI/CD integration
```

### Personal Overrides

```yaml
# ~/.wgo/config.yaml
# Personal settings, not committed

providers:
  aws:
    profile: "my-personal-profile"
    
  gcp:
    credentials_file: "/Users/me/.gcp/personal-key.json"

output:
  format: "table"  # Prefer table for interactive use
  colors: true
```

## Advanced Configuration

### Custom Templates

```yaml
templates:
  scan_report:
    path: "~/.wgo/templates/scan-report.md"
    variables:
      company: "ACME Corp"
      environment: "{{ .Environment }}"
  
  drift_alert:
    path: "~/.wgo/templates/drift-alert.json"
    variables:
      webhook_url: "{{ .WebhookURL }}"
```

### Plugin Configuration

```yaml
plugins:
  enabled: true
  directory: "~/.wgo/plugins"
  
  custom_collectors:
    - name: "datadog"
      path: "./plugins/datadog-collector"
      config:
        api_key: "{{ .DatadogAPIKey }}"
        app_key: "{{ .DatadogAppKey }}"
```

### Performance Tuning

```yaml
performance:
  # Concurrency settings
  max_concurrent_scans: 5
  max_concurrent_requests: 10
  
  # Memory settings
  max_memory_usage: "1GB"
  gc_percent: 100
  
  # Timeouts
  scan_timeout: "300s"
  request_timeout: "30s"
  
  # Caching
  cache:
    enabled: true
    ttl: "1h"
    max_size: "100MB"
    directory: "~/.wgo/cache"
```

---

**Next:** [Commands Reference →](commands.md)