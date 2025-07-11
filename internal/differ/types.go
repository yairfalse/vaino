package differ

import (
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// Differ interface defines the contract for infrastructure drift detection
type Differ interface {
	Compare(baseline, current *types.Snapshot) (*DriftReport, error)
	CalculateDrift(changes []Change) DriftSummary
	ClassifyChange(change Change) ChangeType
}

// DriftReport represents the complete result of a drift comparison
type DriftReport struct {
	ID              string                 `json:"id"`
	BaselineID      string                 `json:"baseline_id"`
	CurrentID       string                 `json:"current_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Summary         DriftSummary           `json:"summary"`
	ResourceChanges []ResourceDiff         `json:"resource_changes"`
	AllChanges      []Change               `json:"all_changes,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`

	// Enterprise features
	ComplianceReport *ComplianceReport `json:"compliance_report,omitempty"`
	ExecutiveSummary *ExecutiveSummary `json:"executive_summary,omitempty"`
}

// DriftSummary provides high-level statistics about the drift
type DriftSummary struct {
	TotalResources    int                   `json:"total_resources"`
	ChangedResources  int                   `json:"changed_resources"`
	AddedResources    int                   `json:"added_resources"`
	RemovedResources  int                   `json:"removed_resources"`
	ModifiedResources int                   `json:"modified_resources"`
	TotalChanges      int                   `json:"total_changes"`
	ChangesByCategory map[DriftCategory]int `json:"changes_by_category"`
	ChangesBySeverity map[RiskLevel]int     `json:"changes_by_severity"`
	OverallRisk       RiskLevel             `json:"overall_risk"`
	RiskScore         float64               `json:"risk_score"`
	RiskAssessment    string                `json:"risk_assessment"`
	ComplianceStatus  string                `json:"compliance_status"`
	Timestamp         time.Time             `json:"timestamp"`

	// Enterprise summary fields
	CriticalChanges   int                    `json:"critical_changes"`
	HighRiskChanges   int                    `json:"high_risk_changes"`
	MediumRiskChanges int                    `json:"medium_risk_changes"`
	LowRiskChanges    int                    `json:"low_risk_changes"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// ResourceDiff represents changes to a specific resource
type ResourceDiff struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	DriftType    ChangeType             `json:"drift_type"`
	Changes      []Change               `json:"changes"`
	Severity     RiskLevel              `json:"severity"`
	Category     DriftCategory          `json:"category"`
	Categories   []DriftCategory        `json:"categories,omitempty"`
	RiskScore    float64                `json:"risk_score"`
	Description  string                 `json:"description"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DifferChange represents a specific configuration change in the differ context
type DifferChange struct {
	Type        ChangeType    `json:"type"`
	ResourceID  string        `json:"resource_id"`
	Path        string        `json:"path"`
	Field       string        `json:"field"`
	OldValue    interface{}   `json:"old_value"`
	NewValue    interface{}   `json:"new_value"`
	Severity    RiskLevel     `json:"severity"`
	Category    DriftCategory `json:"category"`
	Impact      string        `json:"impact"`
	Description string        `json:"description"`
}

// Change is an alias for backward compatibility
type Change = DifferChange

// ChangeType represents the type of change detected
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeRemoved  ChangeType = "removed"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeMoved    ChangeType = "moved"
)

// RiskLevel represents the severity/risk level of a change
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// DriftCategory represents the category of drift detected
type DriftCategory string

const (
	DriftCategoryConfig   DriftCategory = "configuration"
	DriftCategorySecurity DriftCategory = "security"
	DriftCategoryCost     DriftCategory = "cost"
	DriftCategoryState    DriftCategory = "state"
	DriftCategoryNetwork  DriftCategory = "network"
	DriftCategoryStorage  DriftCategory = "storage"
	DriftCategoryCompute  DriftCategory = "compute"
)

// DiffOptions configures how the comparison is performed
type DiffOptions struct {
	IgnoreFields    []string          `json:"ignore_fields,omitempty"`
	IgnoreResources []string          `json:"ignore_resources,omitempty"`
	IgnoreProviders []string          `json:"ignore_providers,omitempty"`
	IgnoreMetadata  bool              `json:"ignore_metadata,omitempty"`
	MinRiskLevel    RiskLevel         `json:"min_risk_level,omitempty"`
	Categories      []DriftCategory   `json:"categories,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
}

// ResourceMatcher defines how resources are matched between snapshots
type ResourceMatcher interface {
	Match(baseline, current []types.Resource) ([]ResourceMatch, []types.Resource, []types.Resource)
}

// ChangeClassifier categorizes and scores changes
type ChangeClassifier interface {
	ClassifyChange(change Change) (DriftCategory, RiskLevel, float64)
	CalculateResourceRisk(changes []Change) (RiskLevel, float64)
	CalculateOverallRisk(summary DriftSummary) (RiskLevel, float64)
}

// Comparer performs deep comparison of resource configurations
type Comparer interface {
	Compare(baseline, current types.Resource) ([]Change, error)
	CompareResources(baseline, current types.Resource) []Change
	CompareConfiguration(basePath string, baseline, current map[string]interface{}) []Change
}

// ResourceMatch represents a matched pair of resources between snapshots
type ResourceMatch struct {
	Baseline types.Resource
	Current  types.Resource
}
