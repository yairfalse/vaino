# Terraform Drift Detection

Real-world examples of using WGO to detect and manage Terraform drift.

## Scenario: Web Application Infrastructure

You have a web application deployed via Terraform with the following resources:
- EC2 instances (web servers)
- RDS database
- S3 buckets (static assets)
- Load balancer
- Security groups

## Initial Setup

### 1. Scan Current Infrastructure
```bash
# Navigate to your Terraform project
cd ~/projects/webapp-infrastructure

# Initial scan to establish baseline
wgo scan --provider terraform --snapshot-name "webapp-initial"
```

**Output:**
```
Infrastructure Scan
===================
Auto-generated snapshot name: webapp-initial
Collecting resources from terraform...
Processed 12 resources in 2.3ms

Collection completed in 15.2ms
Snapshot ID: terraform-1751980234
Resources found: 12

Resource breakdown:
  - aws_instance: 3
  - aws_rds_instance: 1
  - aws_s3_bucket: 2
  - aws_lb: 1
  - aws_security_group: 3
  - aws_db_subnet_group: 1
  - aws_lb_target_group: 1

Snapshot saved - use 'wgo diff' to detect changes
```

### 2. Create Production Baseline
```bash
# Create a baseline for production environment
wgo baseline create --name "webapp-prod-v1.0" \
  --description "Initial production deployment" \
  --tags "environment=prod,version=1.0"
```

## Detecting Manual Changes

### Scenario: Someone manually changed an EC2 instance type

After a few days, you notice performance issues and want to check if anything changed:

```bash
# Check for any drift
wgo diff
```

**Output:**
```
Infrastructure Changes
=====================
📊 Comparing: webapp-initial → current scan
🏗️  Provider: terraform
⏱️  Duration: 1.8s

Changes detected: 2 resources

┌─────────────────────────┬─────────┬──────────┬─────────────────────────────┐
│ Resource                │ Change  │ Severity │ Details                     │
├─────────────────────────┼─────────┼──────────┼─────────────────────────────┤
│ aws_instance.web_server │ MODIFY  │ HIGH     │ instance_type:              │
│                         │         │          │   t3.medium → t3.small      │
│                         │         │          │ Potential impact: -50% CPU  │
├─────────────────────────┼─────────┼──────────┼─────────────────────────────┤
│ aws_instance.web_server │ MODIFY  │ MEDIUM   │ monitoring: true → false    │
│                         │         │          │ CloudWatch monitoring       │
│                         │         │          │ disabled                    │
└─────────────────────────┴─────────┴──────────┴─────────────────────────────┘

Exit code: 1 (changes detected)

💡 Run 'wgo explain' for detailed analysis
```

### Get AI Analysis
```bash
wgo explain
```

**Output:**
```
🤖 AI Analysis of Infrastructure Changes
======================================

Critical Issues Detected:

🔥 EC2 Instance Downgrade (HIGH RISK)
   Resource: aws_instance.web_server
   Change: t3.medium → t3.small
   
   Impact Analysis:
   • 50% reduction in CPU capacity (2 vCPU → 1 vCPU)
   • 50% reduction in memory (4GB → 2GB)  
   • Performance degradation likely
   • Cost savings: ~$15/month per instance
   
   Root Cause Analysis:
   • Manual change detected (not in Terraform state)
   • Change occurred ~2 days ago
   • No corresponding Terraform apply in recent history
   
   Recommendations:
   1. Immediately check application performance metrics
   2. Revert to t3.medium if performance issues confirmed
   3. Update Terraform configuration if downgrade is intentional
   4. Implement change control process

⚠️ Monitoring Disabled (MEDIUM RISK)
   Resource: aws_instance.web_server
   Change: CloudWatch detailed monitoring disabled
   
   Impact:
   • Reduced visibility into instance performance
   • Metrics collection frequency reduced (5min → 1min intervals)
   • May delay incident detection
   
   Recommendation:
   • Re-enable monitoring for production instances
   • Review monitoring costs vs. observability needs

Next Steps:
1. Run 'terraform plan' to see remediation actions
2. Apply Terraform to restore intended configuration
3. Consider implementing drift detection in CI/CD
```

### Fix the Drift
```bash
# Use Terraform to restore the intended configuration
terraform plan

# Apply the fix
terraform apply

# Verify drift is resolved
wgo diff
```

**Output:**
```
Infrastructure Changes
=====================
📊 Comparing: webapp-initial → current scan
🏗️  Provider: terraform
⏱️  Duration: 1.2s

No changes detected ✅

All infrastructure matches the expected state.
```

## Monitoring for Ongoing Drift

### Set Up Automated Drift Detection

**1. Daily Drift Check Script (`check-drift.sh`):**
```bash
#!/bin/bash
set -e

PROJECT_DIR="/home/deploy/webapp-infrastructure"
BASELINE="webapp-prod-v1.0"
SLACK_WEBHOOK="https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"

cd "$PROJECT_DIR"

echo "🔍 Daily infrastructure drift check - $(date)"

# Scan current state
wgo scan --provider terraform --quiet

# Check for drift against production baseline
if ! wgo diff --baseline "$BASELINE" --quiet; then
    echo "⚠️ Infrastructure drift detected!"
    
    # Generate detailed report
    wgo diff --baseline "$BASELINE" --format markdown > drift-report.md
    
    # Send alert to Slack
    MESSAGE="🚨 Infrastructure drift detected in webapp production environment. Check drift-report.md for details."
    curl -X POST -H 'Content-type: application/json' \
        --data "{\"text\":\"$MESSAGE\"}" \
        "$SLACK_WEBHOOK"
    
    # Optionally fail the check
    exit 1
else
    echo "✅ No drift detected. Infrastructure is in compliance."
fi
```

