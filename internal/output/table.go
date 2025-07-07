package output

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/tabwriter"

	"github.com/yairfalse/wgo/pkg/types"
)

// TableFormatter handles table output formatting
type TableFormatter struct {
	config Config
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(config Config) *TableFormatter {
	return &TableFormatter{config: config}
}

// FormatDriftReport formats a drift report as a table
func (t *TableFormatter) FormatDriftReport(report *types.DriftReport) ([]byte, error) {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Header information
	fmt.Fprintf(w, "Drift Report\n")
	fmt.Fprintf(w, "============\n")
	fmt.Fprintf(w, "ID:\t%s\n", report.ID)
	fmt.Fprintf(w, "Baseline ID:\t%s\n", report.BaselineID)
	fmt.Fprintf(w, "Snapshot ID:\t%s\n", report.SnapshotID)
	fmt.Fprintf(w, "Timestamp:\t%s\n", report.Timestamp.Format(t.config.TimeFormat))
	fmt.Fprintf(w, "Total Changes:\t%d\n", len(report.Changes))
	fmt.Fprintf(w, "\n")

	if len(report.Changes) == 0 {
		fmt.Fprintf(w, "No drift detected - infrastructure matches baseline.\n")
	} else {
		// Changes table
		fmt.Fprintf(w, "Changes:\n")
		fmt.Fprintf(w, "Type\tResource ID\tChange Type\tProperty\tOld Value\tNew Value\n")
		fmt.Fprintf(w, "----\t-----------\t-----------\t--------\t---------\t---------\n")

		for _, change := range report.Changes {
			oldVal := truncateString(fmt.Sprintf("%v", change.OldValue), 20)
			newVal := truncateString(fmt.Sprintf("%v", change.NewValue), 20)
			
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				change.ResourceType,
				change.ResourceID,
				change.Type,
				change.Property,
				oldVal,
				newVal,
			)
		}
	}

	w.Flush()
	return buf.Bytes(), nil
}

// FormatSnapshot formats a snapshot as a table
func (t *TableFormatter) FormatSnapshot(snapshot *types.Snapshot) ([]byte, error) {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	// Header information
	fmt.Fprintf(w, "Snapshot\n")
	fmt.Fprintf(w, "========\n")
	fmt.Fprintf(w, "ID:\t%s\n", snapshot.ID)
	fmt.Fprintf(w, "Provider:\t%s\n", snapshot.Provider)
	fmt.Fprintf(w, "Timestamp:\t%s\n", snapshot.Timestamp.Format(t.config.TimeFormat))
	fmt.Fprintf(w, "Resource Count:\t%d\n", len(snapshot.Resources))
	fmt.Fprintf(w, "Collection Time:\t%s\n", snapshot.Metadata.CollectionTime)
	fmt.Fprintf(w, "\n")

	// Resources by type
	resourcesByType := make(map[string]int)
	for _, resource := range snapshot.Resources {
		resourcesByType[resource.Type]++
	}

	fmt.Fprintf(w, "Resources by Type:\n")
	fmt.Fprintf(w, "Type\tCount\n")
	fmt.Fprintf(w, "----\t-----\n")
	for resourceType, count := range resourcesByType {
		fmt.Fprintf(w, "%s\t%d\n", resourceType, count)
	}

	w.Flush()
	return buf.Bytes(), nil
}

// FormatBaseline formats a baseline as a table
func (t *TableFormatter) FormatBaseline(baseline *types.Baseline) ([]byte, error) {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Baseline\n")
	fmt.Fprintf(w, "========\n")
	fmt.Fprintf(w, "ID:\t%s\n", baseline.ID)
	fmt.Fprintf(w, "Name:\t%s\n", baseline.Name)
	fmt.Fprintf(w, "Description:\t%s\n", baseline.Description)
	fmt.Fprintf(w, "Snapshot ID:\t%s\n", baseline.SnapshotID)
	fmt.Fprintf(w, "Created:\t%s\n", baseline.CreatedAt.Format(t.config.TimeFormat))
	fmt.Fprintf(w, "Version:\t%s\n", baseline.Version)

	if len(baseline.Tags) > 0 {
		fmt.Fprintf(w, "\nTags:\n")
		for key, value := range baseline.Tags {
			fmt.Fprintf(w, "%s:\t%s\n", key, value)
		}
	}

	w.Flush()
	return buf.Bytes(), nil
}

// FormatBaselineList formats a list of baselines as a table
func (t *TableFormatter) FormatBaselineList(baselines []BaselineListItem) ([]byte, error) {
	if len(baselines) == 0 {
		return []byte("No baselines found.\n"), nil
	}

	return t.formatStructList(baselines)
}

// FormatSnapshotList formats a list of snapshots as a table
func (t *TableFormatter) FormatSnapshotList(snapshots []SnapshotListItem) ([]byte, error) {
	if len(snapshots) == 0 {
		return []byte("No snapshots found.\n"), nil
	}

	return t.formatStructList(snapshots)
}

// FormatDriftReportList formats a list of drift reports as a table
func (t *TableFormatter) FormatDriftReportList(reports []DriftReportListItem) ([]byte, error) {
	if len(reports) == 0 {
		return []byte("No drift reports found.\n"), nil
	}

	return t.formatStructList(reports)
}

// formatStructList formats a slice of structs as a table using reflection
func (t *TableFormatter) formatStructList(items interface{}) ([]byte, error) {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)

	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("items must be a slice")
	}

	if v.Len() == 0 {
		return []byte("No items found.\n"), nil
	}

	// Get the first item to determine the structure
	firstItem := v.Index(0)
	itemType := firstItem.Type()

	// Extract headers from struct tags
	var headers []string
	var fieldNames []string
	for i := 0; i < itemType.NumField(); i++ {
		field := itemType.Field(i)
		tableTag := field.Tag.Get("table")
		if tableTag != "" {
			headers = append(headers, tableTag)
			fieldNames = append(fieldNames, field.Name)
		}
	}

	// Write headers
	if t.config.TableHeaders {
		fmt.Fprintf(w, "%s\n", strings.Join(headers, "\t"))
		fmt.Fprintf(w, "%s\n", strings.Join(make([]string, len(headers)), "\t"))
	}

	// Write rows
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		var row []string
		for _, fieldName := range fieldNames {
			fieldValue := item.FieldByName(fieldName)
			value := fmt.Sprintf("%v", fieldValue.Interface())
			// Truncate long values for table display
			if len(value) > 50 {
				value = value[:47] + "..."
			}
			row = append(row, value)
		}
		fmt.Fprintf(w, "%s\n", strings.Join(row, "\t"))
	}

	w.Flush()
	return buf.Bytes(), nil
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}