package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/types"
)

func newHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Browse infrastructure history and snapshots",
		Long: `Browse stored infrastructure snapshots and history.
Provides detailed view of individual snapshots and their contents.`,
		Example: `  # List all snapshots
  vaino history

  # List snapshots since a date
  vaino history --since "june-1"

  # Show details of a specific snapshot
  vaino history show snapshot-id

  # List snapshots in JSON format
  vaino history --output json

  # Show snapshot with resource details
  vaino history show --details snapshot-id`,
		RunE: runHistory,
	}

	// Add subcommands
	cmd.AddCommand(newHistoryShowCommand())
	cmd.AddCommand(newHistoryListCommand())

	// Date/time filters for default list behavior
	cmd.Flags().StringP("since", "s", "", "show history since date/duration")
	cmd.Flags().StringP("until", "u", "", "show history until date")
	cmd.Flags().StringSlice("provider", nil, "filter by provider")
	cmd.Flags().IntP("limit", "l", 20, "limit number of snapshots shown")
	cmd.Flags().BoolP("quiet", "q", false, "quiet mode - show IDs only")

	return cmd
}

func runHistory(cmd *cobra.Command, args []string) error {
	// Default behavior: list snapshots
	return runHistoryList(cmd, args)
}

func newHistoryListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored snapshots",
		Long:  `List all stored infrastructure snapshots with filtering options.`,
		RunE:  runHistoryList,
	}

	cmd.Flags().StringP("since", "s", "", "show snapshots since date/duration")
	cmd.Flags().StringP("until", "u", "", "show snapshots until date")
	cmd.Flags().StringSlice("provider", nil, "filter by provider")
	cmd.Flags().IntP("limit", "l", 20, "limit number of snapshots shown")
	cmd.Flags().BoolP("quiet", "q", false, "quiet mode - show IDs only")

	return cmd
}

func newHistoryShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [snapshot-id]",
		Short: "Show details of a specific snapshot",
		Long: `Show detailed information about a specific snapshot including
resource counts, metadata, and optionally full resource details.`,
		Args: cobra.ExactArgs(1),
		RunE: runHistoryShow,
	}

	cmd.Flags().BoolP("details", "d", false, "show full resource details")
	cmd.Flags().StringSlice("resource-type", nil, "filter resources by type")
	cmd.Flags().StringSlice("region", nil, "filter resources by region")

	return cmd
}

func runHistoryList(cmd *cobra.Command, args []string) error {
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

	// Parse filter options - reuse from timeline.go
	sinceTime, err := parseTimeFilter(cmd, "since")
	if err != nil {
		return fmt.Errorf("invalid --since value: %w", err)
	}

	untilTime, err := parseTimeFilter(cmd, "until")
	if err != nil {
		return fmt.Errorf("invalid --until value: %w", err)
	}

	providers, _ := cmd.Flags().GetStringSlice("provider")
	limit, _ := cmd.Flags().GetInt("limit")
	quiet, _ := cmd.Flags().GetBool("quiet")

	// Filter snapshots - reuse from timeline.go
	filteredSnapshots := filterSnapshots(snapshots, sinceTime, untilTime, "", "", providers)

	// Limit results
	if limit > 0 && len(filteredSnapshots) > limit {
		filteredSnapshots = filteredSnapshots[:limit]
	}

	// Sort by timestamp (newest first for history view)
	sort.Slice(filteredSnapshots, func(i, j int) bool {
		return filteredSnapshots[i].Timestamp.After(filteredSnapshots[j].Timestamp)
	})

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	return displayHistoryList(filteredSnapshots, outputFormat, quiet)
}

func runHistoryShow(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()
	snapshotID := args[0]

	// Initialize storage
	localStorage := storage.NewLocal(cfg.Storage.BasePath)

	// Load the snapshot
	snapshot, err := localStorage.LoadSnapshot(snapshotID)
	if err != nil {
		return fmt.Errorf("failed to load snapshot %s: %w", snapshotID, err)
	}

	// Get options
	showDetails, _ := cmd.Flags().GetBool("details")
	resourceTypes, _ := cmd.Flags().GetStringSlice("resource-type")
	regions, _ := cmd.Flags().GetStringSlice("region")
	outputFormat, _ := cmd.Flags().GetString("output")

	return displaySnapshotDetails(snapshot, outputFormat, showDetails, resourceTypes, regions)
}

