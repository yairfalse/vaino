package catchup

import (
	"fmt"
	"strings"
	"time"
)

// Formatter creates human-friendly, empathetic output from catch-up reports
type Formatter struct {
	comfortMode bool
	colors      ColorScheme
}

// ColorScheme defines colors for different output elements
type ColorScheme struct {
	Success string
	Warning string
	Error   string
	Info    string
	Muted   string
	Reset   string
}

// NewFormatter creates a new report formatter
func NewFormatter(comfortMode bool) *Formatter {
	return &Formatter{
		comfortMode: comfortMode,
		colors: ColorScheme{
			Success: "\033[32m", // Green
			Warning: "\033[33m", // Yellow
			Error:   "\033[31m", // Red
			Info:    "\033[36m", // Cyan
			Muted:   "\033[90m", // Gray
			Reset:   "\033[0m",  // Reset
		},
	}
}

// Format converts a report into a comforting, human-readable string
func (f *Formatter) Format(report *Report) string {
	var output strings.Builder

	// Header with period information
	f.writeHeader(&output, report.Period)

	// Comfort introduction
	if f.comfortMode {
		f.writeComfortIntro(&output, report)
	}

	// Executive summary
	f.writeExecutiveSummary(&output, report)

	// Security status (always show for peace of mind)
	f.writeSecurityStatus(&output, report.SecurityStatus)

	// Team activity summary
	f.writeTeamActivity(&output, report.TeamActivity)

	// Changes breakdown
	if report.Summary.TotalChanges > 0 {
		f.writeChangesSection(&output, report)
	}

	// Comfort metrics
	if f.comfortMode {
		f.writeComfortMetrics(&output, report.ComfortMetrics)
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		f.writeRecommendations(&output, report.Recommendations)
	}

	// Closing message
	f.writeClosingMessage(&output, report)

	return output.String()
}

// writeHeader adds the header section
func (f *Formatter) writeHeader(output *strings.Builder, period Period) {
	output.WriteString(f.colors.Info)
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	output.WriteString("                    🔍 Infrastructure Catch-Up Report\n")
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	output.WriteString(f.colors.Reset)

	output.WriteString(fmt.Sprintf("\n%sWhile you were away%s (%s to %s):\n",
		f.colors.Info,
		f.colors.Reset,
		period.Start.Format("Jan 2, 15:04"),
		period.End.Format("Jan 2, 15:04")))

	output.WriteString(fmt.Sprintf("%sAbsence duration:%s %s\n\n",
		f.colors.Muted,
		f.colors.Reset,
		f.formatDuration(period.Duration)))
}

// writeComfortIntro adds a reassuring introduction
func (f *Formatter) writeComfortIntro(output *strings.Builder, report *Report) {
	output.WriteString(f.colors.Success)

	// Personalized comfort messages based on metrics
	if report.ComfortMetrics.OverallConfidence >= 0.9 {
		output.WriteString("✨ Welcome back! Everything went smoothly while you were away.\n")
		output.WriteString("   Your infrastructure remained stable and your team did an excellent job.\n\n")
	} else if report.ComfortMetrics.OverallConfidence >= 0.7 {
		output.WriteString("👋 Welcome back! Your infrastructure is in good shape.\n")
		output.WriteString("   There were a few changes, but everything was handled well.\n\n")
	} else {
		output.WriteString("🤗 Welcome back! Let's get you up to speed.\n")
		output.WriteString("   There's been some activity, but don't worry - we'll walk through it together.\n\n")
	}

	output.WriteString(f.colors.Reset)
}

