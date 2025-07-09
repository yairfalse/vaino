# Multi-Cloud Infrastructure Monitoring

Real-world example of using WGO to monitor infrastructure across AWS, GCP, and Kubernetes in a multi-cloud environment.

## Scenario: Global SaaS Platform

Your company runs a global SaaS platform with:
- **AWS**: Primary compute and storage (US regions)
- **GCP**: Data analytics and machine learning (Global)
- **Kubernetes**: Container orchestration across both clouds
- **Terraform**: Infrastructure as Code for everything

## Architecture Overview

```
Global SaaS Platform
├── AWS (Primary)
│   ├── us-east-1 (Production)
│   ├── us-west-2 (DR/Backup)
│   └── eu-west-1 (EU customers)
├── GCP (Analytics)
│   ├── us-central1 (Data processing)
│   ├── europe-west1 (EU data)
│   └── asia-southeast1 (APAC)
└── Kubernetes
    ├── EKS clusters (AWS)
    └── GKE clusters (GCP)
```

## Initial Setup

### 1. Configure Multi-Cloud Access

**AWS Configuration:**
```bash
# Configure AWS profiles for different regions/accounts
aws configure --profile prod-us-east-1
aws configure --profile prod-us-west-2  
aws configure --profile prod-eu-west-1

# Set up cross-account roles
export AWS_PROFILE=prod-us-east-1
```

**GCP Configuration:**
```bash
# Set up service accounts for each project
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/prod-us-central1-sa.json"

# Configure different projects
gcloud config set project prod-data-platform-us
gcloud config set project prod-data-platform-eu
gcloud config set project prod-data-platform-asia
```

**Kubernetes Configuration:**
```bash
# Configure contexts for all clusters
kubectl config set-context prod-us-east-1-eks --cluster=prod-us-east-1-eks
kubectl config set-context prod-eu-west-1-eks --cluster=prod-eu-west-1-eks
kubectl config set-context prod-us-central1-gke --cluster=prod-us-central1-gke
kubectl config set-context prod-europe-west1-gke --cluster=prod-europe-west1-gke
```

### 2. WGO Multi-Cloud Configuration

**`~/.wgo/config.yaml`:**
```yaml
# Multi-cloud configuration
providers:
  terraform:
    state_paths:
      - "./infrastructure/aws"
      - "./infrastructure/gcp" 
      - "./infrastructure/kubernetes"
    remote_state:
      enabled: true
      backends: ["s3", "gcs"]
  
  aws:
    regions: ["us-east-1", "us-west-2", "eu-west-1"]
    profiles:
      us-east-1: "prod-us-east-1"
      us-west-2: "prod-us-west-2"
      eu-west-1: "prod-eu-west-1"
    include_services: ["ec2", "rds", "s3", "lambda", "iam"]
  
  gcp:
    projects:
      - "prod-data-platform-us"
      - "prod-data-platform-eu" 
      - "prod-data-platform-asia"
    regions: ["us-central1", "europe-west1", "asia-southeast1"]
    include_services: ["compute", "storage", "bigquery", "dataflow"]
  
  kubernetes:
    contexts:
      - "prod-us-east-1-eks"
      - "prod-eu-west-1-eks"
      - "prod-us-central1-gke"
      - "prod-europe-west1-gke"
    namespaces: ["default", "monitoring", "data-processing"]

# Global settings
output:
  format: "table"
  pretty: true

storage:
  base_path: "~/.wgo"
  retention_days: 90

# Drift detection settings
drift:
  sensitivity: "medium"
  ignore_patterns:
    - "*.last_modified"
    - "*.created_at"
    - "tags.LastUpdated"
```

### 3. Initial Infrastructure Scan

```bash
# Scan all providers to establish baseline
wgo scan --all --snapshot-name "initial-multi-cloud-baseline"
```

**Output:**
```
Infrastructure Scan - All Providers
===================================
Scanning providers in parallel...

✅ Terraform: 45 resources (3.2s)
   • AWS resources: 28
   • GCP resources: 12  
   • Kubernetes resources: 5

✅ AWS Direct: 67 resources (4.1s)
   • us-east-1: 32 resources
   • us-west-2: 18 resources
   • eu-west-1: 17 resources

✅ GCP Direct: 34 resources (2.8s)
   • prod-data-platform-us: 15 resources
   • prod-data-platform-eu: 11 resources
   • prod-data-platform-asia: 8 resources

✅ Kubernetes: 89 resources (5.2s)
   • prod-us-east-1-eks: 25 resources
   • prod-eu-west-1-eks: 22 resources
   • prod-us-central1-gke: 23 resources
   • prod-europe-west1-gke: 19 resources

Total Resources: 235
Scan Duration: 5.2s (parallel execution)
Snapshot ID: multi-cloud-1751981234
```

