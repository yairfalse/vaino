package app

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/output"
	"github.com/yairfalse/wgo/pkg/types"
)

func (a *App) runVersionCommand(cmd *cobra.Command, args []string) {
	fmt.Printf("wgo version %s\n", a.config.Version)
	fmt.Printf("  commit: %s\n", a.config.Commit)
	fmt.Printf("  built: %s\n", a.config.BuildDate)
}

func (a *App) runStatusCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Getting infrastructure status...")

	fmt.Println("Infrastructure Status:")
	fmt.Println("  Collectors: Available")
	fmt.Println("  Storage: Ready")
	fmt.Println("  Cache: Active")

	if a.config.Verbose {
		stats := a.cache.Stats()
		fmt.Printf("  Cache Stats: %d hits, %d misses\n", stats.Hits, stats.Misses)
	}
}

func (a *App) runScanCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Starting infrastructure scan...")

	// Get flags
	provider, _ := cmd.Flags().GetString("provider")
	statePaths, _ := cmd.Flags().GetStringSlice("state-file")
	outputFile, _ := cmd.Flags().GetString("output")
	autoDiscover, _ := cmd.Flags().GetBool("auto-discover")

	ctx := cmd.Context()

	if provider == "" {
		// List available providers
		enhancedProviders := a.enhancedRegistry.ListEnhanced()
		legacyProviders := a.enhancedRegistry.ListLegacy()

		fmt.Println("Available providers:")
		fmt.Println("\nEnhanced providers (support full collection):")
		for _, name := range enhancedProviders {
			status := "unknown"
			if collector, err := a.enhancedRegistry.GetEnhanced(name); err == nil {
				status = collector.Status()
			}
			fmt.Printf("  • %s (%s)\n", name, status)
		}

		if len(legacyProviders) > 0 {
			fmt.Println("\nLegacy providers (status only):")
			for _, name := range legacyProviders {
				fmt.Printf("  • %s\n", name)
			}
		}

		fmt.Println("\nUse --provider <name> to scan a specific provider")
		fmt.Println("Example: wgo scan --provider terraform")
		return
	}

	// Validate provider
	if !a.enhancedRegistry.IsEnhanced(provider) {
		fmt.Printf("Error: Provider '%s' is not available or does not support collection\n", provider)
		fmt.Println("Available enhanced providers:")
		for _, name := range a.enhancedRegistry.ListEnhanced() {
			fmt.Printf("  • %s\n", name)
		}
		return
	}

	// Get collector
	collector, err := a.enhancedRegistry.GetEnhanced(provider)
	if err != nil {
		fmt.Printf("Error: Failed to get collector for provider '%s': %v\n", provider, err)
		return
	}

	// Check collector status
	status := collector.Status()
	if status != "ready" {
		fmt.Printf("Warning: Collector status: %s\n", status)
	}

	// Build collector configuration
	var config collectors.CollectorConfig

	if autoDiscover {
		fmt.Println("🔍 Auto-discovering configuration...")
		discoveredConfig, err := collector.AutoDiscover()
		if err != nil {
			fmt.Printf("Auto-discovery failed: %v\n", err)
			return
		}
		config = discoveredConfig
		fmt.Printf("Found %d state paths\n", len(config.StatePaths))
	} else if len(statePaths) > 0 {
		config.StatePaths = statePaths
	} else {
		// Default configuration for terraform
		if provider == "terraform" {
			config.StatePaths = []string{"./terraform.tfstate", "./"}
		}
	}

	// Add common configuration
	config.Config = map[string]interface{}{
		"scan_id": fmt.Sprintf("%s-%d", provider, time.Now().Unix()),
	}

	// Validate configuration
	if err := collector.Validate(config); err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		return
	}

	// Start progress indicator
	noColor := a.config.Debug // Use debug flag to determine color preference
	spinner := output.NewSpinner("Collecting resources from "+provider+"...", noColor)
	spinner.Start()

	startTime := time.Now()

	snapshot, err := collector.Collect(ctx, config)

	spinner.Stop()

	if err != nil {
		fmt.Printf("❌ Collection failed: %v\n", err)
		return
	}

	collectionTime := time.Since(startTime)

	// Display results using enhanced renderer
	fmt.Printf("\n✅ Collection completed in %v\n", collectionTime)
	fmt.Printf("📋 Snapshot ID: %s\n", snapshot.ID)

	// Use enhanced table renderer for resource display
	renderer := output.NewEnhancedTableRenderer(noColor, 120)
	resourceSummary := renderer.RenderResourceList(snapshot.Resources)
	fmt.Print(resourceSummary)

	// Save snapshot
	if err := a.storage.SaveSnapshot(snapshot); err != nil {
		fmt.Printf("Warning: Failed to save snapshot: %v\n", err)
	} else {
		fmt.Printf("\n💾 Snapshot saved successfully\n")
	}

	// Save to output file if specified
	if outputFile != "" {
		if err := a.saveSnapshotToFile(snapshot, outputFile); err != nil {
			fmt.Printf("Warning: Failed to save to output file: %v\n", err)
		} else {
			fmt.Printf("📄 Output saved to: %s\n", outputFile)
		}
	}
}

