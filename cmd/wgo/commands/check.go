package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

func newCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check for infrastructure drift",
		Long: `Check compares the current infrastructure state against a baseline
to detect configuration drift. It can automatically scan the current state
or use a previously captured snapshot for comparison.`,
		Example: `  # Check against latest baseline
  wgo check

  # Check against specific baseline
  wgo check --baseline prod-baseline-2025-01-15

  # Check with current scan and AI analysis
  wgo check --scan --explain

  # Check specific provider only
  wgo check --provider aws --region us-east-1

  # Generate detailed report
  wgo check --baseline prod-v1.0 --output-file drift-report.json --format json`,
		RunE: runCheck,
	}

	// Flags
	cmd.Flags().StringP("baseline", "b", "", "baseline name to compare against")
	cmd.Flags().Bool("scan", false, "perform current scan before comparison")
	cmd.Flags().StringP("provider", "p", "", "limit check to specific provider")
	cmd.Flags().StringSlice("region", []string{}, "limit check to specific regions")
	cmd.Flags().StringSlice("namespace", []string{}, "limit check to specific namespaces")
	cmd.Flags().Bool("explain", false, "get AI analysis of detected drift")
	cmd.Flags().String("output-file", "", "save drift report to file")
	cmd.Flags().Float64("risk-threshold", 0.0, "minimum risk score to report (0.0-1.0)")
	cmd.Flags().Bool("fail-on-drift", false, "exit with non-zero code if drift detected")
	cmd.Flags().StringSlice("ignore-fields", []string{}, "ignore changes in specified fields")
	cmd.Flags().Bool("summary-only", false, "show only summary, not detailed changes")

	return cmd
}

// Helper functions

func createPlaceholderSnapshot(provider string) *types.Snapshot {
	// Create a simple placeholder snapshot for demo purposes
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("current-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  provider,
		Resources: []types.Resource{
			{
				ID:       "demo-resource-1",
				Type:     "instance",
				Name:     "demo-instance",
				Provider: "aws",
				Region:   "us-east-1",
				Configuration: map[string]interface{}{
					"instance_type": "t3.medium",
					"state":         "running",
				},
				Tags: map[string]string{
					"Environment": "production",
					"Application": "web-server",
				},
			},
		},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			CollectionTime:   time.Second * 5,
			ResourceCount:    1,
		},
	}
	return snapshot
}

func displayDriftReport(report *differ.DriftReport, summaryOnly bool) {
	fmt.Println("📊 Drift Summary")
	fmt.Println("=================")
	fmt.Printf("Total Resources: %d\n", report.Summary.TotalResources)
	fmt.Printf("Changed Resources: %d\n", report.Summary.ChangedResources)
	fmt.Printf("Added Resources: %d\n", report.Summary.AddedResources)
	fmt.Printf("Removed Resources: %d\n", report.Summary.RemovedResources)
	fmt.Printf("Modified Resources: %d\n", report.Summary.ModifiedResources)
	fmt.Printf("Overall Risk: %s (%.2f)\n", report.Summary.OverallRisk, report.Summary.RiskScore)

	if len(report.Summary.ChangesBySeverity) > 0 {
		fmt.Println("\n📈 Changes by Severity:")
		for severity, count := range report.Summary.ChangesBySeverity {
			if count > 0 {
				fmt.Printf("  %s: %d\n", severity, count)
			}
		}
	}

	if len(report.Summary.ChangesByCategory) > 0 {
		fmt.Println("\n📋 Changes by Category:")
		for category, count := range report.Summary.ChangesByCategory {
			if count > 0 {
				fmt.Printf("  %s: %d\n", category, count)
			}
		}
	}

	if !summaryOnly && len(report.ResourceChanges) > 0 {
		fmt.Println("\n🔍 Detailed Changes")
		fmt.Println("====================")

		for _, resourceChange := range report.ResourceChanges {
			fmt.Printf("\n📦 Resource: %s (%s)\n", resourceChange.ResourceID, resourceChange.ResourceType)
			fmt.Printf("   Provider: %s\n", resourceChange.Provider)
			fmt.Printf("   Change Type: %s\n", resourceChange.DriftType)
			fmt.Printf("   Severity: %s\n", resourceChange.Severity)
			fmt.Printf("   Risk Score: %.2f\n", resourceChange.RiskScore)
			fmt.Printf("   Description: %s\n", resourceChange.Description)

			if len(resourceChange.Changes) > 0 {
				fmt.Println("   Changes:")
				for _, change := range resourceChange.Changes {
					fmt.Printf("     • %s: %v → %v (%s)\n",
						change.Field, change.OldValue, change.NewValue, change.Severity)
				}
			}
		}
	}

	if report.Summary.ChangedResources == 0 {
		fmt.Println("\n✅ No drift detected - infrastructure matches baseline")
	} else {
		fmt.Printf("\n⚠️  Drift detected in %d resources\n", report.Summary.ChangedResources)
	}
}