### 4. Create Regional Baselines

```bash
# Create baselines for each major region/cloud
wgo baseline create --name "aws-us-baseline" \
  --description "AWS US regions baseline" \
  --tags "cloud=aws,region=us,environment=prod"

wgo baseline create --name "aws-eu-baseline" \
  --description "AWS EU region baseline" \
  --tags "cloud=aws,region=eu,environment=prod"

wgo baseline create --name "gcp-global-baseline" \
  --description "GCP all regions baseline" \
  --tags "cloud=gcp,environment=prod"

wgo baseline create --name "k8s-all-clusters-baseline" \
  --description "All Kubernetes clusters baseline" \
  --tags "cloud=multi,platform=kubernetes,environment=prod"
```

## Multi-Cloud Drift Detection

### Daily Multi-Cloud Health Check

**`scripts/multi-cloud-drift-check.sh`:**
```bash
#!/bin/bash
set -e

# Multi-cloud drift detection script
TIMESTAMP=$(date +"%Y-%m-%d %H:%M:%S")
DRIFT_DETECTED=false
REPORT_DIR="./reports/$(date +%Y%m%d)"

mkdir -p "$REPORT_DIR"

echo "🌐 Multi-Cloud Infrastructure Drift Check - $TIMESTAMP"
echo "================================================================"

# Function to check individual cloud provider
check_provider() {
    local provider=$1
    local baseline=$2
    local description=$3
    
    echo "🔍 Checking $description..."
    
    if ! wgo diff --provider "$provider" --baseline "$baseline" --quiet; then
        echo "⚠️ Drift detected in $description"
        DRIFT_DETECTED=true
        
        # Generate detailed report
        wgo diff --provider "$provider" --baseline "$baseline" \
          --format json > "$REPORT_DIR/$provider-drift.json"
        
        # Generate human-readable report
        wgo diff --provider "$provider" --baseline "$baseline" \
          --format markdown > "$REPORT_DIR/$provider-drift.md"
          
        return 1
    else
        echo "✅ $description is compliant"
        return 0
    fi
}

# Check each cloud provider
check_provider "aws" "aws-us-baseline" "AWS US Regions"
check_provider "gcp" "gcp-global-baseline" "GCP Global Infrastructure"
check_provider "kubernetes" "k8s-all-clusters-baseline" "Kubernetes Clusters"

# Cross-cloud correlation check
echo ""
echo "🔗 Cross-Cloud Correlation Analysis"
echo "==================================="

if [ "$DRIFT_DETECTED" = true ]; then
    echo "🔍 Analyzing cross-cloud changes for correlation..."
    
    # Combine all drift reports for correlation analysis
    jq -s 'add' "$REPORT_DIR"/*-drift.json > "$REPORT_DIR/combined-drift.json"
    
    # Look for correlated changes (same timestamp windows)
    python3 scripts/analyze-correlations.py "$REPORT_DIR/combined-drift.json" > \
      "$REPORT_DIR/correlation-analysis.txt"
    
    echo "📊 Cross-cloud correlation analysis complete"
fi

# Generate summary report
echo ""
echo "📋 Drift Detection Summary"
echo "========================="

if [ "$DRIFT_DETECTED" = true ]; then
    echo "🚨 Infrastructure drift detected across multiple clouds"
    echo "📁 Detailed reports available in: $REPORT_DIR"
    
    # Send notifications
    send_drift_notifications "$REPORT_DIR"
    
    exit 1
else
    echo "✅ All cloud infrastructure is in compliance"
    echo "📈 Total resources monitored: $(wgo scan --all --format json | jq '.metadata.resource_count')"
    exit 0
fi
```

### Advanced Cross-Cloud Correlation

