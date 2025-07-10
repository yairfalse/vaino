package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/differ"
	vainoerrors "github.com/yairfalse/vaino/internal/errors"
	"github.com/yairfalse/vaino/internal/output"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/types"
	"gopkg.in/yaml.v3"
)

func newDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show infrastructure changes (like 'git diff' for infrastructure)",
		Long: `Show changes in your infrastructure state - just like 'git diff' but for infrastructure.

Works great with Unix tools and scripts. Exit codes: 0 = no changes, 1 = changes detected.

By default, compares current infrastructure state with the last scan automatically.`,
		Example: `  # See what changed in your infrastructure
  vaino diff

  # Just list what changed (like git diff --name-only)
  vaino diff --name-only

  # Show change statistics (like git diff --stat)  
  vaino diff --stat

  # Silent mode for scripts (like git diff --quiet)
  vaino diff --quiet && echo "All good!" || echo "Changes detected!"

  # Compare with specific snapshot
  vaino diff --from prod-v1.0

  # Compare two specific snapshots
  vaino diff --from snapshot-1.json --to snapshot-2.json

  # Use in CI/CD pipelines
  if ! vaino diff --quiet; then
    echo "WARNING: Infrastructure drift detected!"
    vaino diff --stat
  fi`,
		RunE: runDiff,
	}

	// Flags - Unix-style like git diff
	// Removed baseline flag - use --from instead
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

	output.WriteString("Drift Summary\n")
	output.WriteString("=================\n")
	output.WriteString(fmt.Sprintf("Total Resources: %d\n", report.Summary.TotalResources))
	output.WriteString(fmt.Sprintf("Changed Resources: %d\n", report.Summary.ChangedResources))
	output.WriteString(fmt.Sprintf("Added Resources: %d\n", report.Summary.AddedResources))
	output.WriteString(fmt.Sprintf("Removed Resources: %d\n", report.Summary.RemovedResources))
	output.WriteString(fmt.Sprintf("Modified Resources: %d\n", report.Summary.ModifiedResources))
	output.WriteString(fmt.Sprintf("Overall Risk: %s (%.2f)\n", report.Summary.OverallRisk, report.Summary.RiskScore))

	if len(report.Summary.ChangesBySeverity) > 0 {
		output.WriteString("\nChanges by Severity:\n")
		for severity, count := range report.Summary.ChangesBySeverity {
			if count > 0 {
				output.WriteString(fmt.Sprintf("  %s: %d\n", severity, count))
			}
		}
	}

	if len(report.Summary.ChangesByCategory) > 0 {
		output.WriteString("\nChanges by Category:\n")
		for category, count := range report.Summary.ChangesByCategory {
			if count > 0 {
				output.WriteString(fmt.Sprintf("  %s: %d\n", category, count))
			}
		}
	}

	if !summaryOnly && len(report.ResourceChanges) > 0 {
		output.WriteString("\nDetailed Changes\n")
		output.WriteString("====================\n")

		for _, resourceChange := range report.ResourceChanges {
			output.WriteString(fmt.Sprintf("\nResource: %s (%s)\n", resourceChange.ResourceID, resourceChange.ResourceType))
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
		output.WriteString("\nNo drift detected - infrastructure matches reference snapshot\n")
	} else {
		output.WriteString(fmt.Sprintf("\nDrift detected in %d resources\n", report.Summary.ChangedResources))
	}

	return output.String()
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Parse flags
	// baseline flag removed
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
	if from == "" && to == "" {
		// Try to find the most recent scan in ~/.vaino
		homeDir, _ := os.UserHomeDir()
		vainoDir := filepath.Join(homeDir, ".vaino")

		// Find all last-scan files
		matches, _ := filepath.Glob(filepath.Join(vainoDir, "last-scan-*.json"))
		if len(matches) == 0 {
			// No scans found - provide helpful guidance
			return vainoerrors.New(vainoerrors.ErrorTypeFileSystem, vainoerrors.ProviderUnknown,
				"No previous scans found").
				WithCause("No snapshot files in ~/.vaino").
				WithSolutions(
					"Run 'vaino scan' to create your first snapshot",
					"Specify snapshots manually with --from and --to",
				).
				WithHelp("vaino scan --help")
		}

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
				tempFile, err := os.CreateTemp("", "vaino-scan-*.json")
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

				// Add timeout for scan execution
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				done := make(chan error, 1)
				go func() {
					done <- scanCmd.Execute()
				}()

				select {
				case err := <-done:
					if err != nil {
						// Check if it's a known error type
						if vainoErr, ok := err.(*vainoerrors.VAINOError); ok {
							return vainoErr
						}
						return vainoerrors.New(vainoerrors.ErrorTypeProvider, vainoerrors.Provider(providerName),
							"Failed to scan current infrastructure").
							WithCause(err.Error()).
							WithSolutions(
								fmt.Sprintf("Run 'vaino scan --provider %s' manually to debug", providerName),
								"Check provider authentication with 'vaino check-config'",
							).
							WithHelp("vaino check-config")
					}
				case <-ctx.Done():
					return vainoerrors.New(vainoerrors.ErrorTypeNetwork, vainoerrors.Provider(providerName),
						"Scan operation timed out after 30 seconds").
						WithSolutions(
							"Try limiting the scan scope with specific namespaces",
							"Run 'vaino scan --provider kubernetes --namespace test-workloads' for targeted scanning",
							"Check cluster connectivity with 'kubectl get nodes'",
						).
						WithHelp("vaino check-config")
				}

				// Set from and to for comparison
				from = mostRecent
				to = tempPath
			}
		}
	}

	// Validate inputs after auto-detection
	if from == "" || to == "" {
		return vainoerrors.New(vainoerrors.ErrorTypeValidation, vainoerrors.ProviderUnknown,
			"Missing required arguments").
			WithCause("Must specify snapshots to compare").
			WithSolutions(
				"Run 'vaino diff' (auto-detects last scan)",
				"Use 'vaino diff --from snapshot1.json --to snapshot2.json'",
				"Use 'vaino diff --from prod-v1.0'",
			).
			WithHelp("vaino diff --help")
	}

	// Removed baseline validation

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
	_, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		return vainoerrors.New(vainoerrors.ErrorTypeFileSystem, vainoerrors.ProviderUnknown,
			"Storage initialization failed").
			WithCause(err.Error()).
			WithSolutions(
				"Check directory permissions",
				"Ensure disk has available space",
				"Run 'vaino check-config' to diagnose",
			).
			WithHelp("vaino check-config")
	}

	// Load snapshots
	var fromSnapshot, toSnapshot *types.Snapshot

	// Loading snapshots for comparison
	fromSnapshot, err = loadSnapshotFromFile(from)
	if err != nil {
		if os.IsNotExist(err) {
			return vainoerrors.New(vainoerrors.ErrorTypeFileSystem, vainoerrors.ProviderUnknown,
				fmt.Sprintf("Snapshot file not found: %s", from)).
				WithCause("File does not exist").
				WithSolutions(
					"Check file path and spelling",
					"Use absolute paths for clarity",
					"List available snapshots: ls ~/.vaino/history/",
				).
				WithHelp("vaino scan --help")
		}
		return vainoerrors.New(vainoerrors.ErrorTypeFileSystem, vainoerrors.ProviderUnknown,
			"Failed to load snapshot").
			WithCause(err.Error()).
			WithSolutions(
				"Ensure file is valid JSON",
				"Check file permissions",
			).
			WithHelp("vaino diff --help")
	}
	toSnapshot, err = loadSnapshotFromFile(to)
	if err != nil {
		if os.IsNotExist(err) {
			return vainoerrors.New(vainoerrors.ErrorTypeFileSystem, vainoerrors.ProviderUnknown,
				fmt.Sprintf("Snapshot file not found: %s", to)).
				WithCause("File does not exist").
				WithSolutions(
					"Check file path and spelling",
					"Use absolute paths for clarity",
					"Run 'vaino scan' to create new snapshot",
				).
				WithHelp("vaino scan --help")
		}
		return vainoerrors.New(vainoerrors.ErrorTypeFileSystem, vainoerrors.ProviderUnknown,
			"Failed to load snapshot").
			WithCause(err.Error()).
			WithSolutions(
				"Ensure file is valid JSON",
				"Check file permissions",
			).
			WithHelp("vaino diff --help")
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
		return vainoerrors.New(vainoerrors.ErrorTypeValidation, vainoerrors.ProviderUnknown,
			"Comparison failed").
			WithCause(err.Error()).
			WithSolutions(
				"Ensure snapshots are from compatible VAINO versions",
				"Check that snapshots contain valid resource data",
				"Try regenerating snapshots with 'vaino scan'",
			).
			WithHelp("vaino scan --help")
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
