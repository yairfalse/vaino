#!/bin/bash
# Setup Homebrew Tap Repository

set -e

REPO_OWNER="yairfalse"
REPO_NAME="homebrew-wgo"

echo "🍺 Setting up Homebrew Tap Repository"
echo "====================================="

# Create repository if it doesn't exist
if ! gh repo view "$REPO_OWNER/$REPO_NAME" >/dev/null 2>&1; then
    echo "Creating Homebrew tap repository..."
    gh repo create "$REPO_OWNER/$REPO_NAME" --public --description "Homebrew tap for WGO"
    echo "Repository created: https://github.com/$REPO_OWNER/$REPO_NAME"
else
    echo "Repository already exists: https://github.com/$REPO_OWNER/$REPO_NAME"
fi

# Create initial README
cat > /tmp/README.md << 'EOF'
# WGO Homebrew Tap

This is the official Homebrew tap for WGO (What's Going On).

## Installation

```bash
brew tap yairfalse/wgo
brew install wgo
```

## Available Formulas

- `wgo` - Git diff for infrastructure

## Repository

This tap is automatically updated when new releases are published.

For more information, visit: https://github.com/yairfalse/wgo
EOF

echo "Setup complete! Next steps:"
echo "1. Add HOMEBREW_TAP_GITHUB_TOKEN to GitHub secrets"
echo "2. The tap will be automatically updated on release"