// writeExecutiveSummary adds the high-level summary
func (f *Formatter) writeExecutiveSummary(output *strings.Builder, report *Report) {
	output.WriteString(fmt.Sprintf("%s📊 Executive Summary%s\n", f.colors.Info, f.colors.Reset))
	output.WriteString(strings.Repeat("─", 50) + "\n")

	// Critical systems status
	statusColor := f.colors.Success
	if report.Summary.CriticalSystems != "All stable" {
		statusColor = f.colors.Warning
	}
	output.WriteString(fmt.Sprintf("  %s●%s Critical Systems: %s%s%s\n",
		statusColor, f.colors.Reset,
		statusColor, report.Summary.CriticalSystems, f.colors.Reset))

	// Change summary
	if report.Summary.TotalChanges == 0 {
		output.WriteString(fmt.Sprintf("  %s●%s Changes: None - your infrastructure remained unchanged! 🎯\n",
			f.colors.Success, f.colors.Reset))
	} else {
		output.WriteString(fmt.Sprintf("  %s●%s Total Changes: %d\n",
			f.colors.Info, f.colors.Reset, report.Summary.TotalChanges))

		if report.Summary.PlannedCount > 0 {
			output.WriteString(fmt.Sprintf("    %s◦%s Planned: %d (%.0f%%)\n",
				f.colors.Success, f.colors.Reset,
				report.Summary.PlannedCount,
				float64(report.Summary.PlannedCount)/float64(report.Summary.TotalChanges)*100))
		}

		if report.Summary.UnplannedCount > 0 {
			output.WriteString(fmt.Sprintf("    %s◦%s Unplanned: %d (%.0f%%)\n",
				f.colors.Warning, f.colors.Reset,
				report.Summary.UnplannedCount,
				float64(report.Summary.UnplannedCount)/float64(report.Summary.TotalChanges)*100))
		}

		if report.Summary.RoutineCount > 0 {
			output.WriteString(fmt.Sprintf("    %s◦%s Routine: %d (%.0f%%)\n",
				f.colors.Muted, f.colors.Reset,
				report.Summary.RoutineCount,
				float64(report.Summary.RoutineCount)/float64(report.Summary.TotalChanges)*100))
		}
	}

	// Team efficiency
	efficiencyColor := f.colors.Success
	if report.Summary.TeamEfficiency == "Adequate" {
		efficiencyColor = f.colors.Warning
	}
	output.WriteString(fmt.Sprintf("  %s●%s Team Performance: %s%s%s\n\n",
		efficiencyColor, f.colors.Reset,
		efficiencyColor, report.Summary.TeamEfficiency, f.colors.Reset))
}

// writeSecurityStatus adds the security section
func (f *Formatter) writeSecurityStatus(output *strings.Builder, status SecurityStatus) {
	output.WriteString(fmt.Sprintf("%s🛡️  Security Status%s\n", f.colors.Info, f.colors.Reset))
	output.WriteString(strings.Repeat("─", 50) + "\n")

	if status.IncidentCount == 0 {
		output.WriteString(fmt.Sprintf("  %s✅ No security incidents occurred%s\n",
			f.colors.Success, f.colors.Reset))
		output.WriteString(fmt.Sprintf("  %s✅ Compliance maintained at %.0f%%%s\n",
			f.colors.Success, status.ComplianceScore, f.colors.Reset))
	} else {
		output.WriteString(fmt.Sprintf("  %s⚠️  %d security incident(s) were handled%s\n",
			f.colors.Warning, status.IncidentCount, f.colors.Reset))
		output.WriteString(fmt.Sprintf("  %s●%s Compliance Score: %.0f%%%s\n",
			f.colors.Info, f.colors.Reset, status.ComplianceScore, f.colors.Reset))

		if len(status.Vulnerabilities) > 0 {
			output.WriteString(fmt.Sprintf("  %s●%s Vulnerabilities addressed:\n", f.colors.Info, f.colors.Reset))
			for _, vuln := range status.Vulnerabilities {
				output.WriteString(fmt.Sprintf("    %s- %s%s\n", f.colors.Muted, vuln, f.colors.Reset))
			}
		}
	}

	output.WriteString(fmt.Sprintf("  %s●%s Last security audit: %s\n\n",
		f.colors.Muted, f.colors.Reset,
		status.LastAudit.Format("Jan 2, 2006")))
}

