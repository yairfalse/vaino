#!/bin/bash

# Script to remove Datadog API key from git history
# This uses git filter-branch to remove the specific file

echo "=== Git Secret Removal Script ==="
echo "This will remove pkg/providers/datadog.go from git history"
echo "WARNING: This rewrites git history!"
echo ""

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Create a backup branch
echo "Creating backup branch..."
git branch backup-before-secret-removal

# Remove the file from all commits
echo "Removing pkg/providers/datadog.go from history..."
git filter-branch --force --index-filter \
    'git rm --cached --ignore-unmatch pkg/providers/datadog.go' \
    --prune-empty --tag-name-filter cat -- --all

# Clean up
echo "Cleaning up..."
rm -rf .git/refs/original/
git reflog expire --expire=now --all
git gc --prune=now --aggressive

echo ""
echo "✓ Secret removal complete!"
echo ""
echo "IMPORTANT NEXT STEPS:"
echo "1. Review the changes:"
echo "   git log --oneline --all --grep='datadog'"
echo ""
echo "2. Force push to GitHub (coordinate with team first!):"
echo "   git push origin --force --all"
echo "   git push origin --force --tags"
echo ""
echo "3. IMMEDIATELY revoke the exposed Datadog API key:"
echo "   - Log into Datadog"
echo "   - Navigate to Organization Settings > API Keys"
echo "   - Revoke the key starting with 'dd...'"
echo "   - Generate a new key"
echo ""
echo "4. Update GitHub secret scanning:"
echo "   - Go to Settings > Security > Secret scanning"
echo "   - Mark the alert as resolved"
echo ""
echo "5. Notify all team members to:"
echo "   - Delete their local copies"
echo "   - Re-clone the repository"
echo ""
echo "Backup branch created: backup-before-secret-removal"