**`scripts/analyze-correlations.py`:**
```python
#!/usr/bin/env python3
"""
Cross-cloud drift correlation analyzer
Identifies changes that might be related across different cloud providers
"""

import json
import sys
from datetime import datetime, timedelta
from collections import defaultdict

def parse_timestamp(ts_string):
    """Parse various timestamp formats"""
    formats = [
        "%Y-%m-%dT%H:%M:%S.%fZ",
        "%Y-%m-%dT%H:%M:%SZ", 
        "%Y-%m-%d %H:%M:%S"
    ]
    
    for fmt in formats:
        try:
            return datetime.strptime(ts_string, fmt)
        except ValueError:
            continue
    return None

def analyze_correlations(drift_data):
    """Analyze correlations between changes across clouds"""
    
    correlations = []
    changes_by_time = defaultdict(list)
    
    # Group changes by time windows (5-minute intervals)
    for change in drift_data.get('changes', []):
        timestamp = parse_timestamp(change.get('timestamp', ''))
        if timestamp:
            # Round to 5-minute intervals
            interval = timestamp.replace(minute=timestamp.minute//5*5, second=0, microsecond=0)
            changes_by_time[interval].append(change)
    
    # Look for correlations
    for interval, changes in changes_by_time.items():
        if len(changes) > 1:
            # Multiple changes in same time window - potential correlation
            clouds = set(change.get('provider', 'unknown') for change in changes)
            
            if len(clouds) > 1:
                correlations.append({
                    'timestamp': interval.isoformat(),
                    'clouds_affected': list(clouds),
                    'change_count': len(changes),
                    'changes': changes,
                    'correlation_type': determine_correlation_type(changes)
                })
    
    return correlations

def determine_correlation_type(changes):
    """Determine the type of correlation between changes"""
    
    resource_types = set(change.get('resource_type', '') for change in changes)
    change_types = set(change.get('change_type', '') for change in changes)
    
    if 'deployment' in ' '.join(resource_types).lower():
        return "deployment_related"
    elif 'scaling' in ' '.join(str(change.get('details', '')) for change in changes).lower():
        return "scaling_event"
    elif len(change_types) == 1 and 'CREATE' in change_types:
        return "coordinated_provisioning"
    elif 'security' in ' '.join(resource_types).lower():
        return "security_update"
    else:
        return "unknown_correlation"

def main():
    if len(sys.argv) != 2:
        print("Usage: analyze-correlations.py <drift-report.json>")
        sys.exit(1)
    
    with open(sys.argv[1], 'r') as f:
        drift_data = json.load(f)
    
    correlations = analyze_correlations(drift_data)
    
    print("Cross-Cloud Correlation Analysis")
    print("=" * 50)
    
    if not correlations:
        print("No cross-cloud correlations detected.")
        return
    
    for i, correlation in enumerate(correlations, 1):
        print(f"\nCorrelation #{i}:")
        print(f"  Time: {correlation['timestamp']}")
        print(f"  Clouds: {', '.join(correlation['clouds_affected'])}")
        print(f"  Changes: {correlation['change_count']}")
        print(f"  Type: {correlation['correlation_type']}")
        
        print("  Details:")
        for change in correlation['changes']:
            provider = change.get('provider', 'unknown')
            resource = change.get('resource', 'unknown')
            change_type = change.get('change_type', 'unknown')
            print(f"    - {provider}: {resource} ({change_type})")

if __name__ == "__main__":
    main()
```

## Provider-Specific Monitoring

### AWS Multi-Region Monitoring

```bash
#!/bin/bash
# aws-regional-drift-check.sh

AWS_REGIONS=("us-east-1" "us-west-2" "eu-west-1")
AWS_PROFILES=("prod-us-east-1" "prod-us-west-2" "prod-eu-west-1")

echo "🔍 AWS Multi-Region Drift Check"

for i in "${!AWS_REGIONS[@]}"; do
    region="${AWS_REGIONS[$i]}"
    profile="${AWS_PROFILES[$i]}"
    
    echo "Checking AWS $region (profile: $profile)..."
    
    # Set AWS profile for this region
    export AWS_PROFILE="$profile"
    
    # Scan specific region
    wgo scan --provider aws --region "$region" \
      --snapshot-name "aws-$region-$(date +%Y%m%d)"
    
    # Check against region-specific baseline
    if ! wgo diff --baseline "aws-$region-baseline" --quiet; then
        echo "⚠️ Drift detected in AWS $region"
        
        # Generate region-specific report
        wgo diff --baseline "aws-$region-baseline" \
          --format json > "reports/aws-$region-drift.json"
        
        # Check for cross-region impact
        check_cross_region_impact "$region"
    else
        echo "✅ AWS $region is compliant"
    fi
done
```

### GCP Multi-Project Monitoring

