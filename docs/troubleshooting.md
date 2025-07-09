# Troubleshooting Guide

Common issues and solutions when using VAINO.

## 🔧 Installation Issues

### Binary Not Found

**Problem:** `wgo: command not found`

**Solutions:**
```bash
# Check if VAINO is in PATH
which wgo

# Add to PATH if installed manually
export PATH="$PATH:/usr/local/bin"
echo 'export PATH="$PATH:/usr/local/bin"' >> ~/.bashrc

# Reinstall via package manager
brew uninstall wgo && brew install yairfalse/wgo/wgo

# Or reinstall via script
curl -sSL https://install.wgo.sh | bash
```

### Permission Errors

**Problem:** `Permission denied` when running VAINO

**Solutions:**
```bash
# Make binary executable
chmod +x /usr/local/bin/wgo

# Check ownership
ls -la /usr/local/bin/wgo

# Fix ownership if needed
sudo chown $(whoami):$(whoami) /usr/local/bin/wgo

# For Docker usage
docker run --rm -v $(pwd):/workspace yairfalse/wgo:latest scan
```

### Version Conflicts

**Problem:** Multiple versions installed

**Solutions:**
```bash
# Check installed versions
which -a wgo
wgo version

# Remove old versions
sudo rm /usr/local/bin/wgo
brew uninstall wgo

# Clean install
curl -sSL https://install.wgo.sh | bash
```

## ⚙️ Configuration Issues

### Config File Problems

**Problem:** `Configuration file not found` or `Invalid configuration`

**Solutions:**
```bash
# Check config file location
wgo configure --show

# Validate configuration
wgo check-config

# Reset configuration
rm -rf ~/.wgo
wgo configure

# Manual config creation
mkdir -p ~/.wgo
cat > ~/.wgo/config.yaml << EOF
providers:
  terraform:
    enabled: true
  aws:
    enabled: false
  gcp:
    enabled: false
  kubernetes:
    enabled: false
EOF
```

### Provider Configuration

**Problem:** Provider not working despite configuration

**Solutions:**
```bash
# Test specific provider
wgo status --provider aws
wgo status --provider gcp
wgo status --provider kubernetes

# Debug provider issues
wgo --debug configure aws
wgo --debug scan --provider aws

# Check credentials
aws sts get-caller-identity
gcloud auth list
kubectl config current-context
```

## 🔐 Authentication Issues

### AWS Authentication

**Problem:** `InvalidAccessKeyId` or `SignatureDoesNotMatch`

**Solutions:**
```bash
# Check AWS credentials
aws configure list
aws sts get-caller-identity

# Set correct profile
export AWS_PROFILE=your-profile
wgo configure aws

# Check permissions
aws iam get-user
aws iam list-attached-user-policies

# Test with minimal policy
cat > wgo-minimal-policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "s3:GetBucketLocation",
        "s3:ListBucket"
      ],
      "Resource": "*"
    }
  ]
}
EOF
```

### GCP Authentication

**Problem:** `Application Default Credentials not found`

**Solutions:**
```bash
# Set service account key
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json

# Or use gcloud auth
gcloud auth application-default login

# Check current auth
gcloud auth list
gcloud config list

# Test permissions
gcloud projects list
gcloud compute instances list --limit=1

# Service account setup
gcloud iam service-accounts create wgo-monitor \
  --description="VAINO monitoring service account" \
  --display-name="VAINO Monitor"

gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:wgo-monitor@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/viewer"
```

### Kubernetes Authentication

**Problem:** `Unable to connect to cluster`

**Solutions:**
```bash
# Check kubectl configuration
kubectl config view
kubectl config current-context

# Test connectivity
kubectl cluster-info
kubectl get nodes

# Check permissions
kubectl auth can-i get pods
kubectl auth can-i list deployments

# Fix context issues
kubectl config use-context your-context
kubectl config set-context --current --namespace=default

# Service account token (for CI/CD)
kubectl create serviceaccount wgo-monitor
kubectl create clusterrolebinding wgo-monitor \
  --clusterrole=view --serviceaccount=default:wgo-monitor
```

## 📊 Scanning Issues

### No Resources Found

**Problem:** `No resources found` when scanning

**Solutions:**

**Terraform:**
```bash
# Check state file location
ls -la terraform.tfstate
terraform state list

# Specify state file explicitly
wgo scan --provider terraform --state-file ./terraform.tfstate

# Check working directory
pwd
wgo scan --provider terraform --path ./infrastructure

# For remote state
terraform state pull > local-state.json
wgo scan --provider terraform --state-file local-state.json
```