func displayHistoryList(snapshots []storage.SnapshotInfo, outputFormat string, quiet bool) error {
	if outputFormat == "json" {
		return historyOutputJSON(snapshots)
	}

	if quiet {
		for _, snapshot := range snapshots {
			fmt.Println(snapshot.ID)
		}
		return nil
	}

	// Default text format
	fmt.Printf("Infrastructure History (%d snapshots)\n", len(snapshots))
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// Display in table format
	fmt.Printf("%-25s %-12s %-12s %-8s %-10s\n", "TIMESTAMP", "PROVIDER", "ID", "RESOURCES", "SIZE")
	fmt.Println(strings.Repeat("-", 80))

	for _, snapshot := range snapshots {
		fmt.Printf("%-25s %-12s %-12s %-8d %-10s\n",
			snapshot.Timestamp.Format("2006-01-02 15:04:05"),
			snapshot.Provider,
			truncateString(snapshot.ID, 12),
			snapshot.ResourceCount,
			formatFileSize(snapshot.FileSize))
	}

	return nil
}

func displaySnapshotDetails(snapshot *types.Snapshot, outputFormat string, showDetails bool, resourceTypes, regions []string) error {
	if outputFormat == "json" {
		if !showDetails {
			// Return summary version
			summary := struct {
				ID            string            `json:"id"`
				Timestamp     time.Time         `json:"timestamp"`
				Provider      string            `json:"provider"`
				ResourceCount int               `json:"resource_count"`
				Metadata      map[string]string `json:"metadata"`
			}{
				ID:            snapshot.ID,
				Timestamp:     snapshot.Timestamp,
				Provider:      snapshot.Provider,
				ResourceCount: len(snapshot.Resources),
				Metadata:      snapshot.Metadata.Tags,
			}
			return historyOutputJSON(summary)
		}
		return historyOutputJSON(snapshot)
	}

	// Text format
	fmt.Printf("Snapshot Details\n")
	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("ID:          %s\n", snapshot.ID)
	fmt.Printf("Timestamp:   %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Provider:    %s\n", snapshot.Provider)
	fmt.Printf("Resources:   %d\n", len(snapshot.Resources))

	if len(snapshot.Metadata.Tags) > 0 {
		fmt.Printf("Tags:        ")
		var tags []string
		for k, v := range snapshot.Metadata.Tags {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}
		fmt.Printf("%s\n", strings.Join(tags, ", "))
	}

	fmt.Println()

	// Resource summary by type and region
	resourcesByType := make(map[string]int)
	resourcesByRegion := make(map[string]int)

	for _, resource := range snapshot.Resources {
		resourcesByType[resource.Type]++
		if resource.Region != "" {
			resourcesByRegion[resource.Region]++
		}
	}

	fmt.Println("Resource Summary")
	fmt.Println(strings.Repeat("-", 20))

	fmt.Println("By Type:")
	for resourceType, count := range resourcesByType {
		fmt.Printf("  %-20s %d\n", resourceType, count)
	}

	if len(resourcesByRegion) > 0 {
		fmt.Println("\nBy Region:")
		for region, count := range resourcesByRegion {
			fmt.Printf("  %-20s %d\n", region, count)
		}
	}

	// Show detailed resource list if requested
	if showDetails {
		fmt.Println()
		displayResourceDetails(snapshot.Resources, resourceTypes, regions)
	}

	return nil
}

func displayResourceDetails(resources []types.Resource, resourceTypeFilter, regionFilter []string) {
	// Create filter maps
	typeFilter := make(map[string]bool)
	for _, t := range resourceTypeFilter {
		typeFilter[strings.ToLower(t)] = true
	}

	regFilter := make(map[string]bool)
	for _, r := range regionFilter {
		regFilter[strings.ToLower(r)] = true
	}

	fmt.Println("Resource Details")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Printf("%-20s %-30s %-15s %-15s\n", "TYPE", "NAME", "REGION", "ID")
	fmt.Println(strings.Repeat("-", 80))

	for _, resource := range resources {
		// Apply filters
		if len(typeFilter) > 0 && !typeFilter[strings.ToLower(resource.Type)] {
			continue
		}
		if len(regFilter) > 0 && !regFilter[strings.ToLower(resource.Region)] {
			continue
		}

		fmt.Printf("%-20s %-30s %-15s %-15s\n",
			truncateString(resource.Type, 20),
			truncateString(resource.Name, 30),
			resource.Region,
			truncateString(resource.ID, 15))
	}
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// historyOutputJSON outputs data as pretty-printed JSON (unique name to avoid conflicts)
func historyOutputJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}
