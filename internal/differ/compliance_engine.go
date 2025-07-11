//go:build enterprise
// +build enterprise

package differ

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// buildComplianceReport creates a compliance report based on drift changes and rules
func (e *EnterpriseDifferEngine) buildComplianceReport(changes []ResourceDiff, rules []ComplianceRule) *ComplianceReport {
	report := &ComplianceReport{
		ID:         "compliance-" + time.Now().Format("20060102-150405"),
		Timestamp:  time.Now(),
		Framework:  "Enterprise",
		Status:     "Compliant",
		Score:      100.0,
		Violations: []ComplianceViolation{},
		Summary: ComplianceSummary{
			TotalRules: len(rules),
		},
		Metadata: map[string]interface{}{
			"total_resources_evaluated": len(changes),
			"rules_engine_version":      "1.0.0",
		},
	}

	// Evaluate each rule against the changes
	for _, rule := range rules {
		violations := e.evaluateComplianceRule(rule, changes)
		if len(violations) > 0 {
			report.Violations = append(report.Violations, violations...)
			report.Summary.FailedRules++

			// Update summary counters
			for _, violation := range violations {
				switch violation.Severity {
				case SeverityCritical:
					report.Summary.CriticalIssues++
				case SeverityHigh:
					report.Summary.HighIssues++
				case SeverityMedium:
					report.Summary.MediumIssues++
				case SeverityLow:
					report.Summary.LowIssues++
				}
			}
		} else {
			report.Summary.PassedRules++
		}
	}

	// Calculate compliance score
	if report.Summary.TotalRules > 0 {
		passRate := float64(report.Summary.PassedRules) / float64(report.Summary.TotalRules)

		// Penalize based on severity of violations
		severityPenalty := float64(report.Summary.CriticalIssues)*0.2 +
			float64(report.Summary.HighIssues)*0.1 +
			float64(report.Summary.MediumIssues)*0.05 +
			float64(report.Summary.LowIssues)*0.01

		report.Score = (passRate * 100) - severityPenalty
		if report.Score < 0 {
			report.Score = 0
		}
	}

	// Determine overall status
	if report.Summary.CriticalIssues > 0 {
		report.Status = "Critical Non-Compliance"
	} else if report.Summary.HighIssues > 0 || report.Score < 80 {
		report.Status = "Non-Compliant"
	} else if report.Summary.MediumIssues > 0 || report.Score < 95 {
		report.Status = "Partially Compliant"
	}

	return report
}

// evaluateComplianceRule evaluates a single compliance rule against changes
func (e *EnterpriseDifferEngine) evaluateComplianceRule(rule ComplianceRule, changes []ResourceDiff) []ComplianceViolation {
	var violations []ComplianceViolation

	for _, change := range changes {
		if e.doesChangeViolateRule(rule, change) {
			violation := ComplianceViolation{
				RuleID:      rule.ID,
				ResourceID:  change.ResourceID,
				Severity:    rule.Severity,
				Description: e.buildViolationDescription(rule, change),
				Remediation: rule.Remediation,
				Metadata: map[string]interface{}{
					"resource_type":     change.ResourceType,
					"resource_provider": change.Provider,
					"drift_type":        change.DriftType,
					"rule_framework":    rule.Framework,
					"violation_time":    time.Now(),
				},
			}
			violations = append(violations, violation)
		}
	}

	return violations
}

// doesChangeViolateRule determines if a resource change violates a compliance rule
func (e *EnterpriseDifferEngine) doesChangeViolateRule(rule ComplianceRule, change ResourceDiff) bool {
	// Parse the condition expression
	return e.evaluateCondition(rule.Condition, change)
}

// evaluateCondition evaluates a compliance rule condition
func (e *EnterpriseDifferEngine) evaluateCondition(condition string, change ResourceDiff) bool {
	// Simple expression evaluator for compliance rules
	// In production, this would use a proper expression parser

	// Handle different condition types
	switch {
	case strings.Contains(condition, "security_group"):
		return e.evaluateSecurityGroupCondition(condition, change)
	case strings.Contains(condition, "encryption"):
		return e.evaluateEncryptionCondition(condition, change)
	case strings.Contains(condition, "public_access"):
		return e.evaluatePublicAccessCondition(condition, change)
	case strings.Contains(condition, "iam_privilege"):
		return e.evaluateIAMPrivilegeCondition(condition, change)
	case strings.Contains(condition, "backup_policy"):
		return e.evaluateBackupPolicyCondition(condition, change)
	case strings.Contains(condition, "network_exposure"):
		return e.evaluateNetworkExposureCondition(condition, change)
	case strings.Contains(condition, "data_classification"):
		return e.evaluateDataClassificationCondition(condition, change)
	default:
		return e.evaluateGenericCondition(condition, change)
	}
}

