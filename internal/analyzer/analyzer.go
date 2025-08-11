package analyzer

import (
	"fmt"

	"github.com/yairfalse/vaino/pkg/types"
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
	// Categorize changes for analysis
	categories := a.CategorizeChanges(report.Changes)

	// Calculate risk score based on changes
	riskScore := a.CalculateRiskScore(report.Changes)

	// Generate contextual recommendations
	recommendations := a.generateRecommendations(categories, riskScore)

	// Identify critical changes that need immediate attention
	criticalChanges := a.identifyCriticalChanges(report.Changes)

	analysis := &types.Analysis{
		RiskScore:       riskScore,
		Categories:      categories,
		Recommendations: recommendations,
		CriticalChanges: criticalChanges,
		Summary:         a.generateSummary(report, categories, riskScore),
	}

	return analysis, nil
}

// CalculateRiskScore calculates a risk score based on changes
func (a *StandardAnalyzer) CalculateRiskScore(changes []types.Change) float64 {
	if len(changes) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, change := range changes {
		var changeScore float64

		// Assign scores based on change type and criticality
		switch change.ChangeType {
		case "deleted":
			changeScore = 0.8 // Deletions are high risk
		case "added":
			changeScore = 0.3 // Additions are medium risk
		case "modified":
			// Score based on field importance
			changeScore = a.getFieldRiskScore(change.Field)
		default:
			changeScore = 0.2
		}

		// Adjust score for resource type criticality
		changeScore *= a.getResourceTypeCriticality(change.ResourceType)

		totalScore += changeScore
	}

	// Normalize score to 0-1 range
	normalizedScore := totalScore / float64(len(changes))
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	}

	return normalizedScore
}

// CategorizeChanges groups changes by category
func (a *StandardAnalyzer) CategorizeChanges(changes []types.Change) map[string][]types.Change {
	categories := make(map[string][]types.Change)

	for _, change := range changes {
		category := a.determineCategory(change)
		categories[category] = append(categories[category], change)
	}

	return categories
}

// determineCategory determines the category for a change
func (a *StandardAnalyzer) determineCategory(change types.Change) string {
	// Map fields to categories
	fieldCategoryMap := map[string]string{
		"tags":               "tagging",
		"security_groups":    "security",
		"security_group_ids": "security",
		"iam_role":           "security",
		"iam_policy":         "security",
		"instance_type":      "sizing",
		"cpu":                "sizing",
		"memory":             "sizing",
		"disk_size":          "storage",
		"volume_size":        "storage",
		"network":            "networking",
		"subnet":             "networking",
		"vpc":                "networking",
		"replicas":           "scaling",
		"min_size":           "scaling",
		"max_size":           "scaling",
		"desired_capacity":   "scaling",
		"image":              "configuration",
		"ami_id":             "configuration",
		"user_data":          "configuration",
		"environment":        "configuration",
	}

	if category, exists := fieldCategoryMap[change.Field]; exists {
		return category
	}

	// Resource type based categorization
	resourceTypeMap := map[string]string{
		"aws_security_group":    "security",
		"aws_iam_role":          "security",
		"aws_iam_policy":        "security",
		"aws_instance":          "compute",
		"aws_autoscaling_group": "scaling",
		"aws_lb":                "networking",
		"aws_db_instance":       "database",
		"aws_s3_bucket":         "storage",
	}

	if category, exists := resourceTypeMap[change.ResourceType]; exists {
		return category
	}

	return "other"
}

// getFieldRiskScore returns a risk score for a specific field
func (a *StandardAnalyzer) getFieldRiskScore(field string) float64 {
	riskScores := map[string]float64{
		"security_groups":    0.9,
		"security_group_ids": 0.9,
		"iam_role":           0.9,
		"iam_policy":         0.9,
		"public_ip":          0.8,
		"encryption":         0.8,
		"instance_type":      0.4,
		"replicas":           0.5,
		"tags":               0.2,
		"description":        0.1,
	}

	if score, exists := riskScores[field]; exists {
		return score
	}
	return 0.3 // Default medium risk
}

// getResourceTypeCriticality returns a criticality multiplier for resource types
func (a *StandardAnalyzer) getResourceTypeCriticality(resourceType string) float64 {
	criticality := map[string]float64{
		"aws_security_group":    1.5,
		"aws_iam_role":          1.5,
		"aws_iam_policy":        1.5,
		"aws_db_instance":       1.3,
		"kubernetes_secret":     1.4,
		"aws_instance":          1.0,
		"aws_s3_bucket":         1.1,
		"aws_autoscaling_group": 0.9,
	}

	if crit, exists := criticality[resourceType]; exists {
		return crit
	}
	return 1.0 // Default criticality
}

// generateRecommendations creates contextual recommendations based on analysis
func (a *StandardAnalyzer) generateRecommendations(categories map[string][]types.Change, riskScore float64) []string {
	recommendations := []string{}

	// High risk recommendations
	if riskScore > 0.7 {
		recommendations = append(recommendations, "CRITICAL: High-risk changes detected. Immediate review required.")
	}

	// Category-based recommendations
	if securityChanges, exists := categories["security"]; exists && len(securityChanges) > 0 {
		recommendations = append(recommendations, "Review security group and IAM changes for compliance")
	}

	if scalingChanges, exists := categories["scaling"]; exists && len(scalingChanges) > 0 {
		recommendations = append(recommendations, "Verify auto-scaling configurations match expected capacity")
	}

	if networkChanges, exists := categories["networking"]; exists && len(networkChanges) > 0 {
		recommendations = append(recommendations, "Check network configuration changes for connectivity impact")
	}

	if storageChanges, exists := categories["storage"]; exists && len(storageChanges) > 0 {
		recommendations = append(recommendations, "Validate storage changes and backup requirements")
	}

	// Add general recommendations if none were added
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Review all changes and update baseline if intentional")
	}

	return recommendations
}

// identifyCriticalChanges finds changes that need immediate attention
func (a *StandardAnalyzer) identifyCriticalChanges(changes []types.Change) []types.Change {
	critical := []types.Change{}

	for _, change := range changes {
		// Deletions are always critical
		if change.ChangeType == "deleted" {
			critical = append(critical, change)
			continue
		}

		// Security-related changes are critical
		if a.determineCategory(change) == "security" {
			critical = append(critical, change)
			continue
		}

		// High-risk field changes
		if a.getFieldRiskScore(change.Field) >= 0.8 {
			critical = append(critical, change)
		}
	}

	return critical
}

// generateSummary creates a human-readable summary of the analysis
func (a *StandardAnalyzer) generateSummary(report *types.DriftReport, categories map[string][]types.Change, riskScore float64) string {
	totalChanges := len(report.Changes)

	if totalChanges == 0 {
		return "No drift detected. Infrastructure matches baseline."
	}

	riskLevel := "Low"
	if riskScore > 0.7 {
		riskLevel = "High"
	} else if riskScore > 0.4 {
		riskLevel = "Medium"
	}

	summary := fmt.Sprintf("Detected %d changes across %d categories. Risk level: %s (%.2f).",
		totalChanges, len(categories), riskLevel, riskScore)

	// Add category breakdown
	if len(categories) > 0 {
		summary += " Changes by category:"
		for category, changes := range categories {
			summary += fmt.Sprintf(" %s(%d)", category, len(changes))
		}
	}

	return summary
}