// writeTeamActivity adds the team activity section
func (f *Formatter) writeTeamActivity(output *strings.Builder, activity TeamActivity) {
	output.WriteString(fmt.Sprintf("%s👥 Team Activity%s\n", f.colors.Info, f.colors.Reset))
	output.WriteString(strings.Repeat("─", 50) + "\n")

	if f.comfortMode {
		output.WriteString(fmt.Sprintf("  %s✨ Your team handled %d actions while you were away%s\n",
			f.colors.Success, activity.TotalActions, f.colors.Reset))
	} else {
		output.WriteString(fmt.Sprintf("  %s●%s Total Actions: %d\n",
			f.colors.Info, f.colors.Reset, activity.TotalActions))
	}

	if len(activity.TopContributors) > 0 {
		output.WriteString(fmt.Sprintf("  %s●%s Top Contributors:\n", f.colors.Info, f.colors.Reset))
		for i, contributor := range activity.TopContributors {
			emoji := "🥇"
			if i == 1 {
				emoji = "🥈"
			} else if i == 2 {
				emoji = "🥉"
			}
			output.WriteString(fmt.Sprintf("    %s %s\n", emoji, contributor))
		}
	}

	handlingColor := f.colors.Success
	if activity.IncidentHandling == "Needs improvement" {
		handlingColor = f.colors.Warning
	}
	output.WriteString(fmt.Sprintf("  %s●%s Incident Handling: %s%s%s\n",
		handlingColor, f.colors.Reset,
		handlingColor, activity.IncidentHandling, f.colors.Reset))

	if len(activity.KeyDecisions) > 0 && len(activity.KeyDecisions) <= 3 {
		output.WriteString(fmt.Sprintf("  %s●%s Key Decisions Made:\n", f.colors.Info, f.colors.Reset))
		for _, decision := range activity.KeyDecisions {
			output.WriteString(fmt.Sprintf("    %s◦ %s%s\n", f.colors.Muted, decision, f.colors.Reset))
		}
	}

	output.WriteString("\n")
}

// writeChangesSection adds detailed changes information
func (f *Formatter) writeChangesSection(output *strings.Builder, report *Report) {
	output.WriteString(fmt.Sprintf("%s📋 Changes Breakdown%s\n", f.colors.Info, f.colors.Reset))
	output.WriteString(strings.Repeat("─", 50) + "\n")

	// Planned changes
	if len(report.PlannedChanges) > 0 {
		output.WriteString(fmt.Sprintf("\n%s  📅 Planned Changes (%d)%s\n",
			f.colors.Success, len(report.PlannedChanges), f.colors.Reset))
		f.writeChangesList(output, report.PlannedChanges, 3) // Show top 3
	}

	// Unplanned changes
	if len(report.UnplannedChanges) > 0 {
		output.WriteString(fmt.Sprintf("\n%s  🚨 Unplanned Changes (%d)%s\n",
			f.colors.Warning, len(report.UnplannedChanges), f.colors.Reset))
		f.writeChangesList(output, report.UnplannedChanges, 5) // Show more unplanned
	}

	// Routine changes (summarize unless very few)
	if len(report.RoutineChanges) > 0 {
		output.WriteString(fmt.Sprintf("\n%s  🔄 Routine Operations (%d)%s\n",
			f.colors.Muted, len(report.RoutineChanges), f.colors.Reset))
		if len(report.RoutineChanges) <= 3 {
			f.writeChangesList(output, report.RoutineChanges, 3)
		} else {
			// Summarize routine changes by type
			summary := f.summarizeRoutineChanges(report.RoutineChanges)
			for changeType, count := range summary {
				output.WriteString(fmt.Sprintf("     %s- %s: %d%s\n",
					f.colors.Muted, changeType, count, f.colors.Reset))
			}
		}
	}

	output.WriteString("\n")
}

// writeChangesList writes a list of changes with a limit
func (f *Formatter) writeChangesList(output *strings.Builder, changes []Change, limit int) {
	displayed := 0
	for i, change := range changes {
		if i >= limit {
			remaining := len(changes) - limit
			output.WriteString(fmt.Sprintf("     %s... and %d more%s\n",
				f.colors.Muted, remaining, f.colors.Reset))
			break
		}

		timestamp := change.Timestamp.Format("Jan 2 15:04")
		icon := f.getChangeIcon(change)

		output.WriteString(fmt.Sprintf("     %s %s[%s]%s %s\n",
			icon,
			f.colors.Muted, timestamp, f.colors.Reset,
			change.Description))

		if change.Impact != "" && f.comfortMode {
			output.WriteString(fmt.Sprintf("       %s↳ Impact: %s%s\n",
				f.colors.Muted, change.Impact, f.colors.Reset))
		}

		displayed++
	}
}

