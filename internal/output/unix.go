package output

import (
	"fmt"
	"strings"

	"github.com/yairfalse/wgo/internal/differ"
)

// UnixFormatter provides simple Unix-style output similar to git diff
type UnixFormatter struct {
	noColor bool
}

// NewUnixFormatter creates a new Unix-style formatter
func NewUnixFormatter(noColor bool) *UnixFormatter {
	return &UnixFormatter{
		noColor: noColor,
	}
}

// FormatDriftReport formats a drift report in Unix style
func (u *UnixFormatter) FormatDriftReport(report *differ.DriftReport) ([]byte, error) {
	var output strings.Builder

	if len(report.ResourceChanges) == 0 {
		// No changes - exit silently like git diff
		return []byte{}, nil
	}

	// Group changes by resource
	changesByResource := make(map[string][]differ.Change)
	for _, resourceChange := range report.ResourceChanges {
		resourceKey := fmt.Sprintf("%s/%s", resourceChange.ResourceType, resourceChange.ResourceID)
		changesByResource[resourceKey] = resourceChange.Changes
	}

	// Output changes in a git diff-like format
	for resource, changes := range changesByResource {
		output.WriteString(fmt.Sprintf("--- %s\n", resource))
		output.WriteString(fmt.Sprintf("+++ %s\n", resource))

		for _, change := range changes {
			// Show the change in a simple format
			output.WriteString(fmt.Sprintf("@@ %s @@\n", change.Field))

			// Old value
			if change.OldValue != nil {
				output.WriteString(fmt.Sprintf("-%s: %v\n", change.Field, change.OldValue))
			}

			// New value
			if change.NewValue != nil {
				output.WriteString(fmt.Sprintf("+%s: %v\n", change.Field, change.NewValue))
			}
		}
		output.WriteString("\n")
	}

	// Summary at the end (like git diff --stat)
	if report.Summary.AddedResources > 0 || report.Summary.RemovedResources > 0 || report.Summary.ModifiedResources > 0 {
		output.WriteString(fmt.Sprintf("%d additions, %d deletions, %d modifications\n",
			report.Summary.AddedResources,
			report.Summary.RemovedResources,
			report.Summary.ModifiedResources))
	}

	return []byte(output.String()), nil
}

// FormatSimple provides an even simpler output for when just the exit code matters
func (u *UnixFormatter) FormatSimple(report *differ.DriftReport) ([]byte, error) {
	if len(report.ResourceChanges) == 0 {
		return []byte{}, nil
	}

	var output strings.Builder

	// Just list what changed
	for _, resourceChange := range report.ResourceChanges {
		switch resourceChange.DriftType {
		case differ.ChangeTypeAdded:
			output.WriteString(fmt.Sprintf("A %s/%s\n", resourceChange.ResourceType, resourceChange.ResourceID))
		case differ.ChangeTypeRemoved:
			output.WriteString(fmt.Sprintf("D %s/%s\n", resourceChange.ResourceType, resourceChange.ResourceID))
		case differ.ChangeTypeModified:
			output.WriteString(fmt.Sprintf("M %s/%s\n", resourceChange.ResourceType, resourceChange.ResourceID))
		}
	}

	return []byte(output.String()), nil
}

// FormatNameOnly lists only the names of changed resources (like git diff --name-only)
func (u *UnixFormatter) FormatNameOnly(report *differ.DriftReport) ([]byte, error) {
	if len(report.ResourceChanges) == 0 {
		return []byte{}, nil
	}

	var resources []string

	for _, resourceChange := range report.ResourceChanges {
		key := fmt.Sprintf("%s/%s", resourceChange.ResourceType, resourceChange.ResourceID)
		resources = append(resources, key)
	}

	return []byte(strings.Join(resources, "\n") + "\n"), nil
}

// FormatStat provides statistics output (like git diff --stat)
func (u *UnixFormatter) FormatStat(report *differ.DriftReport) ([]byte, error) {
	if len(report.ResourceChanges) == 0 {
		return []byte{}, nil
	}

	var output strings.Builder

	// Count changes per resource
	for _, resourceChange := range report.ResourceChanges {
		resource := fmt.Sprintf("%s/%s", resourceChange.ResourceType, resourceChange.ResourceID)
		changeCount := len(resourceChange.Changes)
		output.WriteString(fmt.Sprintf(" %s | %d %s\n", resource, changeCount, plural(changeCount, "change")))
	}

	// Summary
	totalResources := len(report.ResourceChanges)
	totalChanges := 0
	for _, rc := range report.ResourceChanges {
		totalChanges += len(rc.Changes)
	}

	output.WriteString(fmt.Sprintf(" %d %s changed, %d %s\n",
		totalResources,
		plural(totalResources, "resource"),
		totalChanges,
		plural(totalChanges, "modification")))

	return []byte(output.String()), nil
}

func plural(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}