```bash
#!/bin/bash
# gcp-multi-project-drift-check.sh

GCP_PROJECTS=(
    "prod-data-platform-us:us-central1"
    "prod-data-platform-eu:europe-west1"
    "prod-data-platform-asia:asia-southeast1"
)

echo "🔍 GCP Multi-Project Drift Check"

for project_region in "${GCP_PROJECTS[@]}"; do
    IFS=':' read -r project region <<< "$project_region"
    
    echo "Checking GCP $project in $region..."
    
    # Scan specific project and region
    wgo scan --provider gcp \
      --project "$project" \
      --region "$region" \
      --snapshot-name "gcp-$project-$(date +%Y%m%d)"
    
    # Check against project-specific baseline
    if ! wgo diff --baseline "gcp-$project-baseline" --quiet; then
        echo "⚠️ Drift detected in GCP $project"
        
        # Generate project-specific report
        wgo diff --baseline "gcp-$project-baseline" \
          --format json > "reports/gcp-$project-drift.json"
        
        # Check for data pipeline impacts
        if [[ "$project" == *"data-platform"* ]]; then
            check_data_pipeline_impact "$project" "$region"
        fi
    else
        echo "✅ GCP $project is compliant"
    fi
done
```

### Kubernetes Multi-Cluster Monitoring

```bash
#!/bin/bash
# k8s-multi-cluster-drift-check.sh

K8S_CLUSTERS=(
    "prod-us-east-1-eks:aws:us-east-1"
    "prod-eu-west-1-eks:aws:eu-west-1"  
    "prod-us-central1-gke:gcp:us-central1"
    "prod-europe-west1-gke:gcp:europe-west1"
)

echo "🔍 Kubernetes Multi-Cluster Drift Check"

for cluster_info in "${K8S_CLUSTERS[@]}"; do
    IFS=':' read -r cluster cloud region <<< "$cluster_info"
    
    echo "Checking K8s cluster $cluster ($cloud $region)..."
    
    # Scan specific cluster
    wgo scan --provider kubernetes \
      --context "$cluster" \
      --snapshot-name "k8s-$cluster-$(date +%Y%m%d)"
    
    # Check against cluster-specific baseline
    if ! wgo diff --baseline "k8s-$cluster-baseline" --quiet; then
        echo "⚠️ Drift detected in cluster $cluster"
        
        # Generate cluster-specific report
        wgo diff --baseline "k8s-$cluster-baseline" \
          --format json > "reports/k8s-$cluster-drift.json"
        
        # Check for cross-cluster impact
        check_cross_cluster_impact "$cluster" "$cloud" "$region"
    else
        echo "✅ Cluster $cluster is compliant"
    fi
done
```

## Automated Multi-Cloud Monitoring

### GitHub Actions Workflow

**`.github/workflows/multi-cloud-drift.yml`:**
```yaml
name: Multi-Cloud Drift Detection

on:
  schedule:
    # Run every 2 hours during business hours (UTC)
    - cron: '0 6-18/2 * * 1-5'
  workflow_dispatch:

jobs:
  multi-cloud-drift:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        provider: [aws, gcp, kubernetes]
      fail-fast: false
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Install WGO
        run: curl -sSL https://install.wgo.sh | bash
      
      - name: Configure AWS credentials
        if: matrix.provider == 'aws'
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: us-east-1
      
      - name: Configure GCP credentials
        if: matrix.provider == 'gcp'
        uses: google-github-actions/auth@v1
        with:
          credentials_json: ${{ secrets.GCP_SERVICE_ACCOUNT_KEY }}
      
      - name: Configure kubectl
        if: matrix.provider == 'kubernetes'
        uses: azure/k8s-set-context@v1
        with:
          method: kubeconfig
          kubeconfig: ${{ secrets.KUBECONFIG }}
      
      - name: Scan infrastructure
        run: |
          wgo scan --provider ${{ matrix.provider }} \
            --snapshot-name "github-actions-$(date +%Y%m%d-%H%M)"
      
      - name: Check for drift
        id: drift-check
        run: |
          case "${{ matrix.provider }}" in
            aws)
              baseline="aws-global-baseline"
              ;;
            gcp)
              baseline="gcp-global-baseline"
              ;;
            kubernetes)
              baseline="k8s-all-clusters-baseline"
              ;;
          esac
          
          if ! wgo diff --baseline "$baseline" --quiet; then
            echo "drift=true" >> $GITHUB_OUTPUT
            wgo diff --baseline "$baseline" \
              --format json > ${{ matrix.provider }}-drift-report.json
          else
            echo "drift=false" >> $GITHUB_OUTPUT
          fi
      
      - name: Upload drift report
        if: steps.drift-check.outputs.drift == 'true'
        uses: actions/upload-artifact@v3
        with:
          name: ${{ matrix.provider }}-drift-report
          path: ${{ matrix.provider }}-drift-report.json
      
      - name: Send Slack notification
        if: steps.drift-check.outputs.drift == 'true'
        uses: 8398a7/action-slack@v3
        with:
          status: failure
          text: "🚨 Infrastructure drift detected in ${{ matrix.provider }} environment"
          webhook_url: ${{ secrets.SLACK_WEBHOOK_URL }}

  correlation-analysis:
    needs: multi-cloud-drift
    runs-on: ubuntu-latest
    if: always()
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Download all drift reports
        uses: actions/download-artifact@v3
        with:
          path: ./drift-reports
      
      - name: Analyze cross-cloud correlations
        run: |
          python3 scripts/analyze-correlations.py ./drift-reports/ > \
            correlation-analysis.txt
      
      - name: Upload correlation analysis
        uses: actions/upload-artifact@v3
        with:
          name: correlation-analysis
          path: correlation-analysis.txt
```