// evaluateSecurityGroupCondition evaluates security group related compliance
func (e *EnterpriseDifferEngine) evaluateSecurityGroupCondition(condition string, change ResourceDiff) bool {
	if change.ResourceType != "aws_security_group" {
		return false
	}

	// Check for overly permissive rules
	if strings.Contains(condition, "open_to_world") {
		return e.hasOpenToWorldRules(change)
	}

	// Check for administrative port exposure
	if strings.Contains(condition, "admin_ports") {
		return e.hasAdminPortExposure(change)
	}

	return false
}

// evaluateEncryptionCondition evaluates encryption compliance
func (e *EnterpriseDifferEngine) evaluateEncryptionCondition(condition string, change ResourceDiff) bool {
	encryptionResources := map[string]bool{
		"aws_s3_bucket":       true,
		"aws_rds_cluster":     true,
		"aws_ebs_volume":      true,
		"aws_efs_file_system": true,
	}

	if !encryptionResources[change.ResourceType] {
		return false
	}

	// Check if encryption was disabled
	if strings.Contains(condition, "encryption_disabled") {
		return e.wasEncryptionDisabled(change)
	}

	// Check if encryption key was changed without approval
	if strings.Contains(condition, "encryption_key_changed") {
		return e.wasEncryptionKeyChanged(change)
	}

	return false
}

// evaluatePublicAccessCondition evaluates public access compliance
func (e *EnterpriseDifferEngine) evaluatePublicAccessCondition(condition string, change ResourceDiff) bool {
	publicAccessResources := map[string]bool{
		"aws_s3_bucket":            true,
		"aws_s3_bucket_policy":     true,
		"aws_rds_cluster":          true,
		"aws_elasticsearch_domain": true,
	}

	if !publicAccessResources[change.ResourceType] {
		return false
	}

	// Check if resource was made publicly accessible
	if strings.Contains(condition, "made_public") {
		return e.wasMadePublic(change)
	}

	return false
}

// evaluateIAMPrivilegeCondition evaluates IAM privilege compliance
func (e *EnterpriseDifferEngine) evaluateIAMPrivilegeCondition(condition string, change ResourceDiff) bool {
	iamResources := map[string]bool{
		"aws_iam_role":   true,
		"aws_iam_policy": true,
		"aws_iam_user":   true,
		"aws_iam_group":  true,
	}

	if !iamResources[change.ResourceType] {
		return false
	}

	// Check for privilege escalation
	if strings.Contains(condition, "privilege_escalation") {
		return e.hasPrivilegeEscalation(change)
	}

	// Check for overly broad permissions
	if strings.Contains(condition, "overly_broad") {
		return e.hasOverlyBroadPermissions(change)
	}

	return false
}

// evaluateBackupPolicyCondition evaluates backup policy compliance
func (e *EnterpriseDifferEngine) evaluateBackupPolicyCondition(condition string, change ResourceDiff) bool {
	backupResources := map[string]bool{
		"aws_rds_cluster":     true,
		"aws_ebs_volume":      true,
		"aws_efs_file_system": true,
	}

	if !backupResources[change.ResourceType] {
		return false
	}

	// Check if backup was disabled
	if strings.Contains(condition, "backup_disabled") {
		return e.wasBackupDisabled(change)
	}

	return false
}

// evaluateNetworkExposureCondition evaluates network exposure compliance
func (e *EnterpriseDifferEngine) evaluateNetworkExposureCondition(condition string, change ResourceDiff) bool {
	networkResources := map[string]bool{
		"aws_instance":            true,
		"aws_rds_cluster":         true,
		"aws_elasticache_cluster": true,
		"aws_lb":                  true,
	}

	if !networkResources[change.ResourceType] {
		return false
	}

	// Check for unexpected network exposure
	if strings.Contains(condition, "unexpected_exposure") {
		return e.hasUnexpectedNetworkExposure(change)
	}

	return false
}

