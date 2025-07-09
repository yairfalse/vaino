package differ

import (
	"strings"
)

// DefaultClassifier implements change classification and risk assessment
type DefaultClassifier struct {
	rules map[string]ClassificationRule
}

// ClassificationRule defines how to classify a change
type ClassificationRule struct {
	Category    DriftCategory
	Severity    RiskLevel
	RiskScore   float64
	Description string
}

// NewDefaultClassifier creates a classifier with default rules
func NewDefaultClassifier() *DefaultClassifier {
	classifier := &DefaultClassifier{
		rules: make(map[string]ClassificationRule),
	}

	classifier.initializeRules()
	return classifier
}

// initializeRules sets up the default classification rules
func (c *DefaultClassifier) initializeRules() {
	// Security-related changes (CRITICAL)
	c.rules["security_groups"] = ClassificationRule{
		Category:    DriftCategorySecurity,
		Severity:    RiskLevelCritical,
		RiskScore:   0.95,
		Description: "Security group changes affect network access controls",
	}
	c.rules["iam_role"] = ClassificationRule{
		Category:    DriftCategorySecurity,
		Severity:    RiskLevelCritical,
		RiskScore:   0.90,
		Description: "IAM role changes affect access permissions",
	}
	c.rules["vpc_security_group"] = ClassificationRule{
		Category:    DriftCategorySecurity,
		Severity:    RiskLevelHigh,
		RiskScore:   0.85,
		Description: "VPC security group changes affect network isolation",
	}
	c.rules["network_acl"] = ClassificationRule{
		Category:    DriftCategorySecurity,
		Severity:    RiskLevelHigh,
		RiskScore:   0.80,
		Description: "Network ACL changes affect subnet-level security",
	}

	// Network-related changes (HIGH)
	c.rules["subnet_id"] = ClassificationRule{
		Category:    DriftCategoryNetwork,
		Severity:    RiskLevelHigh,
		RiskScore:   0.75,
		Description: "Subnet changes affect network topology",
	}
	c.rules["vpc_id"] = ClassificationRule{
		Category:    DriftCategoryNetwork,
		Severity:    RiskLevelHigh,
		RiskScore:   0.70,
		Description: "VPC changes affect network isolation",
	}
	c.rules["load_balancer"] = ClassificationRule{
		Category:    DriftCategoryNetwork,
		Severity:    RiskLevelMedium,
		RiskScore:   0.60,
		Description: "Load balancer changes affect traffic distribution",
	}

	// Cost-related changes (MEDIUM to HIGH)
	c.rules["instance_type"] = ClassificationRule{
		Category:    DriftCategoryCost,
		Severity:    RiskLevelHigh,
		RiskScore:   0.70,
		Description: "Instance type changes affect performance and cost",
	}
	c.rules["storage_size"] = ClassificationRule{
		Category:    DriftCategoryCost,
		Severity:    RiskLevelMedium,
		RiskScore:   0.50,
		Description: "Storage size changes affect cost",
	}
	c.rules["instance_count"] = ClassificationRule{
		Category:    DriftCategoryCost,
		Severity:    RiskLevelMedium,
		RiskScore:   0.55,
		Description: "Instance count changes affect cost and capacity",
	}

	// Compute-related changes (MEDIUM)
	c.rules["cpu"] = ClassificationRule{
		Category:    DriftCategoryCompute,
		Severity:    RiskLevelMedium,
		RiskScore:   0.60,
		Description: "CPU configuration changes affect performance",
	}
	c.rules["memory"] = ClassificationRule{
		Category:    DriftCategoryCompute,
		Severity:    RiskLevelMedium,
		RiskScore:   0.55,
		Description: "Memory configuration changes affect performance",
	}
	c.rules["replicas"] = ClassificationRule{
		Category:    DriftCategoryCompute,
		Severity:    RiskLevelMedium,
		RiskScore:   0.50,
		Description: "Replica count changes affect availability and resource usage",
	}

	// Storage-related changes (MEDIUM)
	c.rules["volume_size"] = ClassificationRule{
		Category:    DriftCategoryStorage,
		Severity:    RiskLevelMedium,
		RiskScore:   0.45,
		Description: "Volume size changes affect storage capacity",
	}
	c.rules["storage_type"] = ClassificationRule{
		Category:    DriftCategoryStorage,
		Severity:    RiskLevelMedium,
		RiskScore:   0.50,
		Description: "Storage type changes affect performance and cost",
	}

	// Configuration changes (LOW to MEDIUM)
	c.rules["tags"] = ClassificationRule{
		Category:    DriftCategoryConfig,
		Severity:    RiskLevelLow,
		RiskScore:   0.10,
		Description: "Tag changes are usually informational",
	}
	c.rules["name"] = ClassificationRule{
		Category:    DriftCategoryConfig,
		Severity:    RiskLevelLow,
		RiskScore:   0.15,
		Description: "Name changes are cosmetic but may affect identification",
	}
	c.rules["description"] = ClassificationRule{
		Category:    DriftCategoryConfig,
		Severity:    RiskLevelLow,
		RiskScore:   0.05,
		Description: "Description changes are informational",
	}

	// State changes (HIGH)
	c.rules["existence"] = ClassificationRule{
		Category:    DriftCategoryState,
		Severity:    RiskLevelHigh,
		RiskScore:   0.80,
		Description: "Resource creation or deletion affects infrastructure state",
	}
}

