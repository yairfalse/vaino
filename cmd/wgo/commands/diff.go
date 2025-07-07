package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/output"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
	"gopkg.in/yaml.v3"
)

func newDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare infrastructure states",
		Long: `Compare two infrastructure states (snapshots or baselines) to see
detailed differences. Supports multiple output formats for different use cases.`,
		Example: `  # Compare current state with baseline
  wgo diff --baseline prod-v1.0

  # Compare two snapshots
  wgo diff --from snapshot-1.json --to snapshot-2.json

  # Compare with specific format
  wgo diff --baseline prod-v1.0 --format markdown --output-file changes.md

  # Compare specific provider only
  wgo diff --baseline prod-v1.0 --provider aws --region us-east-1

  # Show only high-impact changes
  wgo diff --baseline prod-v1.0 --min-severity medium`,
		RunE: runDiff,
	}

	// Flags - Unix-style like git diff
	cmd.Flags().String("baseline", "", "baseline name to compare against")
	cmd.Flags().String("from", "", "source snapshot file")
	cmd.Flags().String("to", "", "target snapshot file")
	cmd.Flags().String("format", "", "output format (unix, simple, name-only, stat, json, yaml)")
	cmd.Flags().StringP("output", "o", "", "output file (use '-' for stdout)")
	
	// Unix-style options
	cmd.Flags().Bool("name-only", false, "show only names of changed resources")
	cmd.Flags().Bool("stat", false, "show diffstat")
	cmd.Flags().BoolP("quiet", "q", false, "suppress all output, exit with status only")
	
	// Filtering options
	cmd.Flags().StringP("provider", "p", "", "limit diff to specific provider")
	cmd.Flags().StringSlice("region", []string{}, "limit diff to specific regions")
	cmd.Flags().StringSlice("namespace", []string{}, "limit diff to specific namespaces")
	cmd.Flags().StringSlice("resource-type", []string{}, "limit to specific resource types")
	cmd.Flags().StringSlice("ignore-fields", []string{}, "ignore changes in specified fields")
	cmd.Flags().String("min-severity", "low", "minimum severity level to show (low, medium, high, critical)")

	return cmd
}

// Helper function to load snapshot from file
func loadSnapshotFromFile(filename string) (*types.Snapshot, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var snapshot types.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot: %w", err)
	}
	
	return &snapshot, nil
}

// Helper function to parse risk level from string
func parseRiskLevel(severity string) differ.RiskLevel {
	switch strings.ToLower(severity) {
	case "critical":
		return differ.RiskLevelCritical
	case "high":
		return differ.RiskLevelHigh
	case "medium":
		return differ.RiskLevelMedium
	case "low":
		return differ.RiskLevelLow
	default:
		return differ.RiskLevelLow
	}
}

// Helper function to format drift report in different formats
func formatDiffReport(report *differ.DriftReport, format string, summaryOnly bool, showUnchanged bool) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
		
	case "yaml":
		data, err := yaml.Marshal(report)
		if err != nil {
			return "", err
		}
		return string(data), nil
		
	case "markdown":
		return formatMarkdownReport(report, summaryOnly, showUnchanged), nil
		
	case "table":
		return formatTableReport(report, summaryOnly, showUnchanged), nil
	default:
		return formatTableReport(report, summaryOnly, showUnchanged), nil
	}
}

