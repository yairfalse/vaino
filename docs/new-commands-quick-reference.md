# WGO New Commands - Quick Reference

## Core Commands

### `wgo scan` - Scan Infrastructure
```bash
wgo scan                          # Scan and show drift (if previous state exists)
wgo scan --save                   # Scan and save as reference point
wgo scan --save-as prod-v1.2      # Scan and save with custom name
wgo scan --provider gcp           # Scan specific provider
wgo scan --snapshot-only          # Just scan, don't compare
```

### `wgo drift` - Detect Changes
```bash
wgo drift                         # Compare to last saved state
wgo drift --since prod-v1.0       # Compare to named reference
wgo drift --since yesterday       # Compare to time-based reference
wgo drift --since "3 days ago"    # Relative time
wgo drift --since 2025-01-15      # Specific date

wgo drift --from A --to B         # Compare two specific states
wgo drift --severity high         # Show only high severity changes
wgo drift --quiet                 # Exit code only (for CI/CD)
wgo drift --format json --output drift.json  # Export results
```

### `wgo snapshots` - Manage Saved States
```bash
wgo snapshots list                # List all saved states
wgo snapshots show prod-v1.0      # Show details of specific state
wgo snapshots delete old-state    # Delete a saved state
wgo snapshots prune --older-than 30d  # Clean up old states
wgo snapshots tag latest --as production  # Tag states for easy reference
```

## Common Workflows

### Daily Drift Check
```bash
# Morning standup - what changed overnight?
wgo drift --since yesterday

# Or just check latest
wgo drift
```

### Pre/Post Deployment
```bash
# Before deployment
wgo scan --save-as pre-deploy-v2.0

# After deployment
wgo drift --since pre-deploy-v2.0
```

### CI/CD Pipeline
```bash
# Fail pipeline if drift detected
wgo drift --quiet --fail-on-drift

# Generate drift report
wgo drift --format json --output drift-report.json
```

### Historical Analysis
```bash
# What changed this week?
wgo drift --since "last monday"

# Compare two releases
wgo drift --from v1.0-release --to v2.0-release

# Show drift over time
wgo snapshots list --since "last month"
```

## Flag Reference

### Global Flags (all commands)
- `--provider` - Filter by provider (aws, gcp, terraform, k8s)
- `--region` - Filter by region
- `--format` - Output format (table, json, yaml, markdown)
- `--output` - Save to file instead of stdout
- `--no-color` - Disable colored output
- `--verbose` - Detailed output

### Drift-Specific Flags
- `--since` - Reference point for comparison
- `--from/--to` - Compare two specific states  
- `--severity` - Minimum severity (low, medium, high, critical)
- `--ignore-tags` - Ignore tag-only changes
- `--ignore-fields` - Ignore specific fields
- `--summary` - Summary only, no details
- `--quiet` - Minimal output (exit code indicates drift)
- `--fail-on-drift` - Exit 1 if drift detected

### Scan-Specific Flags
- `--save` - Save scan as reference point
- `--save-as NAME` - Save with specific name
- `--snapshot-only` - Skip drift detection
- `--auto-discover` - Auto-discover infrastructure
- `--rescan` - Force fresh scan (ignore cache)

## Time Reference Examples

```bash
# Relative times
--since yesterday
--since "last week"  
--since "last month"
--since "3 days ago"
--since "12 hours ago"

# Absolute times
--since 2025-01-15
--since "2025-01-15 14:30"
--since "Jan 15, 2025"

# Named references
--since prod-v1.0
--since pre-migration
--since last-known-good
```

## Migration from Old Commands

| Old Command | New Command |
|------------|-------------|
| `wgo baseline create --name X` | `wgo scan --save-as X` |
| `wgo baseline list` | `wgo snapshots list` |
| `wgo diff --baseline X` | `wgo drift --since X` |
| `wgo check --baseline X` | `wgo drift --since X` |
| `wgo diff --from A --to B` | `wgo drift --from A --to B` |

## Quick Tips

1. **Default behavior is smart**: Just run `wgo drift` - it figures out what to compare
2. **Time-based queries are natural**: Use "yesterday", "last week" etc.
3. **Save states have names**: Use meaningful names like "prod-v1.2" or "pre-migration"
4. **Quiet mode for scripts**: Use `--quiet` flag and check exit codes
5. **Filter noise**: Use `--ignore-tags` or `--severity high` to focus on important changes