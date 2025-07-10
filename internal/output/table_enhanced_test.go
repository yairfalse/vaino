package output

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

func TestEnhancedTableRenderer_RenderDriftReport_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		report   *differ.DriftReport
		noColor  bool
		maxWidth int
		checks   []func(string) error
	}{
		{
			name:     "complete drift report with all severity levels",
			report:   createComprehensiveDriftReport(),
			noColor:  true,
			maxWidth: 120,
			checks: []func(string) error{
				func(output string) error {
					if !strings.Contains(output, "Infrastructure Drift Report") {
						return fmt.Errorf("missing report header")
					}
					return nil
				},
				func(output string) error {
					if !strings.Contains(output, "┌") || !strings.Contains(output, "┐") {
						return fmt.Errorf("missing table borders")
					}
					return nil
				},
				func(output string) error {
					severities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}
					for _, severity := range severities {
						if !strings.Contains(output, severity) {
							return fmt.Errorf("missing severity level: %s", severity)
						}
					}
					return nil
				},
				func(output string) error {
					categories := []string{"Security", "Cost", "Network", "Storage"}
					for _, category := range categories {
						if !strings.Contains(output, category) {
							return fmt.Errorf("missing category: %s", category)
						}
					}
					return nil
				},
				func(output string) error {
					if !strings.Contains(output, "Change Summary") {
						return fmt.Errorf("missing summary section")
					}
					return nil
				},
			},
		},
		{
			name:     "empty report",
			report:   createEmptyDriftReport(),
			noColor:  true,
			maxWidth: 80,
			checks: []func(string) error{
				func(output string) error {
					if !strings.Contains(output, "No drift detected") {
						return fmt.Errorf("missing no-drift message")
					}
					return nil
				},
			},
		},
		{
			name:     "single change report",
			report:   createSingleChangeDriftReport(),
			noColor:  false, // Test with colors
			maxWidth: 100,
			checks: []func(string) error{
				func(output string) error {
					if !strings.Contains(output, "test-resource-1") {
						return fmt.Errorf("missing resource identifier")
					}
					return nil
				},
				func(output string) error {
					if !strings.Contains(output, "Modified") {
						return fmt.Errorf("missing change type")
					}
					return nil
				},
			},
		},
		{
			name:     "narrow width test",
			report:   createComprehensiveDriftReport(),
			noColor:  true,
			maxWidth: 60, // Narrow width to test truncation
			checks: []func(string) error{
				func(output string) error {
					lines := strings.Split(output, "\n")
					for _, line := range lines {
						if len(line) > 80 { // Allow some margin for Unicode chars
							return fmt.Errorf("line too long for narrow width: %d chars", len(line))
						}
					}
					return nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewEnhancedTableRenderer(tt.noColor, tt.maxWidth)
			output := renderer.RenderDriftReport(tt.report)

			for i, check := range tt.checks {
				if err := check(output); err != nil {
					t.Errorf("Check %d failed: %v", i+1, err)
					t.Logf("Output:\n%s", output)
				}
			}
		})
	}
}

func TestEnhancedTableRenderer_RenderResourceList_Comprehensive(t *testing.T) {
	tests := []struct {
		name      string
		resources []types.Resource
		noColor   bool
		expected  []string
	}{
		{
			name:      "mixed providers and types",
			resources: createMixedResourceList(),
			noColor:   true,
			expected: []string{
				"Found 6 resources",
				"AWS (4 resources)",
				"KUBERNETES (2 resources)",
				"instance: 2",
				"security_group: 1",
				"s3_bucket: 1",
				"deployment: 1",
				"service: 1",
			},
		},
		{
			name:      "single provider",
			resources: createSingleProviderResourceList(),
			noColor:   true,
			expected: []string{
				"Found 3 resources",
				"AWS (3 resources)",
				"instance: 3",
			},
		},
		{
			name:      "empty resource list",
			resources: []types.Resource{},
			noColor:   true,
			expected: []string{
				"No resources found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewEnhancedTableRenderer(tt.noColor, 120)
			output := renderer.RenderResourceList(tt.resources)

			for _, expected := range tt.expected {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain: %s", expected)
					t.Logf("Actual output:\n%s", output)
				}
			}
		})
	}
}

func TestEnhancedTableRenderer_ColorHandling(t *testing.T) {
	report := createSingleChangeDriftReport()

	// Test with colors enabled
	rendererColor := NewEnhancedTableRenderer(false, 120)
	outputColor := rendererColor.RenderDriftReport(report)

	// Test with colors disabled
	rendererNoColor := NewEnhancedTableRenderer(true, 120)
	outputNoColor := rendererNoColor.RenderDriftReport(report)

	// Color output should be longer due to ANSI escape codes
	if len(outputColor) <= len(outputNoColor) {
		t.Error("Expected colored output to be longer than no-color output")
	}

	// No-color output should not contain ANSI escape codes
	if strings.Contains(outputNoColor, "\x1b[") {
		t.Error("No-color output should not contain ANSI escape codes")
	}
}