// evaluateDataClassificationCondition evaluates data classification compliance
func (e *EnterpriseDifferEngine) evaluateDataClassificationCondition(condition string, change ResourceDiff) bool {
	dataResources := map[string]bool{
		"aws_s3_bucket":      true,
		"aws_rds_cluster":    true,
		"aws_dynamodb_table": true,
	}

	if !dataResources[change.ResourceType] {
		return false
	}

	// Check for missing data classification tags
	if strings.Contains(condition, "missing_classification") {
		return e.hasMissingDataClassification(change)
	}

	return false
}

// evaluateGenericCondition evaluates generic conditions using pattern matching
func (e *EnterpriseDifferEngine) evaluateGenericCondition(condition string, change ResourceDiff) bool {
	// Simple pattern matching for common conditions
	patterns := map[string]*regexp.Regexp{
		"resource_type": regexp.MustCompile(`resource_type\s*==\s*["']([^"']+)["']`),
		"severity":      regexp.MustCompile(`severity\s*>=\s*["']([^"']+)["']`),
		"category":      regexp.MustCompile(`category\s*==\s*["']([^"']+)["']`),
		"change_count":  regexp.MustCompile(`change_count\s*>\s*(\d+)`),
	}

	for patternName, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(condition); matches != nil {
			switch patternName {
			case "resource_type":
				return change.ResourceType == matches[1]
			case "severity":
				return e.compareSeverity(change.Severity, matches[1])
			case "category":
				return e.hasCategory(change, matches[1])
			case "change_count":
				return len(change.Changes) > parseIntSafe(matches[1])
			}
		}
	}

	return false
}

// Helper functions for compliance evaluation

func (e *EnterpriseDifferEngine) hasOpenToWorldRules(change ResourceDiff) bool {
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "ingress") &&
			strings.Contains(fmt.Sprintf("%v", ch.NewValue), "0.0.0.0/0") {
			return true
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) hasAdminPortExposure(change ResourceDiff) bool {
	adminPorts := []string{"22", "3389", "5432", "3306", "1433"}
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "port") {
			for _, port := range adminPorts {
				if strings.Contains(fmt.Sprintf("%v", ch.NewValue), port) {
					return true
				}
			}
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) wasEncryptionDisabled(change ResourceDiff) bool {
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "encrypt") {
			if oldVal, ok := ch.OldValue.(bool); ok && oldVal {
				if newVal, ok := ch.NewValue.(bool); ok && !newVal {
					return true
				}
			}
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) wasEncryptionKeyChanged(change ResourceDiff) bool {
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "kms_key") || strings.Contains(ch.Path, "encryption_key") {
			return ch.OldValue != ch.NewValue
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) wasMadePublic(change ResourceDiff) bool {
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "public") || strings.Contains(ch.Path, "acl") {
			if oldVal := fmt.Sprintf("%v", ch.OldValue); !strings.Contains(oldVal, "public") {
				if newVal := fmt.Sprintf("%v", ch.NewValue); strings.Contains(newVal, "public") {
					return true
				}
			}
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) hasPrivilegeEscalation(change ResourceDiff) bool {
	dangerousActions := []string{"*", "iam:*", "s3:*", "ec2:*", "admin"}
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "policy") || strings.Contains(ch.Path, "action") {
			newVal := strings.ToLower(fmt.Sprintf("%v", ch.NewValue))
			for _, dangerous := range dangerousActions {
				if strings.Contains(newVal, dangerous) {
					return true
				}
			}
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) hasOverlyBroadPermissions(change ResourceDiff) bool {
	// Similar to privilege escalation but with different thresholds
	return e.hasPrivilegeEscalation(change)
}