// Helper function to format report as markdown
func formatMarkdownReport(report *differ.DriftReport, summaryOnly bool, showUnchanged bool) string {
	var output strings.Builder
	
	output.WriteString("# Infrastructure Drift Report\n\n")
	output.WriteString("## Summary\n\n")
	output.WriteString(fmt.Sprintf("- **Total Resources**: %d\n", report.Summary.TotalResources))
	output.WriteString(fmt.Sprintf("- **Changed Resources**: %d\n", report.Summary.ChangedResources))
	output.WriteString(fmt.Sprintf("- **Added Resources**: %d\n", report.Summary.AddedResources))
	output.WriteString(fmt.Sprintf("- **Removed Resources**: %d\n", report.Summary.RemovedResources))
	output.WriteString(fmt.Sprintf("- **Modified Resources**: %d\n", report.Summary.ModifiedResources))
	output.WriteString(fmt.Sprintf("- **Overall Risk**: %s (%.2f)\n\n", report.Summary.OverallRisk, report.Summary.RiskScore))
	
	if len(report.Summary.ChangesBySeverity) > 0 {
		output.WriteString("### Changes by Severity\n\n")
		for severity, count := range report.Summary.ChangesBySeverity {
			if count > 0 {
				output.WriteString(fmt.Sprintf("- **%s**: %d\n", severity, count))
			}
		}
		output.WriteString("\n")
	}
	
	if !summaryOnly && len(report.ResourceChanges) > 0 {
		output.WriteString("## Detailed Changes\n\n")
		
		for _, resourceChange := range report.ResourceChanges {
			output.WriteString(fmt.Sprintf("### %s (%s)\n\n", resourceChange.ResourceID, resourceChange.ResourceType))
			output.WriteString(fmt.Sprintf("- **Provider**: %s\n", resourceChange.Provider))
			output.WriteString(fmt.Sprintf("- **Change Type**: %s\n", resourceChange.DriftType))
			output.WriteString(fmt.Sprintf("- **Severity**: %s\n", resourceChange.Severity))
			output.WriteString(fmt.Sprintf("- **Risk Score**: %.2f\n", resourceChange.RiskScore))
			output.WriteString(fmt.Sprintf("- **Description**: %s\n\n", resourceChange.Description))
			
			if len(resourceChange.Changes) > 0 {
				output.WriteString("#### Changes\n\n")
				for _, change := range resourceChange.Changes {
					output.WriteString(fmt.Sprintf("- **%s**: `%v` → `%v` (%s)\n", 
						change.Field, change.OldValue, change.NewValue, change.Severity))
				}
				output.WriteString("\n")
			}
		}
	}
	
	return output.String()
}

