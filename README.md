# WGO - Git Diff for Infrastructure

**Simple infrastructure drift detection that actually makes sense.**

Think `git diff` but for your infrastructure - see what changed, when it changed, and get clear explanations.

## 🚀 Quick Start

```bash
# See what changed in your infrastructure
wgo diff

# List just the changed resources  
wgo diff --name-only

# Show change statistics
wgo diff --stat

# Silent mode for scripts (exit code: 0=no changes, 1=changes)
wgo diff --quiet
```

## 💡 Why WGO?

- **Familiar** - Works like `git diff` you already know
- **Simple** - No complex configuration needed
- **Scriptable** - Perfect for CI/CD and automation
- **Clear** - Shows exactly what changed, no confusing scores

## 📋 Examples

### Basic Usage
```bash
# See infrastructure changes (git diff style)
$ wgo diff
--- aws_instance/i-1234567890abcdef0
+++ aws_instance/i-1234567890abcdef0
@@ instance_type @@
-instance_type: t2.micro
+instance_type: t2.small
```

### List Changed Resources
```bash
$ wgo diff --name-only
aws_instance/i-1234567890abcdef0
aws_security_group/sg-0123456789abcdef0
```

### Change Statistics
```bash
$ wgo diff --stat
 aws_instance/i-1234567890abcdef0 | 1 change
 2 resources changed, 2 modifications
```

### Use in Scripts
```bash
# Check for drift and alert if found
if ! wgo diff --quiet; then
    echo "⚠️ Infrastructure drift detected!"
    wgo diff --stat
    exit 1
fi
echo "✅ Infrastructure is clean"
```

## 🔧 Installation

```bash
# Download latest release
curl -L https://github.com/yairfalse/wgo/releases/latest/download/wgo -o wgo
chmod +x wgo

# Or build from source
git clone https://github.com/yairfalse/wgo.git
cd wgo
go build -o wgo ./cmd/wgo
```

## 🎯 Getting Started

1. **See what changed**: `wgo diff`
2. **Scan your infrastructure**: `wgo scan` 
3. **Auto-setup**: `wgo setup`

## 🤝 Works With

- ✅ Terraform state files
- ✅ AWS resources
- ✅ Kubernetes clusters  
- ✅ Unix tools and scripts
- ✅ CI/CD pipelines

## 📖 More Help

```bash
wgo --help           # General help
wgo diff --help      # Diff command help
wgo setup --help     # Setup help
```

---

**Goal**: Make infrastructure drift detection as simple and familiar as `git diff`.