**AWS:**
```bash
# Check region
aws configure get region
wgo scan --provider aws --region us-east-1

# List available regions
aws ec2 describe-regions --output table

# Check for resources
aws ec2 describe-instances --region us-east-1
aws s3 ls
```

**GCP:**
```bash
# Check project
gcloud config get-value project
wgo scan --provider gcp --project your-project-id

# List projects
gcloud projects list

# Check for resources
gcloud compute instances list
gcloud storage buckets list
```

**Kubernetes:**
```bash
# Check namespaces
kubectl get namespaces
wgo scan --provider kubernetes --namespace default

# Check context
kubectl config current-context
wgo scan --provider kubernetes --context your-context

# List resources
kubectl get all --all-namespaces
```

### Slow Scanning

**Problem:** Scanning takes very long

**Solutions:**
```bash
# Scan specific regions only
wgo scan --provider aws --region us-east-1

# Exclude unnecessary resources
wgo configure aws
# Select only needed services

# Use parallel scanning
wgo scan --provider aws &
wgo scan --provider gcp &
wait

# Enable caching
# ~/.wgo/config.yaml
cache:
  enabled: true
  ttl: 300s
```

### Resource Filtering

**Problem:** Too many irrelevant resources

**Solutions:**
```yaml
# ~/.wgo/config.yaml
providers:
  aws:
    include_services: ["ec2", "rds", "s3"]
    exclude_resources: ["aws_cloudwatch_log_group"]
    
  kubernetes:
    include_resources: ["Deployment", "Service", "Ingress"]
    exclude_resources: ["Event", "Endpoints"]
    
  terraform:
    exclude_patterns:
      - "*.random_*"
      - "data.*"
```

## 🔍 Diff and Comparison Issues

### No Changes Detected

**Problem:** Expected changes not showing up

**Solutions:**
```bash
# Check snapshot history
wgo scan list

# Compare specific snapshots
wgo diff snapshot-1 snapshot-2

# Check baseline
wgo baseline list
wgo diff --baseline your-baseline

# Force new scan
wgo scan --force

# Check time windows
wgo diff --from "1 hour ago" --to "now"
```

### False Positives

**Problem:** Changes detected for unchanged resources

**Solutions:**
```yaml
# ~/.wgo/config.yaml
drift:
  ignore_patterns:
    - "*.last_modified"
    - "*.created_at"
    - "tags.LastUpdated"
    - "metadata.generation"
```

```bash
# Filter by severity
wgo diff --severity high

# Exclude known noisy resources
wgo diff --exclude-resource-type "aws_cloudwatch_log_group"
```

### Baseline Issues

**Problem:** Baseline comparison fails

**Solutions:**
```bash
# List all baselines
wgo baseline list

# Check baseline details
wgo baseline show your-baseline

# Recreate baseline
wgo baseline delete old-baseline
wgo scan
wgo baseline create --name new-baseline

# Validate baseline integrity
wgo check-config --baselines
```

## 🚀 Performance Issues

### High Memory Usage

**Problem:** VAINO using too much memory

**Solutions:**
```yaml
# ~/.wgo/config.yaml
performance:
  max_resources: 10000
  batch_size: 100
  
cache:
  max_size: 50MB
  
storage:
  compression: true
```

```bash
# Monitor memory usage
top -p $(pgrep wgo)

# Reduce scan scope
wgo scan --provider terraform --path ./specific-module

# Clear cache
rm -rf ~/.wgo/cache/*
```

### Network Timeouts

**Problem:** API calls timing out

**Solutions:**
```yaml
# ~/.wgo/config.yaml
providers:
  aws:
    timeout: 30s
    retry_attempts: 3
    
  gcp:
    timeout: 60s
    retry_attempts: 5
```

```bash
# Check network connectivity
ping amazonaws.com
ping googleapis.com

# Use specific endpoints
export AWS_ENDPOINT_URL=https://ec2.us-east-1.amazonaws.com
```

### Disk Space Issues

**Problem:** Running out of disk space

**Solutions:**
```bash
# Check disk usage
du -sh ~/.wgo/

# Clean old snapshots
find ~/.wgo/snapshots -name "*.json" -mtime +30 -delete

# Clean old logs
find ~/.wgo/logs -name "*.log" -mtime +7 -delete

# Configure retention
# ~/.wgo/config.yaml
storage:
  retention_days: 30
  max_snapshots: 100
```

## 📱 Integration Issues

### CI/CD Problems

**Problem:** VAINO fails in CI/CD pipeline

**Solutions:**

**GitHub Actions:**
```yaml
# Add debugging
- name: Debug VAINO
  run: |
    wgo version
    wgo check-config
    wgo status --verbose
    
# Check permissions
- name: Test credentials
  run: |
    aws sts get-caller-identity
    gcloud auth list
    kubectl cluster-info
```

