package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yairfalse/wgo/pkg/types"
	"gopkg.in/yaml.v3"
)

// Formatter defines the interface for output formatting
type Formatter interface {
	FormatSnapshot(snapshot *types.Snapshot, writer io.Writer) error
	FormatDriftReport(report *types.DriftReport, writer io.Writer) error
	FormatBaseline(baseline *types.Baseline, writer io.Writer) error
}

// TableFormatter formats output as tables
type TableFormatter struct{}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	Pretty bool
}

// YAMLFormatter formats output as YAML
type YAMLFormatter struct{}

// MarkdownFormatter formats output as Markdown
type MarkdownFormatter struct{}

// NewFormatter creates a formatter based on format type
func NewFormatter(format string, pretty bool) (Formatter, error) {
	switch format {
	case "table":
		return &TableFormatter{}, nil
	case "json":
		return &JSONFormatter{Pretty: pretty}, nil
	case "yaml", "yml":
		return &YAMLFormatter{}, nil
	case "markdown", "md":
		return &MarkdownFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatSnapshot formats a snapshot for output
func (f *TableFormatter) FormatSnapshot(snapshot *types.Snapshot, writer io.Writer) error {
	fmt.Fprintf(writer, "Snapshot: %s\n", snapshot.ID)
	fmt.Fprintf(writer, "Provider: %s\n", snapshot.Provider)
	fmt.Fprintf(writer, "Timestamp: %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "Resources: %d\n", len(snapshot.Resources))
	return nil
}

// FormatDriftReport formats a drift report for output
func (f *TableFormatter) FormatDriftReport(report *types.DriftReport, writer io.Writer) error {
	fmt.Fprintf(writer, "Drift Report: %s\n", report.ID)
	fmt.Fprintf(writer, "Baseline: %s\n", report.BaselineID)
	fmt.Fprintf(writer, "Current: %s\n", report.CurrentID)
	fmt.Fprintf(writer, "Changes: %d\n", len(report.Changes))
	return nil
}

// FormatBaseline formats a baseline for output
func (f *TableFormatter) FormatBaseline(baseline *types.Baseline, writer io.Writer) error {
	fmt.Fprintf(writer, "Baseline: %s\n", baseline.Name)
	fmt.Fprintf(writer, "ID: %s\n", baseline.ID)
	fmt.Fprintf(writer, "Description: %s\n", baseline.Description)
	fmt.Fprintf(writer, "Created: %s\n", baseline.CreatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

// JSON Formatter implementations
func (f *JSONFormatter) FormatSnapshot(snapshot *types.Snapshot, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	if f.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(snapshot)
}

func (f *JSONFormatter) FormatDriftReport(report *types.DriftReport, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	if f.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(report)
}

func (f *JSONFormatter) FormatBaseline(baseline *types.Baseline, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	if f.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(baseline)
}

// YAML Formatter implementations
func (f *YAMLFormatter) FormatSnapshot(snapshot *types.Snapshot, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()
	return encoder.Encode(snapshot)
}

func (f *YAMLFormatter) FormatDriftReport(report *types.DriftReport, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()
	return encoder.Encode(report)
}

func (f *YAMLFormatter) FormatBaseline(baseline *types.Baseline, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()
	return encoder.Encode(baseline)
}

// Markdown Formatter implementations
func (f *MarkdownFormatter) FormatSnapshot(snapshot *types.Snapshot, writer io.Writer) error {
	fmt.Fprintf(writer, "# Snapshot: %s\n\n", snapshot.ID)
	fmt.Fprintf(writer, "- **Provider**: %s\n", snapshot.Provider)
	fmt.Fprintf(writer, "- **Timestamp**: %s\n", snapshot.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "- **Resources**: %d\n\n", len(snapshot.Resources))
	return nil
}

func (f *MarkdownFormatter) FormatDriftReport(report *types.DriftReport, writer io.Writer) error {
	fmt.Fprintf(writer, "# Drift Report: %s\n\n", report.ID)
	fmt.Fprintf(writer, "- **Baseline**: %s\n", report.BaselineID)
	fmt.Fprintf(writer, "- **Current**: %s\n", report.CurrentID)
	fmt.Fprintf(writer, "- **Changes**: %d\n\n", len(report.Changes))
	return nil
}

func (f *MarkdownFormatter) FormatBaseline(baseline *types.Baseline, writer io.Writer) error {
	fmt.Fprintf(writer, "# Baseline: %s\n\n", baseline.Name)
	fmt.Fprintf(writer, "- **ID**: %s\n", baseline.ID)
	fmt.Fprintf(writer, "- **Description**: %s\n", baseline.Description)
	fmt.Fprintf(writer, "- **Created**: %s\n\n", baseline.CreatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

// WriteToFile writes formatted output to a file
func WriteToFile(formatter Formatter, data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	switch v := data.(type) {
	case *types.Snapshot:
		return formatter.FormatSnapshot(v, file)
	case *types.DriftReport:
		return formatter.FormatDriftReport(v, file)
	case *types.Baseline:
		return formatter.FormatBaseline(v, file)
	default:
		return fmt.Errorf("unsupported data type for formatting")
	}
}