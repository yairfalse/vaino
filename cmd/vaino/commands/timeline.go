package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/storage"
)

func newTimelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Browse infrastructure snapshots chronologically",
		Long: `Browse stored infrastructure snapshots in chronological order.
This provides a simple view of when snapshots were taken and basic statistics.

For advanced change timeline with correlation analysis, use: vaino changes --timeline
For change comparison between snapshots, use: vaino diff`,
		Example: `  # Show snapshot timeline
  vaino timeline

  # Show snapshots from last 2 weeks
  vaino timeline --since "2 weeks ago"

  # Show snapshots for specific provider
  vaino timeline --provider kubernetes

  # Export timeline as JSON
  vaino timeline --output json`,
		RunE: runTimeline,
	}

	// Date/time filters
	cmd.Flags().StringP("since", "s", "", "show snapshots since date/duration (e.g., '2 weeks ago', '2024-01-01')")
	cmd.Flags().StringP("until", "u", "", "show snapshots until date (e.g., '2024-01-31')")

	// Provider filters
	cmd.Flags().StringSlice("provider", nil, "filter by provider (aws, gcp, kubernetes, terraform)")

	// Output options
	cmd.Flags().BoolP("stats", "", false, "show snapshot statistics")
	cmd.Flags().BoolP("quiet", "q", false, "quiet mode - show timestamps only")
	cmd.Flags().IntP("limit", "l", 50, "limit number of snapshots shown")

	return cmd
}

func runTimeline(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Initialize storage
	localStorage := storage.NewLocal(cfg.Storage.BasePath)

	// Get all snapshots
	snapshots, err := localStorage.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found. Run 'vaino scan' to create your first snapshot.")
		return nil
	}

	// Parse filter options
	sinceTime, err := parseTimeFilter(cmd, "since")
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}

	untilTime, err := parseTimeFilter(cmd, "until")
	if err != nil {
		return fmt.Errorf("invalid --until value: %w", err)
	}

	providers, _ := cmd.Flags().GetStringSlice("provider")
	showStats, _ := cmd.Flags().GetBool("stats")
	quiet, _ := cmd.Flags().GetBool("quiet")
	limit, _ := cmd.Flags().GetInt("limit")

	// Filter snapshots
	filteredSnapshots := filterSnapshots(snapshots, sinceTime, untilTime, "", "", providers)

	// Limit results
	if limit > 0 && len(filteredSnapshots) > limit {
		filteredSnapshots = filteredSnapshots[:limit]
	}

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	// Display timeline
	return displaySnapshotTimeline(filteredSnapshots, outputFormat, showStats, quiet)
}

func parseTimeFilter(cmd *cobra.Command, flagName string) (*time.Time, error) {
	value, _ := cmd.Flags().GetString(flagName)
	if value == "" {
		return nil, nil
	}

	// Try parsing as duration first (e.g., "2 weeks ago", "3 days ago")
	if strings.Contains(value, "ago") {
		return parseDurationAgo(value)
	}

	// Try parsing as date
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unable to parse time: %s", value)
}

func parseDurationAgo(value string) (*time.Time, error) {
	// Simple parser for durations like "2 weeks ago", "3 days ago"
	parts := strings.Fields(value)
	if len(parts) < 3 || parts[len(parts)-1] != "ago" {
		return nil, fmt.Errorf("invalid duration format: %s", value)
	}

	amountStr := parts[0]
	unit := parts[1]

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %s", amountStr)
	}

	var duration time.Duration
	switch strings.ToLower(unit) {
	case "minute", "minutes":
		duration = time.Duration(amount) * time.Minute
	case "hour", "hours":
		duration = time.Duration(amount) * time.Hour
	case "day", "days":
		duration = time.Duration(amount) * 24 * time.Hour
	case "week", "weeks":
		duration = time.Duration(amount) * 7 * 24 * time.Hour
	case "month", "months":
		duration = time.Duration(amount) * 30 * 24 * time.Hour
	default:
		return nil, fmt.Errorf("unsupported time unit: %s", unit)
	}

	t := time.Now().Add(-duration)
	return &t, nil
}

