# VAINO Command Redesign Implementation Roadmap

## Overview

This roadmap outlines how to transition from the current baseline-centric commands to the new drift-centric design while maintaining backward compatibility.

## Phase 1: Foundation (No Breaking Changes)

### 1.1 Internal Refactoring
- [ ] Rename internal concepts (keep storage format unchanged):
  - `baseline` → `reference_state` in code
  - `BaselineManager` → `StateManager`
- [ ] Add timestamp index to snapshot storage for time-based queries
- [ ] Create unified state resolution logic (by name, time, or ID)

### 1.2 Add New `drift` Command
```go
// cmd/wgo/commands/drift.go
- Implement core drift detection logic
- Support --since with time-based queries
- Support --from/--to pattern
- Add --quiet and --summary flags
```

### 1.3 Enhance `scan` Command
```go
// cmd/wgo/commands/scan.go enhancements
- Add --save flag (immediate save without separate baseline step)
- Add --save-as NAME flag
- Show drift summary after scan (if previous states exist)
- Keep existing behavior as default
```

### 1.4 Add `snapshots` Command
```go
// cmd/wgo/commands/snapshots.go
- Implement list, show, delete subcommands
- Add time-based filtering
- Support tagging/labeling states
```

## Phase 2: Migration Helpers (Deprecation Warnings)

### 2.1 Add Compatibility Layer
```go
// When user runs old commands, show helpful messages:

$ wgo baseline create --name prod
⚠️  Note: 'baseline' commands are deprecated. Use 'wgo scan --save-as prod' instead.
Creating baseline 'prod'...
✅ Done! Next time, try: wgo scan --save-as prod

$ wgo diff --baseline prod  
⚠️  Note: This command is deprecated. Use 'wgo drift --since prod' instead.
[continue with normal operation]
```

### 2.2 Update Help Text
- Add deprecation notices to old commands
- Update examples to show new patterns
- Add migration guide to documentation

### 2.3 Smart Command Routing
```go
// Internal routing for backward compatibility
baseline create --name X → scan --save-as X
diff --baseline X → drift --since X
check --baseline X → drift --since X
```

## Phase 3: Transition Period (6-12 months)

### 3.1 Documentation Updates
- [ ] Rewrite all documentation to use new commands
- [ ] Create migration guide for existing users
- [ ] Update tutorials and examples
- [ ] Add comparison table (old vs new)

### 3.2 Communication
- [ ] Blog post explaining the changes
- [ ] GitHub release notes with migration guide
- [ ] Update README with new examples

### 3.3 Metrics and Monitoring
- [ ] Track usage of deprecated commands
- [ ] Monitor user feedback and issues
- [ ] Adjust transition timeline based on adoption

## Phase 4: Cleanup (Major Version Release)

### 4.1 Remove Deprecated Commands
- [ ] Remove `baseline` subcommands
- [ ] Remove old `diff` logic (keep as alias to `drift`)
- [ ] Simplify `check` or remove entirely

### 4.2 Final State
```bash
# Clean command structure
wgo scan      # Scan and detect drift
wgo drift     # Show drift details
wgo snapshots # Manage saved states
wgo explain   # AI analysis
wgo auth      # Authentication
wgo version   # Version info
```

## Implementation Details

### State Storage Migration
```go
// Keep existing storage format, add metadata
type StateMetadata struct {
    ID          string
    Name        string    // User-provided name (optional)
    Timestamp   time.Time
    Provider    string
    Tags        map[string]string
    
    // New fields for better UX
    AutoName    string    // Auto-generated descriptive name
    GitCommit   string    // Git commit hash if available
    Environment string    // Detected environment
}
```

### Time-Based Query Implementation
```go
func parseTimeReference(ref string) (time.Time, error) {
    // Handle relative times
    switch strings.ToLower(ref) {
    case "yesterday":
        return time.Now().AddDate(0, 0, -1), nil
    case "last week":
        return time.Now().AddDate(0, 0, -7), nil
    }
    
    // Handle "X days/hours ago"
    if matches := regexp.MustCompile(`(\d+)\s+(day|hour)s?\s+ago`).FindStringSubmatch(ref); matches != nil {
        // Parse and calculate
    }
    
    // Handle absolute dates
    formats := []string{
        "2006-01-02",
        "2006-01-02 15:04",
        "Jan 2, 2006",
    }
    for _, format := range formats {
        if t, err := time.Parse(format, ref); err == nil {
            return t, nil
        }
    }
    
    return time.Time{}, fmt.Errorf("cannot parse time reference: %s", ref)
}
```

### Auto-Detection Logic
```go
func (s *Scanner) detectDriftAutomatically() (*DriftReport, error) {
    // Get latest state
    states, err := s.storage.ListStates()
    if err != nil || len(states) < 2 {
        return nil, nil // No previous state to compare
    }
    
    // Compare current scan to previous
    previous := states[1] // Second most recent
    return s.compareStates(s.currentScan, previous)
}
```

## Testing Strategy

### 1. Unit Tests
- Test time parsing logic
- Test state resolution (by name, time, ID)
- Test backward compatibility routing

### 2. Integration Tests
- Test full workflows with new commands
- Test migration paths from old to new
- Test CI/CD scenarios

### 3. User Acceptance Testing
- Beta release with volunteers
- Gather feedback on new UX
- Iterate based on real usage

## Rollback Plan

If issues arise:
1. Keep old commands functional throughout transition
2. Can disable new commands via feature flag
3. Storage format unchanged, so no data migration needed
4. Documentation versioning to support both patterns

## Success Criteria

- [ ] 80% of active users migrated to new commands
- [ ] Positive feedback on improved UX
- [ ] Reduced support questions about "baselines"
- [ ] Improved time-to-first-drift-detection for new users
- [ ] No data loss or breaking changes for existing users

## Timeline

- **Month 1-2**: Phase 1 implementation
- **Month 3**: Phase 2 with beta release
- **Month 4-9**: Phase 3 transition period
- **Month 10-12**: Monitor and adjust
- **Next major version**: Phase 4 cleanup