## Best Practices for Multi-Cloud Monitoring

### 1. Coordinated Baseline Management

```bash
#!/bin/bash
# update-all-baselines.sh

echo "🔄 Coordinated baseline update across all clouds"

# Update after planned maintenance window
BASELINE_DATE=$(date +%Y%m%d)
DESCRIPTION="Post-maintenance baseline - $BASELINE_DATE"

# Scan all providers
wgo scan --all --snapshot-name "maintenance-complete-$BASELINE_DATE"

# Create new baselines for each provider
wgo baseline create --name "aws-baseline-$BASELINE_DATE" \
  --description "$DESCRIPTION" \
  --tags "type=post-maintenance,date=$BASELINE_DATE"

wgo baseline create --name "gcp-baseline-$BASELINE_DATE" \
  --description "$DESCRIPTION" \
  --tags "type=post-maintenance,date=$BASELINE_DATE"

wgo baseline create --name "k8s-baseline-$BASELINE_DATE" \
  --description "$DESCRIPTION" \
  --tags "type=post-maintenance,date=$BASELINE_DATE"

# Archive old baselines
ARCHIVE_DATE=$(date -d "30 days ago" +%Y%m%d)
wgo baseline delete "aws-baseline-$ARCHIVE_DATE" || true
wgo baseline delete "gcp-baseline-$ARCHIVE_DATE" || true
wgo baseline delete "k8s-baseline-$ARCHIVE_DATE" || true

echo "✅ Baseline update complete"
```

### 2. Cost Impact Analysis

```bash
#!/bin/bash
# cost-impact-analysis.sh

echo "💰 Analyzing cost impact of infrastructure changes"

# Check for cost-impacting changes
wgo diff --all --format json | \
  jq '.changes[] | select(.severity == "HIGH" and (.details | contains("instance_type") or contains("machine_type") or contains("replicas")))' > \
  cost-impacting-changes.json

if [ -s cost-impacting-changes.json ]; then
    echo "💸 Cost-impacting changes detected:"
    
    # Calculate estimated cost impact
    python3 scripts/calculate-cost-impact.py cost-impacting-changes.json
    
    # Send to finance team
    mail -s "Infrastructure Cost Impact Alert" finance@company.com < cost-impact-report.txt
fi
```

### 3. Security Compliance Monitoring

```bash
#!/bin/bash
# security-compliance-check.sh

echo "🔒 Multi-cloud security compliance check"

# Focus on security-sensitive resources
SECURITY_RESOURCES="SecurityGroup,Role,RoleBinding,Secret,IAMRole,IAMPolicy"

# Check each provider for security changes
for provider in aws gcp kubernetes; do
    wgo diff --provider "$provider" \
      --resource-type "$SECURITY_RESOURCES" \
      --severity high \
      --format json > "security-changes-$provider.json"
    
    if [ -s "security-changes-$provider.json" ]; then
        echo "🚨 Security changes detected in $provider"
        
        # Send to security team
        curl -X POST "$SECURITY_WEBHOOK_URL" \
          -H "Content-Type: application/json" \
          -d "{
            \"alert\": \"Security configuration changes detected\",
            \"provider\": \"$provider\",
            \"severity\": \"high\"
          }"
    fi
done
```

This comprehensive multi-cloud setup demonstrates how WGO can provide unified visibility and drift detection across complex, distributed cloud infrastructures while maintaining the ability to drill down into provider-specific details.