func saveDriftReport(report *differ.DriftReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func runCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Infrastructure Drift Check")
	fmt.Println("=============================")

	// Parse flags
	baseline, _ := cmd.Flags().GetString("baseline")
	scan, _ := cmd.Flags().GetBool("scan")
	provider, _ := cmd.Flags().GetString("provider")
	explain, _ := cmd.Flags().GetBool("explain")
	outputFile, _ := cmd.Flags().GetString("output-file")
	riskThreshold, _ := cmd.Flags().GetFloat64("risk-threshold")
	failOnDrift, _ := cmd.Flags().GetBool("fail-on-drift")
	ignoreFields, _ := cmd.Flags().GetStringSlice("ignore-fields")
	summaryOnly, _ := cmd.Flags().GetBool("summary-only")

	// Initialize storage
	storageConfig := storage.Config{BaseDir: "./snapshots"}
	localStorage, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Load baseline snapshot
	var baselineSnapshot *types.Snapshot
	if baseline == "" {
		fmt.Println("📋 Finding latest baseline...")
		baselines, err := localStorage.ListBaselines()
		if err != nil {
			return fmt.Errorf("failed to list baselines: %w", err)
		}
		if len(baselines) == 0 {
			fmt.Println("❌ No Baselines Found")
			fmt.Println("====================")
			fmt.Println()
			fmt.Println("You need to create a baseline first!")
			fmt.Println()
			fmt.Println("🎯 DO THIS NOW:")
			fmt.Println()
			fmt.Println("  1. Scan your infrastructure (if not done already):")
			fmt.Println("     wgo scan --provider terraform")
			fmt.Println("     wgo scan --provider aws --region us-east-1")
			fmt.Println("     wgo scan --provider gcp --project YOUR-PROJECT")
			fmt.Println()
			fmt.Println("  2. Create a baseline:")
			fmt.Println("     wgo baseline create --name prod-baseline")
			fmt.Println()
			fmt.Println("  3. Then check for drift:")
			fmt.Println("     wgo check")
			fmt.Println()
			fmt.Println("💡 TIP: The baseline is your 'known good' state")
			return nil
		}
		// Use the most recent baseline
		fmt.Printf("📋 Using baseline: %s\n", baselines[0].Name)
		baselineSnapshot, err = localStorage.LoadSnapshot(baselines[0].SnapshotID)
		if err != nil {
			return fmt.Errorf("failed to load baseline snapshot: %w", err)
		}
	} else {
		fmt.Printf("📋 Loading baseline: %s\n", baseline)
		baselineData, err := localStorage.LoadBaseline(baseline)
		if err != nil {
			return fmt.Errorf("failed to load baseline '%s': %w", baseline, err)
		}
		baselineSnapshot, err = localStorage.LoadSnapshot(baselineData.SnapshotID)
		if err != nil {
			return fmt.Errorf("failed to load baseline snapshot: %w", err)
		}
	}

	// Get current snapshot
	var currentSnapshot *types.Snapshot
	if scan {
		fmt.Println("🔍 Performing current state scan...")
		// TODO: Implement actual scanning
		// For now, create a placeholder snapshot
		currentSnapshot = createPlaceholderSnapshot(provider)
	} else {
		fmt.Println("📊 Loading latest snapshot...")
		snapshots, err := localStorage.ListSnapshots()
		if err != nil {
			return fmt.Errorf("failed to list snapshots: %w", err)
		}
		if len(snapshots) == 0 {
			fmt.Println("❌ No Infrastructure Snapshots Found")
			fmt.Println("=====================================")
			fmt.Println()
			fmt.Println("You need to scan your infrastructure first!")
			fmt.Println()
			fmt.Println("🎯 DO THIS NOW (choose one):")
			fmt.Println()
			fmt.Println("  wgo scan --provider terraform")
			fmt.Println("  wgo scan --provider aws --region us-east-1")
			fmt.Println("  wgo scan --provider gcp --project YOUR-PROJECT")
			fmt.Println("  wgo scan --provider kubernetes")
			fmt.Println()
			fmt.Println("💡 TIP: Having auth issues? Run 'wgo auth status'")
			return nil
		}
		// Load the most recent snapshot
		currentSnapshot, err = localStorage.LoadSnapshot(snapshots[0].ID)
		if err != nil {
			return fmt.Errorf("failed to load current snapshot: %w", err)
		}
	}

	// Configure differ options
	options := differ.DiffOptions{
		IgnoreFields: ignoreFields,
		MinRiskLevel: differ.RiskLevel(fmt.Sprintf("%.1f", riskThreshold)),
	}

	if provider != "" {
		options.IgnoreProviders = []string{}
		// Filter to only the specified provider
	}

	// Create differ engine
	differ := differ.NewDifferEngine(options)

	fmt.Println("\n🔍 Comparing snapshots...")
	startTime := time.Now()

	// Perform comparison
	report, err := differ.Compare(baselineSnapshot, currentSnapshot)
	if err != nil {
		return fmt.Errorf("drift comparison failed: %w", err)
	}

	comparisonTime := time.Since(startTime)
	fmt.Printf("✅ Comparison completed in %v\n\n", comparisonTime)

	// Display results
	displayDriftReport(report, summaryOnly)

	// Save report if requested
	if outputFile != "" {
		if err := saveDriftReport(report, outputFile); err != nil {
			fmt.Printf("⚠️  Failed to save report: %v\n", err)
		} else {
			fmt.Printf("💾 Report saved to: %s\n", outputFile)
		}
	}

	// AI explanation if requested
	if explain && len(report.ResourceChanges) > 0 {
		fmt.Println("\n🤖 AI Analysis")
		fmt.Println("===============")
		// TODO: Integrate with Claude AI for analysis
		fmt.Println("AI analysis integration pending...")
	}

	// Exit with error if drift detected and fail-on-drift is enabled
	if failOnDrift && report.Summary.ChangedResources > 0 {
		if report.Summary.OverallRisk == "high" || report.Summary.OverallRisk == "critical" {
			return fmt.Errorf("infrastructure drift detected with %s risk level", report.Summary.OverallRisk)
		}
	}

	return nil
}
