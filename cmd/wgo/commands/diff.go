package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/differ"
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

	// Flags
	cmd.Flags().String("baseline", "", "baseline name to compare against")
	cmd.Flags().String("from", "", "source snapshot file")
	cmd.Flags().String("to", "", "target snapshot file")
	cmd.Flags().String("format", "table", "output format (table, json, yaml, markdown)")
	cmd.Flags().String("output-file", "", "save diff to file")
	cmd.Flags().StringP("provider", "p", "", "limit diff to specific provider")
	cmd.Flags().StringSlice("region", []string{}, "limit diff to specific regions")
	cmd.Flags().StringSlice("namespace", []string{}, "limit diff to specific namespaces")
	cmd.Flags().String("min-severity", "low", "minimum severity to show (low, medium, high, critical)")
	cmd.Flags().StringSlice("resource-type", []string{}, "limit to specific resource types")
	cmd.Flags().StringSlice("ignore-fields", []string{}, "ignore changes in specified fields")
	cmd.Flags().Bool("summary-only", false, "show only summary statistics")
	cmd.Flags().Bool("show-unchanged", false, "include unchanged resources")

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
	fmt.Println("📊 Infrastructure State Comparison")
	fmt.Println("==================================")
	
	// Parse flags
	baseline, _ := cmd.Flags().GetString("baseline")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output-file")
	provider, _ := cmd.Flags().GetString("provider")
	minSeverity, _ := cmd.Flags().GetString("min-severity")
	summaryOnly, _ := cmd.Flags().GetBool("summary-only")
	ignoreFields, _ := cmd.Flags().GetStringSlice("ignore-fields")
	showUnchanged, _ := cmd.Flags().GetBool("show-unchanged")
	
	// Validate inputs
	if baseline == "" && (from == "" || to == "") {
		return fmt.Errorf("must specify either --baseline or both --from and --to")
	}
	
	if baseline != "" && (from != "" || to != "") {
		return fmt.Errorf("cannot use --baseline with --from/--to")
	}
	
	// Validate format
	validFormats := []string{"table", "json", "yaml", "markdown"}
	formatValid := false
	for _, validFormat := range validFormats {
		if format == validFormat {
			formatValid = true
			break
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
		fmt.Printf("📋 Loading baseline: %s\n", baseline)
		baselineData, err := localStorage.LoadBaseline(baseline)
		if err != nil {
			return fmt.Errorf("failed to load baseline '%s': %w", baseline, err)
		}
		fromSnapshot, err = localStorage.LoadSnapshot(baselineData.SnapshotID)
		if err != nil {
			return fmt.Errorf("failed to load baseline snapshot: %w", err)
		}
		
		fmt.Println("📊 Loading current snapshot...")
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
		fmt.Printf("📋 Comparing baseline '%s' with current state\n", baseline)
	} else {
		fmt.Printf("📊 Loading snapshots: %s → %s\n", from, to)
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
		fmt.Printf("🔧 Provider filter: %s\n", provider)
	}
	
	// Create differ engine
	differ := differ.NewDifferEngine(options)
	
	fmt.Println("\n🔍 Comparing snapshots...")
	startTime := time.Now()
	
	// Perform comparison
	report, err := differ.Compare(fromSnapshot, toSnapshot)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}
	
	comparisonTime := time.Since(startTime)
	fmt.Printf("✅ Comparison completed in %v\n\n", comparisonTime)
	
	// Format and display results
	output, err := formatDiffReport(report, format, summaryOnly, showUnchanged)
	if err != nil {
		return fmt.Errorf("failed to format report: %w", err)
	}
	
	fmt.Print(output)
	
	// Save output if requested
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			fmt.Printf("⚠️  Failed to save output: %v\n", err)
		} else {
			fmt.Printf("\n💾 Output saved to: %s\n", outputFile)
		}
	}
	
	return nil
}