func filterSnapshots(snapshots []storage.SnapshotInfo, since, until *time.Time, fromID, toID string, providers []string) []storage.SnapshotInfo {
	var filtered []storage.SnapshotInfo

	// Create provider filter map
	providerFilter := make(map[string]bool)
	for _, p := range providers {
		providerFilter[strings.ToLower(p)] = true
	}

	for _, snapshot := range snapshots {
		// Time filters
		if since != nil && snapshot.Timestamp.Before(*since) {
			continue
		}
		if until != nil && snapshot.Timestamp.After(*until) {
			continue
		}

		// Provider filter
		if len(providerFilter) > 0 && !providerFilter[strings.ToLower(snapshot.Provider)] {
			continue
		}

		filtered = append(filtered, snapshot)
	}

	// Sort by timestamp (oldest first for timeline view)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	return filtered
}

func displaySnapshotTimeline(snapshots []storage.SnapshotInfo, outputFormat string, showStats, quiet bool) error {
	if outputFormat == "json" {
		return timelineOutputJSON(snapshots)
	}

	if quiet {
		return displayTimelineQuiet(snapshots)
	}

	// Default text format
	fmt.Printf("Infrastructure Snapshot Timeline (%d snapshots)\n", len(snapshots))
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	for _, snapshot := range snapshots {
		fmt.Printf("📅 %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Provider: %s\n", snapshot.Provider)
		fmt.Printf("   Resources: %d\n", snapshot.ResourceCount)
		fmt.Printf("   ID: %s\n", snapshot.ID)

		if len(snapshot.Tags) > 0 {
			fmt.Print("   Tags: ")
			var tags []string
			for k, v := range snapshot.Tags {
				tags = append(tags, fmt.Sprintf("%s=%s", k, v))
			}
			fmt.Println(strings.Join(tags, ", "))
		}
		fmt.Println()
	}

	if showStats {
		displaySnapshotStats(snapshots)
	}

	fmt.Println("💡 For advanced change timeline with correlation analysis, use:")
	fmt.Println("   vaino changes --timeline")

	return nil
}

func timelineOutputJSON(snapshots []storage.SnapshotInfo) error {
	// Convert to a simple JSON structure
	type TimelineEntry struct {
		Timestamp     time.Time         `json:"timestamp"`
		Provider      string            `json:"provider"`
		ResourceCount int               `json:"resource_count"`
		ID            string            `json:"id"`
		Tags          map[string]string `json:"tags,omitempty"`
	}

	var entries []TimelineEntry
	for _, snapshot := range snapshots {
		entries = append(entries, TimelineEntry{
			Timestamp:     snapshot.Timestamp,
			Provider:      snapshot.Provider,
			ResourceCount: snapshot.ResourceCount,
			ID:            snapshot.ID,
			Tags:          snapshot.Tags,
		})
	}

	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

func displayTimelineQuiet(snapshots []storage.SnapshotInfo) error {
	for _, snapshot := range snapshots {
		fmt.Printf("%s %s %d\n",
			snapshot.Timestamp.Format("2006-01-02T15:04:05"),
			snapshot.Provider,
			snapshot.ResourceCount)
	}
	return nil
}

func displaySnapshotStats(snapshots []storage.SnapshotInfo) {
	if len(snapshots) == 0 {
		return
	}

	fmt.Println("Snapshot Statistics")
	fmt.Println(strings.Repeat("-", 30))

	// Provider distribution
	providers := make(map[string]int)
	totalResources := 0

	for _, snapshot := range snapshots {
		providers[snapshot.Provider]++
		totalResources += snapshot.ResourceCount
	}

	fmt.Printf("Total snapshots: %d\n", len(snapshots))
	fmt.Printf("Date range: %s to %s\n",
		snapshots[0].Timestamp.Format("2006-01-02"),
		snapshots[len(snapshots)-1].Timestamp.Format("2006-01-02"))
	fmt.Printf("Total resources: %d\n", totalResources)
	fmt.Printf("Average resources per snapshot: %.1f\n", float64(totalResources)/float64(len(snapshots)))

	fmt.Println("\nProvider distribution:")
	for provider, count := range providers {
		fmt.Printf("  %s: %d snapshots\n", provider, count)
	}
	fmt.Println()
}
