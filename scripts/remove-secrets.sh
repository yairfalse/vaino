#!/bin/bash

# Script to remove secrets from git history
# WARNING: This will rewrite git history!

echo "This script will remove sensitive data from git history."
echo "WARNING: This is a destructive operation that will rewrite history!"
echo "Make sure you have a backup and coordinate with your team."
echo ""
read -p "Are you sure you want to continue? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Operation cancelled."
    exit 1
fi

# Install BFG Repo-Cleaner if not already installed
if ! command -v bfg &> /dev/null; then
    echo "BFG Repo-Cleaner not found. Please install it first:"
    echo "  brew install bfg"
    echo "  or download from: https://rtyley.github.io/bfg-repo-cleaner/"
    exit 1
fi

# Create a backup
echo "Creating backup..."
cp -r .git .git-backup

# Remove the file from history
echo "Removing pkg/providers/datadog.go from history..."
bfg --delete-files datadog.go

# Clean up the repository
echo "Cleaning up repository..."
git reflog expire --expire=now --all
git gc --prune=now --aggressive

echo ""
echo "Secret removal complete!"
echo "Next steps:"
echo "1. Review the changes with: git log --oneline"
echo "2. Force push to all branches: git push --force-with-lease --all"
echo "3. Force push tags: git push --force-with-lease --tags"
echo "4. Contact all team members to re-clone the repository"
echo "5. Revoke the exposed Datadog API key immediately"
echo ""
echo "Backup saved in .git-backup"