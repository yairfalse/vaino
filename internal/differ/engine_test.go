package differ

import (
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestDifferEngine_Compare(t *testing.T) {
	tests := []struct {
		name         string
		baseline     *types.Snapshot
		current      *types.Snapshot
		options      DiffOptions
		expectError  bool
		expectedType string
	}{
		{
			name: "identical snapshots",
			baseline: &types.Snapshot{
				ID:        "baseline-1",
				Timestamp: time.Now(),
				Resources: []types.Resource{
					{
						ID:       "resource-1",
						Type:     "instance",
						Name:     "test-instance",
						Provider: "aws",
						Region:   "us-east-1",
						Configuration: map[string]interface{}{
							"instance_type": "t3.micro",
							"state":         "running",
						},
					},
				},
			},
			current: &types.Snapshot{
				ID:        "current-1",
				Timestamp: time.Now(),
				Resources: []types.Resource{
					{
						ID:       "resource-1",
						Type:     "instance",
						Name:     "test-instance",
						Provider: "aws",
						Region:   "us-east-1",
						Configuration: map[string]interface{}{
							"instance_type": "t3.micro",
							"state":         "running",
						},
					},
				},
			},
			options:      DiffOptions{},
			expectError:  false,
			expectedType: "no_drift",
		},
		{
			name: "resource added",
			baseline: &types.Snapshot{
				ID:        "baseline-2",
				Timestamp: time.Now(),
				Resources: []types.Resource{},
			},
			current: &types.Snapshot{
				ID:        "current-2",
				Timestamp: time.Now(),
				Resources: []types.Resource{
					{
						ID:       "resource-1",
						Type:     "instance",
						Name:     "new-instance",
						Provider: "aws",
						Region:   "us-east-1",
						Configuration: map[string]interface{}{
							"instance_type": "t3.micro",
							"state":         "running",
						},
					},
				},
			},
			options:      DiffOptions{},
			expectError:  false,
			expectedType: "resource_added",
		},
		{
			name: "resource removed",
			baseline: &types.Snapshot{
				ID:        "baseline-3",
				Timestamp: time.Now(),
				Resources: []types.Resource{
					{
						ID:       "resource-1",
						Type:     "instance",
						Name:     "old-instance",
						Provider: "aws",
						Region:   "us-east-1",
						Configuration: map[string]interface{}{
							"instance_type": "t3.micro",
							"state":         "running",
						},
					},
				},
			},
			current: &types.Snapshot{
				ID:        "current-3",
				Timestamp: time.Now(),
				Resources: []types.Resource{},
			},
			options:      DiffOptions{},
			expectError:  false,
			expectedType: "resource_removed",
		},
		{
			name: "resource modified",
			baseline: &types.Snapshot{
				ID:        "baseline-4",
				Timestamp: time.Now(),
				Resources: []types.Resource{
					{
						ID:       "resource-1",
						Type:     "instance",
						Name:     "test-instance",
						Provider: "aws",
						Region:   "us-east-1",
						Configuration: map[string]interface{}{
							"instance_type": "t3.micro",
							"state":         "running",
						},
					},
				},
			},
			current: &types.Snapshot{
				ID:        "current-4",
				Timestamp: time.Now(),
				Resources: []types.Resource{
					{
						ID:       "resource-1",
						Type:     "instance",
						Name:     "test-instance",
						Provider: "aws",
						Region:   "us-east-1",
						Configuration: map[string]interface{}{
							"instance_type": "t3.medium", // Changed from t3.micro
							"state":         "running",
						},
					},
				},
			},
			options:      DiffOptions{},
			expectError:  false,
			expectedType: "resource_modified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewDifferEngine(tt.options)

			report, err := engine.Compare(tt.baseline, tt.current)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if report == nil {
				t.Errorf("expected report but got nil")
				return
			}

			// Verify report structure
			if report.BaselineID != tt.baseline.ID {
				t.Errorf("expected BaselineID %s, got %s", tt.baseline.ID, report.BaselineID)
			}

			if report.CurrentID != tt.current.ID {
				t.Errorf("expected CurrentID %s, got %s", tt.current.ID, report.CurrentID)
			}

			// Verify expectations based on test type
			switch tt.expectedType {
			case "no_drift":
				if report.Summary.ChangedResources != 0 {
					t.Errorf("expected no changed resources, got %d", report.Summary.ChangedResources)
				}
			case "resource_added":
				if report.Summary.AddedResources != 1 {
					t.Errorf("expected 1 added resource, got %d", report.Summary.AddedResources)
				}
			case "resource_removed":
				if report.Summary.RemovedResources != 1 {
					t.Errorf("expected 1 removed resource, got %d", report.Summary.RemovedResources)
				}
			case "resource_modified":
				if report.Summary.ModifiedResources != 1 {
					t.Errorf("expected 1 modified resource, got %d", report.Summary.ModifiedResources)
				}
			}
		})
	}
}

