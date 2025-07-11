package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// enhanceDiffCommand adds enterprise features to the diff command
func enhanceDiffCommand(cmd *cobra.Command) {
	// Enterprise flags
	cmd.Flags().Bool("enterprise", false, "use enterprise diff engine")
	cmd.Flags().String("compliance-report", "", "generate compliance report")
	cmd.Flags().Bool("executive-summary", false, "generate executive summary")
	cmd.Flags().Bool("correlation", false, "enable change correlation analysis")
	cmd.Flags().Bool("risk-assessment", false, "enable advanced risk assessment")
	cmd.Flags().Bool("streaming", false, "enable real-time change streaming")
	cmd.Flags().Int("max-workers", 0, "number of parallel workers (0 = auto)")
	cmd.Flags().Bool("progress", false, "show progress during analysis")
	cmd.Flags().String("policy-framework", "", "compliance framework (SOC2, PCI-DSS, NIST)")

	// Performance flags
	cmd.Flags().Bool("enable-caching", true, "enable result caching")
	cmd.Flags().Bool("enable-indexing", true, "enable resource indexing")
	cmd.Flags().Float64("parallel-threshold", 100, "minimum resources to enable parallel processing")
}

// runEnterpriseDiff handles enterprise diff operations
func runEnterpriseDiff(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get enterprise flags
	_, _ = cmd.Flags().GetBool("enterprise") // useEnterprise
	complianceReport, _ := cmd.Flags().GetString("compliance-report")
	executiveSummary, _ := cmd.Flags().GetBool("executive-summary")
	enableCorrelation, _ := cmd.Flags().GetBool("correlation")
	enableRiskAssessment, _ := cmd.Flags().GetBool("risk-assessment")
	enableStreaming, _ := cmd.Flags().GetBool("streaming")
	maxWorkers, _ := cmd.Flags().GetInt("max-workers")
	showProgress, _ := cmd.Flags().GetBool("progress")
	policyFramework, _ := cmd.Flags().GetString("policy-framework")
	enableCaching, _ := cmd.Flags().GetBool("enable-caching")
	enableIndexing, _ := cmd.Flags().GetBool("enable-indexing")
	parallelThreshold, _ := cmd.Flags().GetFloat64("parallel-threshold")

	// Standard flags
	quiet, _ := cmd.Flags().GetBool("quiet")
	outputFile, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")

	// Progress callback
	var progressCallback func(float64, string)
	if showProgress && !quiet {
		progressCallback = func(progress float64, message string) {
			fmt.Printf("\r[%.0f%%] %s", progress*100, message)
			if progress >= 1.0 {
				fmt.Println()
			}
		}
	}

	// Get snapshots for comparison
	baseline, current, err := getSnapshotsForComparison(cmd)
	if err != nil {
		return fmt.Errorf("failed to get snapshots: %w", err)
	}

	// Configure enterprise engine
	enterpriseOptions := differ.EnterpriseDiffOptions{
		DiffOptions: differ.DiffOptions{
			IgnoreMetadata: true,
		},
		MaxWorkers:        maxWorkers,
		ParallelThreshold: int(parallelThreshold),
		StreamingEnabled:  enableStreaming,
		EnableCaching:     enableCaching,
		EnableIndexing:    enableIndexing,
		EnableCorrelation: enableCorrelation,
		EnableRiskScoring: enableRiskAssessment,
		EnableCompliance:  complianceReport != "" || policyFramework != "",
		ComplianceRules:   getComplianceRules(policyFramework),
		ProgressCallback:  progressCallback,
	}

	// Set output format for executive summary
	if executiveSummary {
		enterpriseOptions.OutputFormat = "executive"
	}

	// Create enterprise engine
	enterpriseEngine := differ.NewEnterpriseDifferEngine(enterpriseOptions)

	// Perform diff analysis
	report, err := enterpriseEngine.CompareWithContext(ctx, baseline, current)

	if err != nil {
		return fmt.Errorf("enterprise diff analysis failed: %w", err)
	}

	// Handle streaming output
	if enableStreaming {
		go handleChangeStream(enterpriseEngine.GetChangeStream())
	}

	// Generate compliance report if requested
	if complianceReport != "" {
		err = generateComplianceReport(report, complianceReport, policyFramework)
		if err != nil {
			return fmt.Errorf("failed to generate compliance report: %w", err)
		}
	}

	// Exit early for quiet mode
	if quiet {
		return getExitCode(report)
	}

	// Output results
	return outputEnterpriseResults(report, format, outputFile, executiveSummary)
}