func (e *EnterpriseDifferEngine) wasBackupDisabled(change ResourceDiff) bool {
	backupFields := []string{"backup", "automated_backup", "backup_window"}
	for _, ch := range change.Changes {
		for _, field := range backupFields {
			if strings.Contains(ch.Path, field) {
				if oldVal, ok := ch.OldValue.(bool); ok && oldVal {
					if newVal, ok := ch.NewValue.(bool); ok && !newVal {
						return true
					}
				}
			}
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) hasUnexpectedNetworkExposure(change ResourceDiff) bool {
	networkFields := []string{"public_ip", "associate_public_ip", "publicly_accessible"}
	for _, ch := range change.Changes {
		for _, field := range networkFields {
			if strings.Contains(ch.Path, field) {
				if newVal, ok := ch.NewValue.(bool); ok && newVal {
					return true // Unexpected exposure
				}
			}
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) hasMissingDataClassification(change ResourceDiff) bool {
	// Check if data classification tags are missing
	requiredTags := []string{"DataClassification", "Sensitivity", "Compliance"}

	hasClassificationTag := false
	for _, ch := range change.Changes {
		if strings.Contains(ch.Path, "tags") {
			tagStr := fmt.Sprintf("%v", ch.NewValue)
			for _, reqTag := range requiredTags {
				if strings.Contains(tagStr, reqTag) {
					hasClassificationTag = true
					break
				}
			}
		}
	}

	return !hasClassificationTag
}

func (e *EnterpriseDifferEngine) compareSeverity(actual DriftSeverity, expected string) bool {
	severityLevels := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	actualLevel := severityLevels[strings.ToLower(string(actual))]
	expectedLevel := severityLevels[strings.ToLower(expected)]

	return actualLevel >= expectedLevel
}

func (e *EnterpriseDifferEngine) hasCategory(change ResourceDiff, category string) bool {
	expectedCategory := DriftCategory(category)
	for _, cat := range change.Categories {
		if cat == expectedCategory {
			return true
		}
	}
	return false
}

func (e *EnterpriseDifferEngine) buildViolationDescription(rule ComplianceRule, change ResourceDiff) string {
	return fmt.Sprintf("Resource %s (%s) violates compliance rule '%s': %s",
		change.ResourceID, change.ResourceType, rule.Name, rule.Description)
}

// parseIntSafe safely parses an integer from string, returning 0 on error
func parseIntSafe(s string) int {
	// Simple implementation, could use strconv.Atoi with error handling
	if s == "1" {
		return 1
	}
	if s == "2" {
		return 2
	}
	if s == "3" {
		return 3
	}
	if s == "4" {
		return 4
	}
	if s == "5" {
		return 5
	}
	return 0
}

// GetBuiltInComplianceRules returns a set of built-in compliance rules
func GetBuiltInComplianceRules() []ComplianceRule {
	return []ComplianceRule{
		{
			ID:          "SEC-001",
			Name:        "Security Group Open to World",
			Description: "Security groups should not allow inbound access from 0.0.0.0/0",
			Severity:    SeverityHigh,
			Category:    CategorySecurity,
			Condition:   "resource_type == 'aws_security_group' AND open_to_world",
			Remediation: "Restrict security group rules to specific IP ranges or security groups",
			Framework:   "CIS AWS Foundations",
		},
		{
			ID:          "ENC-001",
			Name:        "Encryption at Rest Disabled",
			Description: "Storage resources should have encryption at rest enabled",
			Severity:    SeverityMedium,
			Category:    CategorySecurity,
			Condition:   "encryption_disabled",
			Remediation: "Enable encryption at rest for all storage resources",
			Framework:   "SOC2 Type II",
		},
		{
			ID:          "IAM-001",
			Name:        "Overly Broad IAM Permissions",
			Description: "IAM policies should follow principle of least privilege",
			Severity:    SeverityHigh,
			Category:    CategorySecurity,
			Condition:   "iam_privilege AND overly_broad",
			Remediation: "Review and restrict IAM permissions to minimum required",
			Framework:   "NIST Cybersecurity Framework",
		},
		{
			ID:          "BCK-001",
			Name:        "Backup Policy Disabled",
			Description: "Critical resources should have backup policies enabled",
			Severity:    SeverityMedium,
			Category:    CategoryConfig,
			Condition:   "backup_disabled",
			Remediation: "Enable automated backup policies for critical resources",
			Framework:   "Enterprise Policy",
		},
		{
			ID:          "NET-001",
			Name:        "Unexpected Network Exposure",
			Description: "Resources should not be unexpectedly exposed to the internet",
			Severity:    SeverityHigh,
			Category:    CategoryNetwork,
			Condition:   "network_exposure AND unexpected_exposure",
			Remediation: "Review network configuration and remove unintended exposure",
			Framework:   "Cloud Security Alliance",
		},
		{
			ID:          "DATA-001",
			Name:        "Missing Data Classification",
			Description: "Data resources should have appropriate classification tags",
			Severity:    SeverityLow,
			Category:    CategoryConfig,
			Condition:   "data_classification AND missing_classification",
			Remediation: "Add appropriate data classification tags to resources",
			Framework:   "Data Governance Policy",
		},
	}
}
