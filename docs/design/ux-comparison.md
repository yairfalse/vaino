# VAINO User Experience Comparison

## Scenario 1: First-Time User Checking for Drift

### Current Experience (Confusing)
```bash
$ wgo check
❌ No Baselines Found
====================

You need to create a baseline first!

🎯 DO THIS NOW:

  1. Scan your infrastructure (if not done already):
     wgo scan --provider terraform

  2. Create a baseline:
     wgo baseline create --name prod-baseline

  3. Then check for drift:
     wgo check
```

**Problems:**
- User must understand "baseline" concept
- Multiple steps required
- Not intuitive what a baseline is or why it's needed

### New Experience (Intuitive)
```bash
$ wgo drift
ℹ️  No previous state found for comparison

💡 TIP: Save a reference state first:
    wgo scan --save
```

**Benefits:**
- Clear, simple message
- Only one additional step needed
- No confusing terminology

## Scenario 2: Daily Drift Check

### Current Experience
```bash
# Must remember baseline name
$ wgo check --baseline prod-baseline-2025-01-14

# Or hope the "latest" baseline is what you want
$ wgo check
```

### New Experience
```bash
# Just run drift - automatically uses last saved state
$ wgo drift
🔍 Comparing to: 18 hours ago (auto-selected)
⚠️  Drift detected in 3 resources
```

## Scenario 3: Creating Reference Points

### Current Experience
```bash
# Scan first
$ wgo scan --provider terraform
✅ Collection completed
📋 Snapshot ID: snapshot-1234567890

# Then create baseline (must remember snapshot ID or rely on defaults)
$ wgo baseline create --name prod-v1.0 --description "Production baseline v1.0"

# Now can check drift
$ wgo diff --baseline prod-v1.0
```

### New Experience
```bash
# Single command
$ wgo scan --save-as prod-v1.0
✅ Scanned 142 resources
💾 Saved as: prod-v1.0
⚠️  Drift detected in 3 resources (compared to previous scan)
```

## Scenario 4: Historical Comparisons

### Current Experience
```bash
# Must know exact baseline names
$ wgo baseline list
$ wgo diff --baseline prod-baseline-2025-01-10

# No time-based queries
# No relative references
```

### New Experience
```bash
# Natural time-based queries
$ wgo drift --since yesterday
$ wgo drift --since "last week"
$ wgo drift --since 2025-01-10

# Or use saved names
$ wgo drift --since prod-release-v2.1
```

## Scenario 5: Quick Status Check

### Current Experience
```bash
# No quick way to check drift status
# Must run full check or diff command
$ wgo check
# ... lots of output ...
```

### New Experience
```bash
# Quick, quiet check (perfect for CI/CD)
$ wgo drift --quiet
$ echo $?
1  # Exit code indicates drift detected

# Or get just summary
$ wgo drift --summary
⚠️  3 resources drifted (2 modified, 1 added)
```

## Scenario 6: Comparing Specific States

### Current Experience
```bash
# Must use snapshots files or baseline names
$ wgo diff --from snapshot-1.json --to snapshot-2.json

# Or baseline to snapshot
$ wgo diff --baseline prod --to snapshot-new.json
```

### New Experience
```bash
# Consistent interface for all comparisons
$ wgo drift --from prod-v1 --to prod-v2
$ wgo drift --from yesterday --to today
$ wgo drift --from snapshot1.json --to snapshot2.json
```

## Command Comparison Summary

| Task | Current Commands | New Commands |
|------|-----------------|--------------|
| First scan | `wgo scan` | `wgo scan` |
| Save reference | `wgo scan` + `wgo baseline create --name X` | `wgo scan --save-as X` |
| Check drift | `wgo check` or `wgo diff --baseline X` | `wgo drift` |
| List saved states | `wgo baseline list` | `wgo snapshots list` |
| Compare states | `wgo diff --from X --to Y` | `wgo drift --from X --to Y` |
| Time-based comparison | Not supported | `wgo drift --since yesterday` |

## Mental Model Shift

### Current: Process-Oriented
```
Scan → Create Baseline → Compare to Baseline → Manage Baselines
```
Users must understand the entire workflow and terminology.

### New: Goal-Oriented
```
What changed? → wgo drift
```
Users can accomplish their goal with minimal understanding of internals.

## Benefits Summary

1. **Fewer Commands to Learn**: 3 main commands vs 5+
2. **Natural Language**: "drift since yesterday" vs "baseline management"
3. **Smart Defaults**: Auto-comparison to last state
4. **Progressive Disclosure**: Simple by default, powerful when needed
5. **CI/CD Friendly**: `--quiet` and `--fail-on-drift` flags
6. **Time Awareness**: Built-in time-based comparisons