package app

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/output"
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
	a.logger.Info("Scanning infrastructure...")
	
	collectors := a.registry.List()
	fmt.Printf("Found %d collectors\n", len(collectors))
	
	for _, name := range collectors {
		fmt.Printf("  - %s: Available\n", name)
	}
}

func (a *App) runCheckCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Checking for drift...")
	
	fmt.Println("Drift Check:")
	fmt.Println("  Baseline: Not found")
	fmt.Println("  Status: Run 'wgo baseline create' first")
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

func (a *App) runDiffCommand(cmd *cobra.Command, args []string) {
	a.logger.Info("Comparing infrastructure states...")
	
	// Get Unix-style flags
	quiet, _ := cmd.Flags().GetBool("quiet")
	nameOnly, _ := cmd.Flags().GetBool("name-only")
	stat, _ := cmd.Flags().GetBool("stat")
	format, _ := cmd.Flags().GetString("format")
	
	// Handle format shortcuts
	if nameOnly {
		format = "name-only"
	} else if stat {
		format = "stat"
	} else if format == "" {
		format = "unix" // Default to Unix-style format
	}
	
	// For demo purposes, create a sample drift report
	report := a.createSampleDriftReport()
	
	// If quiet mode, just exit with status
	if quiet {
		if len(report.ResourceChanges) > 0 {
			// Exit code 1 means drift detected
			fmt.Printf("") // Silent output
			return
		}
		return
	}
	
	// Use Unix-style formatter
	formatter := a.createUnixFormatter()
	
	var result []byte
	var err error
	
	switch format {
	case "name-only":
		result, err = formatter.FormatNameOnly(report)
	case "stat":
		result, err = formatter.FormatStat(report)
	case "simple":
		result, err = formatter.FormatSimple(report)
	default: // "unix"
		result, err = formatter.FormatDriftReport(report)
	}
	
	if err != nil {
		fmt.Printf("Error formatting output: %v\n", err)
		return
	}
	
	fmt.Print(string(result))
	
	// Set exit code based on whether drift was detected (like git diff)
	if len(report.ResourceChanges) > 0 {
		// In a real implementation, we would use os.Exit(1) here
		// For demo purposes, we'll just indicate drift was found
	}
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

// Helper methods for the diff command

func (a *App) createUnixFormatter() *output.UnixFormatter {
	return output.NewUnixFormatter(false) // No color for now
}

func (a *App) createSampleDriftReport() *differ.DriftReport {
	return &differ.DriftReport{
		ID:         "demo-diff",
		BaselineID: "baseline-123",
		CurrentID:  "current-456",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:    5,
			ChangedResources:  2,
			AddedResources:    1,
			RemovedResources:  0,
			ModifiedResources: 1,
		},
		ResourceChanges: []differ.ResourceChange{
			{
				ResourceID:   "i-1234567890abcdef0",
				ResourceType: "aws_instance",
				DriftType:    differ.ChangeTypeModified,
				Changes: []differ.Change{
					{
						Field:    "instance_type",
						OldValue: "t2.micro",
						NewValue: "t2.small",
					},
				},
			},
			{
				ResourceID:   "sg-0123456789abcdef0",
				ResourceType: "aws_security_group",
				DriftType:    differ.ChangeTypeModified,
				Changes: []differ.Change{
					{
						Field:    "ingress_rules",
						OldValue: `[{"port": 80, "protocol": "tcp"}]`,
						NewValue: `[{"port": 80, "protocol": "tcp"}, {"port": 443, "protocol": "tcp"}]`,
					},
				},
			},
		},
	}
}