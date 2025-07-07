package output

import (
	"fmt"
	"io"
	"os"

	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

// Renderer implements the Outputter interface
type Renderer struct {
	config     Config
	jsonOut    *JSONFormatter
	tableOut   *TableFormatter
	markdownOut *MarkdownFormatter
}

// NewRenderer creates a new output renderer
func NewRenderer(config Config) *Renderer {
	if config.TimeFormat == "" {
		config.TimeFormat = "2006-01-02 15:04:05"
	}
	
	return &Renderer{
		config:      config,
		jsonOut:     NewJSONFormatter(),
		tableOut:    NewTableFormatter(config),
		markdownOut: NewMarkdownFormatter(config),
	}
}

// FormatDriftReport formats a drift report in the specified format
func (r *Renderer) FormatDriftReport(report *types.DriftReport, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return r.jsonOut.FormatDriftReport(report)
	case FormatTable:
		return r.tableOut.FormatDriftReport(report)
	case FormatMarkdown:
		return r.markdownOut.FormatDriftReport(report)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatSnapshot formats a snapshot in the specified format
func (r *Renderer) FormatSnapshot(snapshot *types.Snapshot, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return r.jsonOut.FormatSnapshot(snapshot)
	case FormatTable:
		return r.tableOut.FormatSnapshot(snapshot)
	case FormatMarkdown:
		return r.markdownOut.FormatSnapshot(snapshot)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatBaseline formats a baseline in the specified format
func (r *Renderer) FormatBaseline(baseline *types.Baseline, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return r.jsonOut.FormatBaseline(baseline)
	case FormatTable:
		return r.tableOut.FormatBaseline(baseline)
	case FormatMarkdown:
		return r.markdownOut.FormatBaseline(baseline)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatBaselineList formats a list of baselines
func (r *Renderer) FormatBaselineList(baselines []BaselineListItem, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return r.jsonOut.FormatBaselineList(baselines)
	case FormatTable:
		return r.tableOut.FormatBaselineList(baselines)
	case FormatMarkdown:
		return r.markdownOut.FormatBaselineList(baselines)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatSnapshotList formats a list of snapshots
func (r *Renderer) FormatSnapshotList(snapshots []SnapshotListItem, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return r.jsonOut.FormatSnapshotList(snapshots)
	case FormatTable:
		return r.tableOut.FormatSnapshotList(snapshots)
	case FormatMarkdown:
		return r.markdownOut.FormatSnapshotList(snapshots)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatDriftReportList formats a list of drift reports
func (r *Renderer) FormatDriftReportList(reports []DriftReportListItem, format OutputFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return r.jsonOut.FormatDriftReportList(reports)
	case FormatTable:
		return r.tableOut.FormatDriftReportList(reports)
	case FormatMarkdown:
		return r.markdownOut.FormatDriftReportList(reports)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// DisplayProgress shows a progress message
func (r *Renderer) DisplayProgress(message string) {
	if r.config.EnableColors {
		fmt.Printf("\033[36m🔄 %s\033[0m\n", message)
	} else {
		fmt.Printf("🔄 %s\n", message)
	}
}

// DisplayError shows an error message
func (r *Renderer) DisplayError(err error) {
	if r.config.EnableColors {
		fmt.Printf("\033[31m❌ Error: %v\033[0m\n", err)
	} else {
		fmt.Printf("❌ Error: %v\n", err)
	}
}

// DisplaySuccess shows a success message
func (r *Renderer) DisplaySuccess(message string) {
	if r.config.EnableColors {
		fmt.Printf("\033[32m✅ %s\033[0m\n", message)
	} else {
		fmt.Printf("✅ %s\n", message)
	}
}

// DisplayWarning shows a warning message
func (r *Renderer) DisplayWarning(message string) {
	if r.config.EnableColors {
		fmt.Printf("\033[33m⚠️  %s\033[0m\n", message)
	} else {
		fmt.Printf("⚠️  %s\n", message)
	}
}

// DisplayInfo shows an info message
func (r *Renderer) DisplayInfo(message string) {
	if r.config.EnableColors {
		fmt.Printf("\033[34mℹ️  %s\033[0m\n", message)
	} else {
		fmt.Printf("ℹ️  %s\n", message)
	}
}

// WriteToFile writes data to a file
func (r *Renderer) WriteToFile(data []byte, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filename, err)
	}

	return nil
}

// WriteTo writes data to a writer
func (r *Renderer) WriteTo(data []byte, writer io.Writer) error {
	_, err := writer.Write(data)
	return err
}

// ConvertBaselineInfoToListItem converts storage.BaselineInfo to BaselineListItem
func ConvertBaselineInfoToListItem(info storage.BaselineInfo, snapshotResourceCount int) BaselineListItem {
	return BaselineListItem{
		ID:            info.ID,
		Name:          info.Name,
		Description:   info.Description,
		SnapshotID:    info.SnapshotID,
		CreatedAt:     info.CreatedAt.Format("2006-01-02 15:04:05"),
		ResourceCount: snapshotResourceCount,
		FileSize:      formatFileSize(info.FileSize),
	}
}

// ConvertSnapshotInfoToListItem converts storage.SnapshotInfo to SnapshotListItem
func ConvertSnapshotInfoToListItem(info storage.SnapshotInfo) SnapshotListItem {
	return SnapshotListItem{
		ID:            info.ID,
		Provider:      info.Provider,
		Timestamp:     info.Timestamp.Format("2006-01-02 15:04:05"),
		ResourceCount: info.ResourceCount,
		FileSize:      formatFileSize(info.FileSize),
	}
}

// ConvertDriftReportInfoToListItem converts storage.DriftReportInfo to DriftReportListItem
func ConvertDriftReportInfoToListItem(info storage.DriftReportInfo) DriftReportListItem {
	status := "No Changes"
	if info.ChangeCount > 0 {
		status = "Has Drift"
	}

	return DriftReportListItem{
		ID:          info.ID,
		BaselineID:  info.BaselineID,
		SnapshotID:  info.SnapshotID,
		CreatedAt:   info.CreatedAt.Format("2006-01-02 15:04:05"),
		ChangeCount: info.ChangeCount,
		Status:      status,
		FileSize:    formatFileSize(info.FileSize),
	}
}

// formatFileSize formats file size in human-readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParseOutputFormat parses a string into OutputFormat
func ParseOutputFormat(format string) (OutputFormat, error) {
	switch format {
	case "json":
		return FormatJSON, nil
	case "table":
		return FormatTable, nil
	case "markdown":
		return FormatMarkdown, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unsupported format: %s (supported: json, table, markdown, yaml)", format)
	}
}