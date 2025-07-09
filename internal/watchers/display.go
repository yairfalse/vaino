package watchers

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yairfalse/wgo/internal/analyzer"
)

// displayChanges shows detected changes in the configured format
func (w *Watcher) displayChanges(event *DisplayEvent) {
	switch w.outputFormat {
	case "table":
		w.displayTableFormat(event)
	case "json":
		w.displayJSONFormat(event)
	case "quiet":
		w.displayQuietFormat(event)
	default:
		w.displayTableFormat(event)
	}
}

// displayTableFormat shows changes in a human-readable table format
func (w *Watcher) displayTableFormat(event *DisplayEvent) {
	timestamp := event.Timestamp.Format("15:04:05")

	// Summary line
	fmt.Printf("[%s] %d changes detected (%d added, %d modified, %d removed)\n",
		timestamp,
		event.Summary.Total,
		event.Summary.Added,
		event.Summary.Modified,
		event.Summary.Removed)

	// Show correlated groups if available
	if len(event.CorrelatedGroups) > 0 {
		fmt.Printf("┌─ 🔗 Correlated Changes:\n")

		for _, group := range event.CorrelatedGroups {
			// Skip low confidence groups if only showing high confidence
			if w.onlyHighConf && group.Confidence != "high" {
				continue
			}

			confidence := w.getConfidenceIndicator(group.Confidence)
			fmt.Printf("├─ %s %s %s (%d changes)\n",
				confidence,
				group.Title,
				group.Description,
				len(group.Changes))

			// Show individual changes in the group
			for i, change := range group.Changes {
				prefix := "├──"
				if i == len(group.Changes)-1 {
					prefix = "└──"
				}

				fmt.Printf("│  %s %s %s/%s\n",
					prefix,
					w.getChangeTypeIcon(change.Type),
					change.ResourceType,
					change.ResourceName)
			}
		}
		fmt.Printf("└─\n")
	} else {
		// Show individual changes if no correlation
		fmt.Printf("┌─ Individual Changes:\n")
		for i, change := range event.RawChanges {
			prefix := "├─"
			if i == len(event.RawChanges)-1 {
				prefix = "└─"
			}

			fmt.Printf("%s %s %s/%s in %s\n",
				prefix,
				w.getChangeTypeIcon(change.Type),
				change.ResourceType,
				change.ResourceName,
				change.Namespace)
		}
	}

	fmt.Printf("\n")
}

// displayJSONFormat outputs changes as JSON for automation
func (w *Watcher) displayJSONFormat(event *DisplayEvent) {
	// Filter groups by confidence if needed
	filteredEvent := *event
	if w.onlyHighConf {
		var filteredGroups []analyzer.ChangeGroup
		for _, group := range event.CorrelatedGroups {
			if group.Confidence == "high" {
				filteredGroups = append(filteredGroups, group)
			}
		}
		filteredEvent.CorrelatedGroups = filteredGroups
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(filteredEvent)
}

// displayQuietFormat shows minimal output for scripts
func (w *Watcher) displayQuietFormat(event *DisplayEvent) {
	timestamp := event.Timestamp.Format("15:04:05")

	// Only show high-confidence correlated changes in quiet mode
	for _, group := range event.CorrelatedGroups {
		if group.Confidence == "high" {
			fmt.Printf("[%s] %s\n", timestamp, group.Title)
		}
	}

	// If no high-confidence correlations, show summary
	if len(event.CorrelatedGroups) == 0 {
		fmt.Printf("[%s] %d changes\n", timestamp, event.Summary.Total)
	}
}

// getConfidenceIndicator returns a visual indicator for confidence level
func (w *Watcher) getConfidenceIndicator(confidence string) string {
	switch confidence {
	case "high":
		return "[H]" // High confidence
	case "medium":
		return "[M]" // Medium confidence
	case "low":
		return "[L]" // Low confidence
	default:
		return "[?]"
	}
}

// getChangeTypeIcon returns an icon for the change type
func (w *Watcher) getChangeTypeIcon(changeType string) string {
	switch changeType {
	case "added":
		return "+"
	case "removed":
		return "-"
	case "modified":
		return "~"
	default:
		return "?"
	}
}

// displayWatchHeader shows initial watch information
func (w *Watcher) displayWatchHeader() {
	if w.quiet {
		return
	}

	fmt.Printf("🔍 WGO Watch Mode\n")
	fmt.Printf("================\n")
	fmt.Printf("Monitoring: %s\n", strings.Join(w.providers, ", "))
	fmt.Printf("Interval: %v\n", w.interval)
	fmt.Printf("Output: %s\n", w.outputFormat)

	if w.onlyHighConf {
		fmt.Printf("Filter: High confidence only\n")
	}

	if w.webhookURL != "" {
		fmt.Printf("Webhook: Enabled\n")
	}

	fmt.Printf("\nPress Ctrl+C to stop watching\n")
	fmt.Printf(strings.Repeat("─", 50) + "\n\n")
}

// displayStatistics shows periodic statistics (called every N intervals)
func (w *Watcher) displayStatistics(totalChecks, totalChanges int) {
	if w.quiet {
		return
	}

	fmt.Printf("Statistics: %d checks performed, %d change events detected\n\n",
		totalChecks, totalChanges)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
