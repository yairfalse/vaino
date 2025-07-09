package output

import (
	"encoding/json"

	"github.com/yairfalse/vaino/pkg/types"
)

// JSONFormatter handles JSON output formatting
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// FormatDriftReport formats a drift report as JSON
func (j *JSONFormatter) FormatDriftReport(report *types.DriftReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// FormatSnapshot formats a snapshot as JSON
func (j *JSONFormatter) FormatSnapshot(snapshot *types.Snapshot) ([]byte, error) {
	return json.MarshalIndent(snapshot, "", "  ")
}

// FormatBaseline formats a baseline as JSON
func (j *JSONFormatter) FormatBaseline(baseline *types.Baseline) ([]byte, error) {
	return json.MarshalIndent(baseline, "", "  ")
}

// FormatBaselineList formats a list of baselines as JSON
func (j *JSONFormatter) FormatBaselineList(baselines []BaselineListItem) ([]byte, error) {
	return json.MarshalIndent(baselines, "", "  ")
}

// FormatSnapshotList formats a list of snapshots as JSON
func (j *JSONFormatter) FormatSnapshotList(snapshots []SnapshotListItem) ([]byte, error) {
	return json.MarshalIndent(snapshots, "", "  ")
}

// FormatDriftReportList formats a list of drift reports as JSON
func (j *JSONFormatter) FormatDriftReportList(reports []DriftReportListItem) ([]byte, error) {
	return json.MarshalIndent(reports, "", "  ")
}
