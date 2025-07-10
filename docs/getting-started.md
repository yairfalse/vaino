# Getting Started with VAINO

**VAINO** (*"What's Going On?"*) is the infrastructure drift detection tool that works like `git diff` for your infrastructure. This guide will get you up and running in under 5 minutes.

## Prerequisites

Before you begin, make sure you have VAINO installed. See the [Installation Guide](installation.md) for all installation methods.

**Quick install:**
```bash
curl -sSL https://install.wgo.sh | bash
```

**Verify installation:**
```bash
wgo version
wgo --help
```

## First Scan - 30 Seconds

### Auto-Discovery (Easiest)
VAINO automatically detects your infrastructure:

```bash
wgo scan
```

This will:
- Find Terraform state files in current directory
- Detect cloud provider configurations  
- Scan available infrastructure
- Create your first snapshot

### Provider-Specific Scans

**Terraform:**
```bash
wgo scan --provider terraform
```

**GCP:**
```bash
wgo scan --provider gcp --project my-project-123
```

**AWS:**
```bash
wgo scan --provider aws --region us-east-1
```

**Kubernetes:**
```bash
wgo scan --provider kubernetes --namespace default
```

## See What Changed

The magic happens with drift detection:

```bash
# See what changed since last scan
wgo diff

# Compare specific snapshots
wgo diff snapshot-1 snapshot-2

# Show statistics only
wgo diff --stat
```

## Example Output

```bash
$ wgo diff

Infrastructure Changes
=====================
📊 Comparing: scan-2024-01-15 → scan-2024-01-16
🏗️  Provider: terraform
⏱️  Duration: 1.2s

Changes detected: 3 resources

┌─────────────────────┬─────────┬──────────┬─────────────────────────┐
│ Resource            │ Change  │ Severity │ Details                 │
├─────────────────────┼─────────┼──────────┼─────────────────────────┤
│ aws_instance.web    │ MODIFY  │ HIGH     │ instance_type:          │
│                     │         │          │   t3.medium → t3.large  │
├─────────────────────┼─────────┼──────────┼─────────────────────────┤
│ aws_s3_bucket.data  │ MODIFY  │ MEDIUM   │ versioning: false →     │
│                     │         │          │ true                    │
├─────────────────────┼─────────┼──────────┼─────────────────────────┤
│ aws_rds_instance.db │ CREATE  │ HIGH     │ New database instance   │
└─────────────────────┴─────────┴──────────┴─────────────────────────┘

Exit code: 1 (changes detected)

💡 Run 'wgo explain' for AI-powered analysis
📖 Run 'wgo help' for more options
```

## Common First Steps

### 1. Create a Baseline
Save your current infrastructure state as a reference point:

```bash
wgo scan --snapshot-name "production-baseline-$(date +%Y%m%d)"
```

### 2. Set Up Daily Monitoring
Check for drift every day:

```bash
# Add to cron or CI/CD
wgo diff --quiet && echo "✅ No changes" || echo "⚠️ Drift detected"
```

### 3. Configure for Your Environment
Create a config file for your team:

```bash
wgo configure
# Interactive wizard to set up providers, regions, etc.
```

## What's Next?

### Essential Commands
- `wgo scan` - Capture current infrastructure state
- `wgo diff` - See what changed  
- `wgo status` - Check provider connectivity
- `wgo configure` - Set up configuration

### Learn More
- [Installation Methods](installation.md) - All installation options
- [Configuration](configuration.md) - Detailed configuration guide
- [Commands Reference](commands.md) - Complete command documentation
- [Examples](examples/) - Real-world usage examples

### Get Help
- `wgo help [command]` - Built-in help
- [Troubleshooting](troubleshooting.md) - Common issues
- [FAQ](faq.md) - Frequently asked questions

## 5-Minute Tutorial

Let's walk through a complete example:

```bash
# 1. Navigate to your infrastructure directory
cd my-terraform-project

# 2. Take first snapshot
wgo scan --snapshot-name "initial-state"

# 3. Make some changes to your infrastructure
terraform apply

# 4. See what changed
wgo diff

# 5. Get AI analysis of changes
wgo explain

# 6. Create new baseline after confirming changes
wgo scan --snapshot-name "post-deployment-$(date +%Y%m%d)"
```

## Key Benefits

✅ **Instant Setup** - Works without configuration  
✅ **Familiar Interface** - Like `git diff` but for infrastructure  
✅ **Multi-Provider** - Terraform, AWS, GCP, Kubernetes  
✅ **Professional Output** - Clean, scriptable, no emojis in CI  
✅ **Actionable Errors** - Clear guidance when things go wrong  

## Need Help?

- **Quick Help**: `wgo help`
- **Command Help**: `wgo [command] --help`
- **Documentation**: All guides in [`docs/`](.)
- **Issues**: [GitHub Issues](https://github.com/yairfalse/vaino/issues)

---

**Next:** [Installation Methods →](installation.md)