func TestDifferEngine_CalculateDrift(t *testing.T) {
	engine := NewDifferEngine(DiffOptions{})

	changes := []Change{
		{
			Type:       ChangeTypeAdded,
			ResourceID: "resource-1",
			Field:      "instance_type",
			Severity:   RiskLevelHigh,
			Category:   DriftCategoryConfig,
		},
		{
			Type:       ChangeTypeModified,
			ResourceID: "resource-2",
			Field:      "security_group",
			Severity:   RiskLevelCritical,
			Category:   DriftCategorySecurity,
		},
	}

	summary := engine.CalculateDrift(changes)

	if summary.ChangedResources != 2 {
		t.Errorf("expected 2 changed resources, got %d", summary.ChangedResources)
	}

	if summary.ChangesByCategory[DriftCategoryConfig] != 1 {
		t.Errorf("expected 1 config change, got %d", summary.ChangesByCategory[DriftCategoryConfig])
	}

	if summary.ChangesByCategory[DriftCategorySecurity] != 1 {
		t.Errorf("expected 1 security change, got %d", summary.ChangesByCategory[DriftCategorySecurity])
	}

	if summary.ChangesBySeverity[RiskLevelHigh] != 1 {
		t.Errorf("expected 1 high risk change, got %d", summary.ChangesBySeverity[RiskLevelHigh])
	}

	if summary.ChangesBySeverity[RiskLevelCritical] != 1 {
		t.Errorf("expected 1 critical risk change, got %d", summary.ChangesBySeverity[RiskLevelCritical])
	}
}

func TestDifferEngine_WithIgnoreFields(t *testing.T) {
	baseline := &types.Snapshot{
		ID: "baseline",
		Resources: []types.Resource{
			{
				ID:       "resource-1",
				Type:     "instance",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"timestamp":     "2024-01-01T00:00:00Z",
					"state":         "running",
				},
			},
		},
	}

	current := &types.Snapshot{
		ID: "current",
		Resources: []types.Resource{
			{
				ID:       "resource-1",
				Type:     "instance",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"instance_type": "t3.micro",
					"timestamp":     "2024-01-02T00:00:00Z", // Changed timestamp
					"state":         "running",
				},
			},
		},
	}

	// Test without ignore fields - should detect timestamp change
	engine1 := NewDifferEngine(DiffOptions{})
	report1, err := engine1.Compare(baseline, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test with ignore fields - should ignore timestamp change
	engine2 := NewDifferEngine(DiffOptions{
		IgnoreFields: []string{"timestamp"},
	})
	report2, err := engine2.Compare(baseline, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without ignore fields, should detect change
	if report1.Summary.ChangedResources == 0 {
		t.Errorf("expected changes without ignore fields")
	}

	// With ignore fields, should not detect change
	if report2.Summary.ChangedResources != 0 {
		t.Errorf("expected no changes with ignore fields, got %d", report2.Summary.ChangedResources)
	}
}
