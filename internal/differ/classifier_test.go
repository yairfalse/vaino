package differ

import (
	"testing"
)

func TestDefaultClassifier_ClassifyChange(t *testing.T) {
	classifier := &DefaultClassifier{}

	tests := []struct {
		name             string
		change           Change
		expectedCategory DriftCategory
		expectedRisk     RiskLevel
		expectedMinScore float64
	}{
		{
			name: "instance type change",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "i-123",
				Field:      "instance_type",
				OldValue:   "t3.micro",
				NewValue:   "t3.xlarge",
			},
			expectedCategory: DriftCategoryCost,
			expectedRisk:     RiskLevelHigh,
			expectedMinScore: 0.5,
		},
		{
			name: "security group change",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "sg-123",
				Field:      "security_groups",
				OldValue:   []interface{}{},
				NewValue:   []interface{}{map[string]interface{}{"port": 22}},
			},
			expectedCategory: DriftCategorySecurity,
			expectedRisk:     RiskLevelCritical,
			expectedMinScore: 0.9,
		},
		{
			name: "public IP change",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "i-123",
				Field:      "public_ip",
				OldValue:   "1.2.3.4",
				NewValue:   "1.2.3.5",
			},
			expectedCategory: DriftCategoryNetwork,
			expectedRisk:     RiskLevelMedium,
			expectedMinScore: 0.4,
		},
		{
			name: "storage size change",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "vol-123",
				Field:      "size",
				OldValue:   "100GB",
				NewValue:   "500GB",
			},
			expectedCategory: DriftCategoryStorage,
			expectedRisk:     RiskLevelHigh,
			expectedMinScore: 0.6,
		},
		{
			name: "cost related change",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "i-123",
				Field:      "billing_mode",
				OldValue:   "on-demand",
				NewValue:   "reserved",
			},
			expectedCategory: DriftCategoryCost,
			expectedRisk:     RiskLevelMedium,
			expectedMinScore: 0.3,
		},
		{
			name: "tag change - low risk",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "i-123",
				Field:      "tags.Environment",
				OldValue:   "dev",
				NewValue:   "staging",
			},
			expectedCategory: DriftCategoryConfig,
			expectedRisk:     RiskLevelLow,
			expectedMinScore: 0.1,
		},
		{
			name: "state change",
			change: Change{
				Type:       ChangeTypeModified,
				ResourceID: "i-123",
				Field:      "state",
				OldValue:   "running",
				NewValue:   "stopped",
			},
			expectedCategory: DriftCategoryState,
			expectedRisk:     RiskLevelHigh,
			expectedMinScore: 0.7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, risk, score := classifier.ClassifyChange(tt.change)

			if category != tt.expectedCategory {
				t.Errorf("expected category %s, got %s", tt.expectedCategory, category)
			}

			if risk != tt.expectedRisk {
				t.Errorf("expected risk %s, got %s", tt.expectedRisk, risk)
			}

			if score < tt.expectedMinScore {
				t.Errorf("expected score >= %f, got %f", tt.expectedMinScore, score)
			}
		})
	}
}

