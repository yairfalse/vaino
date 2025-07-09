package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

// newDriftCommand creates the new simplified drift detection command
func newDriftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift",
		Short: "Detect infrastructure drift",
		Long: `Detect changes in your infrastructure by comparing current state 
to previously saved reference points.

By default, compares to the most recent saved state. Use --since to compare
to specific reference points by name, date, or relative time.`,
		Example: `  # Show drift from last saved state
  wgo drift
  
  # Compare to specific saved state
  wgo drift --since prod-release-v2.1
  wgo drift --since yesterday
  wgo drift --since "3 days ago"
  wgo drift --since 2025-01-10
  
  # Compare between two states
  wgo drift --from prod-v1 --to prod-v2
  
  # Filter results
  wgo drift --severity high --provider gcp
  
  # Export results
  wgo drift --format json --output drift-report.json`,
		RunE: runDrift,
	}

	// Comparison flags
	cmd.Flags().String("since", "", "reference point to compare against (name, date, or relative time)")
	cmd.Flags().String("from", "", "source state for comparison")
	cmd.Flags().String("to", "", "target state for comparison (defaults to current)")

	// Filter flags
	cmd.Flags().StringP("provider", "p", "", "filter by provider")
	cmd.Flags().StringSlice("region", []string{}, "filter by regions")
	cmd.Flags().String("severity", "", "minimum severity to show (low, medium, high, critical)")
	cmd.Flags().StringSlice("resource-type", []string{}, "filter by resource types")
	cmd.Flags().Bool("ignore-tags", false, "ignore tag changes")
	cmd.Flags().StringSlice("ignore-fields", []string{}, "ignore specific fields")

	// Output flags
	cmd.Flags().StringP("format", "f", "table", "output format (table, json, yaml, markdown)")
	cmd.Flags().StringP("output", "o", "", "save results to file")
	cmd.Flags().Bool("summary", false, "show summary only")
	cmd.Flags().Bool("quiet", false, "minimal output (exit code indicates drift)")

	// Advanced flags
	cmd.Flags().Bool("rescan", false, "force new scan before comparison")
	cmd.Flags().Bool("fail-on-drift", false, "exit with error if drift detected")

	return cmd
}

func runDrift(cmd *cobra.Command, args []string) error {
	// Parse flags
	since, _ := cmd.Flags().GetString("since")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")
	quiet, _ := cmd.Flags().GetBool("quiet")
	rescan, _ := cmd.Flags().GetBool("rescan")

	if !quiet {
		fmt.Println("🔍 Drift Detection")
		fmt.Println("==================")
	}

	// Initialize storage
	localStorage, err := storage.NewLocalStorage(storage.Config{BaseDir: "./wgo-states"})
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Determine reference and current states
	var referenceState, currentState *types.Snapshot

	// Handle --from/--to pattern
	if from != "" {
		referenceState, err = loadState(localStorage, from)
		if err != nil {
			return fmt.Errorf("failed to load 'from' state: %w", err)
		}

		if to != "" {
			currentState, err = loadState(localStorage, to)
			if err != nil {
				return fmt.Errorf("failed to load 'to' state: %w", err)
			}
		}
	} else {
		// Handle --since pattern (most common)
		if since == "" {
			// Default: use most recent saved state as reference
			states, err := localStorage.ListSnapshots()
			if err != nil {
				return fmt.Errorf("failed to list states: %w", err)
			}

			if len(states) < 2 {
				if !quiet {
					fmt.Println("ℹ️  No previous state found for comparison")
					fmt.Println("\n💡 TIP: Save a reference state first:")
					fmt.Println("    wgo scan --save")
				}
				return nil
			}

			// states[0] is current, states[1] is previous
			referenceState, err = localStorage.LoadSnapshot(states[1].ID)
			if err != nil {
				return fmt.Errorf("failed to load reference state: %w", err)
			}

			if !quiet {
				fmt.Printf("📌 Comparing to: %s (auto-selected)\n",
					formatTimestamp(referenceState.Timestamp))
			}
		} else {
			// Parse --since value
			referenceState, err = parseReferencePoint(localStorage, since)
			if err != nil {
				return fmt.Errorf("failed to load reference state '%s': %w", since, err)
			}

			if !quiet {
				fmt.Printf("📌 Comparing to: %s\n", since)
			}
		}
	}

	// Get current state
	if currentState == nil {
		if rescan || !hasRecentScan(localStorage) {
			if !quiet {
				fmt.Println("🔄 Scanning current infrastructure...")
			}
			// TODO: Trigger scan
			currentState = createPlaceholderSnapshot("current")
		} else {
			// Use most recent scan
			states, _ := localStorage.ListSnapshots()
			if len(states) > 0 {
				currentState, _ = localStorage.LoadSnapshot(states[0].ID)
			}
		}
	}

	// Perform drift detection
	differ := differ.NewDifferEngine(differ.DiffOptions{
		IgnoreFields: getIgnoreFields(cmd),
		MinRiskLevel: parseSeverity(cmd),
	})

	report, err := differ.Compare(referenceState, currentState)
	if err != nil {
		return fmt.Errorf("drift detection failed: %w", err)
	}

	// Handle quiet mode
	if quiet {
		if report.Summary.ChangedResources > 0 {
			return fmt.Errorf("drift detected")
		}
		return nil
	}

	// Display results
	if format == "table" && outputFile == "" {
		displayDriftSummary(report)
	} else {
		exportDrift(report, format, outputFile)
	}

	// Handle fail-on-drift
	failOnDrift, _ := cmd.Flags().GetBool("fail-on-drift")
	if failOnDrift && report.Summary.ChangedResources > 0 {
		return fmt.Errorf("drift detected in %d resources", report.Summary.ChangedResources)
	}

	return nil
}

