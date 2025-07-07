package analyzer

import (
	"fmt"
	"strings"
	"time"
)

// FormatCorrelatedChanges formats grouped changes for display
func FormatCorrelatedChanges(groups []ChangeGroup) string {
	var output strings.Builder

	output.WriteString("📊 Correlated Infrastructure Changes\n")
	output.WriteString("====================================\n\n")

	for i, group := range groups {
		if i > 0 {
			output.WriteString("\n")
		}

		// Group header
		confidenceIcon := "○"
		if group.Confidence == "high" {
			confidenceIcon = "●"
		} else if group.Confidence == "medium" {
			confidenceIcon = "◐"
		}
		
		output.WriteString(fmt.Sprintf("%s 🔗 %s\n", confidenceIcon, group.Title))
		output.WriteString(fmt.Sprintf("   %s\n", group.Description))
		output.WriteString(fmt.Sprintf("   Time: %s\n", group.Timestamp.Format("15:04:05")))
		
		if group.Reason != "" {
			output.WriteString(fmt.Sprintf("   Reason: %s\n", group.Reason))
		}
		
		output.WriteString("\n")

		// Changes in this group
		for _, change := range group.Changes {
			switch change.Type {
			case "added":
				output.WriteString(fmt.Sprintf("   + %s (%s)\n", change.ResourceName, change.ResourceType))
			case "removed":
				output.WriteString(fmt.Sprintf("   - %s (%s)\n", change.ResourceName, change.ResourceType))
			case "modified":
				output.WriteString(fmt.Sprintf("   ~ %s (%s)\n", change.ResourceName, change.ResourceType))
				// Show key changes
				for _, detail := range change.Details {
					output.WriteString(fmt.Sprintf("     • %s: %v → %v\n", 
						detail.Field, detail.OldValue, detail.NewValue))
				}
			}
		}
	}

	return output.String()
}

// FormatChangeTimeline creates a visual timeline of changes
func FormatChangeTimeline(groups []ChangeGroup, duration time.Duration) string {
	var output strings.Builder

	output.WriteString("📅 Change Timeline\n")
	output.WriteString("==================\n\n")

	if len(groups) == 0 {
		output.WriteString("No changes in this time period\n")
		return output.String()
	}

	// Find time bounds
	earliest := groups[0].Timestamp
	latest := groups[0].Timestamp
	
	for _, group := range groups {
		if group.Timestamp.Before(earliest) {
			earliest = group.Timestamp
		}
		if group.Timestamp.After(latest) {
			latest = group.Timestamp
		}
	}

	// Create timeline
	timeRange := latest.Sub(earliest)
	if timeRange == 0 {
		timeRange = 1 * time.Minute // Minimum range
	}

	output.WriteString(fmt.Sprintf("%s ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ %s\n",
		earliest.Format("15:04"),
		latest.Format("15:04")))

	// Plot changes
	lineWidth := 50
	for _, group := range groups {
		position := int(float64(group.Timestamp.Sub(earliest)) / float64(timeRange) * float64(lineWidth))
		if position >= lineWidth {
			position = lineWidth - 1
		}
		
		// Create marker line
		marker := strings.Repeat(" ", position) + "▲"
		label := strings.Repeat(" ", position) + "|"
		
		output.WriteString(fmt.Sprintf("%s\n%s %s (%d changes)\n", 
			marker, label, group.Title, len(group.Changes)))
	}

	return output.String()
}