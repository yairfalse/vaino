# WGO Simple Usage Guide

## Quick Start

### 1. First Time Setup

```bash
# For Terraform projects
cd your-terraform-project
wgo scan --provider terraform

# For Google Cloud
wgo scan --provider gcp --project your-project-id

# For AWS
wgo scan --provider aws --region us-east-1
```

### 2. Create a Baseline

```bash
# After your first scan, create a baseline
wgo baseline create
```

### 3. Check for Drift

```bash
# Run this anytime to check for changes
wgo check

# See detailed differences
wgo diff
```

## Common Issues & Fixes

### GCP Authentication Error
If you see "GCP Authentication Failed", run:
```bash
wgo auth gcp
```

### AWS Authentication Error  
If you see "AWS Authentication Failed", run:
```bash
wgo auth aws
```

### Check Authentication Status
```bash
wgo auth status
```

## Basic Workflow

1. **Scan** your infrastructure → `wgo scan --provider <provider>`
2. **Create baseline** when happy → `wgo baseline create`  
3. **Check for drift** regularly → `wgo check`
4. **View differences** when drift found → `wgo diff`

## Tips

- WGO works without AI setup (Claude API is optional)
- Use `wgo auth <provider>` if you get authentication errors
- Each command has `--help` with examples
- The auth helper will guide you through fixing issues