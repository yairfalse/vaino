# WGO Command Redesign Proposal

## Current Problems

The current command structure is confusing because:
1. Users must understand the "baseline" concept
2. Multiple commands (`diff`, `check`) do similar things
3. The workflow requires too many steps
4. Terminology doesn't match user mental models

## Proposed Command Structure

### Core Principle: Drift Detection First

Users think in terms of:
- "What changed?"
- "Show me the drift"
- "Compare to yesterday/last week/last deployment"

### New Command Structure

```bash
# Primary Commands
wgo scan           # Scan infrastructure and detect drift
wgo drift          # Show drift (replaces diff/check)
wgo snapshots      # Manage saved infrastructure states

# Secondary Commands  
wgo auth           # Authentication management
wgo explain        # AI-powered drift analysis
wgo version        # Version information
```

## Detailed Command Design

### 1. `wgo scan` - Simplified Scanning

```bash
# Basic scan with automatic drift detection
wgo scan
# → Scans infrastructure
# → Automatically compares to last saved state
# → Shows drift immediately

# Scan and save reference point
wgo scan --save
wgo scan --save-as prod-2025-01-15

# Scan specific providers
wgo scan --provider terraform
wgo scan --provider gcp --project my-project

# Scan without drift comparison (snapshot only)
wgo scan --snapshot-only
```

### 2. `wgo drift` - Primary Drift Detection Command

```bash
# Show drift from last saved state (most common use case)
wgo drift
# → Automatically uses most recent saved state as reference

# Compare to specific saved state
wgo drift --since prod-2025-01-15
wgo drift --since yesterday
wgo drift --since "last week"
wgo drift --since 2025-01-10

# Compare between two states
wgo drift --from prod-v1 --to prod-v2
wgo drift --from snapshot1.json --to snapshot2.json

# Filter drift results
wgo drift --severity high
wgo drift --provider gcp
wgo drift --ignore-tags

# Output formats
wgo drift --format json
wgo drift --output drift-report.md
```

### 3. `wgo snapshots` - Manage Saved States

```bash
# List saved states
wgo snapshots list
wgo snapshots ls

# Show details
wgo snapshots show prod-2025-01-15
wgo snapshots show latest

# Delete old states
wgo snapshots delete old-snapshot
wgo snapshots prune --older-than 30d

# Tag/label states
wgo snapshots tag latest --as production
wgo snapshots tag latest --add version=1.2.3
```

## Implementation Plan

### Phase 1: Add New Commands (Backward Compatible)

1. Implement `wgo drift` as the primary command
2. Implement `wgo snapshots` for state management  
3. Update `wgo scan` to show drift by default

### Phase 2: Deprecate Old Commands

1. Mark `baseline` commands as deprecated
2. Add warnings suggesting new commands
3. Update documentation

### Phase 3: Remove Old Commands (Major Version)

1. Remove `baseline` subcommands
2. Remove redundant `diff` command
3. Simplify `check` or merge into `drift`

## User Experience Examples

### Example 1: Daily Drift Check
```bash
# Morning routine - check what changed overnight
$ wgo drift
🔍 Comparing current state to: prod-2025-01-14-18:00
📊 Drift detected in 3 resources:

  ✗ gcp_compute_instance.web-server-1
    → instance_type: n1-standard-2 → n1-standard-4
  
  ✗ kubernetes_deployment.api
    → replicas: 3 → 5
  
  + aws_s3_bucket.new-backup
    → New resource detected

💡 Run 'wgo explain' for AI analysis of these changes
```

### Example 2: Pre-deployment Check
```bash
# Save current production state
$ wgo scan --save-as pre-deploy-v2.1

# ... deploy changes ...

# Check what actually changed
$ wgo drift --since pre-deploy-v2.1
```

### Example 3: Historical Comparison
```bash
# What changed this week?
$ wgo drift --since "last monday"

# Compare two specific versions
$ wgo drift --from prod-v1.0 --to prod-v2.0
```

## Benefits of New Design

1. **Intuitive**: Commands match user mental models
2. **Fewer Steps**: `wgo scan` shows drift immediately
3. **Flexible**: Time-based, tag-based, or file-based comparisons
4. **Clear Purpose**: Each command has one clear job
5. **Progressive Disclosure**: Simple defaults, advanced options available

## Migration Guide for Users

```bash
# Old way
wgo scan
wgo baseline create --name prod
wgo diff --baseline prod

# New way
wgo scan --save-as prod
wgo drift --since prod

# Or even simpler
wgo scan  # Shows drift automatically
```

## Technical Implementation Notes

### Internal Changes

1. **Rename concepts**:
   - "baseline" → "reference state" or "saved state"
   - "baseline management" → "snapshot management"

2. **Default behaviors**:
   - `scan` automatically shows drift if previous states exist
   - `drift` automatically uses most recent state if no --since specified

3. **State storage**:
   - Keep same storage format
   - Add metadata for time-based queries
   - Support automatic state pruning

### Backward Compatibility

During transition period:
- `wgo baseline create` → internally calls `wgo scan --save`
- `wgo diff --baseline X` → internally calls `wgo drift --since X`
- Show deprecation warnings with new command suggestions