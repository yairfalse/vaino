package types

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ChangeType represents the type of change detected in a drift analysis
type ChangeType string

const (
	// ChangeTypeCreated indicates a resource was created
	ChangeTypeCreated ChangeType = "created"
	// ChangeTypeModified indicates a resource was modified
	ChangeTypeModified ChangeType = "modified"
	// ChangeTypeDeleted indicates a resource was deleted
	ChangeTypeDeleted ChangeType = "deleted"
)

// IsValid checks if the ChangeType is valid
func (c ChangeType) IsValid() bool {
	switch c {
	case ChangeTypeCreated, ChangeTypeModified, ChangeTypeDeleted:
		return true
	default:
		return false
	}
}

// String returns the string representation of the ChangeType
func (c ChangeType) String() string {
	return string(c)
}

// Change represents a single change detected in drift analysis
type Change struct {
	ResourceID  string      `json:"resource_id"`
	Type        ChangeType  `json:"type"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	Description string      `json:"description"`
}

// Validate checks if the Change has all required fields and valid values
func (c *Change) Validate() error {
	if strings.TrimSpace(c.ResourceID) == "" {
		return errors.New("change resource ID is required")
	}
	if !c.Type.IsValid() {
		return errors.New("change type is invalid")
	}
	if strings.TrimSpace(c.Description) == "" {
		return errors.New("change description is required")
	}

	// Validate change type specific requirements
	switch c.Type {
	case ChangeTypeCreated:
		if c.NewValue == nil {
			return errors.New("created change must have a new value")
		}
		if c.OldValue != nil {
			return errors.New("created change should not have an old value")
		}
	case ChangeTypeDeleted:
		if c.OldValue == nil {
			return errors.New("deleted change must have an old value")
		}
		if c.NewValue != nil {
			return errors.New("deleted change should not have a new value")
		}
	case ChangeTypeModified:
		if c.OldValue == nil || c.NewValue == nil {
			return errors.New("modified change must have both old and new values")
		}
	}

	return nil
}

// IsBreakingChange determines if this change could be considered breaking
func (c *Change) IsBreakingChange() bool {
	switch c.Type {
	case ChangeTypeDeleted:
		return true
	case ChangeTypeModified:
		// This is a simplified check - in reality this would be more sophisticated
		// based on the specific resource type and field being changed
		return strings.Contains(strings.ToLower(c.Description), "security") ||
			strings.Contains(strings.ToLower(c.Description), "access") ||
			strings.Contains(strings.ToLower(c.Description), "network") ||
			strings.Contains(strings.ToLower(c.Description), "public")
	default:
		return false
	}
}

// DriftReport represents a comprehensive drift analysis report
type DriftReport struct {
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	BaselineID string    `json:"baseline_id"`
	CurrentID  string    `json:"current_id"`
	Changes    []Change  `json:"changes"`
}

// Validate checks if the DriftReport has all required fields and valid values
func (d *DriftReport) Validate() error {
	if strings.TrimSpace(d.ID) == "" {
		return errors.New("drift report ID is required")
	}
	if d.Timestamp.IsZero() {
		return errors.New("drift report timestamp is required")
	}
	if strings.TrimSpace(d.BaselineID) == "" {
		return errors.New("drift report baseline ID is required")
	}
	if strings.TrimSpace(d.CurrentID) == "" {
		return errors.New("drift report current ID is required")
	}
	if d.Changes == nil {
		return errors.New("drift report changes cannot be nil")
	}

	// Validate each change in the report
	for i, change := range d.Changes {
		if err := change.Validate(); err != nil {
			return fmt.Errorf("change at index %d is invalid: %w", i, err)
		}
	}

	return nil
}

// HasChanges returns true if the drift report contains any changes
func (d *DriftReport) HasChanges() bool {
	return len(d.Changes) > 0
}

// ChangeCount returns the total number of changes in the report
func (d *DriftReport) ChangeCount() int {
	return len(d.Changes)
}

// GetChangesByType returns all changes of a specific type
func (d *DriftReport) GetChangesByType(changeType ChangeType) []Change {
	var changes []Change
	for _, change := range d.Changes {
		if change.Type == changeType {
			changes = append(changes, change)
		}
	}
	return changes
}

// GetChangesByResourceID returns all changes for a specific resource
func (d *DriftReport) GetChangesByResourceID(resourceID string) []Change {
	var changes []Change
	for _, change := range d.Changes {
		if change.ResourceID == resourceID {
			changes = append(changes, change)
		}
	}
	return changes
}

// GetBreakingChanges returns all changes that are considered breaking
func (d *DriftReport) GetBreakingChanges() []Change {
	var changes []Change
	for _, change := range d.Changes {
		if change.IsBreakingChange() {
			changes = append(changes, change)
		}
	}
	return changes
}

// HasBreakingChanges returns true if the report contains any breaking changes
func (d *DriftReport) HasBreakingChanges() bool {
	return len(d.GetBreakingChanges()) > 0
}

// GetCreatedResourceCount returns the number of resources that were created
func (d *DriftReport) GetCreatedResourceCount() int {
	return len(d.GetChangesByType(ChangeTypeCreated))
}

// GetModifiedResourceCount returns the number of resources that were modified
func (d *DriftReport) GetModifiedResourceCount() int {
	return len(d.GetChangesByType(ChangeTypeModified))
}

// GetDeletedResourceCount returns the number of resources that were deleted
func (d *DriftReport) GetDeletedResourceCount() int {
	return len(d.GetChangesByType(ChangeTypeDeleted))
}

// GetSummary returns a summary of the drift report
func (d *DriftReport) GetSummary() map[string]int {
	return map[string]int{
		"total":    d.ChangeCount(),
		"created":  d.GetCreatedResourceCount(),
		"modified": d.GetModifiedResourceCount(),
		"deleted":  d.GetDeletedResourceCount(),
		"breaking": len(d.GetBreakingChanges()),
	}
}

// AddChange adds a change to the drift report
func (d *DriftReport) AddChange(change *Change) error {
	if err := change.Validate(); err != nil {
		return err
	}
	d.Changes = append(d.Changes, *change)
	return nil
}

// String returns a string representation of the drift report
func (d *DriftReport) String() string {
	summary := d.GetSummary()
	return fmt.Sprintf("Drift Report %s - Total: %d, Created: %d, Modified: %d, Deleted: %d, Breaking: %d",
		d.ID, summary["total"], summary["created"], summary["modified"], summary["deleted"], summary["breaking"])
}
