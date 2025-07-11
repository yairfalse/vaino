//go:build !enterprise
// +build !enterprise

package differ

import (
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// ComplianceRule stub for non-enterprise builds
type ComplianceRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Severity    RiskLevel              `json:"severity"`
	Category    DriftCategory          `json:"category"`
	Condition   string                 `json:"condition"`
	Remediation string                 `json:"remediation"`
	Framework   string                 `json:"framework"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GetBuiltInComplianceRules returns empty rules for non-enterprise
func GetBuiltInComplianceRules() []ComplianceRule {
	return []ComplianceRule{}
}

// DefaultResourceMatcher is a simple resource matcher for non-enterprise builds
type DefaultResourceMatcher struct{}

// NewSmartResourceMatcher creates a default resource matcher (stub)
func NewSmartResourceMatcher() ResourceMatcher {
	return &DefaultResourceMatcher{}
}

// Match implements simple ID-based matching
func (m *DefaultResourceMatcher) Match(baseline, current []types.Resource) ([]ResourceMatch, []types.Resource, []types.Resource) {
	var matches []ResourceMatch
	var added []types.Resource
	var removed []types.Resource

	baselineMap := make(map[string]types.Resource)
	currentMap := make(map[string]types.Resource)

	for _, r := range baseline {
		baselineMap[r.ID] = r
	}
	for _, r := range current {
		currentMap[r.ID] = r
	}

	// Find matches
	for _, curr := range current {
		if base, exists := baselineMap[curr.ID]; exists {
			matches = append(matches, ResourceMatch{
				Baseline: base,
				Current:  curr,
			})
		} else {
			added = append(added, curr)
		}
	}

	// Find removed
	for _, base := range baseline {
		if _, exists := currentMap[base.ID]; !exists {
			removed = append(removed, base)
		}
	}

	return matches, added, removed
}

// DefaultClassifier is a simple classifier for non-enterprise builds
type DefaultClassifier struct{}

// ClassifyChange performs basic classification
func (c *DefaultClassifier) ClassifyChange(change Change) (DriftCategory, RiskLevel, float64) {
	// Simple classification logic
	category := DriftCategoryConfig
	severity := RiskLevelMedium
	riskScore := 0.5

	// Basic security classification
	if change.Field == "security_groups" || change.Field == "iam_role" {
		category = DriftCategorySecurity
		severity = RiskLevelHigh
		riskScore = 0.8
	}

	return category, severity, riskScore
}

// CalculateResourceRisk calculates risk for a resource
func (c *DefaultClassifier) CalculateResourceRisk(changes []Change) (RiskLevel, float64) {
	if len(changes) == 0 {
		return RiskLevelLow, 0.0
	}

	totalRisk := 0.0
	for _, change := range changes {
		_, _, risk := c.ClassifyChange(change)
		totalRisk += risk
	}
	avgRisk := totalRisk / float64(len(changes))

	if avgRisk > 0.8 {
		return RiskLevelCritical, avgRisk
	} else if avgRisk > 0.6 {
		return RiskLevelHigh, avgRisk
	} else if avgRisk > 0.4 {
		return RiskLevelMedium, avgRisk
	}
	return RiskLevelLow, avgRisk
}

// CalculateOverallRisk calculates overall risk
func (c *DefaultClassifier) CalculateOverallRisk(summary DriftSummary) (RiskLevel, float64) {
	return RiskLevelMedium, 0.5
}

// SmartComparer is a stub comparer for non-enterprise builds
type SmartComparer struct {
	options DiffOptions
}

// NewSmartComparer creates a new smart comparer (stub)
func NewSmartComparer(options DiffOptions) *SmartComparer {
	return &SmartComparer{options: options}
}

// Compare performs basic comparison
func (c *SmartComparer) Compare(baseline, current types.Resource) ([]Change, error) {
	changes := c.CompareResources(baseline, current)
	return changes, nil
}

// CompareResources performs basic resource comparison
func (c *SmartComparer) CompareResources(baseline, current types.Resource) []Change {
	var changes []Change

	// Simple comparison - just check if configurations are different
	if !compareConfigurations(baseline.Configuration, current.Configuration) {
		changes = append(changes, Change{
			Type:        ChangeTypeModified,
			ResourceID:  current.ID,
			Path:        "configuration",
			Field:       "config",
			OldValue:    baseline.Configuration,
			NewValue:    current.Configuration,
			Description: "Configuration changed",
		})
	}

	return changes
}

// CompareConfiguration compares configurations
func (c *SmartComparer) CompareConfiguration(basePath string, baseline, current map[string]interface{}) []Change {
	return []Change{} // Stub implementation
}

// NewAdvancedClassifier creates a default classifier (stub)
func NewAdvancedClassifier() ChangeClassifier {
	return &DefaultClassifier{}
}

// Helper function to compare configurations
func compareConfigurations(baseline, current map[string]interface{}) bool {
	if len(baseline) != len(current) {
		return false
	}
	for k, v := range baseline {
		if currentV, exists := current[k]; !exists || currentV != v {
			return false
		}
	}
	return true
}
