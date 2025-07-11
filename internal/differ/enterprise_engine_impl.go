//go:build enterprise
// +build enterprise

package differ

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// processMatchedResource processes changes for a matched resource pair
func (e *EnterpriseDifferEngine) processMatchedResource(match ResourceMatch) processingResult {
	// Check cache first if enabled
	if e.options.EnableCaching && e.cache != nil {
		cacheKey := e.generateCacheKey(match.Baseline, match.Current)
		if cached, found := e.cache.Get(cacheKey); found {
			atomic.AddInt64(&e.metrics.CacheHits, 1)
			return cached.(processingResult)
		}
		atomic.AddInt64(&e.metrics.CacheMisses, 1)
	}

	// Perform deep comparison
	changes, err := e.comparer.Compare(match.Baseline, match.Current)
	if err != nil {
		return processingResult{err: err}
	}

	if len(changes) == 0 {
		return processingResult{} // No changes
	}

	// Classify and score each change
	var highestSeverity DriftSeverity = SeverityLow
	var categories []DriftCategory
	categoryMap := make(map[DriftCategory]bool)
	totalRiskScore := 0.0

	for i := range changes {
		category, severity, riskScore := e.classifier.ClassifyChange(changes[i])
		changes[i].Category = category
		changes[i].Severity = severity

		if !categoryMap[category] {
			categoryMap[category] = true
			categories = append(categories, category)
		}

		if severity > highestSeverity {
			highestSeverity = severity
		}

		totalRiskScore += riskScore
	}

	// Build resource diff
	resourceDiff := ResourceDiff{
		ResourceID:   match.Current.ID,
		ResourceType: match.Current.Type,
		Provider:     match.Current.Provider,
		DriftType:    ChangeTypeModified,
		Changes:      changes,
		Severity:     highestSeverity,
		Categories:   categories,
		RiskScore:    totalRiskScore / float64(len(changes)),
		Description:  e.generateChangeDescription(changes),
		Metadata: map[string]interface{}{
			"change_count":    len(changes),
			"comparison_time": time.Now(),
			"resource_name":   match.Current.Name,
			"resource_region": match.Current.Region,
			"resource_tags":   match.Current.Tags,
		},
	}

	// Apply risk assessment if enabled
	if e.options.EnableRiskScoring && e.riskAssessor != nil {
		resourceDiff.RiskScore = e.riskAssessor.AssessResourceRisk(resourceDiff, match.Baseline, match.Current)
	}

	result := processingResult{
		resourceDiff: &resourceDiff,
		changes:      changes,
	}

	// Cache the result if enabled
	if e.options.EnableCaching && e.cache != nil {
		cacheKey := e.generateCacheKey(match.Baseline, match.Current)
		e.cache.Set(cacheKey, result)
	}

	// Update metrics for high-risk changes
	if resourceDiff.RiskScore > 0.7 || highestSeverity >= SeverityHigh {
		atomic.AddInt64(&e.metrics.HighRiskChanges, 1)
	}

	return result
}

// processAddedResource processes a newly added resource
func (e *EnterpriseDifferEngine) processAddedResource(resource types.Resource) processingResult {
	change := Change{
		Type:        ChangeTypeAdded,
		ResourceID:  resource.ID,
		Path:        "resource",
		Field:       "existence",
		OldValue:    nil,
		NewValue:    resource,
		Description: fmt.Sprintf("New %s resource '%s' added", resource.Type, resource.Name),
	}

	category, severity, riskScore := e.classifier.ClassifyChange(change)
	change.Category = category
	change.Severity = severity

	// Assess risk for new resources
	if e.options.EnableRiskScoring && e.riskAssessor != nil {
		riskScore = e.riskAssessor.AssessNewResourceRisk(resource)
	}

	resourceDiff := ResourceDiff{
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
		Provider:     resource.Provider,
		DriftType:    ChangeTypeAdded,
		Changes:      []Change{change},
		Severity:     severity,
		Categories:   []DriftCategory{category},
		RiskScore:    riskScore,
		Description:  change.Description,
		Metadata: map[string]interface{}{
			"resource_name":   resource.Name,
			"resource_region": resource.Region,
			"resource_tags":   resource.Tags,
			"created_at":      time.Now(),
		},
	}

	return processingResult{
		resourceDiff: &resourceDiff,
		changes:      []Change{change},
	}
}

