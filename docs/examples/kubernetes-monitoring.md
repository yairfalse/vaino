# Kubernetes Infrastructure Monitoring

Real-world examples of using WGO to monitor Kubernetes infrastructure changes and detect configuration drift.

## Scenario: Multi-Tenant SaaS Platform

You're running a SaaS platform on Kubernetes with:
- Multiple customer namespaces
- Shared services (monitoring, logging, ingress)
- Critical workloads (API, database, cache)
- Different environments (prod, staging, dev)

## Initial Setup

### 1. Configure WGO for Kubernetes
```bash
# Configure WGO for your cluster
wgo configure kubernetes
```

**Interactive Configuration:**
```
🔧 Kubernetes Configuration
==========================

Current kubeconfig: ~/.kube/config
Available contexts:
[1] production-cluster
[2] staging-cluster  
[3] development-cluster

Select context to scan: 1

Available namespaces in production-cluster:
[✓] default
[✓] kube-system
[✓] monitoring
[✓] customer-a
[✓] customer-b
[✓] api-services
[ ] kube-public (excluded by default)

Configuration saved to ~/.wgo/config.yaml
```

### 2. Initial Infrastructure Scan
```bash
# Scan production cluster
wgo scan --provider kubernetes --context production-cluster
```

**Output:**
```
Infrastructure Scan
===================
Context: production-cluster
Namespaces: default, kube-system, monitoring, customer-a, customer-b, api-services

Collecting resources from kubernetes...
Processed 47 resources in 2.1s

Collection completed in 2.3s
Snapshot ID: kubernetes-1751980456
Resources found: 47

Resource breakdown:
  - Deployment: 12
  - Service: 8
  - ConfigMap: 10
  - Secret: 6
  - Ingress: 4
  - PersistentVolumeClaim: 3
  - ServiceAccount: 4

Snapshot saved - use 'wgo diff' to detect changes
```

### 3. Create Environment Baselines
```bash
# Create baseline for production
wgo baseline create --name "k8s-prod-baseline" \
  --description "Production Kubernetes baseline" \
  --tags "environment=prod,cluster=production"

# Scan and baseline staging
wgo scan --provider kubernetes --context staging-cluster
wgo baseline create --name "k8s-staging-baseline" \
  --description "Staging Kubernetes baseline" \
  --tags "environment=staging,cluster=staging"
```

## Detecting Configuration Changes

### Scenario: Unauthorized Resource Scaling

A developer manually scaled a deployment, bypassing the normal GitOps process:

```bash
# Check for changes
wgo diff --provider kubernetes
```

**Output:**
```
Infrastructure Changes
=====================
📊 Comparing: k8s-prod-baseline → current scan
🏗️  Provider: kubernetes
⏱️  Duration: 1.8s

Changes detected: 3 resources

┌─────────────────────────────┬─────────┬──────────┬─────────────────────────┐
│ Resource                    │ Change  │ Severity │ Details                 │
├─────────────────────────────┼─────────┼──────────┼─────────────────────────┤
│ Deployment/api-backend      │ MODIFY  │ HIGH     │ replicas: 3 → 6         │
│ (namespace: api-services)   │         │          │ Resource scaling        │
│                             │         │          │ without approval        │
├─────────────────────────────┼─────────┼──────────┼─────────────────────────┤
│ ConfigMap/app-config        │ MODIFY  │ MEDIUM   │ data.LOG_LEVEL:         │
│ (namespace: api-services)   │         │          │ INFO → DEBUG            │
├─────────────────────────────┼─────────┼──────────┼─────────────────────────┤
│ Secret/database-credentials │ MODIFY  │ HIGH     │ data.password: CHANGED  │
│ (namespace: api-services)   │         │          │ Credential rotation     │
│                             │         │          │ detected                │
└─────────────────────────────┴─────────┴──────────┴─────────────────────────┘

Exit code: 1 (changes detected)
```

### Investigate Changes
```bash
# Get detailed analysis
wgo explain --provider kubernetes
```

