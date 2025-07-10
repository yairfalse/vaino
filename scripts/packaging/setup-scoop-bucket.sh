#!/bin/bash
# Setup Scoop Bucket Repository

set -e

REPO_OWNER="yairfalse"
REPO_NAME="scoop-wgo"

echo "🪣 Setting up Scoop Bucket Repository"
echo "====================================="

# Create repository if it doesn't exist
if ! gh repo view "$REPO_OWNER/$REPO_NAME" >/dev/null 2>&1; then
    echo "Creating Scoop bucket repository..."
    gh repo create "$REPO_OWNER/$REPO_NAME" --public --description "Scoop bucket for WGO"
    echo "Repository created: https://github.com/$REPO_OWNER/$REPO_NAME"
else
    echo "Repository already exists: https://github.com/$REPO_OWNER/$REPO_NAME"
fi

# Create initial README
cat > /tmp/README.md << 'EOF'
# WGO Scoop Bucket

This is the official Scoop bucket for WGO (What's Going On).

## Installation

```powershell
scoop bucket add wgo https://github.com/yairfalse/scoop-wgo
scoop install wgo
```

## Available Apps

- `wgo` - Git diff for infrastructure

## Repository

This bucket is automatically updated when new releases are published.

For more information, visit: https://github.com/yairfalse/vaino
EOF

echo "Setup complete! Next steps:"
echo "1. Add SCOOP_GITHUB_TOKEN to GitHub secrets"
echo "2. The bucket will be automatically updated on release"