// processRemovedResource processes a removed resource
func (e *EnterpriseDifferEngine) processRemovedResource(resource types.Resource) processingResult {
	change := Change{
		Type:        ChangeTypeRemoved,
		ResourceID:  resource.ID,
		Path:        "resource",
		Field:       "existence",
		OldValue:    resource,
		NewValue:    nil,
		Description: fmt.Sprintf("%s resource '%s' was removed", resource.Type, resource.Name),
	}

	category, severity, riskScore := e.classifier.ClassifyChange(change)
	change.Category = category
	change.Severity = severity

	// Removed resources often have higher risk
	if e.options.EnableRiskScoring && e.riskAssessor != nil {
		riskScore = e.riskAssessor.AssessRemovedResourceRisk(resource)
	}

	resourceDiff := ResourceDiff{
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
		Provider:     resource.Provider,
		DriftType:    ChangeTypeRemoved,
		Changes:      []Change{change},
		Severity:     severity,
		Categories:   []DriftCategory{category},
		RiskScore:    riskScore,
		Description:  change.Description,
		Metadata: map[string]interface{}{
			"resource_name":   resource.Name,
			"resource_region": resource.Region,
			"resource_tags":   resource.Tags,
			"removed_at":      time.Now(),
		},
	}

	// Mark as high risk if critical resource
	if isCriticalResource(resource) {
		resourceDiff.Severity = SeverityCritical
		atomic.AddInt64(&e.metrics.HighRiskChanges, 1)
	}

	return processingResult{
		resourceDiff: &resourceDiff,
		changes:      []Change{change},
	}
}

// sequentialProcessChanges processes changes sequentially (for smaller workloads)
func (e *EnterpriseDifferEngine) sequentialProcessChanges(ctx context.Context, matches []ResourceMatch, added, removed []types.Resource) ([]ResourceDiff, []Change) {
	var resourceChanges []ResourceDiff
	var allChanges []Change

	// Process matches
	for _, match := range matches {
		result := e.processMatchedResource(match)
		if result.err == nil && result.resourceDiff != nil {
			resourceChanges = append(resourceChanges, *result.resourceDiff)
			allChanges = append(allChanges, result.changes...)
		}
	}

	// Process added
	for _, resource := range added {
		result := e.processAddedResource(resource)
		if result.err == nil && result.resourceDiff != nil {
			resourceChanges = append(resourceChanges, *result.resourceDiff)
			allChanges = append(allChanges, result.changes...)
		}
	}

	// Process removed
	for _, resource := range removed {
		result := e.processRemovedResource(resource)
		if result.err == nil && result.resourceDiff != nil {
			resourceChanges = append(resourceChanges, *result.resourceDiff)
			allChanges = append(allChanges, result.changes...)
		}
	}

	return resourceChanges, allChanges
}

// correlateChanges applies correlation analysis to identify related changes
func (e *EnterpriseDifferEngine) correlateChanges(changes []ResourceDiff) []ResourceDiff {
	if e.correlator == nil {
		return changes
	}

	// Group changes by correlation patterns
	correlations := e.correlator.Correlate(changes)

	// Enhance resource diffs with correlation information
	for i := range changes {
		for _, correlation := range correlations {
			if containsResourceID(correlation.ResourceIDs, changes[i].ResourceID) {
				changes[i].Metadata["correlation_id"] = correlation.ID
				changes[i].Metadata["correlation_confidence"] = correlation.Confidence
				changes[i].Metadata["correlation_pattern"] = correlation.Pattern

				// Adjust risk score based on correlation
				if correlation.Pattern == "cascading_failure" || correlation.Pattern == "security_breach" {
					changes[i].RiskScore *= 1.5 // Increase risk for correlated security/failure patterns
					if changes[i].RiskScore > 1.0 {
						changes[i].RiskScore = 1.0
					}
				}
			}
		}
	}

	return changes
}