func TestEnhancedTableRenderer_HelperMethods(t *testing.T) {
	renderer := NewEnhancedTableRenderer(true, 120)

	// Test truncateResource
	tests := []struct {
		id           string
		resourceType string
		expected     string
	}{
		{"short-id", "instance", "instance:short-id"},
		{"very-long-resource-identifier-that-exceeds-limit", "instance", "instance:very-long-reso..."},
		{"id", "very-long-type-name", "very-long-type-name:id"},
	}

	for _, test := range tests {
		result := renderer.truncateResource(test.id, test.resourceType)
		if result != test.expected {
			t.Errorf("truncateResource(%s, %s) = %s, expected %s",
				test.id, test.resourceType, result, test.expected)
		}
	}

	// Test padString
	padTests := []struct {
		input    string
		width    int
		expected int
	}{
		{"test", 10, 10},
		{"verylongstring", 5, 14}, // Should not truncate, just return as-is
		{"exact", 5, 5},
	}

	for _, test := range padTests {
		result := renderer.padString(test.input, test.width)
		if len(result) != test.expected {
			t.Errorf("padString(%s, %d) length = %d, expected %d",
				test.input, test.width, len(result), test.expected)
		}
	}
}

func TestEnhancedTableRenderer_SeverityOrdering(t *testing.T) {
	// Create report with changes in mixed severity order
	report := &differ.DriftReport{
		ID:         "severity-test",
		BaselineID: "baseline",
		CurrentID:  "current",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:   4,
			ChangedResources: 4,
			OverallRisk:      differ.RiskLevelHigh,
			RiskScore:        0.8,
		},
		ResourceChanges: []differ.ResourceDiff{
			{ResourceID: "low-resource", Severity: differ.RiskLevelLow, DriftType: differ.ChangeTypeModified},
			{ResourceID: "critical-resource", Severity: differ.RiskLevelCritical, DriftType: differ.ChangeTypeModified},
			{ResourceID: "medium-resource", Severity: differ.RiskLevelMedium, DriftType: differ.ChangeTypeModified},
			{ResourceID: "high-resource", Severity: differ.RiskLevelHigh, DriftType: differ.ChangeTypeModified},
		},
	}

	renderer := NewEnhancedTableRenderer(true, 120)
	output := renderer.RenderDriftReport(report)

	// Check that critical appears before high, high before medium, etc.
	criticalPos := strings.Index(output, "critical-resource")
	highPos := strings.Index(output, "high-resource")
	mediumPos := strings.Index(output, "medium-resource")
	lowPos := strings.Index(output, "low-resource")

	if criticalPos == -1 || highPos == -1 || mediumPos == -1 || lowPos == -1 {
		t.Fatal("Not all resources found in output")
	}

	if !(criticalPos < highPos && highPos < mediumPos && mediumPos < lowPos) {
		t.Error("Resources not ordered by severity (critical -> high -> medium -> low)")
		t.Logf("Positions: critical=%d, high=%d, medium=%d, low=%d",
			criticalPos, highPos, mediumPos, lowPos)
	}
}

func TestEnhancedTableRenderer_TableStructure(t *testing.T) {
	report := createSingleChangeDriftReport()
	renderer := NewEnhancedTableRenderer(true, 120)
	output := renderer.RenderDriftReport(report)

	// Check for Unicode table characters
	requiredChars := []string{"┌", "┐", "├", "┤", "└", "┘", "┬", "┴", "┼", "─", "│"}
	for _, char := range requiredChars {
		if !strings.Contains(output, char) {
			t.Errorf("Table missing required Unicode character: %s", char)
		}
	}

	// Check table headers
	headers := []string{"Resource", "Change", "Severity", "Category", "Impact"}
	for _, header := range headers {
		if !strings.Contains(output, header) {
			t.Errorf("Table missing header: %s", header)
		}
	}
}

// Helper functions to create test data

