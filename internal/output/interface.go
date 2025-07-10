package output

import (
	"io"

	"github.com/yairfalse/vaino/pkg/types"
)

// OutputFormat represents the available output formats
type OutputFormat string

const (
	FormatJSON     OutputFormat = "json"
	FormatTable    OutputFormat = "table"
	FormatMarkdown OutputFormat = "markdown"
	FormatYAML     OutputFormat = "yaml"
)

// Outputter defines the interface for formatting and displaying output
type Outputter interface {
	// Format drift reports
	FormatDriftReport(report *types.DriftReport, format OutputFormat) ([]byte, error)

	// Format snapshots
	FormatSnapshot(snapshot *types.Snapshot, format OutputFormat) ([]byte, error)

	// Format baselines
	FormatBaseline(baseline *types.Baseline, format OutputFormat) ([]byte, error)

	// Format lists
	FormatBaselineList(baselines []BaselineListItem, format OutputFormat) ([]byte, error)
	FormatSnapshotList(snapshots []SnapshotListItem, format OutputFormat) ([]byte, error)
	FormatDriftReportList(reports []DriftReportListItem, format OutputFormat) ([]byte, error)

	// Display methods for interactive use
	DisplayProgress(message string)
	DisplayError(err error)
	DisplaySuccess(message string)
	DisplayWarning(message string)

	// Writer methods
	WriteToFile(data []byte, filename string) error
	WriteTo(data []byte, writer io.Writer) error
}

// BaselineListItem represents a baseline in list output
type BaselineListItem struct {
	ID            string `json:"id" table:"ID"`
	Name          string `json:"name" table:"Name"`
	Description   string `json:"description,omitempty" table:"Description"`
	SnapshotID    string `json:"snapshot_id" table:"Snapshot ID"`
	CreatedAt     string `json:"created_at" table:"Created"`
	ResourceCount int    `json:"resource_count" table:"Resources"`
	FileSize      string `json:"file_size" table:"Size"`
}

// SnapshotListItem represents a snapshot in list output
type SnapshotListItem struct {
	ID            string `json:"id" table:"ID"`
	Provider      string `json:"provider" table:"Provider"`
	Timestamp     string `json:"timestamp" table:"Timestamp"`
	ResourceCount int    `json:"resource_count" table:"Resources"`
	FileSize      string `json:"file_size" table:"Size"`
}

// DriftReportListItem represents a drift report in list output
type DriftReportListItem struct {
	ID          string `json:"id" table:"ID"`
	BaselineID  string `json:"baseline_id" table:"Baseline"`
	SnapshotID  string `json:"snapshot_id" table:"Snapshot"`
	CreatedAt   string `json:"created_at" table:"Created"`
	ChangeCount int    `json:"change_count" table:"Changes"`
	Status      string `json:"status" table:"Status"`
	FileSize    string `json:"file_size" table:"Size"`
}

// Config holds output configuration
type Config struct {
	// Default format when none specified
	DefaultFormat OutputFormat `json:"default_format"`

	// Color settings
	EnableColors bool `json:"enable_colors"`

	// Table settings
	TableHeaders bool `json:"table_headers"`
	TableBorders bool `json:"table_borders"`

	// Pagination
	PageSize int `json:"page_size"`

	// Timestamp format
	TimeFormat string `json:"time_format"`
}