**Output:**
```
🤖 Kubernetes Infrastructure Analysis
====================================

Critical Changes Detected:

🚨 Unauthorized Scaling (HIGH RISK)
   Resource: Deployment/api-backend in api-services namespace
   Change: Replica count increased from 3 to 6 (+100%)
   
   Impact Analysis:
   • Resource consumption doubled (CPU: 6 cores, Memory: 12GB)
   • Monthly cost increase: ~$240
   • Potential cascade effects on cluster capacity
   
   Investigation:
   • Change occurred: 23 minutes ago
   • No recent GitOps commits detected
   • kubectl edit or direct API call suspected
   
   Recommendations:
   1. Check cluster resource utilization
   2. Verify if scaling was justified by traffic
   3. Implement RBAC restrictions on deployment editing
   4. Set up resource quotas per namespace

🔑 Credential Rotation (HIGH IMPACT)
   Resource: Secret/database-credentials
   Change: Password field updated
   
   Analysis:
   • Standard credential rotation detected
   • Change aligns with security policy (monthly rotation)
   • No action required unless causing connectivity issues

⚠️ Debug Mode Enabled (MEDIUM RISK)  
   Resource: ConfigMap/app-config
   Change: LOG_LEVEL changed from INFO to DEBUG
   
   Impact:
   • Increased log verbosity
   • Potential performance impact
   • Sensitive data might be logged
   
   Recommendation:
   • Verify if debug logging is still needed
   • Monitor log volume and costs
   • Ensure no sensitive data in debug logs
```

### Remediate Issues
```bash
# Scale back the deployment
kubectl scale deployment api-backend --replicas=3 -n api-services

# Revert logging level
kubectl patch configmap app-config -n api-services \
  --patch '{"data":{"LOG_LEVEL":"INFO"}}'

# Verify changes are resolved
wgo diff --provider kubernetes
```

## Continuous Monitoring

### Real-Time Monitoring with Watch Mode
```bash
# Monitor production cluster in real-time
wgo watch --provider kubernetes --context production-cluster --interval 60s
```

**Output:**
```
🔍 WGO Watch Mode - Kubernetes Monitoring
=========================================
Context: production-cluster | Interval: 60s | Started: 2024-01-15 14:30:00

14:30:00 ✅ Scan completed - No changes (47 resources)
14:31:00 ✅ Scan completed - No changes (47 resources)  
14:32:00 ⚠️  Changes detected! 2 resources modified
         📊 Deployment/frontend: replicas 5 → 3
         📊 ConfigMap/feature-flags: data.NEW_FEATURE true → false
         🔗 Recent activity: kubectl apply detected (1 minute ago)
14:33:00 ✅ Scan completed - No changes (47 resources)

Press Ctrl+C to stop monitoring...
```

### Automated Alerts with Webhooks
```bash
# Monitor with Slack notifications
wgo watch --provider kubernetes \
  --webhook https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX \
  --interval 300s
```

**Webhook Configuration in `~/.wgo/config.yaml`:**
```yaml
providers:
  kubernetes:
    contexts: ["production-cluster"]
    namespaces: ["api-services", "customer-a", "customer-b"]

webhooks:
  enabled: true
  drift_detected:
    url: "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX"
    method: "POST"
    headers:
      Content-Type: "application/json"
    template: |
      {
        "text": "🚨 Kubernetes drift detected in {{ .Context }}",
        "attachments": [
          {
            "color": "danger",
            "fields": [
              {
                "title": "Changes",
                "value": "{{ .ChangeCount }} resources modified",
                "short": true
              },
              {
                "title": "Time", 
                "value": "{{ .Timestamp }}",
                "short": true
              }
            ]
          }
        ]
      }
```

## Multi-Environment Monitoring

### Environment-Specific Monitoring Script
```bash
#!/bin/bash
# k8s-drift-check.sh

ENVIRONMENTS=("production-cluster:prod" "staging-cluster:staging" "development-cluster:dev")
DRIFT_DETECTED=false

for env_pair in "${ENVIRONMENTS[@]}"; do
    IFS=':' read -r context env <<< "$env_pair"
    
    echo "🔍 Checking $env environment ($context)..."
    
    # Scan current state  
    wgo scan --provider kubernetes --context "$context" \
      --snapshot-name "$env-daily-$(date +%Y%m%d)"
    
    # Check against baseline
    if ! wgo diff --baseline "k8s-$env-baseline" --quiet; then
        echo "⚠️ Drift detected in $env environment"
        DRIFT_DETECTED=true
        
        # Generate detailed report
        wgo diff --baseline "k8s-$env-baseline" --format json > \
          "reports/$env-drift-$(date +%Y%m%d).json"
        
        # Send notification
        send_alert "$env" "$context"
    else
        echo "✅ $env environment is compliant"
    fi
done

if [ "$DRIFT_DETECTED" = true ]; then
    echo "🚨 Drift detected in one or more environments"
    exit 1
fi
```

