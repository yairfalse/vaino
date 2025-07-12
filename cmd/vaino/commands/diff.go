package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
		SilenceUsage: true,
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
  
  # Compare against latest baseline
  vaino diff --baseline
  
  # Compare against specific baseline
  vaino diff --baseline production

  # Use in CI/CD pipelines
  if ! vaino diff --quiet; then
    echo "WARNING: Infrastructure drift detected!"
    vaino diff --stat
  fi`,
		RunE: runDiff,
	}

	// Flags - Unix-style like git diff
	cmd.Flags().String("from", "", "source snapshot file")
	cmd.Flags().String("to", "", "target snapshot file")

	// Baseline support (transparent alternative to --from)
	cmd.Flags().String("baseline", "", "compare against baseline (name or 'auto' for latest)")
	cmd.Flags().Bool("list-baselines", false, "list available baselines and exit")
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
	cmd.Flags().BoolP("verbose", "v", false, "show detailed progress information")

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
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	baseline, _ := cmd.Flags().GetString("baseline")
	listBaselines, _ := cmd.Flags().GetBool("list-baselines")
	format, _ := cmd.Flags().GetString("format")
	outputFile, _ := cmd.Flags().GetString("output")
	provider, _ := cmd.Flags().GetString("provider")
	quiet, _ := cmd.Flags().GetBool("quiet")
	nameOnly, _ := cmd.Flags().GetBool("name-only")
	stat, _ := cmd.Flags().GetBool("stat")
	ignoreFields, _ := cmd.Flags().GetStringSlice("ignore-fields")
	minSeverity, _ := cmd.Flags().GetString("min-severity")

	// Handle --list-baselines flag
	if listBaselines {
		return listAvailableBaselines(provider)
	}

	// Check for no-color flag from global flags
	noColor := cmd.Flag("no-color") != nil && cmd.Flag("no-color").Value.String() == "true"

	// Handle format shortcuts
	if nameOnly {
		format = "name-only"
	} else if stat {
		format = "stat"
	}

	// Handle baseline flag by resolving to appropriate --from value
	if baseline != "" {
		if from != "" {
			return fmt.Errorf("cannot use both --baseline and --from flags")
		}
		baselineSnapshot, err := findBaseline(baseline, provider)
		if err != nil {
			return fmt.Errorf("baseline resolution failed: %w", err)
		}
		from = baselineSnapshot
		// Remove verbose baseline messages - only show in verbose mode
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			if baseline == "auto" || baseline == "" {
				fmt.Fprintf(os.Stderr, "Using latest baseline as comparison source\n")
			} else {
				fmt.Fprintf(os.Stderr, "Using baseline '%s' as comparison source\n", baseline)
			}
		}
	}

	// Auto-detect snapshots if no inputs provided
	if from == "" && to == "" {
		// Try to find existing snapshots first
		homeDir, _ := os.UserHomeDir()
		vainoDir := filepath.Join(homeDir, ".vaino")
		historyDir := filepath.Join(vainoDir, "history")

		// Look for snapshots in history directory first
		historyFiles, _ := filepath.Glob(filepath.Join(historyDir, "*.json"))
		if len(historyFiles) >= 2 {
			// Sort by modification time
			sort.Slice(historyFiles, func(i, j int) bool {
				stat1, _ := os.Stat(historyFiles[i])
				stat2, _ := os.Stat(historyFiles[j])
				return stat1.ModTime().After(stat2.ModTime())
			})
			// Use two most recent snapshots
			from = historyFiles[1] // Second most recent as baseline
			to = historyFiles[0]   // Most recent as current
			if verbose, _ := cmd.Flags().GetBool("verbose"); verbose && !quiet {
				fmt.Fprintf(os.Stderr, "Comparing %s with %s\n", filepath.Base(from), filepath.Base(to))
			}
		} else if len(historyFiles) == 1 {
			// Only one snapshot exists - nothing to compare against
			return vainoerrors.New(vainoerrors.ErrorTypeValidation, vainoerrors.ProviderUnknown,
				"Only one snapshot found - nothing to compare").
				WithCause("Diff requires at least two snapshots for comparison").
				WithSolutions(
					"Run 'vaino scan' again to create a new snapshot for comparison",
					"Wait for infrastructure changes, then scan again",
					"Use 'vaino status' to check current state without comparison",
				).
				WithHelp("vaino scan --help")
		} else {
			// Fallback to last-scan files with auto-scanning
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
			} else {
				// Auto-scan mode: compare last scan with new scan
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

					// Only show progress in verbose mode
					if verbose, _ := cmd.Flags().GetBool("verbose"); verbose && !quiet {
						fmt.Fprintf(os.Stderr, "Comparing %s infrastructure...\n", providerName)
					}

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

					// Show scanning progress if not quiet
					if !quiet {
						fmt.Fprintf(os.Stderr, "Scanning %s infrastructure for comparison...\n", providerName)
					}

					// Add timeout for scan execution with proper context
					ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
					defer cancel()

					// Set context on scan command
					scanCmd.SetContext(ctx)

					// Execute scan synchronously with timeout
					scanErr := scanCmd.Execute()

					// Check if context was cancelled
					if ctx.Err() == context.DeadlineExceeded {
						return vainoerrors.New(vainoerrors.ErrorTypeNetwork, vainoerrors.Provider(providerName),
							"Scan operation timed out after 30 seconds").
							WithSolutions(
								fmt.Sprintf("Run 'vaino scan --provider %s' manually to check for issues", providerName),
								"Try limiting the scan scope if available",
								fmt.Sprintf("Check %s connectivity and authentication", providerName),
							).
							WithHelp("vaino check-config")
					}

					if scanErr != nil {
						// Handle specific auto-discovery failures gracefully
						if strings.Contains(scanErr.Error(), "auto-discovery failed") || 
						   strings.Contains(scanErr.Error(), "No terraform state files found") {
							if !quiet {
								fmt.Println("No current infrastructure found - nothing to compare")
								fmt.Println("Run 'vaino scan' to create a current snapshot")
							}
							return nil
						}
						
						// Check if it's a known error type
						if vainoErr, ok := scanErr.(*vainoerrors.VAINOError); ok {
							return vainoErr
						}
						return vainoerrors.New(vainoerrors.ErrorTypeProvider, vainoerrors.Provider(providerName),
							"Auto-scan failed, cannot compare with current state").
							WithCause(scanErr.Error()).
							WithSolutions(
								fmt.Sprintf("Run 'vaino scan --provider %s' manually first", providerName),
								"Use existing snapshots: 'vaino diff --from snap1.json --to snap2.json'",
								"Check provider configuration: 'vaino check-config'",
							).
							WithHelp("vaino scan --help")
					}

					// Check if the new scan found any resources
					tempData, err := os.ReadFile(tempPath)
					if err != nil {
						return fmt.Errorf("failed to read scan results: %w", err)
					}
					
					var tempSnapshot types.Snapshot
					if err := json.Unmarshal(tempData, &tempSnapshot); err != nil {
						return fmt.Errorf("failed to parse scan results: %w", err)
					}
					
					// If no resources found in current scan, show appropriate message
					if len(tempSnapshot.Resources) == 0 {
						if !quiet {
							fmt.Println("No current infrastructure found - nothing to compare")
							fmt.Println("Run 'vaino scan' to create a current snapshot")
						}
						return nil
					}

					// Set from and to for comparison
					from = mostRecent
					to = tempPath
				}
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

// findBaseline locates a baseline snapshot by name or automatically finds the latest
func findBaseline(baselineName, provider string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	historyDir := filepath.Join(homeDir, ".vaino", "history")

	// If no specific name, find latest baseline
	if baselineName == "" || baselineName == "auto" {
		return findLatestBaseline(historyDir, provider)
	}

	// Find baseline by name
	return findNamedBaseline(historyDir, baselineName, provider)
}

// findLatestBaseline finds the most recent baseline snapshot
func findLatestBaseline(historyDir, provider string) (string, error) {
	files, err := filepath.Glob(filepath.Join(historyDir, "*.json"))
	if err != nil {
		return "", fmt.Errorf("could not list history files: %w", err)
	}

	var latestBaseline string
	var latestTime time.Time

	for _, file := range files {
		// Skip if provider filter is specified and doesn't match
		if provider != "" && !strings.Contains(file, provider) {
			continue
		}

		snapshot, err := loadSnapshotFromFile(file)
		if err != nil {
			continue // Skip invalid files
		}

		if snapshot.IsBaselineSnapshot() && snapshot.Timestamp.After(latestTime) {
			latestBaseline = file
			latestTime = snapshot.Timestamp
		}
	}

	if latestBaseline == "" {
		return "", fmt.Errorf("no baseline snapshots found")
	}

	return latestBaseline, nil
}

// findNamedBaseline finds a baseline snapshot by name
func findNamedBaseline(historyDir, baselineName, provider string) (string, error) {
	files, err := filepath.Glob(filepath.Join(historyDir, "*.json"))
	if err != nil {
		return "", fmt.Errorf("could not list history files: %w", err)
	}

	for _, file := range files {
		// Skip if provider filter is specified and doesn't match
		if provider != "" && !strings.Contains(file, provider) {
			continue
		}

		snapshot, err := loadSnapshotFromFile(file)
		if err != nil {
			continue // Skip invalid files
		}

		if snapshot.IsBaselineSnapshot() {
			name, _, _ := snapshot.GetBaselineInfo()
			if name == baselineName {
				return file, nil
			}
		}
	}

	return "", fmt.Errorf("baseline '%s' not found", baselineName)
}

// listAvailableBaselines lists all available baseline snapshots
func listAvailableBaselines(provider string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get home directory: %w", err)
	}

	historyDir := filepath.Join(homeDir, ".vaino", "history")
	files, err := filepath.Glob(filepath.Join(historyDir, "*.json"))
	if err != nil {
		return fmt.Errorf("could not list history files: %w", err)
	}

	type baselineInfo struct {
		name      string
		reason    string
		provider  string
		timestamp time.Time
		file      string
	}

	var baselines []baselineInfo

	for _, file := range files {
		// Skip if provider filter is specified and doesn't match
		if provider != "" && !strings.Contains(file, provider) {
			continue
		}

		snapshot, err := loadSnapshotFromFile(file)
		if err != nil {
			continue // Skip invalid files
		}

		if snapshot.IsBaselineSnapshot() {
			name, reason, _ := snapshot.GetBaselineInfo()
			baselines = append(baselines, baselineInfo{
				name:      name,
				reason:    reason,
				provider:  snapshot.Provider,
				timestamp: snapshot.Timestamp,
				file:      filepath.Base(file),
			})
		}
	}

	if len(baselines) == 0 {
		if provider != "" {
			fmt.Printf("No baselines found for provider '%s'\n", provider)
		} else {
			fmt.Println("No baselines found")
		}
		fmt.Println("\nCreate a baseline with:")
		fmt.Println("  vaino scan --baseline --baseline-name my-baseline")
		return nil
	}

	fmt.Println("Available Baselines:")
	fmt.Println("===================")
	fmt.Println()

	for _, baseline := range baselines {
		name := baseline.name
		if name == "" {
			name = "(unnamed)"
		}

		fmt.Printf("• %s\n", name)
		fmt.Printf("  Provider: %s\n", baseline.provider)
		fmt.Printf("  Created:  %s\n", baseline.timestamp.Format("2006-01-02 15:04:05"))
		if baseline.reason != "" {
			fmt.Printf("  Reason:   %s\n", baseline.reason)
		}
		fmt.Printf("  File:     %s\n", baseline.file)
		fmt.Println()
	}

	fmt.Println("Usage:")
	if provider != "" {
		fmt.Printf("  vaino diff --baseline <name>      # Compare against named baseline\n")
		fmt.Printf("  vaino diff --baseline auto        # Compare against latest baseline\n")
	} else {
		fmt.Printf("  vaino diff --baseline <name>      # Compare against named baseline\n")
		fmt.Printf("  vaino diff --baseline auto        # Compare against latest baseline\n")
		fmt.Printf("  vaino diff --baseline auto -p %s  # Limit to specific provider\n", baselines[0].provider)
	}

	return nil
}