**Docker Issues:**
```bash
# Mount correct volumes
docker run --rm \
  -v $(pwd):/workspace \
  -v ~/.aws:/home/wgo/.aws:ro \
  -v ~/.kube:/home/wgo/.kube:ro \
  yairfalse/wgo:latest scan

# Check container logs
docker logs container-id

# Run interactively for debugging
docker run -it --entrypoint=/bin/bash yairfalse/wgo:latest
```

### Webhook Failures

**Problem:** Webhooks not working

**Solutions:**
```bash
# Test webhook manually
curl -X POST https://hooks.slack.com/services/... \
  -H "Content-Type: application/json" \
  -d '{"text":"Test message"}'

# Check webhook configuration
wgo configure --show | grep webhook

# Debug webhook calls
wgo --debug watch --webhook https://...

# Validate JSON format
echo '{"text":"test"}' | jq .
```

## 🛠️ Debug Mode

### Enable Debugging

**Global Debug:**
```bash
# Enable debug mode
wgo --debug scan
wgo --debug diff
wgo --debug configure

# Verbose output
wgo --verbose status
wgo --verbose scan --provider aws

# Log level control
export VAINO_LOG_LEVEL=debug
wgo scan
```

### Debug Configuration

```yaml
# ~/.wgo/config.yaml
logging:
  level: debug
  file: ~/.wgo/logs/debug.log
  
debug:
  enabled: true
  api_calls: true
  timing: true
```

### Log Analysis

```bash
# View recent logs
tail -f ~/.wgo/logs/wgo.log

# Search for errors
grep -i error ~/.wgo/logs/wgo.log

# API call tracing
grep "API:" ~/.wgo/logs/debug.log

# Timing analysis
grep "Duration:" ~/.wgo/logs/debug.log
```

## 🆘 Getting Help

### Diagnostic Information

**Collect System Info:**
```bash
#!/bin/bash
# collect-diagnostics.sh

echo "=== VAINO Diagnostics ==="
echo "Date: $(date)"
echo "OS: $(uname -a)"
echo

echo "=== VAINO Version ==="
wgo version
echo

echo "=== Configuration ==="
wgo check-config
echo

echo "=== Provider Status ==="
wgo status --verbose
echo

echo "=== Environment ==="
env | grep -E "(AWS|GCP|GOOGLE|KUBE)" | sort
echo

echo "=== Disk Usage ==="
du -sh ~/.wgo/
echo

echo "=== Recent Logs ==="
tail -20 ~/.wgo/logs/wgo.log
```

### Common Error Codes

| Exit Code | Meaning | Solution |
|-----------|---------|----------|
| 0 | Success | Normal operation |
| 1 | Changes detected | Expected for drift detection |
| 2 | Invalid arguments | Check command syntax |
| 3 | Configuration error | Run `wgo check-config` |
| 4 | Authentication error | Check credentials |
| 5 | Network error | Check connectivity |

### Support Channels

**Before Reporting Issues:**
1. Run `wgo check-config`
2. Try with `--debug` flag
3. Check logs in `~/.wgo/logs/`
4. Collect diagnostic information

**Issue Template:**
```
**VAINO Version:** (output of `wgo version`)
**OS:** (e.g., macOS 13.0, Ubuntu 22.04)
**Provider:** (terraform/aws/gcp/kubernetes)
**Command:** (exact command that failed)

**Expected Behavior:**
What you expected to happen

**Actual Behavior:**
What actually happened

**Error Output:**
```
paste error output here
```

**Configuration:**
```yaml
# Sanitized version of ~/.wgo/config.yaml
paste relevant config sections
```

**Additional Context:**
Any other relevant information
```

## 🔄 Recovery Procedures

### Reset VAINO

**Complete Reset:**
```bash
# Backup current state
cp -r ~/.wgo ~/.wgo.backup

# Remove all VAINO data
rm -rf ~/.wgo

# Reinstall
curl -sSL https://install.wgo.sh | bash

# Reconfigure
wgo configure
```

### Recover Corrupted Data

**Snapshot Recovery:**
```bash
# Check snapshot integrity
wgo scan list --validate

# Remove corrupted snapshots
find ~/.wgo/snapshots -name "*.json" -exec jq . {} \; > /dev/null

# Rebuild from backups
cp ~/.wgo.backup/snapshots/*.json ~/.wgo/snapshots/
```

**Baseline Recovery:**
```bash
# List baselines
wgo baseline list

# Recreate from snapshot
wgo baseline create --from snapshot-123 --name recovered-baseline

# Validate baseline
wgo baseline show recovered-baseline
```

This troubleshooting guide should help resolve most common issues encountered when using VAINO.