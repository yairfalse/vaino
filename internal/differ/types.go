package differ

import "time"

// DriftReport represents the result of comparing infrastructure states
type DriftReport struct {
	ID              string           `json:"id"`
	BaselineID      string           `json:"baseline_id"`
	CurrentID       string           `json:"current_id"`
	Timestamp       time.Time        `json:"timestamp"`
	Summary         DriftSummary     `json:"summary"`
	ResourceChanges []ResourceChange `json:"resource_changes"`
}

// DriftSummary provides a summary of drift findings
type DriftSummary struct {
	TotalResources     int `json:"total_resources"`
	ChangedResources   int `json:"changed_resources"`
	AddedResources     int `json:"added_resources"`
	RemovedResources   int `json:"removed_resources"`
	ModifiedResources  int `json:"modified_resources"`
}

// ResourceChange represents a change to a specific resource
type ResourceChange struct {
	ResourceID   string     `json:"resource_id"`
	ResourceType string     `json:"resource_type"`
	DriftType    ChangeType `json:"drift_type"`
	Changes      []Change   `json:"changes"`
}

// Change represents a specific field change within a resource
type Change struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// ChangeType represents the type of change detected
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeRemoved  ChangeType = "removed"
	ChangeTypeModified ChangeType = "modified"
)