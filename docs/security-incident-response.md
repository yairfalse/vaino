# Security Incident Response: Exposed API Key

## Summary

GitHub's secret scanning detected an exposed Datadog API key in the repository history. This document outlines the steps to remediate this security issue.

## Affected Secret

- **Type**: Datadog API Key
- **File**: `pkg/providers/datadog.go` (already removed from current code)
- **Status**: Exists in git history
- **GitHub Alert**: https://github.com/yairfalse/wgo/security/secret-scanning/1

## Immediate Actions Required

### 1. Revoke the Exposed Key

**Priority: CRITICAL - Do this immediately!**

1. Log into your Datadog account
2. Navigate to **Organization Settings** > **API Keys**
3. Find the exposed key (starting with `dd`)
4. Click **Revoke** to immediately disable the key
5. Generate a new API key for your applications

### 2. Remove Secret from Git History

We've created a script to remove the secret from git history:

```bash
# Run the removal script
./scripts/remove-datadog-secret.sh

# This will:
# - Create a backup branch
# - Remove pkg/providers/datadog.go from all commits
# - Clean up the repository
```

### 3. Force Push Changes

After running the script and reviewing the changes:

```bash
# Force push all branches
git push origin --force --all

# Force push all tags
git push origin --force --tags
```

### 4. Team Communication

Notify all team members to:

1. **Stop any ongoing work** and commit/stash changes
2. **Delete their local repository**
3. **Re-clone the repository**:
   ```bash
   cd ..
   rm -rf wgo
   git clone https://github.com/yairfalse/wgo.git
   cd wgo
   ```

### 5. Update Secret Storage

For future API keys:

1. **Never commit secrets to code**
2. Use environment variables:
   ```go
   apiKey := os.Getenv("DATADOG_API_KEY")
   ```
3. Or use a secrets management service
4. Add to `.gitignore`:
   ```
   # Secrets
   *.key
   *.pem
   .env
   .env.*
   ```

## Prevention Measures

### Pre-commit Hooks

Install a pre-commit hook to prevent secrets:

```bash
# Install pre-commit
brew install pre-commit

# Add to .pre-commit-config.yaml
repos:
  - repo: https://github.com/Yelp/detect-secrets
    rev: v1.4.0
    hooks:
      - id: detect-secrets
        args: ['--baseline', '.secrets.baseline']
```

### GitHub Secret Scanning

1. Enable secret scanning alerts in repository settings
2. Configure push protection to block commits with secrets
3. Set up webhook notifications for security alerts

## Verification

After completing the remediation:

1. Verify the secret is removed:
   ```bash
   git log --all -p | grep -i "dd[0-9a-f]\{32\}"
   # Should return no results
   ```

2. Check GitHub security tab to confirm alert is resolved

3. Test that the old API key no longer works

4. Ensure new API key is properly secured

## Lessons Learned

1. **Always use environment variables** for sensitive configuration
2. **Review code carefully** before committing
3. **Use secret scanning tools** in CI/CD pipeline
4. **Respond quickly** to security alerts

## Contact

If you need assistance with this process:
- Security Team: security@yourcompany.com
- DevOps Team: devops@yourcompany.com

---

Remember: The exposed key can be used by anyone who has accessed the repository history. **Revoke it immediately!**