// ClassifyChange determines the category and risk level of a change
func (c *DefaultClassifier) ClassifyChange(change Change) (DriftCategory, RiskLevel, float64) {
	// Try to find a specific rule for this field/path
	if rule, exists := c.findMatchingRule(change); exists {
		return rule.Category, rule.Severity, rule.RiskScore
	}

	// Apply heuristic classification based on change type and content
	return c.heuristicClassification(change)
}

// findMatchingRule finds the best matching rule for a change
func (c *DefaultClassifier) findMatchingRule(change Change) (ClassificationRule, bool) {
	// Check for exact field matches
	if rule, exists := c.rules[change.Field]; exists {
		return rule, true
	}

	// Check for path pattern matches
	lowerPath := strings.ToLower(change.Path)
	lowerField := strings.ToLower(change.Field)

	for pattern, rule := range c.rules {
		if strings.Contains(lowerPath, pattern) || strings.Contains(lowerField, pattern) {
			return rule, true
		}
	}

	return ClassificationRule{}, false
}

// heuristicClassification applies heuristic rules when no specific rule matches
func (c *DefaultClassifier) heuristicClassification(change Change) (DriftCategory, RiskLevel, float64) {
	lowerPath := strings.ToLower(change.Path)
	lowerField := strings.ToLower(change.Field)

	// Security-related heuristics
	securityKeywords := []string{"security", "auth", "iam", "role", "policy", "permission", "access", "key", "secret", "password", "cert", "ssl", "tls"}
	for _, keyword := range securityKeywords {
		if strings.Contains(lowerPath, keyword) || strings.Contains(lowerField, keyword) {
			return DriftCategorySecurity, RiskLevelHigh, 0.75
		}
	}

	// Network-related heuristics
	networkKeywords := []string{"network", "subnet", "vpc", "route", "gateway", "endpoint", "dns", "ip", "cidr", "port"}
	for _, keyword := range networkKeywords {
		if strings.Contains(lowerPath, keyword) || strings.Contains(lowerField, keyword) {
			return DriftCategoryNetwork, RiskLevelMedium, 0.55
		}
	}

	// Cost-related heuristics
	costKeywords := []string{"size", "type", "tier", "class", "capacity", "count", "scale", "billing"}
	for _, keyword := range costKeywords {
		if strings.Contains(lowerPath, keyword) || strings.Contains(lowerField, keyword) {
			return DriftCategoryCost, RiskLevelMedium, 0.50
		}
	}

	// Storage-related heuristics
	storageKeywords := []string{"storage", "volume", "disk", "backup", "snapshot", "archive"}
	for _, keyword := range storageKeywords {
		if strings.Contains(lowerPath, keyword) || strings.Contains(lowerField, keyword) {
			return DriftCategoryStorage, RiskLevelMedium, 0.45
		}
	}

	// Compute-related heuristics
	computeKeywords := []string{"cpu", "memory", "compute", "instance", "container", "process", "thread"}
	for _, keyword := range computeKeywords {
		if strings.Contains(lowerPath, keyword) || strings.Contains(lowerField, keyword) {
			return DriftCategoryCompute, RiskLevelMedium, 0.50
		}
	}

	// State-related heuristics
	if change.Type == ChangeTypeAdded || change.Type == ChangeTypeRemoved {
		return DriftCategoryState, RiskLevelHigh, 0.70
	}

	// Default classification
	return DriftCategoryConfig, RiskLevelLow, 0.20
}