// getSnapshotsForComparison gets the baseline and current snapshots
func getSnapshotsForComparison(cmd *cobra.Command) (*types.Snapshot, *types.Snapshot, error) {
	fromFile, _ := cmd.Flags().GetString("from")
	toFile, _ := cmd.Flags().GetString("to")

	var baseline, current *types.Snapshot
	var err error

	// Regular snapshot comparison
	if fromFile != "" {
		baseline, err = loadSnapshotFromFile(fromFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load baseline snapshot: %w", err)
		}
	}

	if toFile != "" {
		current, err = loadSnapshotFromFile(toFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load current snapshot: %w", err)
		}
	}

	// If no files specified, use latest snapshots
	if baseline == nil || current == nil {
		baseline, current, err = getLatestSnapshots()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get latest snapshots: %w", err)
		}
	}

	return baseline, current, nil
}

// handleChangeStream processes real-time change streaming
func handleChangeStream(stream <-chan differ.StreamedChange) {
	fmt.Println("\n=== Real-time Change Stream ===")
	for change := range stream {
		fmt.Printf("[%s] %s: %s\n",
			change.Timestamp.Format("15:04:05"),
			change.ResourceDiff.ResourceID,
			change.Change.Description)
	}
	fmt.Println("=== Stream Complete ===")
}

// generateComplianceReport generates a compliance report
func generateComplianceReport(report *differ.DriftReport, filename, framework string) error {
	if report.ComplianceReport == nil {
		return fmt.Errorf("no compliance data available in diff report")
	}

	// Generate detailed compliance report
	complianceData := map[string]interface{}{
		"report_id":      report.ComplianceReport.ID,
		"timestamp":      report.ComplianceReport.Timestamp,
		"framework":      framework,
		"overall_status": report.ComplianceReport.Status,
		"score":          report.ComplianceReport.Score,
		"summary":        report.ComplianceReport.Summary,
		"violations":     report.ComplianceReport.Violations,
		"metadata": map[string]interface{}{
			"total_resources_evaluated": len(report.ResourceChanges),
			"analysis_duration_ms":      report.Metadata["analysis_duration_ms"],
			"engine_version":            report.Metadata["engine_version"],
		},
	}

	// Save to file
	return saveReportToFile(complianceData, filename, "json")
}

// outputEnterpriseResults outputs the enterprise diff results
func outputEnterpriseResults(report *differ.DriftReport, format, outputFile string, executiveSummary bool) error {
	// Determine output format
	if format == "" {
		if executiveSummary {
			format = "executive"
		} else {
			format = "enterprise"
		}
	}

	// Create output writer
	var writer *os.File = os.Stdout
	if outputFile != "" && outputFile != "-" {
		var err error
		writer, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
	}

	// Generate output based on format
	switch format {
	case "executive":
		return outputExecutiveSummary(report, writer)
	case "compliance":
		return outputComplianceReport(report, writer)
	case "enterprise":
		return outputEnterpriseReport(report, writer)
	case "json":
		return outputJSON(report)
	case "yaml":
		return outputYAML(report)
	default:
		return outputJSON(report) // Default to JSON for now
	}
}

