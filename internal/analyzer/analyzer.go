package analyzer

import (
	"github.com/yairfalse/wgo/pkg/types"
)

// Analyzer provides analysis capabilities for drift reports
type Analyzer interface {
	AnalyzeDrift(report *types.DriftReport) (*types.Analysis, error)
	CalculateRiskScore(changes []types.Change) float64
	CategorizeChanges(changes []types.Change) map[string][]types.Change
}

// StandardAnalyzer implements the Analyzer interface
type StandardAnalyzer struct{}

// NewStandardAnalyzer creates a new StandardAnalyzer
func NewStandardAnalyzer() *StandardAnalyzer {
	return &StandardAnalyzer{}
}

// AnalyzeDrift analyzes a drift report and provides insights
func (a *StandardAnalyzer) AnalyzeDrift(report *types.DriftReport) (*types.Analysis, error) {
	// TODO: Implement actual analysis logic
	analysis := &types.Analysis{
		RiskScore:  a.CalculateRiskScore(report.Changes),
		Categories: a.CategorizeChanges(report.Changes),
		Recommendations: []string{
			"Review security group changes",
			"Verify resource tagging",
			"Check configuration drift",
		},
	}

	return analysis, nil
}

// CalculateRiskScore calculates a risk score based on changes
func (a *StandardAnalyzer) CalculateRiskScore(changes []types.Change) float64 {
	// TODO: Implement actual risk calculation
	if len(changes) == 0 {
		return 0.0
	}

	// Simple placeholder logic
	score := float64(len(changes)) * 0.1
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// CategorizeChanges groups changes by category
func (a *StandardAnalyzer) CategorizeChanges(changes []types.Change) map[string][]types.Change {
	categories := make(map[string][]types.Change)

	for _, change := range changes {
		category := "other"
		if change.Field == "tags" {
			category = "tagging"
		} else if change.Field == "security_groups" {
			category = "security"
		} else if change.Field == "instance_type" {
			category = "sizing"
		}

		categories[category] = append(categories[category], change)
	}

	return categories
}