func TestDefaultClassifier_CalculateResourceRisk(t *testing.T) {
	classifier := &DefaultClassifier{}

	tests := []struct {
		name         string
		changes      []Change
		expectedRisk RiskLevel
		minScore     float64
	}{
		{
			name: "multiple critical changes",
			changes: []Change{
				{
					Type:     ChangeTypeModified,
					Field:    "instance_type",
					Severity: RiskLevelCritical,
				},
				{
					Type:     ChangeTypeModified,
					Field:    "security_groups",
					Severity: RiskLevelCritical,
				},
			},
			expectedRisk: RiskLevelCritical,
			minScore:     0.8,
		},
		{
			name: "mixed severity changes",
			changes: []Change{
				{
					Type:     ChangeTypeModified,
					Field:    "public_ip",
					Severity: RiskLevelMedium,
				},
				{
					Type:     ChangeTypeModified,
					Field:    "tags",
					Severity: RiskLevelLow,
				},
			},
			expectedRisk: RiskLevelMedium,
			minScore:     0.3,
		},
		{
			name: "single high risk change",
			changes: []Change{
				{
					Type:     ChangeTypeModified,
					Field:    "state",
					Severity: RiskLevelHigh,
				},
			},
			expectedRisk: RiskLevelHigh,
			minScore:     0.6,
		},
		{
			name: "multiple low risk changes",
			changes: []Change{
				{
					Type:     ChangeTypeModified,
					Field:    "tags.team",
					Severity: RiskLevelLow,
				},
				{
					Type:     ChangeTypeModified,
					Field:    "tags.project",
					Severity: RiskLevelLow,
				},
			},
			expectedRisk: RiskLevelLow,
			minScore:     0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk, score := classifier.CalculateResourceRisk(tt.changes)

			if risk != tt.expectedRisk {
				t.Errorf("expected risk %s, got %s", tt.expectedRisk, risk)
			}

			if score < tt.minScore {
				t.Errorf("expected score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}

func TestDefaultClassifier_CalculateOverallRisk(t *testing.T) {
	classifier := &DefaultClassifier{}

	tests := []struct {
		name         string
		summary      DriftSummary
		expectedRisk RiskLevel
	}{
		{
			name: "high percentage of critical changes",
			summary: DriftSummary{
				ChangedResources: 10,
				ChangesBySeverity: map[RiskLevel]int{
					RiskLevelCritical: 8,
					RiskLevelHigh:     2,
				},
			},
			expectedRisk: RiskLevelCritical,
		},
		{
			name: "majority high risk changes",
			summary: DriftSummary{
				ChangedResources: 10,
				ChangesBySeverity: map[RiskLevel]int{
					RiskLevelHigh:   6,
					RiskLevelMedium: 4,
				},
			},
			expectedRisk: RiskLevelHigh,
		},
		{
			name: "mostly medium risk changes",
			summary: DriftSummary{
				ChangedResources: 10,
				ChangesBySeverity: map[RiskLevel]int{
					RiskLevelMedium: 7,
					RiskLevelLow:    3,
				},
			},
			expectedRisk: RiskLevelMedium,
		},
		{
			name: "only low risk changes",
			summary: DriftSummary{
				ChangedResources: 5,
				ChangesBySeverity: map[RiskLevel]int{
					RiskLevelLow: 5,
				},
			},
			expectedRisk: RiskLevelLow,
		},
		{
			name: "no changes",
			summary: DriftSummary{
				ChangedResources:  0,
				ChangesBySeverity: map[RiskLevel]int{},
			},
			expectedRisk: RiskLevelLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk, _ := classifier.CalculateOverallRisk(tt.summary)

			if risk != tt.expectedRisk {
				t.Errorf("expected risk %s, got %s", tt.expectedRisk, risk)
			}
		})
	}
}

func TestDefaultClassifier_EdgeCases(t *testing.T) {
	classifier := &DefaultClassifier{}

	// Test with empty change
	emptyChange := Change{}
	category, risk, score := classifier.ClassifyChange(emptyChange)

	// Should assign default values
	if category == "" {
		t.Error("expected non-empty category for empty change")
	}
	if risk == "" {
		t.Error("expected non-empty risk for empty change")
	}
	if score < 0 || score > 1 {
		t.Errorf("expected score between 0-1, got %f", score)
	}

	// Test with nil changes slice
	risk2, score2 := classifier.CalculateResourceRisk(nil)
	if risk2 != RiskLevelLow {
		t.Errorf("expected low risk for nil changes, got %s", risk2)
	}
	if score2 != 0.0 {
		t.Errorf("expected 0 score for nil changes, got %f", score2)
	}

	// Test with empty changes slice
	risk3, score3 := classifier.CalculateResourceRisk([]Change{})
	if risk3 != RiskLevelLow {
		t.Errorf("expected low risk for empty changes, got %s", risk3)
	}
	if score3 != 0.0 {
		t.Errorf("expected 0 score for empty changes, got %f", score3)
	}
}

func TestDefaultClassifier_SecurityRules(t *testing.T) {
	classifier := &DefaultClassifier{}

	securityTests := []struct {
		name         string
		field        string
		oldValue     interface{}
		newValue     interface{}
		expectedRisk RiskLevel
	}{
		{
			name:         "SSH port opened",
			field:        "ingress_rules",
			oldValue:     []interface{}{},
			newValue:     []interface{}{map[string]interface{}{"port": 22, "cidr": "0.0.0.0/0"}},
			expectedRisk: RiskLevelCritical,
		},
		{
			name:         "database port exposed",
			field:        "ingress_rules",
			oldValue:     []interface{}{},
			newValue:     []interface{}{map[string]interface{}{"port": 3306, "cidr": "0.0.0.0/0"}},
			expectedRisk: RiskLevelCritical,
		},
		{
			name:         "IAM policy changed",
			field:        "policy_document",
			oldValue:     `{"Version": "2012-10-17", "Statement": []}`,
			newValue:     `{"Version": "2012-10-17", "Statement": [{"Effect": "Allow", "Action": "*", "Resource": "*"}]}`,
			expectedRisk: RiskLevelCritical,
		},
		{
			name:         "encryption disabled",
			field:        "encryption_enabled",
			oldValue:     true,
			newValue:     false,
			expectedRisk: RiskLevelCritical,
		},
		{
			name:         "public access enabled",
			field:        "public_access_block",
			oldValue:     true,
			newValue:     false,
			expectedRisk: RiskLevelHigh,
		},
	}

	for _, tt := range securityTests {
		t.Run(tt.name, func(t *testing.T) {
			change := Change{
				Type:     ChangeTypeModified,
				Field:    tt.field,
				OldValue: tt.oldValue,
				NewValue: tt.newValue,
			}

			category, risk, _ := classifier.ClassifyChange(change)

			if category != DriftCategorySecurity {
				t.Errorf("expected security category, got %s", category)
			}

			if risk != tt.expectedRisk {
				t.Errorf("expected risk %s, got %s", tt.expectedRisk, risk)
			}
		})
	}
}
