# CI/CD Cleanup Plan

## Current State (10 workflows, complex, failing)
- benchmark.yml
- cache-cleanup.yml
- ci.yml (marked as deprecated)
- comprehensive-test.yml
- comprehensive-tests.yml (duplicate?)
- performance-ci.yml
- release.yaml
- release.yml (duplicate?)
- selective-tests.yml
- test.yml

## New Simple Structure (3 workflows)

### 1. ci-simple.yml (Main CI)
- **Purpose**: Run on every push/PR
- **Jobs**:
  - `validate`: Go modules, build, lint, format
  - `test`: Unit tests with coverage
  - `build-matrix`: Test builds on Linux/Mac/Windows
  - `integration`: Optional integration tests
  - `ci-status`: Summary check

### 2. release-simple.yml
- **Purpose**: Create releases on tags
- **Jobs**:
  - Build binaries for all platforms
  - Create GitHub release with artifacts

### 3. performance-simple.yml
- **Purpose**: Weekly performance benchmarks
- **Jobs**:
  - Run benchmarks
  - Store results

## Migration Steps

1. **Test new CI first**:
   ```bash
   # Rename current workflows to .old
   mv .github/workflows/ci.yml .github/workflows/ci.yml.old
   mv .github/workflows/selective-tests.yml .github/workflows/selective-tests.yml.old
   
   # Rename new simple CI to active
   mv .github/workflows/ci-simple.yml .github/workflows/ci.yml
   ```

2. **Fix any issues in the simple CI**

3. **Once working, archive old workflows**:
   ```bash
   mkdir .github/workflows/archive
   mv .github/workflows/*.old .github/workflows/archive/
   ```

4. **Clean up completely**:
   - Remove all old workflows
   - Keep only the 3 simple ones

## Key Principles
1. **One workflow per purpose** (CI, Release, Performance)
2. **Fast feedback** - validate/build/lint runs first
3. **Conditional heavy tests** - integration only when needed
4. **Clear job dependencies** - if validate fails, stop
5. **Single source of truth** - one GO_VERSION env var

## Benefits
- **Simplicity**: 3 files instead of 10
- **Speed**: Parallel jobs where possible
- **Clarity**: Easy to understand what runs when
- **Maintainability**: Less duplication, clear structure