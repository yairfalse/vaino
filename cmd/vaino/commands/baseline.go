package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/storage"
)

func newBaselineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Manage infrastructure baselines",
		Long: `Manage infrastructure baselines used for drift detection.
Baselines represent known good states of your infrastructure.`,
		Example: `  # Create baseline from current state
  vaino baseline create --name prod-v1.0 --description "Production baseline v1.0"

  # Create baseline from existing snapshot
  vaino baseline create --from-snapshot snapshot-123.json --name staging-v2.1

  # List all baselines
  vaino baseline list

  # Show baseline details
  vaino baseline show prod-v1.0

  # Delete baseline
  vaino baseline delete old-baseline`,
	}

	// Subcommands
	cmd.AddCommand(newBaselineCreateCommand())
	cmd.AddCommand(newBaselineListCommand())
	cmd.AddCommand(newBaselineShowCommand())
	cmd.AddCommand(newBaselineDeleteCommand())

	return cmd
}

func newBaselineCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new baseline",
		Long: `Create a new baseline from current infrastructure state or existing snapshot.
Baselines are used as reference points for drift detection.`,
		RunE: runBaselineCreate,
	}

	cmd.Flags().StringP("name", "n", "", "baseline name (required)")
	cmd.Flags().StringP("description", "d", "", "baseline description")
	cmd.Flags().String("from-snapshot", "", "create baseline from existing snapshot")
	cmd.Flags().StringSlice("tags", []string{}, "baseline tags (key=value)")
	cmd.Flags().StringSlice("provider", []string{}, "limit to specific providers")
	cmd.Flags().StringSlice("region", []string{}, "limit to specific regions")

	cmd.MarkFlagRequired("name")

	return cmd
}

func newBaselineListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all baselines",
		Long:  `List all stored baselines with their metadata.`,
		RunE:  runBaselineList,
	}

	cmd.Flags().StringP("filter", "f", "", "filter baselines by name pattern")
	cmd.Flags().StringSlice("tags", []string{}, "filter by tags (key=value)")
	cmd.Flags().String("sort", "created", "sort by (name, created, updated)")
	cmd.Flags().Bool("reverse", false, "reverse sort order")

	return cmd
}

func newBaselineShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [baseline-name]",
		Short: "Show baseline details",
		Long:  `Display detailed information about a specific baseline.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runBaselineShow,
	}

	cmd.Flags().Bool("resources", false, "show detailed resource information")
	cmd.Flags().StringSlice("provider", []string{}, "filter resources by provider")

	return cmd
}

func newBaselineDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [baseline-name]",
		Short: "Delete a baseline",
		Long:  `Delete a baseline permanently. This action cannot be undone.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runBaselineDelete,
	}

	cmd.Flags().Bool("force", false, "force deletion without confirmation")

	return cmd
}

func runBaselineCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	fromSnapshot, _ := cmd.Flags().GetString("from-snapshot")

	// Check if any snapshots exist
	// TODO: Actually check for snapshots
	hasSnapshots := false

	if !hasSnapshots && fromSnapshot == "" {
		fmt.Println("❌ No Infrastructure Snapshots Found")
		fmt.Println("=====================================")
		fmt.Println()
		fmt.Println("You need to scan your infrastructure first!")
		fmt.Println()
		fmt.Println("🎯 DO THIS NOW:")
		fmt.Println()
		fmt.Println("  1. Run a scan (choose one):")
		fmt.Println("     vaino scan --provider terraform")
		fmt.Println("     vaino scan --provider aws --region us-east-1")
		fmt.Println("     vaino scan --provider gcp --project YOUR-PROJECT")
		fmt.Println()
		fmt.Println("  2. Then create your baseline:")
		fmt.Printf("     vaino baseline create --name %s", name)
		if description != "" {
			fmt.Printf(" --description \"%s\"", description)
		}
		fmt.Println()
		fmt.Println()
		fmt.Println("💡 TIP: Having auth issues? Run 'vaino auth status'")
		return nil
	}

	fmt.Println("📋 Creating Baseline")
	fmt.Println("===================")

	fmt.Printf("🏷️  Name: %s\n", name)
	if description != "" {
		fmt.Printf("📝 Description: %s\n", description)
	}
	if fromSnapshot != "" {
		fmt.Printf("📊 From snapshot: %s\n", fromSnapshot)
	}

	fmt.Println("\n⚠️  Baseline creation not yet implemented")

	return nil
}

