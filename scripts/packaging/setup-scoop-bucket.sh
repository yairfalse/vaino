#!/bin/bash
# Setup Scoop Bucket Repository

set -e

REPO_OWNER="yairfalse"
REPO_NAME="scoop-vaino"

echo "🪣 Setting up Scoop Bucket Repository"
echo "====================================="

# Create repository if it doesn't exist
if ! gh repo view "$REPO_OWNER/$REPO_NAME" >/dev/null 2>&1; then
    echo "Creating Scoop bucket repository..."
    gh repo create "$REPO_OWNER/$REPO_NAME" --public --description "Scoop bucket for VAINO"
    echo "Repository created: https://github.com/$REPO_OWNER/$REPO_NAME"
else
    echo "Repository already exists: https://github.com/$REPO_OWNER/$REPO_NAME"
fi

# Create initial README
cat > /tmp/README.md << 'EOF'
# VAINO Scoop Bucket

This is the official Scoop bucket for VAINO (What's Going On).

## Installation

```powershell
scoop bucket add vaino https://github.com/yairfalse/scoop-vaino
scoop install vaino
```

## Available Apps

- `vaino` - Git diff for infrastructure

## Repository

This bucket is automatically updated when new releases are published.

For more information, visit: https://github.com/yairfalse/vaino
EOF

echo "Setup complete! Next steps:"
echo "1. Add SCOOP_GITHUB_TOKEN to GitHub secrets"
echo "2. The bucket will be automatically updated on release"