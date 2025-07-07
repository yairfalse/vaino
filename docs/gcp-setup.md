# GCP Setup Guide for WGO

This guide will help you configure Google Cloud Platform (GCP) authentication for WGO.

## Prerequisites

1. A Google Cloud Platform account
2. A GCP project (you already have `taskmate-46a1721`)
3. `gcloud` CLI installed (optional but recommended)

## Authentication Methods

WGO supports two authentication methods for GCP:

### Method 1: Application Default Credentials (Recommended for Development)

1. **Install gcloud CLI** (if not already installed):
   ```bash
   # macOS
   brew install --cask google-cloud-sdk
   
   # Or download from: https://cloud.google.com/sdk/docs/install
   ```

2. **Authenticate with gcloud**:
   ```bash
   gcloud auth application-default login
   ```

3. **Set your project**:
   ```bash
   gcloud config set project taskmate-46a1721
   ```

4. **Run WGO scan**:
   ```bash
   ./wgo scan --provider gcp --project taskmate-46a1721
   ```

### Method 2: Service Account Key (Recommended for Production)

1. **Create a Service Account**:
   ```bash
   # Create service account
   gcloud iam service-accounts create wgo-scanner \
     --display-name="WGO Scanner" \
     --project=taskmate-46a1721
   ```

2. **Grant necessary permissions**:
   ```bash
   # Grant viewer role (read-only access)
   gcloud projects add-iam-policy-binding taskmate-46a1721 \
     --member="serviceAccount:wgo-scanner@taskmate-46a1721.iam.gserviceaccount.com" \
     --role="roles/viewer"
   
   # For more specific permissions, grant these roles:
   # - roles/compute.viewer (for Compute Engine resources)
   # - roles/storage.objectViewer (for Cloud Storage)
   # - roles/iam.viewer (for IAM resources)
   ```

3. **Create and download key**:
   ```bash
   gcloud iam service-accounts keys create ~/wgo-gcp-key.json \
     --iam-account=wgo-scanner@taskmate-46a1721.iam.gserviceaccount.com
   ```

4. **Use the key with WGO**:
   ```bash
   # Option 1: Pass credentials file directly
   ./wgo scan --provider gcp --project taskmate-46a1721 --credentials ~/wgo-gcp-key.json
   
   # Option 2: Set environment variable
   export GOOGLE_APPLICATION_CREDENTIALS=~/wgo-gcp-key.json
   ./wgo scan --provider gcp --project taskmate-46a1721
   ```

## Required Permissions

The GCP collector requires these permissions to scan resources:

- `compute.instances.list`
- `compute.disks.list`
- `compute.instanceGroups.list`
- `compute.networks.list`
- `compute.subnetworks.list`
- `compute.firewalls.list`
- `compute.regions.list`
- `compute.zones.list`
- `storage.buckets.list`
- `storage.buckets.get`
- `iam.serviceAccounts.list`
- `resourcemanager.projects.get`

## Troubleshooting

### Error: "The caller does not have permission"
- Ensure you're authenticated: `gcloud auth list`
- Check your project: `gcloud config get-value project`
- Verify permissions: `gcloud projects get-iam-policy taskmate-46a1721`

### Error: "Could not find default credentials"
- Run: `gcloud auth application-default login`
- Or set: `export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json`

### Error: "Invalid project ID"
- Verify project exists: `gcloud projects describe taskmate-46a1721`
- Ensure you have access: `gcloud projects list`

## Quick Test

Once authenticated, test with:

```bash
# Basic scan
./wgo scan --provider gcp --project taskmate-46a1721

# Scan specific regions
./wgo scan --provider gcp --project taskmate-46a1721 --region us-central1,us-east1

# Save output
./wgo scan --provider gcp --project taskmate-46a1721 --output-file gcp-snapshot.json
```

## Security Best Practices

1. **Use Service Accounts** for production/CI environments
2. **Grant minimal permissions** - use viewer roles when possible
3. **Rotate keys regularly** if using service account keys
4. **Store keys securely** - never commit them to version control
5. **Use Workload Identity** if running in GKE

## Example Output

When successfully authenticated, you should see:

```
🔍 Infrastructure Scan
=====================
📝 Auto-generated snapshot name: scan-wgo-2025-07-07-22-33
📊 Collecting resources from gcp...

✅ Collection completed in 5.2s
📋 Snapshot ID: gcp-1234567890
📊 Resources found: 42

📈 Resource breakdown:
  • instance: 15
  • disk: 20
  • network: 3
  • subnet: 4

💾 Snapshot ready for baseline/drift analysis
```