// CalculateResourceRisk calculates the overall risk for a resource based on its changes
func (c *DefaultClassifier) CalculateResourceRisk(changes []Change) (RiskLevel, float64) {
	if len(changes) == 0 {
		return RiskLevelLow, 0.0
	}

	var totalRiskScore float64
	maxRiskScore := 0.0
	criticalChanges := 0
	highChanges := 0
	mediumChanges := 0

	for _, change := range changes {
		_, severity, riskScore := c.ClassifyChange(change)

		totalRiskScore += riskScore
		if riskScore > maxRiskScore {
			maxRiskScore = riskScore
		}

		switch severity {
		case RiskLevelCritical:
			criticalChanges++
		case RiskLevelHigh:
			highChanges++
		case RiskLevelMedium:
			mediumChanges++
		}
	}

	// Calculate weighted average with emphasis on highest severity
	avgRiskScore := totalRiskScore / float64(len(changes))
	weightedScore := (avgRiskScore * 0.6) + (maxRiskScore * 0.4)

	// Determine severity based on score and change counts
	if criticalChanges > 0 || weightedScore >= 0.80 {
		return RiskLevelCritical, weightedScore
	}
	if highChanges > 0 || weightedScore >= 0.60 {
		return RiskLevelHigh, weightedScore
	}
	if mediumChanges > 0 || weightedScore >= 0.30 {
		return RiskLevelMedium, weightedScore
	}

	return RiskLevelLow, weightedScore
}

// CalculateOverallRisk calculates the overall risk for the entire drift report
func (c *DefaultClassifier) CalculateOverallRisk(summary DriftSummary) (RiskLevel, float64) {
	if summary.TotalResources == 0 {
		return RiskLevelLow, 0.0
	}

	// Calculate risk based on severity distribution
	criticalCount := summary.ChangesBySeverity[RiskLevelCritical]
	highCount := summary.ChangesBySeverity[RiskLevelHigh]
	mediumCount := summary.ChangesBySeverity[RiskLevelMedium]
	lowCount := summary.ChangesBySeverity[RiskLevelLow]

	totalChanges := criticalCount + highCount + mediumCount + lowCount
	if totalChanges == 0 {
		return RiskLevelLow, 0.0
	}

	// Weighted risk score
	riskScore := (float64(criticalCount)*0.90 + float64(highCount)*0.70 + float64(mediumCount)*0.40 + float64(lowCount)*0.10) / float64(totalChanges)

	// Factor in the proportion of changed resources
	changeRatio := float64(summary.ChangedResources) / float64(summary.TotalResources)
	riskScore *= (0.5 + changeRatio*0.5) // Scale by change ratio

	// Factor in state changes (added/removed resources)
	stateChangeRatio := float64(summary.AddedResources+summary.RemovedResources) / float64(summary.TotalResources)
	riskScore += stateChangeRatio * 0.2 // Add risk for state changes

	// Determine overall severity
	if criticalCount > 0 || riskScore >= 0.75 {
		return RiskLevelCritical, riskScore
	}
	if highCount > 0 || riskScore >= 0.55 {
		return RiskLevelHigh, riskScore
	}
	if mediumCount > 0 || riskScore >= 0.25 {
		return RiskLevelMedium, riskScore
	}

	return RiskLevelLow, riskScore
}

// AdvancedClassifier provides more sophisticated classification with machine learning concepts
type AdvancedClassifier struct {
	baseClassifier *DefaultClassifier
	contextRules   map[string]ContextRule
	patterns       []PatternRule
}

// ContextRule provides context-sensitive classification
type ContextRule struct {
	ResourceType string
	Provider     string
	Modifier     func(category DriftCategory, severity RiskLevel, score float64) (DriftCategory, RiskLevel, float64)
}

// PatternRule detects patterns across multiple changes
type PatternRule struct {
	Pattern     string
	Description string
	Modifier    func(changes []Change) []Change
}

// NewAdvancedClassifier creates an advanced classifier with context awareness
func NewAdvancedClassifier() *AdvancedClassifier {
	return &AdvancedClassifier{
		baseClassifier: NewDefaultClassifier(),
		contextRules:   make(map[string]ContextRule),
		patterns:       make([]PatternRule, 0),
	}
}

// ClassifyChange uses context-aware classification
func (c *AdvancedClassifier) ClassifyChange(change Change) (DriftCategory, RiskLevel, float64) {
	// Start with base classification
	category, severity, score := c.baseClassifier.ClassifyChange(change)

	// Apply context rules if available
	// This would be enhanced with actual resource context

	return category, severity, score
}

// CalculateResourceRisk uses the base classifier
func (c *AdvancedClassifier) CalculateResourceRisk(changes []Change) (RiskLevel, float64) {
	return c.baseClassifier.CalculateResourceRisk(changes)
}

// CalculateOverallRisk uses the base classifier
func (c *AdvancedClassifier) CalculateOverallRisk(summary DriftSummary) (RiskLevel, float64) {
	return c.baseClassifier.CalculateOverallRisk(summary)
}

// AddContextRule adds a context-sensitive rule
func (c *AdvancedClassifier) AddContextRule(key string, rule ContextRule) {
	c.contextRules[key] = rule
}

// AddPatternRule adds a pattern detection rule
func (c *AdvancedClassifier) AddPatternRule(rule PatternRule) {
	c.patterns = append(c.patterns, rule)
}
