#!/bin/bash
# Setup Homebrew Tap Repository

set -e

REPO_OWNER="yairfalse"
REPO_NAME="homebrew-vaino"

echo "🍺 Setting up Homebrew Tap Repository"
echo "====================================="

# Create repository if it doesn't exist
if ! gh repo view "$REPO_OWNER/$REPO_NAME" >/dev/null 2>&1; then
    echo "Creating Homebrew tap repository..."
    gh repo create "$REPO_OWNER/$REPO_NAME" --public --description "Homebrew tap for VAINO"
    echo "Repository created: https://github.com/$REPO_OWNER/$REPO_NAME"
else
    echo "Repository already exists: https://github.com/$REPO_OWNER/$REPO_NAME"
fi

# Create initial README
cat > /tmp/README.md << 'EOF'
# VAINO Homebrew Tap

This is the official Homebrew tap for VAINO (What's Going On).

## Installation

```bash
brew tap yairfalse/vaino
brew install vaino
```

## Available Formulas

- `vaino` - Git diff for infrastructure

## Repository

This tap is automatically updated when new releases are published.

For more information, visit: https://github.com/yairfalse/vaino
EOF

echo "Setup complete! Next steps:"
echo "1. Add HOMEBREW_TAP_GITHUB_TOKEN to GitHub secrets"
echo "2. The tap will be automatically updated on release"