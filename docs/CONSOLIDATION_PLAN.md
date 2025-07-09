# Documentation Consolidation Plan

## Current State Analysis

We have identified 32 documentation files across the repository with significant duplication and organization issues:

### Major Issues Identified:
1. **Installation instructions** duplicated across 4 files
2. **Basic commands** explained in 6 different files
3. **Authentication setup** scattered across multiple files
4. **Internal/design documents** mixed with user documentation
5. **Example outputs** duplicated between files

## Proposed Documentation Structure

```
docs/
├── README.md                   # Main overview (consolidated)
├── installation.md             # Complete installation guide
├── getting-started.md          # Quick start tutorial
├── configuration.md            # Complete config reference (KEEP)
├── commands.md                 # Command reference (KEEP)
├── gcp-setup.md                # GCP-specific setup (KEEP)
├── unix-style-output-examples.md # Unix philosophy (KEEP)
├── troubleshooting.md          # Problem solving (KEEP)
├── best-practices.md           # Production guidance (KEEP)
├── concurrent-scanning.md      # Feature documentation (KEEP)
├── examples/                   # Real-world examples
│   ├── kubernetes-monitoring.md    # (KEEP)
│   ├── multi-cloud-setup.md        # (KEEP)
│   ├── terraform-drift.md          # (KEEP)
│   └── usage.md                     # (MOVE from /examples/)
├── performance/                # Performance documentation
│   ├── analysis.md                  # (KEEP)
│   ├── ci-integration.md            # (KEEP)
│   └── testing-guide.md             # (KEEP)
├── development/                # Development documentation
│   ├── testing-strategy.md          # (KEEP)
│   ├── ci-configuration.md          # (KEEP)
│   └── architecture.md              # (NEW - consolidate tech docs)
└── design/                     # Design and planning documents
    ├── command-redesign-proposal.md # (MOVE from /docs/)
    ├── implementation-roadmap.md    # (MOVE from /docs/)
    └── ux-comparison.md             # (MOVE from /docs/)
```

## Files to Remove

1. **`SIMPLE_USAGE.md`** - Entirely redundant with other documentation
2. **`IMPROVEMENTS.md`** - Internal implementation notes
3. **`CI_CLEANUP_PLAN.md`** - Internal development documentation
4. **`new-commands-quick-reference.md`** - Consolidate into main docs

## Files to Consolidate

### 1. Create New README.md (Root)
**Sources to merge:**
- Current README.md (project overview, supported providers)
- SIMPLE_USAGE.md (basic workflow - will be removed)
- Getting-started.md (quick start section)

**New content structure:**
```markdown
# WGO - Git for Infrastructure
- Project overview and value proposition
- Supported providers table
- Quick start (3-4 commands max)
- Links to detailed documentation
- Contributing and support
```

### 2. Enhanced installation.md
**Sources to merge:**
- Current INSTALLATION.md (detailed instructions)
- README.md (installation section)
- getting-started.md (installation section)
- quick-reference.md (installation section)

**New content structure:**
```markdown
# Installation Guide
- Package managers (all platforms)
- Docker installation
- Building from source
- Shell completions
- Verification and troubleshooting
```

### 3. Enhanced getting-started.md
**Sources to merge:**
- Current getting-started.md (tutorial)
- README.md (quick start examples)
- SIMPLE_USAGE.md (basic workflow)

**New content structure:**
```markdown
# Getting Started with WGO
- Prerequisites
- First scan tutorial
- Understanding output
- Common workflows
- Next steps
```

### 4. New docs/development/architecture.md
**Sources to merge:**
- Technical content from various files
- Implementation details
- System architecture overview

## Implementation Steps

### Phase 1: Remove Redundant Files
1. Delete `SIMPLE_USAGE.md`
2. Delete `IMPROVEMENTS.md`
3. Delete `CI_CLEANUP_PLAN.md`
4. Delete `new-commands-quick-reference.md`

### Phase 2: Create New Directory Structure
1. Create `docs/performance/` directory
2. Create `docs/development/` directory
3. Create `docs/design/` directory

### Phase 3: Move Files
1. Move `examples/usage.md` → `docs/examples/usage.md`
2. Move `docs/command-redesign-proposal.md` → `docs/design/`
3. Move `docs/implementation-roadmap.md` → `docs/design/`
4. Move `docs/ux-comparison.md` → `docs/design/`
5. Move performance files to `docs/performance/`
6. Move development files to `docs/development/`

### Phase 4: Consolidate Content
1. Create new consolidated README.md
2. Enhance installation.md with all installation content
3. Enhance getting-started.md with tutorial content
4. Create architecture.md with technical details

### Phase 5: Update Links and References
1. Update all internal links to point to new locations
2. Update CI/CD references to documentation paths
3. Update any hardcoded documentation paths in code

## Quality Assurance Checklist

- [ ] All installation methods covered in single file
- [ ] No duplicate command examples
- [ ] Consistent formatting and style
- [ ] All links work correctly
- [ ] Examples are current and tested
- [ ] Internal vs user documentation clearly separated
- [ ] Navigation is intuitive

## Benefits of Consolidation

1. **Reduced maintenance** - Single source of truth for each topic
2. **Better user experience** - Clear navigation and no confusion
3. **Improved accuracy** - No conflicting information
4. **Professional appearance** - Well-organized documentation structure
5. **Easier updates** - Clear ownership of content areas

## Timeline

- **Phase 1-2**: 1 day - Remove files and create structure
- **Phase 3**: 1 day - Move files to new locations
- **Phase 4**: 2 days - Consolidate and write new content
- **Phase 5**: 1 day - Update links and test

**Total estimated time: 5 days**