// Helper functions

func loadState(storage storage.Storage, identifier string) (*types.Snapshot, error) {
	// Try as snapshot ID first
	snapshot, err := storage.LoadSnapshot(identifier)
	if err == nil {
		return snapshot, nil
	}

	// Try as file path
	if strings.HasSuffix(identifier, ".json") {
		return loadSnapshotFromFile(identifier)
	}

	// Try as saved state name
	// TODO: Implement named state lookup

	return nil, fmt.Errorf("state not found: %s", identifier)
}

func parseReferencePoint(storage storage.Storage, reference string) (*types.Snapshot, error) {
	// Handle relative time references
	relativeTime := parseRelativeTime(reference)
	if relativeTime != nil {
		return findSnapshotByTime(storage, *relativeTime)
	}

	// Handle absolute dates
	if timestamp, err := time.Parse("2006-01-02", reference); err == nil {
		return findSnapshotByTime(storage, timestamp)
	}

	// Handle named references
	return loadState(storage, reference)
}

func parseRelativeTime(reference string) *time.Time {
	now := time.Now()
	reference = strings.ToLower(reference)

	switch reference {
	case "yesterday":
		t := now.AddDate(0, 0, -1)
		return &t
	case "last week":
		t := now.AddDate(0, 0, -7)
		return &t
	case "last month":
		t := now.AddDate(0, -1, 0)
		return &t
	}

	// Parse "X days ago", "X hours ago", etc.
	// TODO: Implement more sophisticated parsing

	return nil
}

func findSnapshotByTime(storage storage.Storage, targetTime time.Time) (*types.Snapshot, error) {
	snapshots, err := storage.ListSnapshots()
	if err != nil {
		return nil, err
	}

	// Find the snapshot closest to but before the target time
	for _, meta := range snapshots {
		snapshot, err := storage.LoadSnapshot(meta.ID)
		if err != nil {
			continue
		}

		if snapshot.Timestamp.Before(targetTime) || snapshot.Timestamp.Equal(targetTime) {
			return snapshot, nil
		}
	}

	return nil, fmt.Errorf("no snapshot found before %s", targetTime.Format("2006-01-02"))
}

func hasRecentScan(storage storage.Storage) bool {
	snapshots, err := storage.ListSnapshots()
	if err != nil || len(snapshots) == 0 {
		return false
	}

	// Consider scan recent if less than 5 minutes old
	latestSnapshot, _ := storage.LoadSnapshot(snapshots[0].ID)
	return time.Since(latestSnapshot.Timestamp) < 5*time.Minute
}

func formatTimestamp(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else if duration < 7*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	}

	return t.Format("2006-01-02 15:04")
}

func displayDriftSummary(report *differ.DriftReport) {
	if report.Summary.ChangedResources == 0 {
		fmt.Println("\n✅ No drift detected")
		fmt.Println("Your infrastructure matches the reference state")
		return
	}

	// Summary header
	fmt.Printf("\n⚠️  Drift detected in %d resources\n", report.Summary.ChangedResources)
	fmt.Println()

	// Quick stats
	if report.Summary.AddedResources > 0 {
		fmt.Printf("  + %d added\n", report.Summary.AddedResources)
	}
	if report.Summary.RemovedResources > 0 {
		fmt.Printf("  - %d removed\n", report.Summary.RemovedResources)
	}
	if report.Summary.ModifiedResources > 0 {
		fmt.Printf("  ~ %d modified\n", report.Summary.ModifiedResources)
	}

	// Risk assessment
	fmt.Printf("\n📊 Risk Level: %s (%.1f/10)\n",
		strings.Title(string(report.Summary.OverallRisk)),
		report.Summary.RiskScore*10)

	// Top changes
	if len(report.ResourceChanges) > 0 {
		fmt.Println("\n🔍 Key Changes:")
		shown := 0
		for _, change := range report.ResourceChanges {
			if shown >= 5 {
				remaining := len(report.ResourceChanges) - shown
				if remaining > 0 {
					fmt.Printf("\n   ... and %d more changes\n", remaining)
				}
				break
			}

			icon := "~"
			if change.DriftType == "added" {
				icon = "+"
			} else if change.DriftType == "removed" {
				icon = "-"
			}

			fmt.Printf("\n  %s %s.%s\n", icon, change.Provider, change.ResourceID)
			if change.Description != "" {
				fmt.Printf("    %s\n", change.Description)
			}

			shown++
		}
	}

	// Next steps
	fmt.Println("\n💡 Next Steps:")
	fmt.Println("  • Review changes: wgo drift --format markdown")
	fmt.Println("  • Get AI analysis: wgo explain")
	fmt.Println("  • Save new reference: wgo scan --save")
}

func getIgnoreFields(cmd *cobra.Command) []string {
	ignoreFields, _ := cmd.Flags().GetStringSlice("ignore-fields")
	ignoreTags, _ := cmd.Flags().GetBool("ignore-tags")

	if ignoreTags {
		ignoreFields = append(ignoreFields, "tags", "Tags", "labels", "Labels")
	}

	return ignoreFields
}

func parseSeverity(cmd *cobra.Command) differ.RiskLevel {
	severity, _ := cmd.Flags().GetString("severity")
	switch strings.ToLower(severity) {
	case "critical":
		return differ.RiskLevelCritical
	case "high":
		return differ.RiskLevelHigh
	case "medium":
		return differ.RiskLevelMedium
	default:
		return differ.RiskLevelLow
	}
}

func exportDrift(report *differ.DriftReport, format, outputFile string) {
	// TODO: Implement export functionality
	fmt.Printf("Exporting drift report as %s to %s\n", format, outputFile)
}