**2. Cron Job Setup:**
```bash
# Add to crontab (runs daily at 9 AM)
0 9 * * * /home/deploy/scripts/check-drift.sh >> /var/log/drift-check.log 2>&1
```

**3. CI/CD Integration (`.github/workflows/drift-check.yml`):**
```yaml
name: Infrastructure Drift Check

on:
  schedule:
    # Run twice daily
    - cron: '0 6,18 * * *'
  workflow_dispatch:

jobs:
  drift-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install WGO
        run: curl -sSL https://install.wgo.sh | bash
      
      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1
      
      - name: Check for infrastructure drift
        run: |
          cd terraform/
          wgo scan --provider terraform
          wgo diff --baseline webapp-prod-v1.0 --format json > drift-report.json
      
      - name: Upload drift report
        uses: actions/upload-artifact@v3
        if: failure()
        with:
          name: drift-report
          path: drift-report.json
      
      - name: Notify on drift
        if: failure()
        uses: 8398a7/action-slack@v3
        with:
          status: failure
          text: "🚨 Infrastructure drift detected in production environment"
        env:
          SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```

## Advanced Drift Management

### Multiple Environment Tracking

**Directory Structure:**
```
webapp-infrastructure/
├── environments/
│   ├── prod/
│   │   └── terraform.tfstate
│   ├── staging/
│   │   └── terraform.tfstate
│   └── dev/
│       └── terraform.tfstate
├── scripts/
│   └── check-all-envs.sh
└── baselines/
    ├── prod-baseline.json
    ├── staging-baseline.json
    └── dev-baseline.json
```

**Multi-Environment Check Script:**
```bash
#!/bin/bash

ENVIRONMENTS=("prod" "staging" "dev")
DRIFT_DETECTED=false

for env in "${ENVIRONMENTS[@]}"; do
    echo "🔍 Checking $env environment..."
    
    cd "environments/$env"
    
    # Scan current state
    wgo scan --provider terraform --snapshot-name "$env-daily-$(date +%Y%m%d)"
    
    # Check against baseline
    if ! wgo diff --baseline "$env-baseline" --quiet; then
        echo "⚠️ Drift detected in $env environment"
        DRIFT_DETECTED=true
        
        # Generate environment-specific report
        wgo diff --baseline "$env-baseline" --format json > "../../reports/$env-drift-$(date +%Y%m%d).json"
    else
        echo "✅ $env environment is clean"
    fi
    
    cd - > /dev/null
done

if [ "$DRIFT_DETECTED" = true ]; then
    echo "🚨 Drift detected in one or more environments"
    exit 1
else
    echo "✅ All environments are in compliance"
fi
```

### Change Tracking and Approval

**Track Changes Over Time:**
```bash
# Create dated snapshots before major changes
wgo scan --provider terraform --snapshot-name "pre-release-v2.1-$(date +%Y%m%d)"

# After deployment
wgo scan --provider terraform --snapshot-name "post-release-v2.1-$(date +%Y%m%d)"

# Compare the deployment impact
wgo diff pre-release-v2.1-20240115 post-release-v2.1-20240115 --format markdown > deployment-impact.md
```

**Approval Workflow:**
```bash
#!/bin/bash
# pre-deployment-check.sh

echo "🔍 Pre-deployment infrastructure check"

# Scan current state
wgo scan --provider terraform

# Check for any unexpected drift
if ! wgo diff --baseline "approved-production-state" --quiet; then
    echo "❌ Unexpected drift detected before deployment!"
    echo "Please resolve drift before deploying new changes."
    
    wgo diff --baseline "approved-production-state" --format table
    exit 1
fi

# Validate Terraform plan
terraform plan -out=deployment.tfplan

# Show what will change
echo "📋 Planned infrastructure changes:"
terraform show -json deployment.tfplan | jq '.planned_values.root_module.resources[]'

# Require manual approval
read -p "Proceed with deployment? (yes/no): " response
if [ "$response" != "yes" ]; then
    echo "Deployment cancelled by user"
    exit 1
fi

echo "✅ Pre-deployment checks passed"
```

## Best Practices

### 1. Baseline Management
```bash
# Update baselines after approved changes
terraform apply
wgo scan --provider terraform
wgo baseline create --name "webapp-prod-v1.1" \
  --description "Updated after feature X deployment"

# Archive old baselines
wgo baseline delete "webapp-prod-v1.0"
```

### 2. Configuration Drift Prevention
```bash
# Use remote state locking
# In your Terraform configuration:
terraform {
  backend "s3" {
    bucket         = "webapp-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-locks"
    encrypt        = true
  }
}

# Implement state file monitoring
wgo watch --provider terraform --interval 300s
```

### 3. Incident Response
```bash
# Emergency drift check during incident
wgo scan --provider terraform --snapshot-name "incident-$(date +%Y%m%d-%H%M)"
wgo diff --baseline "last-known-good-state" --severity high

# Quick rollback verification
terraform plan -destroy -target=aws_instance.problematic_instance
wgo scan --provider terraform --snapshot-name "after-rollback"
```

### 4. Reporting and Documentation
```bash
# Generate weekly drift report
wgo diff --from "7 days ago" --to "now" --format markdown > weekly-changes.md

# Track resource growth
wgo scan --format json | jq '.metadata.resource_count' >> resource-count-history.txt

# Export for compliance audits
wgo baseline show prod-baseline --format json > compliance-baseline-$(date +%Y%m%d).json
```

This comprehensive example shows how WGO can be integrated into a real-world Terraform workflow to maintain infrastructure compliance and quickly detect unauthorized changes.