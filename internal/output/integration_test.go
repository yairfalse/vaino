package output

import (
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/types"
)

func TestEnhancedTableRenderer_RenderDriftReport(t *testing.T) {
	// Create a sample drift report
	report := &differ.DriftReport{
		ID:         "test-report-1",
		BaselineID: "baseline-1",
		CurrentID:  "current-1",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:    5,
			ChangedResources:  3,
			AddedResources:    1,
			RemovedResources:  0,
			ModifiedResources: 2,
			OverallRisk:       differ.RiskLevelHigh,
			RiskScore:         0.75,
			ChangesBySeverity: map[differ.RiskLevel]int{
				differ.RiskLevelCritical: 1,
				differ.RiskLevelHigh:     1,
				differ.RiskLevelMedium:   1,
			},
			ChangesByCategory: map[differ.DriftCategory]int{
				differ.DriftCategorySecurity: 1,
				differ.DriftCategoryCost:     1,
				differ.DriftCategoryNetwork:  1,
			},
		},
		ResourceChanges: []differ.ResourceDiff{
			{
				ResourceID:   "i-1234567890",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelCritical,
				Category:     differ.DriftCategorySecurity,
				RiskScore:    0.95,
				Description:  "Security group changed to allow SSH access",
				Changes: []differ.Change{
					{
						Field:    "security_groups",
						OldValue: []string{"sg-web"},
						NewValue: []string{"sg-web", "sg-ssh"},
						Severity: differ.RiskLevelCritical,
					},
				},
			},
			{
				ResourceID:   "i-0987654321",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelHigh,
				Category:     differ.DriftCategoryCost,
				RiskScore:    0.70,
				Description:  "Instance type changed from t3.micro to t3.large",
				Changes: []differ.Change{
					{
						Field:    "instance_type",
						OldValue: "t3.micro",
						NewValue: "t3.large",
						Severity: differ.RiskLevelHigh,
					},
				},
			},
			{
				ResourceID:   "elb-abc123",
				ResourceType: "load_balancer",
				DriftType:    differ.ChangeTypeAdded,
				Severity:     differ.RiskLevelMedium,
				Category:     differ.DriftCategoryNetwork,
				RiskScore:    0.50,
				Description:  "New load balancer added",
				Changes:      []differ.Change{},
			},
		},
	}

	// Test with colors disabled
	renderer := NewEnhancedTableRenderer(true, 100)
	output := renderer.RenderDriftReport(report)

	// Verify output contains expected elements
	if !strings.Contains(output, "Infrastructure Drift Report") {
		t.Error("Expected output to contain report header")
	}

	if !strings.Contains(output, "i-1234567890") {
		t.Error("Expected output to contain resource ID")
	}

	if !strings.Contains(output, "CRITICAL") {
		t.Error("Expected output to contain severity levels")
	}

	if !strings.Contains(output, "Security") {
		t.Error("Expected output to contain category information")
	}

	// Verify table structure (simplified check)
	if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
		t.Error("Expected output to contain table borders")
	}

	// Test summary section
	if !strings.Contains(output, "Change Summary") {
		t.Error("Expected output to contain summary section")
	}

	if !strings.Contains(output, "Total Resources: 5") {
		t.Error("Expected output to contain correct resource count")
	}
}

func TestEnhancedTableRenderer_RenderResourceList(t *testing.T) {
	resources := []types.Resource{
		{
			ID:       "i-1234567890",
			Type:     "instance",
			Name:     "web-server-1",
			Provider: "aws",
			Region:   "us-east-1",
		},
		{
			ID:       "i-0987654321",
			Type:     "instance",
			Name:     "web-server-2",
			Provider: "aws",
			Region:   "us-east-1",
		},
		{
			ID:       "sg-abc123",
			Type:     "security_group",
			Name:     "web-sg",
			Provider: "aws",
			Region:   "us-east-1",
		},
		{
			ID:        "deploy-web",
			Type:      "deployment",
			Name:      "web-deployment",
			Provider:  "kubernetes",
			Namespace: "default",
		},
	}

	renderer := NewEnhancedTableRenderer(true, 100)
	output := renderer.RenderResourceList(resources)

	// Verify output contains expected elements
	if !strings.Contains(output, "Found 4 resources") {
		t.Error("Expected output to contain resource count")
	}

	if !strings.Contains(output, "AWS") {
		t.Error("Expected output to contain provider information")
	}

	if !strings.Contains(output, "instance: 2") {
		t.Error("Expected output to contain resource type counts")
	}

	if !strings.Contains(output, "KUBERNETES") {
		t.Error("Expected output to contain all providers")
	}
}

func TestExportManager_ExportDriftReport(t *testing.T) {
	// Create atomic writer for testing
	atomicWriter := storage.NewAtomicWriter("")
	exportManager := NewExportManager(atomicWriter, true)

	// Create sample report
	report := &differ.DriftReport{
		ID:         "test-report",
		BaselineID: "baseline-1",
		CurrentID:  "current-1",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:   3,
			ChangedResources: 1,
			OverallRisk:      differ.RiskLevelMedium,
			RiskScore:        0.5,
		},
		ResourceChanges: []differ.ResourceDiff{
			{
				ResourceID:   "test-resource",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelMedium,
				Category:     differ.DriftCategoryCost,
				Description:  "Test change",
			},
		},
	}

	// Test JSON export
	options := ExportOptions{
		Format:     "json",
		OutputPath: "-", // stdout
		Pretty:     true,
	}

	err := exportManager.ExportDriftReport(report, options)
	if err != nil {
		t.Errorf("Failed to export to JSON: %v", err)
	}

	// Test YAML export
	options.Format = "yaml"
	err = exportManager.ExportDriftReport(report, options)
	if err != nil {
		t.Errorf("Failed to export to YAML: %v", err)
	}

	// Test Markdown export
	options.Format = "markdown"
	err = exportManager.ExportDriftReport(report, options)
	if err != nil {
		t.Errorf("Failed to export to Markdown: %v", err)
	}

	// Test CSV export
	options.Format = "csv"
	err = exportManager.ExportDriftReport(report, options)
	if err != nil {
		t.Errorf("Failed to export to CSV: %v", err)
	}
}

func TestProgressBar_Basic(t *testing.T) {
	config := ProgressBarConfig{
		Title:       "Test Progress",
		Total:       100,
		Width:       20,
		ShowPercent: true,
		ShowETA:     false,
		NoColor:     true,
	}

	bar := NewProgressBar(config)

	// Test updates
	bar.Update(25)
	bar.Update(50)
	bar.Update(75)
	bar.Finish()

	// Test increment
	bar2 := NewProgressBar(config)
	bar2.Increment(30)
	bar2.Increment(40)
	bar2.Increment(30)

	// No errors should occur
}

func TestSpinner_Basic(t *testing.T) {
	spinner := NewSpinner("Testing...", true)

	spinner.Start()
	spinner.Update("Still testing...")
	spinner.Stop()

	// No errors should occur
}

func TestStepProgress_Basic(t *testing.T) {
	steps := []string{
		"Initialize",
		"Scan resources",
		"Compare states",
		"Generate report",
		"Complete",
	}

	progress := NewStepProgress("Drift Detection", steps, true)

	progress.NextStep() // Initialize
	progress.NextStep() // Scan resources
	progress.SetStep(4) // Jump to Complete
	progress.Finish()

	// No errors should occur
}