// outputExecutiveSummary outputs an executive summary
func outputExecutiveSummary(report *differ.DriftReport, writer *os.File) error {
	if report.ExecutiveSummary == nil {
		return fmt.Errorf("no executive summary available")
	}

	summary := report.ExecutiveSummary

	fmt.Fprintf(writer, "EXECUTIVE INFRASTRUCTURE DRIFT SUMMARY\n")
	fmt.Fprintf(writer, "=====================================\n\n")
	fmt.Fprintf(writer, "Overall Risk Level: %s\n", summary.OverallRisk)
	fmt.Fprintf(writer, "Compliance Status: %s\n", summary.ComplianceStatus)
	fmt.Fprintf(writer, "Analysis Date: %s\n\n", summary.Timestamp.Format("January 2, 2006 15:04 MST"))

	fmt.Fprintf(writer, "KEY FINDINGS:\n")
	for i, finding := range summary.KeyFindings {
		fmt.Fprintf(writer, "  %d. %s\n", i+1, finding)
	}

	fmt.Fprintf(writer, "\nRECOMMENDATIONS:\n")
	for i, rec := range summary.Recommendations {
		fmt.Fprintf(writer, "  %d. %s\n", i+1, rec)
	}

	fmt.Fprintf(writer, "\nKEY METRICS:\n")
	for key, value := range summary.Metrics {
		fmt.Fprintf(writer, "  %s: %v\n", strings.Replace(key, "_", " ", -1), value)
	}

	return nil
}