// buildEnterpriseReport creates a comprehensive drift report with enterprise features
func (e *EnterpriseDifferEngine) buildEnterpriseReport(baseline, current *types.Snapshot, resourceChanges []ResourceDiff, allChanges []Change) *DriftReport {
	// Calculate summary statistics
	summary := e.calculateSummary(resourceChanges, allChanges)

	// Build compliance report if enabled
	var complianceReport *ComplianceReport
	if e.options.EnableCompliance {
		complianceReport = e.buildComplianceReport(resourceChanges, e.options.ComplianceRules)
		e.metrics.ComplianceIssues = int64(len(complianceReport.Violations))
	}

	report := &DriftReport{
		ID:              generateReportID(),
		Timestamp:       time.Now(),
		BaselineID:      baseline.ID,
		CurrentID:       current.ID,
		Summary:         summary,
		ResourceChanges: resourceChanges,
		AllChanges:      allChanges,
		Metadata: map[string]interface{}{
			"baseline_provider":    baseline.Provider,
			"current_provider":     current.Provider,
			"baseline_timestamp":   baseline.Timestamp,
			"current_timestamp":    current.Timestamp,
			"analysis_duration_ms": e.metrics.ProcessingTimeMs,
			"parallel_processing":  e.metrics.WorkersUsed > 1,
			"correlation_enabled":  e.options.EnableCorrelation,
			"risk_scoring_enabled": e.options.EnableRiskScoring,
			"compliance_enabled":   e.options.EnableCompliance,
			"engine_version":       "2.0.0-enterprise",
		},
	}

	// Add compliance report if available
	if complianceReport != nil {
		report.ComplianceReport = complianceReport
	}

	// Add executive summary for C-level reporting
	if e.options.OutputFormat == "executive" {
		report.ExecutiveSummary = e.generateExecutiveSummary(summary, resourceChanges)
	}

	return report
}

// buildResourceIndexes creates indexes for fast lookups
func (e *EnterpriseDifferEngine) buildResourceIndexes(baseline, current *types.Snapshot) {
	if e.resourceIndex == nil {
		return
	}

	// Index baseline resources
	for _, resource := range baseline.Resources {
		e.resourceIndex.AddBaseline(resource)
	}

	// Index current resources
	for _, resource := range current.Resources {
		e.resourceIndex.AddCurrent(resource)
	}

	// Build secondary indexes
	e.resourceIndex.BuildSecondaryIndexes()
}

// generateCacheKey creates a cache key for a resource comparison
func (e *EnterpriseDifferEngine) generateCacheKey(baseline, current types.Resource) string {
	h := md5.New()
	h.Write([]byte(baseline.ID))
	h.Write([]byte(current.ID))
	h.Write([]byte(fmt.Sprintf("%v", baseline.Configuration)))
	h.Write([]byte(fmt.Sprintf("%v", current.Configuration)))
	return hex.EncodeToString(h.Sum(nil))
}

