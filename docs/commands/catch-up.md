# wgo catch-up

Get a comforting summary of infrastructure changes while you were away.

## Overview

The `catch-up` command provides an empathetic, human-friendly summary of all infrastructure changes that occurred during your absence. It's designed to help you quickly understand what happened, distinguish between planned and unplanned changes, and feel confident about the current state of your infrastructure.

## Philosophy

Being away from your infrastructure can be stressful. The catch-up command is built with emotional intelligence to:

- **Reduce anxiety** by immediately showing that critical systems are stable
- **Build confidence** through clear categorization of changes
- **Save time** by highlighting only what matters
- **Provide comfort** through reassuring language and metrics

## Usage

```bash
# Auto-detect absence period and show changes
wgo catch-up

# Show changes from the last 2 weeks
wgo catch-up --since "2 weeks ago"

# Use comfort mode for reassuring tone (default: true)
wgo catch-up --comfort-mode

# Update baselines after reviewing changes
wgo catch-up --sync-state

# Check specific providers only
wgo catch-up --providers aws,kubernetes
```

## Options

- `--since` - Time period to catch up from (e.g., "2 weeks ago", "2024-01-01")
- `--comfort-mode` - Use reassuring tone and emotional intelligence (default: true)
- `--sync-state` - Update baselines after reviewing changes
- `--providers` - Specific providers to check (default: all configured)

## Time Period Formats

The `--since` flag accepts various formats:

### Relative Time
- `"1 hour ago"`
- `"3 days ago"`
- `"2 weeks ago"`
- `"1 month ago"`
- `"6 months ago"`

### Absolute Dates
- `"2024-01-15"`
- `"2024-01-15 14:30:00"`
- `"Jan 15, 2024"`
- `"January 15, 2024"`

### Auto-Detection
When no `--since` is provided, catch-up will intelligently detect your absence period based on:
- Last command execution
- Last baseline update
- System login history
- Git commit activity

## Output Sections

### 1. Executive Summary
Provides immediate comfort with high-level status:
- Critical systems status
- Total changes breakdown
- Team performance rating
- Security incident count

### 2. Security Status
Always shown for peace of mind:
- Security incidents (if any)
- Compliance score
- Vulnerabilities addressed
- Last audit date

### 3. Team Activity
Shows what your team accomplished:
- Total actions taken
- Top contributors
- Incident handling rating
- Key decisions made

### 4. Changes Breakdown
Categorized for easy understanding:

#### Planned Changes
- Scheduled deployments
- Maintenance windows
- Feature releases
- Infrastructure upgrades

#### Unplanned Changes
- Incident responses
- Emergency fixes
- Unexpected failures
- Manual interventions

#### Routine Operations
- Auto-scaling events
- Backup completions
- Log rotations
- Health checks

### 5. Comfort Metrics (Comfort Mode)
Visual representation of system health:
- Stability Score
- Team Performance
- System Resilience
- Overall Confidence

### 6. Recommendations
Actionable next steps based on the analysis.

## Examples

