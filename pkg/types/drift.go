package types

import (
	"fmt"
	"time"
)

// ChangeType represents the type of change detected
type ChangeType string

const (
	// Created indicates a new resource was created
	Created ChangeType = "created"
	// Modified indicates an existing resource was modified
	Modified ChangeType = "modified"
	// Deleted indicates a resource was deleted
	Deleted ChangeType = "deleted"
)

// IsValid checks if the ChangeType is valid
func (ct ChangeType) IsValid() bool {
	switch ct {
	case Created, Modified, Deleted:
		return true
	default:
		return false
	}
}

// String returns the string representation of ChangeType
func (ct ChangeType) String() string {
	return string(ct)
}

// Change represents a single change detected between snapshots
type Change struct {
	ResourceID  string      `json:"resourceId"`
	Type        ChangeType  `json:"type"`
	OldValue    interface{} `json:"oldValue,omitempty"`
	NewValue    interface{} `json:"newValue,omitempty"`
	Description string      `json:"description"`
}

// Validate checks if the Change has all required fields
func (c *Change) Validate() error {
	if c.ResourceID == "" {
		return fmt.Errorf("change resourceID cannot be empty")
	}
	if !c.Type.IsValid() {
		return fmt.Errorf("invalid change type: %s", c.Type)
	}
	if c.Description == "" {
		return fmt.Errorf("change description cannot be empty")
	}
	
	// Validate type-specific requirements
	switch c.Type {
	case Created:
		if c.NewValue == nil {
			return fmt.Errorf("created change must have newValue")
		}
		if c.OldValue != nil {
			return fmt.Errorf("created change should not have oldValue")
		}
	case Deleted:
		if c.OldValue == nil {
			return fmt.Errorf("deleted change must have oldValue")
		}
		if c.NewValue != nil {
			return fmt.Errorf("deleted change should not have newValue")
		}
	case Modified:
		if c.OldValue == nil || c.NewValue == nil {
			return fmt.Errorf("modified change must have both oldValue and newValue")
		}
	}
	
	return nil
}

// DriftReport represents a complete drift analysis between two snapshots
type DriftReport struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	BaselineID string    `json:"baselineId"` // ID of the baseline snapshot
	CurrentID  string    `json:"currentId"`  // ID of the current snapshot
	Changes    []Change  `json:"changes"`
}

// Validate checks if the DriftReport has all required fields
func (dr *DriftReport) Validate() error {
	if dr.ID == "" {
		return fmt.Errorf("drift report ID cannot be empty")
	}
	if dr.Timestamp.IsZero() {
		return fmt.Errorf("drift report timestamp cannot be zero")
	}
	if dr.BaselineID == "" {
		return fmt.Errorf("drift report baselineID cannot be empty")
	}
	if dr.CurrentID == "" {
		return fmt.Errorf("drift report currentID cannot be empty")
	}
	
	// Validate all changes
	for i, change := range dr.Changes {
		if err := change.Validate(); err != nil {
			return fmt.Errorf("invalid change at index %d: %w", i, err)
		}
	}
	
	return nil
}

// ChangeCount returns the total number of changes
func (dr *DriftReport) ChangeCount() int {
	return len(dr.Changes)
}

// ChangesByType returns all changes of a specific type
func (dr *DriftReport) ChangesByType(changeType ChangeType) []Change {
	var filtered []Change
	for _, c := range dr.Changes {
		if c.Type == changeType {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// HasDrift returns true if any changes were detected
func (dr *DriftReport) HasDrift() bool {
	return len(dr.Changes) > 0
}

// Summary returns a summary of changes by type
func (dr *DriftReport) Summary() map[ChangeType]int {
	summary := make(map[ChangeType]int)
	for _, c := range dr.Changes {
		summary[c.Type]++
	}
	return summary
}

// FindChangeByResourceID finds a change by resource ID
func (dr *DriftReport) FindChangeByResourceID(resourceID string) (*Change, bool) {
	for _, c := range dr.Changes {
		if c.ResourceID == resourceID {
			return &c, true
		}
	}
	return nil, false
}