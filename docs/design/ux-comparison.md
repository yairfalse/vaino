# VAINO User Experience Comparison

## Scenario 1: First-Time User Checking for Drift

### Current Experience (Confusing)
```bash
$ vaino check
❌ No Baselines Found
====================

You need to create a baseline first!

🎯 DO THIS NOW:

  1. Scan your infrastructure (if not done already):
     vaino scan --provider terraform

  2. Create a baseline:
     vaino baseline create --name prod-baseline

  3. Then check for drift:
     vaino check
```

**Problems:**
- User must understand "baseline" concept
- Multiple steps required
- Not intuitive what a baseline is or why it's needed

### New Experience (Intuitive)
```bash
$ vaino drift
ℹ️  No previous state found for comparison

💡 TIP: Save a reference state first:
    vaino scan --save
```

**Benefits:**
- Clear, simple message
- Only one additional step needed
- No confusing terminology

## Scenario 2: Daily Drift Check

### Current Experience
```bash
# Must remember baseline name
$ vaino check --baseline prod-baseline-2025-01-14

# Or hope the "latest" baseline is what you want
$ vaino check
```

### New Experience
```bash
# Just run drift - automatically uses last saved state
$ vaino drift
🔍 Comparing to: 18 hours ago (auto-selected)
⚠️  Drift detected in 3 resources
```

## Scenario 3: Creating Reference Points

### Current Experience
```bash
# Scan first
$ vaino scan --provider terraform
✅ Collection completed
📋 Snapshot ID: snapshot-1234567890

# Then create baseline (must remember snapshot ID or rely on defaults)
$ vaino baseline create --name prod-v1.0 --description "Production baseline v1.0"

# Now can check drift
$ vaino diff --baseline prod-v1.0
```

### New Experience
```bash
# Single command
$ vaino scan --save-as prod-v1.0
✅ Scanned 142 resources
💾 Saved as: prod-v1.0
⚠️  Drift detected in 3 resources (compared to previous scan)
```

## Scenario 4: Historical Comparisons

### Current Experience
```bash
# Must know exact baseline names
$ vaino baseline list
$ vaino diff --baseline prod-baseline-2025-01-10

# No time-based queries
# No relative references
```

### New Experience
```bash
# Natural time-based queries
$ vaino drift --since yesterday
$ vaino drift --since "last week"
$ vaino drift --since 2025-01-10

# Or use saved names
$ vaino drift --since prod-release-v2.1
```

## Scenario 5: Quick Status Check

### Current Experience
```bash
# No quick way to check drift status
# Must run full check or diff command
$ vaino check
# ... lots of output ...
```

### New Experience
```bash
# Quick, quiet check (perfect for CI/CD)
$ vaino drift --quiet
$ echo $?
1  # Exit code indicates drift detected

# Or get just summary
$ vaino drift --summary
⚠️  3 resources drifted (2 modified, 1 added)
```

## Scenario 6: Comparing Specific States

### Current Experience
```bash
# Must use snapshots files or baseline names
$ vaino diff --from snapshot-1.json --to snapshot-2.json

# Or baseline to snapshot
$ vaino diff --baseline prod --to snapshot-new.json
```

### New Experience
```bash
# Consistent interface for all comparisons
$ vaino drift --from prod-v1 --to prod-v2
$ vaino drift --from yesterday --to today
$ vaino drift --from snapshot1.json --to snapshot2.json
```

## Command Comparison Summary

| Task | Current Commands | New Commands |
|------|-----------------|--------------|
| First scan | `vaino scan` | `vaino scan` |
| Save reference | `vaino scan` + `vaino baseline create --name X` | `vaino scan --save-as X` |
| Check drift | `vaino check` or `vaino diff --baseline X` | `vaino drift` |
| List saved states | `vaino baseline list` | `vaino snapshots list` |
| Compare states | `vaino diff --from X --to Y` | `vaino drift --from X --to Y` |
| Time-based comparison | Not supported | `vaino drift --since yesterday` |

## Mental Model Shift

### Current: Process-Oriented
```
Scan → Create Baseline → Compare to Baseline → Manage Baselines
```
Users must understand the entire workflow and terminology.

### New: Goal-Oriented
```
What changed? → vaino drift
```
Users can accomplish their goal with minimal understanding of internals.

## Benefits Summary

1. **Fewer Commands to Learn**: 3 main commands vs 5+
2. **Natural Language**: "drift since yesterday" vs "baseline management"
3. **Smart Defaults**: Auto-comparison to last state
4. **Progressive Disclosure**: Simple by default, powerful when needed
5. **CI/CD Friendly**: `--quiet` and `--fail-on-drift` flags
6. **Time Awareness**: Built-in time-based comparisons