### GitOps Integration

**ArgoCD Drift Detection (`.github/workflows/k8s-drift.yml`):**
```yaml
name: Kubernetes Drift Detection

on:
  schedule:
    - cron: '*/15 * * * *'  # Every 15 minutes
  workflow_dispatch:

jobs:
  detect-drift:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [production, staging]
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Install WGO
        run: curl -sSL https://install.wgo.sh | bash
      
      - name: Configure kubectl
        uses: azure/k8s-set-context@v1
        with:
          method: kubeconfig
          kubeconfig: ${{ secrets.KUBECONFIG }}
          context: ${{ matrix.environment }}-cluster
      
      - name: Scan Kubernetes infrastructure
        run: |
          wgo scan --provider kubernetes \
            --context ${{ matrix.environment }}-cluster \
            --snapshot-name "github-actions-$(date +%Y%m%d-%H%M)"
      
      - name: Check for drift
        id: drift-check
        run: |
          if ! wgo diff --baseline "k8s-${{ matrix.environment }}-baseline" --quiet; then
            echo "drift=true" >> $GITHUB_OUTPUT
            wgo diff --baseline "k8s-${{ matrix.environment }}-baseline" \
              --format json > drift-report.json
          else
            echo "drift=false" >> $GITHUB_OUTPUT
          fi
      
      - name: Create GitHub issue on drift
        if: steps.drift-check.outputs.drift == 'true'
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const driftReport = JSON.parse(fs.readFileSync('drift-report.json', 'utf8'));
            
            const title = `🚨 Kubernetes Drift Detected - ${{ matrix.environment }}`;
            const body = `
            ## Infrastructure Drift Detected
            
            **Environment:** ${{ matrix.environment }}
            **Time:** ${new Date().toISOString()}
            **Changes:** ${driftReport.summary.total_changes} resources modified
            
            ### Changes Detected:
            ${driftReport.changes.map(c => `- **${c.resource}**: ${c.change_type} (${c.severity})`).join('\n')}
            
            Please review and remediate these changes.
            `;
            
            github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title,
              body,
              labels: ['drift-detection', 'kubernetes', '${{ matrix.environment }}']
            });
```

## Namespace-Specific Monitoring

### Customer Namespace Isolation
```bash
# Monitor specific customer namespaces
for customer in customer-a customer-b customer-c; do
    echo "Checking $customer namespace..."
    
    wgo scan --provider kubernetes \
      --context production-cluster \
      --namespace "$customer" \
      --snapshot-name "$customer-$(date +%Y%m%d)"
    
    # Compare against customer-specific baseline
    if ! wgo diff --baseline "$customer-baseline" --quiet; then
        echo "⚠️ Changes detected in $customer namespace"
        
        # Notify customer via webhook
        curl -X POST "https://api.customer-portal.com/webhooks/$customer/drift" \
          -H "Content-Type: application/json" \
          -d '{"message": "Infrastructure changes detected in your namespace"}'
    fi
done
```

### Resource Quota Monitoring
```bash
#!/bin/bash
# quota-drift-check.sh

# Check if resource quotas have been modified
wgo scan --provider kubernetes --context production-cluster

# Focus on ResourceQuota and LimitRange objects
wgo diff --baseline prod-quota-baseline \
  --resource-type ResourceQuota,LimitRange \
  --format json > quota-changes.json

if [ -s quota-changes.json ]; then
    echo "🚨 Resource quota changes detected!"
    
    # Alert platform team
    jq '.changes[] | select(.resource_type | contains("ResourceQuota"))' \
      quota-changes.json | \
      mail -s "Resource Quota Changes Detected" platform-team@company.com
fi
```

## Security Monitoring

### RBAC Changes Detection
```bash
# Monitor RBAC configuration
wgo scan --provider kubernetes \
  --context production-cluster \
  --include-resources Role,RoleBinding,ClusterRole,ClusterRoleBinding

# Check for privilege escalation
wgo diff --baseline security-baseline \
  --resource-type Role,RoleBinding,ClusterRole,ClusterRoleBinding \
  --severity high
```

