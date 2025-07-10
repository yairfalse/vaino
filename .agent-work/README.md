# Agent Work Directory

This directory contains the Agent Branch Management System for VAINO, enabling multiple claude-code instances to work simultaneously without conflicts.

## Directory Structure

```
.agent-work/
├── README.md           # This file
├── config.json         # Global configuration
├── tasks/              # Task tracking files
├── agents/             # Agent registration files
└── conflicts/          # Conflict detection logs
```

## How It Works

### 1. Agent Registration
Each claude-code instance registers as an agent with:
- Unique agent ID
- Task description
- File claims
- Branch assignment

### 2. Task Management
Tasks are tracked with:
- Task ID and description
- Assigned agent
- Component and action
- File dependencies
- Status tracking

### 3. Conflict Prevention
The system prevents conflicts by:
- File claim checking
- Branch isolation
- Real-time conflict detection
- Coordination between agents

### 4. Quality Gates
Before PR creation, agents must pass:
- Code formatting checks
- Build validation
- Test execution
- Documentation updates

## Usage

### Start New Agent
```bash
make agent-start
# or
./scripts/agent-branch.sh start
```

### Check Status
```bash
make agent-status
# or
./scripts/agent-branch.sh status
```

### Quality Check
```bash
make agent-check
# or
make pr-ready
```

### Clean Up
```bash
./scripts/agent-branch.sh cleanup
```

## Branch Naming Convention

Branches follow the pattern: `feature/agent-id/component-action`

Examples:
- `feature/agent-123/docs-consolidation`
- `feature/agent-456/collectors-refactor`
- `feature/agent-789/watchers-bugfix`

## File Structure

### config.json
Global configuration for the agent system.

### tasks/task-{id}.json
Individual task tracking files containing:
- Task metadata
- Agent assignment
- Progress tracking
- File dependencies

### agents/agent-{id}.json
Agent registration files containing:
- Agent status
- Branch assignment
- File claims
- Activity tracking

### conflicts/conflict-{timestamp}.json
Conflict detection logs containing:
- Conflicting files
- Involved agents
- Resolution status
- Timestamps

## Integration with Git

The system integrates with Git to:
- Create isolated feature branches
- Track branch assignments
- Prevent merge conflicts
- Coordinate PR creation

## Quality Assurance

Each agent must pass quality gates:
- `make agent-check` - Quick validation
- `make pr-ready` - Comprehensive checks
- Automated testing
- Documentation verification

## Troubleshooting

### Common Issues

1. **File Conflicts**: Use `./scripts/agent-branch.sh status` to see file claims
2. **Agent Cleanup**: Run `./scripts/agent-branch.sh cleanup` to remove inactive agents
3. **Branch Issues**: Ensure you're on the correct agent branch before making changes

### Getting Help

Run `./scripts/agent-branch.sh help` for available commands and options.

## Best Practices

1. **Always register as agent** before starting work
2. **Claim files** you plan to modify
3. **Check for conflicts** before making changes
4. **Run quality checks** before creating PRs
5. **Use descriptive task descriptions**
6. **Keep branches focused** on single components
7. **Clean up** when finished

## Security

- Agent files contain no sensitive information
- All data is stored locally in `.agent-work/`
- No external network calls required
- Compatible with existing Git workflows

## Maintenance

The system is self-cleaning:
- Inactive agents are automatically detected
- Cleanup removes stale registrations
- Conflicts are logged for resolution
- Old tasks are archived automatically