func createComprehensiveDriftReport() *differ.DriftReport {
	return &differ.DriftReport{
		ID:         "comprehensive-test",
		BaselineID: "baseline-comprehensive",
		CurrentID:  "current-comprehensive",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:    10,
			ChangedResources:  6,
			AddedResources:    2,
			RemovedResources:  1,
			ModifiedResources: 3,
			OverallRisk:       differ.RiskLevelCritical,
			RiskScore:         0.9,
			ChangesBySeverity: map[differ.RiskLevel]int{
				differ.RiskLevelCritical: 2,
				differ.RiskLevelHigh:     2,
				differ.RiskLevelMedium:   1,
				differ.RiskLevelLow:      1,
			},
			ChangesByCategory: map[differ.DriftCategory]int{
				differ.DriftCategorySecurity: 2,
				differ.DriftCategoryCost:     2,
				differ.DriftCategoryNetwork:  1,
				differ.DriftCategoryStorage:  1,
			},
		},
		ResourceChanges: []differ.ResourceDiff{
			{
				ResourceID:   "i-critical-security",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelCritical,
				Category:     differ.DriftCategorySecurity,
				RiskScore:    0.95,
				Description:  "SSH access enabled from internet",
			},
			{
				ResourceID:   "sg-critical-rules",
				ResourceType: "security_group",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelCritical,
				Category:     differ.DriftCategorySecurity,
				RiskScore:    0.92,
				Description:  "Security group rules changed",
			},
			{
				ResourceID:   "i-high-cost",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelHigh,
				Category:     differ.DriftCategoryCost,
				RiskScore:    0.75,
				Description:  "Instance type upgraded significantly",
			},
			{
				ResourceID:   "vol-high-storage",
				ResourceType: "volume",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelHigh,
				Category:     differ.DriftCategoryStorage,
				RiskScore:    0.70,
				Description:  "Storage volume expanded",
			},
			{
				ResourceID:   "elb-medium-network",
				ResourceType: "load_balancer",
				DriftType:    differ.ChangeTypeAdded,
				Severity:     differ.RiskLevelMedium,
				Category:     differ.DriftCategoryNetwork,
				RiskScore:    0.50,
				Description:  "New load balancer added",
			},
			{
				ResourceID:   "tag-low-config",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelLow,
				Category:     differ.DriftCategoryConfig,
				RiskScore:    0.20,
				Description:  "Tags updated",
			},
		},
	}
}

func createEmptyDriftReport() *differ.DriftReport {
	return &differ.DriftReport{
		ID:         "empty-test",
		BaselineID: "baseline-empty",
		CurrentID:  "current-empty",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:    5,
			ChangedResources:  0,
			AddedResources:    0,
			RemovedResources:  0,
			ModifiedResources: 0,
			OverallRisk:       differ.RiskLevelLow,
			RiskScore:         0.0,
		},
		ResourceChanges: []differ.ResourceDiff{},
	}
}

func createSingleChangeDriftReport() *differ.DriftReport {
	return &differ.DriftReport{
		ID:         "single-test",
		BaselineID: "baseline-single",
		CurrentID:  "current-single",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:    3,
			ChangedResources:  1,
			ModifiedResources: 1,
			OverallRisk:       differ.RiskLevelMedium,
			RiskScore:         0.5,
			ChangesBySeverity: map[differ.RiskLevel]int{
				differ.RiskLevelMedium: 1,
			},
			ChangesByCategory: map[differ.DriftCategory]int{
				differ.DriftCategoryCost: 1,
			},
		},
		ResourceChanges: []differ.ResourceDiff{
			{
				ResourceID:   "test-resource-1",
				ResourceType: "instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelMedium,
				Category:     differ.DriftCategoryCost,
				RiskScore:    0.5,
				Description:  "Instance type changed",
			},
		},
	}
}

func createMixedResourceList() []types.Resource {
	return []types.Resource{
		{ID: "i-123", Type: "instance", Provider: "aws", Region: "us-east-1"},
		{ID: "i-456", Type: "instance", Provider: "aws", Region: "us-west-2"},
		{ID: "sg-789", Type: "security_group", Provider: "aws", Region: "us-east-1"},
		{ID: "bucket-abc", Type: "s3_bucket", Provider: "aws", Region: "us-east-1"},
		{ID: "deploy-web", Type: "deployment", Provider: "kubernetes", Namespace: "default"},
		{ID: "svc-api", Type: "service", Provider: "kubernetes", Namespace: "api"},
	}
}

func createSingleProviderResourceList() []types.Resource {
	return []types.Resource{
		{ID: "i-111", Type: "instance", Provider: "aws", Region: "us-east-1"},
		{ID: "i-222", Type: "instance", Provider: "aws", Region: "us-east-1"},
		{ID: "i-333", Type: "instance", Provider: "aws", Region: "us-east-1"},
	}
}

// Benchmark tests
func BenchmarkEnhancedTableRenderer_RenderDriftReport(b *testing.B) {
	report := createComprehensiveDriftReport()
	renderer := NewEnhancedTableRenderer(true, 120)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.RenderDriftReport(report)
	}
}

func BenchmarkEnhancedTableRenderer_RenderResourceList(b *testing.B) {
	resources := createMixedResourceList()
	renderer := NewEnhancedTableRenderer(true, 120)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderer.RenderResourceList(resources)
	}
}