// writeComfortMetrics adds the comfort metrics section
func (f *Formatter) writeComfortMetrics(output *strings.Builder, metrics ComfortMetrics) {
	output.WriteString(fmt.Sprintf("%s💪 System Health Metrics%s\n", f.colors.Info, f.colors.Reset))
	output.WriteString(strings.Repeat("─", 50) + "\n")

	// Stability
	f.writeMetricBar(output, "Stability", metrics.StabilityScore)

	// Team Performance
	f.writeMetricBar(output, "Team Performance", metrics.TeamPerformance)

	// System Resilience
	f.writeMetricBar(output, "System Resilience", metrics.SystemResilience)

	// Overall Confidence
	output.WriteString("\n")
	confidenceColor := f.colors.Success
	if metrics.OverallConfidence < 0.8 {
		confidenceColor = f.colors.Warning
	}
	output.WriteString(fmt.Sprintf("  %s⭐ Overall Confidence: %.0f%%%s\n\n",
		confidenceColor, metrics.OverallConfidence*100, f.colors.Reset))
}

// writeMetricBar creates a visual progress bar for a metric
func (f *Formatter) writeMetricBar(output *strings.Builder, name string, value float64) {
	barLength := 20
	filled := int(value * float64(barLength))

	color := f.colors.Success
	if value < 0.8 {
		color = f.colors.Warning
	}
	if value < 0.6 {
		color = f.colors.Error
	}

	bar := color + strings.Repeat("█", filled) + f.colors.Muted + strings.Repeat("░", barLength-filled) + f.colors.Reset

	output.WriteString(fmt.Sprintf("  %-18s %s %.0f%%\n", name+":", bar, value*100))
}

// writeRecommendations adds the recommendations section
func (f *Formatter) writeRecommendations(output *strings.Builder, recommendations []string) {
	output.WriteString(fmt.Sprintf("%s💡 Recommendations%s\n", f.colors.Info, f.colors.Reset))
	output.WriteString(strings.Repeat("─", 50) + "\n")

	for i, rec := range recommendations {
		number := fmt.Sprintf("%d.", i+1)
		output.WriteString(fmt.Sprintf("  %s %s\n", number, rec))
	}

	output.WriteString("\n")
}

// writeClosingMessage adds a personalized closing
func (f *Formatter) writeClosingMessage(output *strings.Builder, report *Report) {
	if f.comfortMode {
		output.WriteString(f.colors.Info)
		output.WriteString(strings.Repeat("━", 70) + "\n")

		if report.ComfortMetrics.OverallConfidence >= 0.9 {
			output.WriteString("🎉 You're all caught up! Your infrastructure is in excellent hands.\n")
			output.WriteString("   Feel free to reach out if you need any clarification.\n")
		} else if report.ComfortMetrics.OverallConfidence >= 0.7 {
			output.WriteString("✅ You're now up to speed! Everything important has been covered.\n")
			output.WriteString("   Take your time reviewing the changes above.\n")
		} else {
			output.WriteString("📚 That's everything that happened while you were away.\n")
			output.WriteString("   Don't hesitate to ask your team if you need more context.\n")
		}

		output.WriteString(f.colors.Reset)
	}

	// Sync state reminder
	if !report.Period.Start.IsZero() {
		output.WriteString(fmt.Sprintf("\n%sRun 'vaino catch-up --sync-state' to update your baselines%s\n",
			f.colors.Muted, f.colors.Reset))
	}
}

// Helper methods

func (f *Formatter) formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24

	if days > 0 {
		if hours > 0 {
			return fmt.Sprintf("%d days, %d hours", days, hours)
		}
		return fmt.Sprintf("%d days", days)
	}

	if hours > 0 {
		return fmt.Sprintf("%d hours", hours)
	}

	return fmt.Sprintf("%d minutes", int(d.Minutes()))
}

func (f *Formatter) getChangeIcon(change Change) string {
	switch change.Type {
	case ChangeTypePlanned:
		return "📅"
	case ChangeTypeUnplanned:
		return "🚨"
	case ChangeTypeRoutine:
		return "🔄"
	default:
		return "•"
	}
}

func (f *Formatter) summarizeRoutineChanges(changes []Change) map[string]int {
	summary := make(map[string]int)

	for _, change := range changes {
		// Extract type from description or resource type
		changeType := "Other"

		desc := strings.ToLower(change.Description)
		switch {
		case strings.Contains(desc, "scaling"):
			changeType = "Auto-scaling"
		case strings.Contains(desc, "backup"):
			changeType = "Backups"
		case strings.Contains(desc, "snapshot"):
			changeType = "Snapshots"
		case strings.Contains(desc, "rotation"):
			changeType = "Rotations"
		case strings.Contains(desc, "health"):
			changeType = "Health checks"
		default:
			changeType = change.Resource.Type
		}

		summary[changeType]++
	}

	return summary
}