// generateChangeDescription creates a human-readable description of changes
func (e *EnterpriseDifferEngine) generateChangeDescription(changes []Change) string {
	if len(changes) == 0 {
		return "No changes detected"
	}

	// Group changes by category
	categoryCount := make(map[DriftCategory]int)
	for _, change := range changes {
		categoryCount[change.Category]++
	}

	var parts []string
	for category, count := range categoryCount {
		parts = append(parts, fmt.Sprintf("%d %s changes", count, category))
	}

	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// calculateSummary generates comprehensive summary statistics
func (e *EnterpriseDifferEngine) calculateSummary(resourceChanges []ResourceDiff, allChanges []Change) DriftSummary {
	summary := DriftSummary{
		TotalResources:   int(e.metrics.TotalResources / 2), // Divide by 2 as we count both snapshots
		ChangedResources: len(resourceChanges),
		TotalChanges:     len(allChanges),
		RiskAssessment:   "Low",
		ComplianceStatus: "Compliant",
		Timestamp:        time.Now(),
	}

	// Count by change type
	for _, change := range resourceChanges {
		switch change.DriftType {
		case ChangeTypeAdded:
			summary.AddedResources++
		case ChangeTypeRemoved:
			summary.RemovedResources++
		case ChangeTypeModified:
			summary.ModifiedResources++
		}
	}

	// Calculate risk metrics
	var totalRisk float64
	severityCount := make(map[DriftSeverity]int)

	for _, change := range resourceChanges {
		totalRisk += change.RiskScore
		severityCount[change.Severity]++
	}

	summary.CriticalChanges = severityCount[SeverityCritical]
	summary.HighRiskChanges = severityCount[SeverityHigh]
	summary.MediumRiskChanges = severityCount[SeverityMedium]
	summary.LowRiskChanges = severityCount[SeverityLow]

	// Determine overall risk assessment
	avgRisk := totalRisk / float64(len(resourceChanges))
	if summary.CriticalChanges > 0 || avgRisk > 0.8 {
		summary.RiskAssessment = "Critical"
	} else if summary.HighRiskChanges > 5 || avgRisk > 0.6 {
		summary.RiskAssessment = "High"
	} else if summary.MediumRiskChanges > 10 || avgRisk > 0.4 {
		summary.RiskAssessment = "Medium"
	}

	// Add enterprise metrics
	summary.Metadata = map[string]interface{}{
		"average_risk_score":    avgRisk,
		"processing_time_ms":    e.metrics.ProcessingTimeMs,
		"cache_hit_rate":        float64(e.metrics.CacheHits) / float64(e.metrics.CacheHits+e.metrics.CacheMisses),
		"parallel_workers_used": e.metrics.WorkersUsed,
		"compliance_violations": e.metrics.ComplianceIssues,
	}

	return summary
}

// generateExecutiveSummary creates a C-level executive summary
func (e *EnterpriseDifferEngine) generateExecutiveSummary(summary DriftSummary, changes []ResourceDiff) *ExecutiveSummary {
	exec := &ExecutiveSummary{
		OverallRisk:      summary.RiskAssessment,
		ComplianceStatus: summary.ComplianceStatus,
		KeyFindings:      []string{},
		Recommendations:  []string{},
		Metrics: map[string]interface{}{
			"total_resources":   summary.TotalResources,
			"changed_resources": summary.ChangedResources,
			"critical_changes":  summary.CriticalChanges,
			"high_risk_changes": summary.HighRiskChanges,
			"compliance_issues": e.metrics.ComplianceIssues,
		},
	}

	// Identify key findings
	criticalResources := filterCriticalChanges(changes)
	if len(criticalResources) > 0 {
		exec.KeyFindings = append(exec.KeyFindings,
			fmt.Sprintf("Found %d critical infrastructure changes requiring immediate attention", len(criticalResources)))
	}

	securityChanges := filterSecurityChanges(changes)
	if len(securityChanges) > 0 {
		exec.KeyFindings = append(exec.KeyFindings,
			fmt.Sprintf("Detected %d security-related configuration changes", len(securityChanges)))
	}

	// Generate recommendations
	if summary.RiskAssessment == "Critical" || summary.RiskAssessment == "High" {
		exec.Recommendations = append(exec.Recommendations,
			"Immediate review and approval of critical changes recommended",
			"Conduct security audit of modified resources",
			"Update baseline after review to prevent drift accumulation")
	}

	if e.metrics.ComplianceIssues > 0 {
		exec.Recommendations = append(exec.Recommendations,
			fmt.Sprintf("Address %d compliance violations to maintain regulatory standards", e.metrics.ComplianceIssues))
	}

	return exec
}

// Helper functions

func isCriticalResource(resource types.Resource) bool {
	criticalTypes := map[string]bool{
		"aws_iam_role":            true,
		"aws_security_group":      true,
		"aws_kms_key":             true,
		"kubernetes_secret":       true,
		"gcp_service_account":     true,
		"aws_rds_cluster":         true,
		"aws_elasticache_cluster": true,
	}
	return criticalTypes[resource.Type]
}

func containsResourceID(ids []string, id string) bool {
	for _, rid := range ids {
		if rid == id {
			return true
		}
	}
	return false
}

func filterCriticalChanges(changes []ResourceDiff) []ResourceDiff {
	var critical []ResourceDiff
	for _, change := range changes {
		if change.Severity == SeverityCritical {
			critical = append(critical, change)
		}
	}
	return critical
}

func filterSecurityChanges(changes []ResourceDiff) []ResourceDiff {
	var security []ResourceDiff
	for _, change := range changes {
		for _, category := range change.Categories {
			if category == CategorySecurity {
				security = append(security, change)
				break
			}
		}
	}
	return security
}

func generateReportID() string {
	h := md5.New()
	h.Write([]byte(time.Now().String()))
	return "drift-" + hex.EncodeToString(h.Sum(nil))[:8]
}

// GetChangeStream returns the change stream channel for real-time monitoring
func (e *EnterpriseDifferEngine) GetChangeStream() <-chan StreamedChange {
	return e.changeStream
}

// GetMetrics returns the current diff metrics
func (e *EnterpriseDifferEngine) GetMetrics() *DiffMetrics {
	return e.metrics
}
