package types

import "time"

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
)

// Change represents a specific configuration change
type Change struct {
	Field       string      `json:"field"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	Severity    string      `json:"severity"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	ChangeType  ChangeType  `json:"change_type"`
}

// DriftReport represents the result of comparing infrastructure states
type DriftReport struct {
	ID         string       `json:"id"`
	Timestamp  time.Time    `json:"timestamp"`
	BaselineID string       `json:"baseline_id"`
	CurrentID  string       `json:"current_id"`
	Changes    []Change     `json:"changes"`
	Summary    DriftSummary `json:"summary"`
	Analysis   *Analysis    `json:"analysis,omitempty"`
}

// DriftSummary provides high-level drift statistics
type DriftSummary struct {
	TotalChanges      int     `json:"total_changes"`
	AddedResources    int     `json:"added_resources"`
	DeletedResources  int     `json:"deleted_resources"`
	ModifiedResources int     `json:"modified_resources"`
	RiskScore         float64 `json:"risk_score"`
	HighRiskChanges   int     `json:"high_risk_changes"`
}

// Analysis provides AI-powered insights about drift
type Analysis struct {
	RiskScore       float64             `json:"risk_score"`
	Categories      map[string][]Change `json:"categories"`
	Recommendations []string            `json:"recommendations"`
	Insights        []Insight           `json:"insights"`
	SecurityImpact  string              `json:"security_impact,omitempty"`
	CostImpact      string              `json:"cost_impact,omitempty"`
}

// Insight represents an AI-generated insight about infrastructure changes
type Insight struct {
	Type        string  `json:"type"`        // "warning", "info", "critical"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`  // 0.0 - 1.0
	Action      string  `json:"action,omitempty"`
}