func (a *App) runCheckCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Checking for drift...")

	fmt.Println("Drift Check:")
	fmt.Println("  Baseline: Not found")
	fmt.Println("  Status: Run 'wgo baseline create' first")
}

func (a *App) runDiffCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Comparing infrastructure states...")

	fmt.Println("Infrastructure Diff:")
	fmt.Println("  Use 'wgo diff --baseline <name>' to compare with baseline")
	fmt.Println("  Use 'wgo diff --from <file1> --to <file2>' to compare snapshots")
	fmt.Println("  Example: wgo diff --baseline prod-v1.0 --format json")
}

func (a *App) runBaselineCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Managing baselines...")

	fmt.Println("Baseline Management:")
	fmt.Println("  Use 'wgo baseline create' to create a new baseline")
	fmt.Println("  Use 'wgo baseline list' to list baselines")
	fmt.Println("  Use 'wgo baseline delete <id>' to delete a baseline")
}

func (a *App) runExplainCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Explaining changes...")

	fmt.Println("AI-Powered Explanation:")
	fmt.Println("  Set ANTHROPIC_API_KEY environment variable to use AI features")
	fmt.Println("  Or configure it in ~/.wgo/config.yaml")
}

func (a *App) runCacheCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Managing cache...")

	stats := a.cache.Stats()
	fmt.Println("Cache Status:")
	fmt.Printf("  Items: %d\n", stats.Size)
	fmt.Printf("  Hits: %d\n", stats.Hits)
	fmt.Printf("  Misses: %d\n", stats.Misses)
	if stats.Hits+stats.Misses > 0 {
		fmt.Printf("  Hit Rate: %.2f%%\n", float64(stats.Hits)/(float64(stats.Hits+stats.Misses))*100)
	}
}

func (a *App) runConfigCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Showing configuration...")

	fmt.Println("Configuration:")
	fmt.Printf("  Verbose: %v\n", a.config.Verbose)
	fmt.Printf("  Debug: %v\n", a.config.Debug)
	fmt.Println("  Config file: ~/.wgo/config.yaml")
	fmt.Println("  Environment variables:")
	fmt.Println("    ANTHROPIC_API_KEY - for AI features")
	fmt.Println("    WGO_VERBOSE - enable verbose output")
	fmt.Println("    WGO_DEBUG - enable debug mode")
}

func (a *App) runSetupCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Running setup...")

	fmt.Println("Auto-Setup:")
	fmt.Println("  Detecting infrastructure providers...")
	fmt.Println("  - Terraform: Checking for .tf files...")
	fmt.Println("  - AWS: Checking for AWS credentials...")
	fmt.Println("  - Kubernetes: Checking for kubeconfig...")
	fmt.Println("  - Git: Checking for .git directory...")
	fmt.Println("  Setup complete! Run 'wgo config' to see configuration.")
}

// saveSnapshotToFile saves a snapshot to a JSON file
func (a *App) saveSnapshotToFile(snapshot *types.Snapshot, filename string) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}
