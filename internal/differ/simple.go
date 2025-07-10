package differ

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// SimpleChange represents a simple change between two states
type SimpleChange struct {
	Type         string    // "added", "removed", "modified"
	ResourceID   string    // e.g., "deployment/frontend"
	ResourceType string    // e.g., "deployment"
	ResourceName string    // e.g., "frontend"
	Namespace    string    // e.g., "test-workloads"
	Timestamp    time.Time // When the change was detected
	Details      []SimpleFieldChange
}

// SimpleFieldChange represents a change in a specific field
type SimpleFieldChange struct {
	Field    string      // e.g., "replicas"
	OldValue interface{} // e.g., 3
	NewValue interface{} // e.g., 5
}

// SimpleChangeReport contains all changes between two snapshots
type SimpleChangeReport struct {
	FromTime time.Time
	ToTime   time.Time
	Changes  []SimpleChange
	Summary  ChangeSummary
}

// ChangeSummary provides counts of changes
type ChangeSummary struct {
	Added    int
	Removed  int
	Modified int
	Total    int
}

// SimpleDiffer compares two snapshots and returns changes
type SimpleDiffer struct{}

// NewSimpleDiffer creates a new simple differ
func NewSimpleDiffer() *SimpleDiffer {
	return &SimpleDiffer{}
}

// Compare compares two snapshots and returns a change report
func (d *SimpleDiffer) Compare(from, to *types.Snapshot) (*SimpleChangeReport, error) {
	report := &SimpleChangeReport{
		FromTime: from.Timestamp,
		ToTime:   to.Timestamp,
		Changes:  []SimpleChange{},
	}

	// Build maps for efficient lookup
	fromMap := make(map[string]types.Resource)
	toMap := make(map[string]types.Resource)

	for _, r := range from.Resources {
		fromMap[r.ID] = r
	}
	for _, r := range to.Resources {
		toMap[r.ID] = r
	}

	// Find removed and modified resources
	for id, fromResource := range fromMap {
		if toResource, exists := toMap[id]; exists {
			// Resource exists in both - check for modifications
			if changes := d.compareResources(fromResource, toResource); len(changes) > 0 {
				report.Changes = append(report.Changes, SimpleChange{
					Type:         "modified",
					ResourceID:   id,
					ResourceType: fromResource.Type,
					ResourceName: fromResource.Name,
					Namespace:    fromResource.Namespace,
					Timestamp:    to.Timestamp,
					Details:      changes,
				})
				report.Summary.Modified++
			}
		} else {
			// Resource was removed
			report.Changes = append(report.Changes, SimpleChange{
				Type:         "removed",
				ResourceID:   id,
				ResourceType: fromResource.Type,
				ResourceName: fromResource.Name,
				Namespace:    fromResource.Namespace,
				Timestamp:    to.Timestamp,
			})
			report.Summary.Removed++
		}
	}

	// Find added resources
	for id, toResource := range toMap {
		if _, exists := fromMap[id]; !exists {
			report.Changes = append(report.Changes, SimpleChange{
				Type:         "added",
				ResourceID:   id,
				ResourceType: toResource.Type,
				ResourceName: toResource.Name,
				Namespace:    toResource.Namespace,
				Timestamp:    to.Timestamp,
			})
			report.Summary.Added++
		}
	}

	report.Summary.Total = len(report.Changes)

	// Sort changes for consistent output
	sort.Slice(report.Changes, func(i, j int) bool {
		// Sort by type (added, modified, removed), then by resource ID
		if report.Changes[i].Type != report.Changes[j].Type {
			typeOrder := map[string]int{"added": 0, "modified": 1, "removed": 2}
			return typeOrder[report.Changes[i].Type] < typeOrder[report.Changes[j].Type]
		}
		return report.Changes[i].ResourceID < report.Changes[j].ResourceID
	})

	return report, nil
}

// compareResources compares two resources and returns field changes
func (d *SimpleDiffer) compareResources(from, to types.Resource) []SimpleFieldChange {
	var changes []SimpleFieldChange

	// Compare configuration
	changes = append(changes, d.compareConfiguration(from.Configuration, to.Configuration, "")...)

	return changes
}

// compareConfiguration recursively compares configuration maps
func (d *SimpleDiffer) compareConfiguration(from, to map[string]interface{}, prefix string) []SimpleFieldChange {
	var changes []SimpleFieldChange

	// Check all keys in 'from'
	for key, fromValue := range from {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		toValue, exists := to[key]
		if !exists {
			changes = append(changes, SimpleFieldChange{
				Field:    path,
				OldValue: fromValue,
				NewValue: nil,
			})
			continue
		}

		// Compare values
		if !d.valuesEqual(fromValue, toValue) {
			changes = append(changes, SimpleFieldChange{
				Field:    path,
				OldValue: fromValue,
				NewValue: toValue,
			})
		}
	}

	// Check for new keys in 'to'
	for key, toValue := range to {
		if _, exists := from[key]; !exists {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			changes = append(changes, SimpleFieldChange{
				Field:    path,
				OldValue: nil,
				NewValue: toValue,
			})
		}
	}

	return changes
}

// valuesEqual compares two values for equality
func (d *SimpleDiffer) valuesEqual(a, b interface{}) bool {
	// Simple comparison - could be enhanced
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// FormatChangeReport formats a change report as a string
func FormatChangeReport(report *SimpleChangeReport) string {
	var output strings.Builder

	// Header
	output.WriteString(fmt.Sprintf("📊 Infrastructure Changes\n"))
	output.WriteString(fmt.Sprintf("========================\n"))
	output.WriteString(fmt.Sprintf("Time range: %s → %s\n\n",
		report.FromTime.Format("2006-01-02 15:04:05"),
		report.ToTime.Format("2006-01-02 15:04:05")))

	// Summary
	output.WriteString(fmt.Sprintf("Summary:\n"))
	output.WriteString(fmt.Sprintf("  Added:    %d\n", report.Summary.Added))
	output.WriteString(fmt.Sprintf("  Modified: %d\n", report.Summary.Modified))
	output.WriteString(fmt.Sprintf("  Removed:  %d\n", report.Summary.Removed))
	output.WriteString(fmt.Sprintf("  Total:    %d changes\n\n", report.Summary.Total))

	if len(report.Changes) == 0 {
		output.WriteString("✅ No changes detected\n")
		return output.String()
	}

	// Group changes by type
	added := []SimpleChange{}
	modified := []SimpleChange{}
	removed := []SimpleChange{}

	for _, change := range report.Changes {
		switch change.Type {
		case "added":
			added = append(added, change)
		case "modified":
			modified = append(modified, change)
		case "removed":
			removed = append(removed, change)
		}
	}

	// Show added resources
	if len(added) > 0 {
		output.WriteString("Added:\n")
		for _, change := range added {
			output.WriteString(fmt.Sprintf("  + %s\n", change.ResourceID))
		}
		output.WriteString("\n")
	}

	// Show modified resources
	if len(modified) > 0 {
		output.WriteString("Modified:\n")
		for _, change := range modified {
			output.WriteString(fmt.Sprintf("  ~ %s\n", change.ResourceID))
			for _, detail := range change.Details {
				output.WriteString(fmt.Sprintf("    - %s: %v → %v\n",
					detail.Field, detail.OldValue, detail.NewValue))
			}
		}
		output.WriteString("\n")
	}

	// Show removed resources
	if len(removed) > 0 {
		output.WriteString("Removed:\n")
		for _, change := range removed {
			output.WriteString(fmt.Sprintf("  - %s\n", change.ResourceID))
		}
	}

	return output.String()
}
