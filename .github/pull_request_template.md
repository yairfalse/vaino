# Pull Request Template

## Summary
<!-- Provide a clear and concise description of what this PR does -->

## Agent Information
- **Agent ID:** <!-- e.g., agent-1234567890-5678 -->
- **Task ID:** <!-- e.g., task-1234567890-123 -->
- **Branch:** <!-- e.g., feature/agent-1234567890-5678/docs-consolidation -->

## Changes Made
<!-- Describe the specific changes in this PR -->

### Component: <!-- e.g., docs, collectors, watchers -->
### Action: <!-- e.g., refactor, add, fix -->

#### Files Modified:
<!-- List all modified files -->
- [ ] `path/to/file1.go`
- [ ] `path/to/file2.md`
- [ ] `path/to/file3.yaml`

#### New Files Added:
<!-- List any new files created -->
- [ ] `path/to/new/file1.go`
- [ ] `path/to/new/file2.md`

## Testing
<!-- Describe how the changes were tested -->

### Manual Testing:
- [ ] Feature works as expected
- [ ] No regressions introduced
- [ ] Edge cases handled

### Automated Testing:
- [ ] All existing tests pass
- [ ] New tests added for new functionality
- [ ] Test coverage maintained or improved

## Quality Checklist
<!-- Complete all items before submitting PR -->

### Code Quality:
- [ ] Code follows Go best practices
- [ ] Code is properly formatted (`gofmt`)
- [ ] Code passes linting (`golint`)
- [ ] No unused imports or variables
- [ ] Proper error handling implemented

### Documentation:
- [ ] Code changes are documented
- [ ] README updated if necessary
- [ ] Examples updated if applicable
- [ ] API documentation updated

### Build & CI:
- [ ] `make agent-check` passes
- [ ] `make pr-ready` passes
- [ ] CI pipeline passes
- [ ] No build warnings

### Agent Workflow:
- [ ] Agent properly registered
- [ ] Files properly claimed
- [ ] No conflicts with other agents
- [ ] Branch follows naming convention

## Performance Impact
<!-- Describe any performance implications -->

- [ ] No performance impact
- [ ] Performance improved
- [ ] Performance regression (explain why acceptable)

## Breaking Changes
<!-- List any breaking changes -->

- [ ] No breaking changes
- [ ] Breaking changes documented
- [ ] Migration guide provided

## Security Review
<!-- Security considerations -->

- [ ] No security implications
- [ ] Security review completed
- [ ] No secrets committed
- [ ] Input validation implemented

## Deployment Notes
<!-- Any special deployment considerations -->

- [ ] No deployment changes needed
- [ ] Configuration changes required
- [ ] Database migrations needed
- [ ] Infrastructure changes required

## Related Issues
<!-- Link to related issues -->

Closes #<!-- issue number -->
Related to #<!-- issue number -->

## Screenshots/Demos
<!-- Add screenshots or demo links if applicable -->

## Reviewer Notes
<!-- Any specific areas that need attention -->

### Areas of Focus:
- [ ] Code architecture
- [ ] Error handling
- [ ] Performance
- [ ] Security
- [ ] Documentation

### Questions for Review:
<!-- Any specific questions for reviewers -->

## Post-Merge Checklist
<!-- Items to complete after merge -->

- [ ] Update documentation
- [ ] Deploy to staging
- [ ] Monitor for issues
- [ ] Clean up agent branch
- [ ] Update tracking systems

---

## Agent System Notes
<!-- Automatically filled by agent system -->

**Agent Registration:** ✅ Verified  
**Conflict Check:** ✅ Passed  
**Quality Gates:** ✅ Passed  
**File Claims:** ✅ Verified  

---

*This PR was created using the WGO Agent Branch Management System. For issues with the agent system, see `.agent-work/README.md`*