// outputComplianceReport outputs a compliance report
func outputComplianceReport(report *differ.DriftReport, writer *os.File) error {
	if report.ComplianceReport == nil {
		return fmt.Errorf("no compliance report available")
	}

	compliance := report.ComplianceReport

	fmt.Fprintf(writer, "COMPLIANCE REPORT\n")
	fmt.Fprintf(writer, "================\n\n")
	fmt.Fprintf(writer, "Framework: %s\n", compliance.Framework)
	fmt.Fprintf(writer, "Status: %s\n", compliance.Status)
	fmt.Fprintf(writer, "Score: %.1f/100\n", compliance.Score)
	fmt.Fprintf(writer, "Date: %s\n\n", compliance.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Fprintf(writer, "SUMMARY:\n")
	fmt.Fprintf(writer, "  Total Rules: %d\n", compliance.Summary.TotalRules)
	fmt.Fprintf(writer, "  Passed: %d\n", compliance.Summary.PassedRules)
	fmt.Fprintf(writer, "  Failed: %d\n", compliance.Summary.FailedRules)
	fmt.Fprintf(writer, "  Critical Issues: %d\n", compliance.Summary.CriticalIssues)
	fmt.Fprintf(writer, "  High Issues: %d\n", compliance.Summary.HighIssues)

	if len(compliance.Violations) > 0 {
		fmt.Fprintf(writer, "\nVIOLATIONS:\n")
		for _, violation := range compliance.Violations {
			fmt.Fprintf(writer, "  [%s] %s\n", violation.Severity, violation.Description)
			fmt.Fprintf(writer, "    Resource: %s\n", violation.ResourceID)
			fmt.Fprintf(writer, "    Remediation: %s\n\n", violation.Remediation)
		}
	}

	return nil
}

// outputEnterpriseReport outputs a comprehensive enterprise report
func outputEnterpriseReport(report *differ.DriftReport, writer *os.File) error {
	fmt.Fprintf(writer, "ENTERPRISE INFRASTRUCTURE ANALYSIS\n")
	fmt.Fprintf(writer, "==================================\n\n")

	// Summary section
	fmt.Fprintf(writer, "SUMMARY:\n")
	fmt.Fprintf(writer, "  Total Resources: %d\n", report.Summary.TotalResources)
	fmt.Fprintf(writer, "  Changed Resources: %d\n", report.Summary.ChangedResources)
	fmt.Fprintf(writer, "  Total Changes: %d\n", report.Summary.TotalChanges)
	fmt.Fprintf(writer, "  Risk Assessment: %s\n", report.Summary.RiskAssessment)
	fmt.Fprintf(writer, "  Analysis Duration: %v ms\n\n", report.Metadata["analysis_duration_ms"])

	// Risk breakdown
	fmt.Fprintf(writer, "RISK BREAKDOWN:\n")
	fmt.Fprintf(writer, "  Critical: %d\n", report.Summary.CriticalChanges)
	fmt.Fprintf(writer, "  High: %d\n", report.Summary.HighRiskChanges)
	fmt.Fprintf(writer, "  Medium: %d\n", report.Summary.MediumRiskChanges)
	fmt.Fprintf(writer, "  Low: %d\n\n", report.Summary.LowRiskChanges)

	// Compliance section if available
	if report.ComplianceReport != nil {
		fmt.Fprintf(writer, "COMPLIANCE:\n")
		fmt.Fprintf(writer, "  Status: %s\n", report.ComplianceReport.Status)
		fmt.Fprintf(writer, "  Score: %.1f/100\n", report.ComplianceReport.Score)
		fmt.Fprintf(writer, "  Violations: %d\n\n", len(report.ComplianceReport.Violations))
	}

	// Performance metrics if available
	if metrics, ok := report.Metadata["metrics"]; ok {
		fmt.Fprintf(writer, "PERFORMANCE METRICS:\n")
		if metricsMap, ok := metrics.(map[string]interface{}); ok {
			for key, value := range metricsMap {
				fmt.Fprintf(writer, "  %s: %v\n", strings.Replace(key, "_", " ", -1), value)
			}
		}
		fmt.Fprintf(writer, "\n")
	}

	// Detailed changes (limited for readability)
	if len(report.ResourceChanges) > 0 {
		fmt.Fprintf(writer, "SIGNIFICANT CHANGES:\n")
		count := 0
		for _, change := range report.ResourceChanges {
			if change.Severity >= differ.SeverityHigh && count < 10 {
				fmt.Fprintf(writer, "  [%s] %s (%s)\n", change.Severity, change.ResourceID, change.ResourceType)
				fmt.Fprintf(writer, "    Risk Score: %.2f\n", change.RiskScore)
				fmt.Fprintf(writer, "    Description: %s\n\n", change.Description)
				count++
			}
		}
		if len(report.ResourceChanges) > count {
			fmt.Fprintf(writer, "  ... and %d more changes\n", len(report.ResourceChanges)-count)
		}
	}

	return nil
}

// Helper functions

func getComplianceRules(framework string) []differ.ComplianceRule {
	rules := differ.GetBuiltInComplianceRules()

	// Filter rules by framework if specified
	if framework != "" {
		var filteredRules []differ.ComplianceRule
		for _, rule := range rules {
			if strings.ToLower(rule.Framework) == strings.ToLower(framework) {
				filteredRules = append(filteredRules, rule)
			}
		}
		if len(filteredRules) > 0 {
			return filteredRules
		}
	}

	return rules
}

func scanCurrentState(ctx context.Context) (*types.Snapshot, error) {
	// This would integrate with the existing scan functionality
	// For now, return a placeholder
	return &types.Snapshot{
		ID:        fmt.Sprintf("current-scan-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "auto-detected",
		Resources: []types.Resource{},
	}, nil
}

func getLatestSnapshots() (*types.Snapshot, *types.Snapshot, error) {
	// This would load the most recent snapshots from storage
	// For now, return placeholders
	baseline := &types.Snapshot{
		ID:        "latest-baseline",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Provider:  "auto-detected",
		Resources: []types.Resource{},
	}

	current := &types.Snapshot{
		ID:        "latest-current",
		Timestamp: time.Now(),
		Provider:  "auto-detected",
		Resources: []types.Resource{},
	}

	return baseline, current, nil
}

func getExitCode(report *differ.DriftReport) error {
	// Return exit code based on changes detected
	if report.Summary.TotalChanges > 0 {
		os.Exit(1) // Changes detected
	}
	return nil // No changes
}

func saveReportToFile(data interface{}, filename, format string) error {
	// Save report data to file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "json":
		return outputJSON(data)
	case "yaml":
		return outputYAML(data)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