// Helper function to format report as table
func formatTableReport(report *differ.DriftReport, summaryOnly bool, showUnchanged bool) string {
	var output strings.Builder
	
	output.WriteString("📊 Drift Summary\n")
	output.WriteString("=================\n")
	output.WriteString(fmt.Sprintf("Total Resources: %d\n", report.Summary.TotalResources))
	output.WriteString(fmt.Sprintf("Changed Resources: %d\n", report.Summary.ChangedResources))
	output.WriteString(fmt.Sprintf("Added Resources: %d\n", report.Summary.AddedResources))
	output.WriteString(fmt.Sprintf("Removed Resources: %d\n", report.Summary.RemovedResources))
	output.WriteString(fmt.Sprintf("Modified Resources: %d\n", report.Summary.ModifiedResources))
	output.WriteString(fmt.Sprintf("Overall Risk: %s (%.2f)\n", report.Summary.OverallRisk, report.Summary.RiskScore))
	
	if len(report.Summary.ChangesBySeverity) > 0 {
		output.WriteString("\n📈 Changes by Severity:\n")
		for severity, count := range report.Summary.ChangesBySeverity {
			if count > 0 {
				output.WriteString(fmt.Sprintf("  %s: %d\n", severity, count))
			}
		}
	}
	
	if len(report.Summary.ChangesByCategory) > 0 {
		output.WriteString("\n📋 Changes by Category:\n")
		for category, count := range report.Summary.ChangesByCategory {
			if count > 0 {
				output.WriteString(fmt.Sprintf("  %s: %d\n", category, count))
			}
		}
	}
	
	if !summaryOnly && len(report.ResourceChanges) > 0 {
		output.WriteString("\n🔍 Detailed Changes\n")
		output.WriteString("====================\n")
		
		for _, resourceChange := range report.ResourceChanges {
			output.WriteString(fmt.Sprintf("\n📦 Resource: %s (%s)\n", resourceChange.ResourceID, resourceChange.ResourceType))
			output.WriteString(fmt.Sprintf("   Provider: %s\n", resourceChange.Provider))
			output.WriteString(fmt.Sprintf("   Change Type: %s\n", resourceChange.DriftType))
			output.WriteString(fmt.Sprintf("   Severity: %s\n", resourceChange.Severity))
			output.WriteString(fmt.Sprintf("   Risk Score: %.2f\n", resourceChange.RiskScore))
			output.WriteString(fmt.Sprintf("   Description: %s\n", resourceChange.Description))
			
			if len(resourceChange.Changes) > 0 {
				output.WriteString("   Changes:\n")
				for _, change := range resourceChange.Changes {
					output.WriteString(fmt.Sprintf("     • %s: %v → %v (%s)\n", 
						change.Field, change.OldValue, change.NewValue, change.Severity))
				}
			}
		}
	}
	
	if report.Summary.ChangedResources == 0 {
		output.WriteString("\n✅ No drift detected - infrastructure matches baseline\n")
	} else {
		output.WriteString(fmt.Sprintf("\n⚠️  Drift detected in %d resources\n", report.Summary.ChangedResources))
	}
	
	return output.String()
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Parse flags
	baseline, _ := cmd.Flags().GetString("baseline")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")
	provider, _ := cmd.Flags().GetString("provider")
	quiet, _ := cmd.Flags().GetBool("quiet")
	nameOnly, _ := cmd.Flags().GetBool("name-only")
	stat, _ := cmd.Flags().GetBool("stat")
	ignoreFields, _ := cmd.Flags().GetStringSlice("ignore-fields")
	minSeverity, _ := cmd.Flags().GetString("min-severity")
	
	// Check for no-color flag from global flags
	noColor := cmd.Flag("no-color") != nil && cmd.Flag("no-color").Value.String() == "true"
	
	// Handle format shortcuts
	if nameOnly {
		format = "name-only"
	} else if stat {
		format = "stat"
	}
	
	// Auto-detect last scan if no inputs provided
	if baseline == "" && from == "" && to == "" {
		// Try to find the most recent scan in ~/.wgo
		homeDir, _ := os.UserHomeDir()
		wgoDir := filepath.Join(homeDir, ".wgo")
		
		// Find all last-scan files
		matches, _ := filepath.Glob(filepath.Join(wgoDir, "last-scan-*.json"))
		if len(matches) > 0 {
			// Use the most recently modified one
			var mostRecent string
			var mostRecentTime time.Time
			
			for _, match := range matches {
				info, err := os.Stat(match)
				if err == nil && info.ModTime().After(mostRecentTime) {
					mostRecent = match
					mostRecentTime = info.ModTime()
				}
			}
			
			if mostRecent != "" {
				// Extract provider from filename
				base := filepath.Base(mostRecent)
				providerName := strings.TrimPrefix(strings.TrimSuffix(base, ".json"), "last-scan-")
				
				fmt.Printf("Comparing %s infrastructure...\n", providerName)
				
				// Create temp file for new scan
				tempFile, err := os.CreateTemp("", "wgo-scan-*.json")
				if err != nil {
					return fmt.Errorf("failed to create temp file: %w", err)
				}
				tempPath := tempFile.Name()
				tempFile.Close()
				defer os.Remove(tempPath)
				
				// Run new scan silently
				scanCmd := newScanCommand()
				scanArgs := []string{"--provider", providerName, "--output-file", tempPath, "--quiet"}
				
				// Add auto-discover for terraform
				if providerName == "terraform" {
					scanArgs = append(scanArgs, "--auto-discover")
				}
				
				scanCmd.SetArgs(scanArgs)
				scanCmd.SetOutput(io.Discard) // Suppress output
				if err := scanCmd.Execute(); err != nil {
					return fmt.Errorf("failed to run scan: %w", err)
				}
				
				// Set from and to for comparison
				from = mostRecent
				to = tempPath
			}
		}
	}
	
	// Validate inputs after auto-detection
	if baseline == "" && (from == "" || to == "") {
		return fmt.Errorf("must specify either --baseline or both --from and --to")
	}
	
	if baseline != "" && (from != "" || to != "") {
		return fmt.Errorf("cannot use --baseline with --from/--to")
	}
	
	// Validate format
	validFormats := []string{"table", "json", "yaml", "markdown", "unix", "simple", "name-only", "stat"}
	formatValid := false
	if format == "" {
		format = "unix" // Default format
		formatValid = true
	} else {
		for _, validFormat := range validFormats {
			if format == validFormat {
				formatValid = true
				break
			}
		}
	}
	if !formatValid {
		return fmt.Errorf("invalid format '%s'. Valid formats: %s", format, strings.Join(validFormats, ", "))
	}
	
	// Initialize storage
	storageConfig := storage.Config{BaseDir: "./snapshots"}
	localStorage, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}
	
	// Load snapshots
	var fromSnapshot, toSnapshot *types.Snapshot
	
	if baseline != "" {
		// Loading baseline
		baselineData, err := localStorage.LoadBaseline(baseline)
		if err != nil {
			return fmt.Errorf("failed to load baseline '%s': %w", baseline, err)
		}
		fromSnapshot, err = localStorage.LoadSnapshot(baselineData.SnapshotID)
		if err != nil {
			return fmt.Errorf("failed to load baseline snapshot: %w", err)
		}
		
		// Loading current snapshot
		snapshots, err := localStorage.ListSnapshots()
		if err != nil {
			return fmt.Errorf("failed to list snapshots: %w", err)
		}
		if len(snapshots) == 0 {
			return fmt.Errorf("no current snapshots found. Run 'wgo scan' first")
		}
		// Load the most recent snapshot
		toSnapshot, err = localStorage.LoadSnapshot(snapshots[0].ID)
		if err != nil {
			return fmt.Errorf("failed to load current snapshot: %w", err)
		}
		// Comparing baseline with current state
	} else {
		// Loading snapshots for comparison
		fromSnapshot, err = loadSnapshotFromFile(from)
		if err != nil {
			return fmt.Errorf("failed to load from snapshot '%s': %w", from, err)
		}
		toSnapshot, err = loadSnapshotFromFile(to)
		if err != nil {
			return fmt.Errorf("failed to load to snapshot '%s': %w", to, err)
		}
	}
	
	// Parse minimum severity
	minRiskLevel := parseRiskLevel(minSeverity)
	
	// Configure differ options
	options := differ.DiffOptions{
		IgnoreFields: ignoreFields,
		MinRiskLevel: minRiskLevel,
	}
	
	if provider != "" {
		// Filter to only include the specified provider
		options.IgnoreProviders = []string{}
		for _, resource := range fromSnapshot.Resources {
			if resource.Provider != provider {
				options.IgnoreProviders = append(options.IgnoreProviders, resource.Provider)
			}
		}
		// Provider filter applied
	}
	
	// Create differ engine
	differ := differ.NewDifferEngine(options)
	
	// Perform comparison
	report, err := differ.Compare(fromSnapshot, toSnapshot)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}
	
	// Comparison completed
	
	// If quiet mode, just exit with status
	if quiet {
		if len(report.ResourceChanges) > 0 {
			os.Exit(1)
		}
		return nil
	}
	
	// Use Unix-style output by default
	formatter := output.NewUnixFormatter(noColor)
	
	var result []byte
	
	// Handle different formats
	switch format {
	case "", "unix":
		// Default Unix-style output
		result, err = formatter.FormatDriftReport(report)
	case "simple":
		result, err = formatter.FormatSimple(report)
	case "name-only":
		result, err = formatter.FormatNameOnly(report)
	case "stat":
		result, err = formatter.FormatStat(report)
	case "json":
		result, err = json.MarshalIndent(report, "", "  ")
	case "yaml":
		result, err = yaml.Marshal(report)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}
	
	// Output the result
	if outputFile == "" || outputFile == "-" {
		fmt.Print(string(result))
	} else {
		if err := os.WriteFile(outputFile, result, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}
	
	// Set exit code based on whether drift was detected
	if len(report.ResourceChanges) > 0 {
		os.Exit(1) // Drift detected - like git diff
	}
	
	return nil
}