### Basic Catch-Up
```bash
$ wgo catch-up

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    🔍 Infrastructure Catch-Up Report
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

While you were away (Jan 8, 09:00 to Jan 15, 16:30):
Absence duration: 7 days, 7 hours

✨ Welcome back! Everything went smoothly while you were away.
   Your infrastructure remained stable and your team did an excellent job.

📊 Executive Summary
──────────────────────────────────────────────────
  ● Critical Systems: All stable
  ● Total Changes: 28
    ◦ Planned: 18 (64%)
    ◦ Unplanned: 3 (11%)
    ◦ Routine: 7 (25%)
  ● Team Performance: Excellent

🛡️  Security Status
──────────────────────────────────────────────────
  ✅ No security incidents occurred
  ✅ Compliance maintained at 100%
  ● Last security audit: Dec 15, 2023

👥 Team Activity
──────────────────────────────────────────────────
  ✨ Your team handled 28 actions while you were away
  ● Top Contributors:
    🥇 Alice (12 actions)
    🥈 Bob (8 actions)
    🥉 Charlie (5 actions)
  ● Incident Handling: Excellent

📋 Changes Breakdown
──────────────────────────────────────────────────

  📅 Planned Changes (18)
     📅 [Jan 10 14:00] Deployed API v2.3.0 with new features
       ↳ Impact: Improved response times by 20%
     📅 [Jan 12 10:00] Database maintenance window completed
       ↳ Impact: Optimized query performance
     📅 [Jan 14 15:30] Kubernetes cluster upgrade to v1.28
     ... and 15 more

  🚨 Unplanned Changes (3)
     🚨 [Jan 11 23:45] Pod crash loop detected and resolved
       ↳ Impact: 5 minutes of degraded service
     🚨 [Jan 13 02:30] Emergency patch for memory leak
     🚨 [Jan 14 19:00] Load balancer failover triggered

  🔄 Routine Operations (7)
     - Auto-scaling: 4
     - Backups: 2
     - Certificate renewal: 1

💪 System Health Metrics
──────────────────────────────────────────────────
  Stability:          ████████████████████ 95%
  Team Performance:   ████████████████████ 98%
  System Resilience:  ████████████████████ 100%

  ⭐ Overall Confidence: 96%

💡 Recommendations
──────────────────────────────────────────────────
  1. Continue the excellent work maintaining infrastructure stability!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎉 You're all caught up! Your infrastructure is in excellent hands.
   Feel free to reach out if you need any clarification.

Run 'wgo catch-up --sync-state' to update your baselines
```

### After Incident Period
```bash
$ wgo catch-up --since "3 days ago"

While you were away (Jan 12, 16:30 to Jan 15, 16:30):
Absence duration: 3 days

🤗 Welcome back! Let's get you up to speed.
   There's been some activity, but don't worry - we'll walk through it together.

📊 Executive Summary
──────────────────────────────────────────────────
  ● Critical Systems: Mostly stable
  ● Total Changes: 15
    ◦ Planned: 5 (33%)
    ◦ Unplanned: 8 (53%)
    ◦ Routine: 2 (13%)
  ● Team Performance: Good

🛡️  Security Status
──────────────────────────────────────────────────
  ⚠️  2 security incident(s) were handled
  ● Compliance Score: 92%
  ● Vulnerabilities addressed:
    - CVE-2024-1234 patched on web servers
    - Unauthorized access attempt blocked

[... rest of report ...]
```

### Sync State After Review
```bash
$ wgo catch-up --sync-state

[... full catch-up report ...]

Updating baselines with current state...
✅ Baselines updated successfully!
```

## Best Practices

1. **Regular Catch-Ups**: Run after any absence longer than a day
2. **Review Before Sync**: Always review changes before updating baselines
3. **Team Communication**: Share reports with team members who were also away
4. **Action Items**: Follow up on any recommendations provided

## Integration with Other Commands

After running catch-up:
- Use `wgo diff` to see detailed changes for specific resources
- Use `wgo explain` to understand complex changes
- Use `wgo status` to verify current system health
- Use `wgo baseline update` if you didn't use `--sync-state`

## Customization

### Business Hours
The classifier considers business hours when categorizing changes. Default is Mon-Fri 9 AM-5 PM, but this can be configured in your WGO config file.

### Change Patterns
You can customize how changes are classified by adding patterns to your configuration:

```yaml
catch_up:
  planned_patterns:
    - "scheduled"
    - "maintenance"
    - "release"
  unplanned_patterns:
    - "emergency"
    - "incident"
    - "failure"
  routine_patterns:
    - "backup"
    - "scaling"
    - "rotation"
```

## Troubleshooting

### No Changes Detected
- Verify snapshots exist for the time period
- Check that providers are properly configured
- Ensure baseline snapshots are being created

### Missing Changes
- Some changes may be classified as routine and summarized
- Use `--comfort-mode=false` for more detailed output
- Check provider-specific logs for collection issues

### Performance
- For long absence periods, the initial scan may take time
- Consider using `--providers` to limit scope
- Snapshots are cached for 15 minutes

## FAQ

**Q: How far back can I look?**
A: As far back as you have snapshots stored. Default retention is 90 days.

**Q: What happens if I was away during an incident?**
A: The report will clearly show all incidents, who handled them, and the current status.

**Q: Can I get alerts for changes while away?**
A: Yes, configure webhooks in watch mode for real-time notifications.

**Q: Is sensitive information hidden?**
A: Yes, the report focuses on metadata and impact, not sensitive configuration details.