func runBaselineList(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Initialize storage
	localStorage := storage.NewLocal(cfg.Storage.BasePath)

	// Get all snapshots
	snapshots, err := localStorage.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Filter for baselines (snapshots with baseline tag)
	var baselines []storage.SnapshotInfo
	filter, _ := cmd.Flags().GetString("filter")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	// Create tag filter map
	tagFilter := make(map[string]string)
	for _, tag := range tags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 2 {
			tagFilter[parts[0]] = parts[1]
		}
	}

	for _, snapshot := range snapshots {
		// Check if it's a baseline
		baselineName, isBaseline := snapshot.Tags["baseline"]
		if !isBaseline {
			continue
		}

		// Apply name filter
		if filter != "" && !strings.Contains(strings.ToLower(baselineName), strings.ToLower(filter)) {
			continue
		}

		// Apply tag filters
		skipSnapshot := false
		for k, v := range tagFilter {
			if snapshot.Tags[k] != v {
				skipSnapshot = true
				break
			}
		}
		if skipSnapshot {
			continue
		}

		baselines = append(baselines, snapshot)
	}

	if len(baselines) == 0 {
		fmt.Println("No baselines found. Create one with 'vaino baseline create'")
		return nil
	}

	// Sort baselines
	sortBy, _ := cmd.Flags().GetString("sort")
	reverse, _ := cmd.Flags().GetBool("reverse")

	sortBaselines(baselines, sortBy, reverse)

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	return displayBaselines(baselines, outputFormat)
}

func runBaselineShow(cmd *cobra.Command, args []string) error {
	baselineName := args[0]

	fmt.Printf("📋 Baseline Details: %s\n", baselineName)
	fmt.Println("================================")

	fmt.Println("\n⚠️  Baseline show not yet implemented")

	return nil
}

func runBaselineDelete(cmd *cobra.Command, args []string) error {
	baselineName := args[0]
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		fmt.Printf("Are you sure you want to delete baseline '%s'? (y/N): ", baselineName)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	fmt.Printf("🗑️  Deleting baseline: %s\n", baselineName)
	fmt.Println("\n⚠️  Baseline deletion not yet implemented")

	return nil
}

// Helper functions for baseline listing

func sortBaselines(baselines []storage.SnapshotInfo, sortBy string, reverse bool) {
	sort.Slice(baselines, func(i, j int) bool {
		var result bool
		switch sortBy {
		case "name":
			result = baselines[i].Tags["baseline"] < baselines[j].Tags["baseline"]
		case "updated":
			// For now, use timestamp as update time
			result = baselines[i].Timestamp.Before(baselines[j].Timestamp)
		default: // "created"
			result = baselines[i].Timestamp.Before(baselines[j].Timestamp)
		}

		if reverse {
			return !result
		}
		return result
	})
}

func displayBaselines(baselines []storage.SnapshotInfo, outputFormat string) error {
	if outputFormat == "json" {
		return baselineOutputJSON(baselines)
	}

	// Default table format
	fmt.Println("📋 Infrastructure Baselines")
	fmt.Println("===========================")
	fmt.Println()

	fmt.Printf("%-25s %-20s %-12s %-8s %-30s\n", "NAME", "CREATED", "PROVIDER", "RESOURCES", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 100))

	for _, baseline := range baselines {
		name := baseline.Tags["baseline"]
		description := baseline.Tags["description"]
		if description == "" {
			description = "-"
		}
		if len(description) > 30 {
			description = description[:27] + "..."
		}

		fmt.Printf("%-25s %-20s %-12s %-8d %-30s\n",
			truncateString(name, 25),
			baseline.Timestamp.Format("2006-01-02 15:04"),
			baseline.Provider,
			baseline.ResourceCount,
			description)
	}

	fmt.Printf("\nTotal: %d baselines\n", len(baselines))
	fmt.Println("\n💡 Use 'vaino baseline show <name>' to see details")

	return nil
}

func baselineOutputJSON(baselines []storage.SnapshotInfo) error {
	// Convert to a cleaner structure for JSON output
	type BaselineInfo struct {
		Name          string            `json:"name"`
		ID            string            `json:"id"`
		Created       time.Time         `json:"created"`
		Provider      string            `json:"provider"`
		ResourceCount int               `json:"resource_count"`
		Description   string            `json:"description,omitempty"`
		Tags          map[string]string `json:"tags,omitempty"`
	}

	var baselineList []BaselineInfo
	for _, baseline := range baselines {
		info := BaselineInfo{
			Name:          baseline.Tags["baseline"],
			ID:            baseline.ID,
			Created:       baseline.Timestamp,
			Provider:      baseline.Provider,
			ResourceCount: baseline.ResourceCount,
			Description:   baseline.Tags["description"],
			Tags:          baseline.Tags,
		}
		baselineList = append(baselineList, info)
	}

	jsonData, err := json.MarshalIndent(baselineList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}