### Secret and ConfigMap Monitoring
```bash
#!/bin/bash
# security-drift-monitor.sh

echo "🔍 Security-focused drift detection"

# Scan secrets and configmaps
wgo scan --provider kubernetes \
  --context production-cluster \
  --include-resources Secret,ConfigMap

# Check for changes in sensitive resources
if ! wgo diff --baseline security-baseline \
  --resource-type Secret,ConfigMap \
  --quiet; then
    
    echo "🚨 Security-sensitive changes detected!"
    
    # Generate detailed security report
    wgo diff --baseline security-baseline \
      --resource-type Secret,ConfigMap \
      --format json > security-drift-report.json
    
    # Send to security team
    curl -X POST "$SECURITY_WEBHOOK_URL" \
      -H "Content-Type: application/json" \
      -d '{
        "alert": "Kubernetes security drift detected",
        "severity": "high",
        "resources": "secrets and configmaps",
        "cluster": "production-cluster"
      }'
    
    # Create security incident ticket
    create_security_incident "K8S-DRIFT" "Unauthorized changes to secrets/configmaps"
fi
```

## Performance Impact Monitoring

### Resource Usage Tracking
```bash
#!/bin/bash
# resource-impact-monitor.sh

# Scan before peak hours
wgo scan --provider kubernetes \
  --context production-cluster \
  --snapshot-name "pre-peak-$(date +%Y%m%d-%H%M)"

# Monitor during peak hours
while [ $(date +%H) -ge 09 ] && [ $(date +%H) -le 17 ]; do
    # Quick scan every 5 minutes during business hours
    wgo scan --provider kubernetes --quiet
    
    # Check for scaling events
    if ! wgo diff --quiet --from "5 minutes ago"; then
        echo "📊 Scaling event detected during peak hours"
        
        # Log the changes
        wgo diff --from "5 minutes ago" --format json >> \
          "logs/peak-hours-changes-$(date +%Y%m%d).log"
    fi
    
    sleep 300  # 5 minutes
done
```

### HPA and VPA Monitoring
```bash
# Monitor autoscaling components
wgo scan --provider kubernetes \
  --context production-cluster \
  --include-resources HorizontalPodAutoscaler,VerticalPodAutoscaler

# Track autoscaling decisions
wgo diff --baseline autoscaling-baseline \
  --resource-type HorizontalPodAutoscaler \
  --format json > hpa-changes.json

# Analyze scaling patterns
jq '.changes[] | select(.change_type == "MODIFY") | 
    select(.details | contains("replicas"))' hpa-changes.json
```

## Best Practices for Kubernetes Monitoring

### 1. Baseline Management
```bash
# Update baselines after planned deployments
kubectl apply -f deployment.yaml
wgo scan --provider kubernetes
wgo baseline create --name "post-deployment-$(date +%Y%m%d)" \
  --description "State after feature X deployment"
```

### 2. Selective Monitoring
```yaml
# ~/.wgo/config.yaml
providers:
  kubernetes:
    contexts: ["production-cluster"]
    # Monitor only critical namespaces
    namespaces: ["api-services", "database", "monitoring"]
    
    # Exclude noisy resources
    exclude_resources: ["Event", "Endpoints", "EndpointSlice"]
    
    # Focus on security-sensitive resources
    include_resources: [
      "Deployment", "Service", "Ingress", 
      "Secret", "ConfigMap", "Role", "RoleBinding"
    ]
```

### 3. Integration with Kubernetes Events
```bash
#!/bin/bash
# correlate-with-events.sh

# Capture current events
kubectl get events --all-namespaces \
  --sort-by='.lastTimestamp' > k8s-events.log

# Scan for changes
wgo scan --provider kubernetes

# Check for drift
if ! wgo diff --quiet; then
    echo "Drift detected - correlating with Kubernetes events..."
    
    # Look for recent events that might explain changes
    kubectl get events --all-namespaces \
      --field-selector involvedObject.kind=Deployment \
      --sort-by='.lastTimestamp' | tail -10
fi
```

This comprehensive example demonstrates how WGO can provide deep visibility into Kubernetes infrastructure changes, helping maintain security, compliance, and operational excellence in complex container environments.