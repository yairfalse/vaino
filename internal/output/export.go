package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
	"gopkg.in/yaml.v3"
)

// ExportManager handles exporting data to various formats
type ExportManager struct {
	atomicWriter *storage.AtomicWriter
	tableRenderer *EnhancedTableRenderer
	noColor      bool
}

// NewExportManager creates a new export manager
func NewExportManager(atomicWriter *storage.AtomicWriter, noColor bool) *ExportManager {
	return &ExportManager{
		atomicWriter:  atomicWriter,
		tableRenderer: NewEnhancedTableRenderer(noColor, 120),
		noColor:      noColor,
	}
}

// ExportOptions configures export behavior
type ExportOptions struct {
	Format      string            // json, yaml, markdown, csv, table
	OutputPath  string            // file path or "-" for stdout
	Compress    bool              // whether to compress output
	Pretty      bool              // pretty print JSON/YAML
	Template    string            // custom template path
	Metadata    map[string]string // additional metadata
	FilterLevel string            // minimum severity level
}

// ExportDriftReport exports a drift report in the specified format
func (e *ExportManager) ExportDriftReport(report *differ.DriftReport, options ExportOptions) error {
	var data []byte
	var err error

	// Filter report if needed
	filteredReport := e.filterReport(report, options.FilterLevel)

	// Generate output based on format
	switch strings.ToLower(options.Format) {
	case "json":
		data, err = e.exportToJSON(filteredReport, options.Pretty)
	case "yaml", "yml":
		data, err = e.exportToYAML(filteredReport)
	case "markdown", "md":
		data, err = e.exportToMarkdown(filteredReport)
	case "csv":
		data, err = e.exportToCSV(filteredReport)
	case "table":
		data = []byte(e.tableRenderer.RenderDriftReport(filteredReport))
	case "html":
		data, err = e.exportToHTML(filteredReport)
	default:
		return fmt.Errorf("unsupported export format: %s", options.Format)
	}

	if err != nil {
		return fmt.Errorf("failed to export to %s: %w", options.Format, err)
	}

	// Write output
	return e.writeOutput(data, options)
}

// ExportSnapshot exports a snapshot in the specified format
func (e *ExportManager) ExportSnapshot(snapshot *types.Snapshot, options ExportOptions) error {
	var data []byte
	var err error

	switch strings.ToLower(options.Format) {
	case "json":
		data, err = e.exportSnapshotToJSON(snapshot, options.Pretty)
	case "yaml", "yml":
		data, err = e.exportSnapshotToYAML(snapshot)
	case "markdown", "md":
		data, err = e.exportSnapshotToMarkdown(snapshot)
	case "csv":
		data, err = e.exportSnapshotToCSV(snapshot)
	default:
		return fmt.Errorf("unsupported export format for snapshot: %s", options.Format)
	}

	if err != nil {
		return fmt.Errorf("failed to export snapshot to %s: %w", options.Format, err)
	}

	return e.writeOutput(data, options)
}

// filterReport filters a drift report based on severity level
func (e *ExportManager) filterReport(report *differ.DriftReport, filterLevel string) *differ.DriftReport {
	if filterLevel == "" {
		return report
	}

	minLevel := e.parseRiskLevel(filterLevel)
	if minLevel == "" {
		return report
	}

	// Create filtered copy
	filtered := &differ.DriftReport{
		ID:        report.ID,
		BaselineID: report.BaselineID,
		CurrentID: report.CurrentID,
		Timestamp: report.Timestamp,
		Metadata:  report.Metadata,
	}

	// Filter resource changes
	var filteredChanges []differ.ResourceDiff
	for _, change := range report.ResourceChanges {
		if e.shouldIncludeChange(change.Severity, minLevel) {
			filteredChanges = append(filteredChanges, change)
		}
	}
	filtered.ResourceChanges = filteredChanges

	// Recalculate summary
	filtered.Summary = e.recalculateSummary(filteredChanges)

	return filtered
}

// exportToJSON exports data to JSON format
func (e *ExportManager) exportToJSON(data interface{}, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}

// exportToYAML exports data to YAML format
func (e *ExportManager) exportToYAML(data interface{}) ([]byte, error) {
	return yaml.Marshal(data)
}

// exportToMarkdown exports drift report to Markdown format
func (e *ExportManager) exportToMarkdown(report *differ.DriftReport) ([]byte, error) {
	var md strings.Builder

	// Header
	md.WriteString("# Infrastructure Drift Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("**Baseline ID:** %s\n", report.BaselineID))
	md.WriteString(fmt.Sprintf("**Current ID:** %s\n", report.CurrentID))
	md.WriteString(fmt.Sprintf("**Overall Risk:** %s (%.2f)\n\n", report.Summary.OverallRisk, report.Summary.RiskScore))

	// Summary
	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Total Resources:** %d\n", report.Summary.TotalResources))
	md.WriteString(fmt.Sprintf("- **Changed Resources:** %d\n", report.Summary.ChangedResources))
	md.WriteString(fmt.Sprintf("- **Added Resources:** %d\n", report.Summary.AddedResources))
	md.WriteString(fmt.Sprintf("- **Removed Resources:** %d\n", report.Summary.RemovedResources))
	md.WriteString(fmt.Sprintf("- **Modified Resources:** %d\n\n", report.Summary.ModifiedResources))

	// Severity breakdown
	if len(report.Summary.ChangesBySeverity) > 0 {
		md.WriteString("### Changes by Severity\n\n")
		severities := []differ.RiskLevel{differ.RiskLevelCritical, differ.RiskLevelHigh, differ.RiskLevelMedium, differ.RiskLevelLow}
		for _, severity := range severities {
			if count, exists := report.Summary.ChangesBySeverity[severity]; exists && count > 0 {
				emoji := e.getRiskEmoji(severity)
				md.WriteString(fmt.Sprintf("- %s **%s:** %d\n", emoji, strings.Title(string(severity)), count))
			}
		}
		md.WriteString("\n")
	}

	// Detailed changes
	if len(report.ResourceChanges) > 0 {
		md.WriteString("## Detailed Changes\n\n")
		
		// Group by severity
		bySeverity := make(map[differ.RiskLevel][]differ.ResourceDiff)
		for _, change := range report.ResourceChanges {
			bySeverity[change.Severity] = append(bySeverity[change.Severity], change)
		}

		severities := []differ.RiskLevel{differ.RiskLevelCritical, differ.RiskLevelHigh, differ.RiskLevelMedium, differ.RiskLevelLow}
		for _, severity := range severities {
			changes, exists := bySeverity[severity]
			if !exists || len(changes) == 0 {
				continue
			}

			emoji := e.getRiskEmoji(severity)
			md.WriteString(fmt.Sprintf("### %s %s Risk Changes\n\n", emoji, strings.Title(string(severity))))

			for _, change := range changes {
				md.WriteString(fmt.Sprintf("#### %s (%s)\n\n", change.ResourceID, change.ResourceType))
				md.WriteString(fmt.Sprintf("- **Change Type:** %s\n", strings.Title(string(change.DriftType))))
				md.WriteString(fmt.Sprintf("- **Category:** %s\n", strings.Title(string(change.Category))))
				md.WriteString(fmt.Sprintf("- **Risk Score:** %.2f\n", change.RiskScore))
				md.WriteString(fmt.Sprintf("- **Description:** %s\n\n", change.Description))

				if len(change.Changes) > 0 {
					md.WriteString("**Field Changes:**\n\n")
					for _, fieldChange := range change.Changes {
						md.WriteString(fmt.Sprintf("- **%s:** `%v` → `%v`\n", fieldChange.Field, fieldChange.OldValue, fieldChange.NewValue))
					}
					md.WriteString("\n")
				}
			}
		}
	}

	return []byte(md.String()), nil
}

// exportToCSV exports drift report to CSV format
func (e *ExportManager) exportToCSV(report *differ.DriftReport) ([]byte, error) {
	var csv strings.Builder

	// Header
	csv.WriteString("ResourceID,ResourceType,ChangeType,Severity,Category,RiskScore,Description\n")

	// Data rows
	for _, change := range report.ResourceChanges {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%.2f,\"%s\"\n",
			e.escapeCSV(change.ResourceID),
			e.escapeCSV(change.ResourceType),
			e.escapeCSV(string(change.DriftType)),
			e.escapeCSV(string(change.Severity)),
			e.escapeCSV(string(change.Category)),
			change.RiskScore,
			e.escapeCSV(change.Description)))
	}

	return []byte(csv.String()), nil
}

// exportToHTML exports drift report to HTML format
func (e *ExportManager) exportToHTML(report *differ.DriftReport) ([]byte, error) {
	var html strings.Builder

	// HTML template
	html.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>Infrastructure Drift Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .changes { margin: 20px 0; }
        .change-item { margin: 10px 0; padding: 15px; border-left: 4px solid; }
        .critical { border-color: #d32f2f; background: #ffebee; }
        .high { border-color: #f57c00; background: #fff3e0; }
        .medium { border-color: #1976d2; background: #e3f2fd; }
        .low { border-color: #388e3c; background: #e8f5e9; }
        .risk-badge { padding: 2px 8px; border-radius: 3px; color: white; font-size: 12px; }
        .risk-critical { background: #d32f2f; }
        .risk-high { background: #f57c00; }
        .risk-medium { background: #1976d2; }
        .risk-low { background: #388e3c; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f5f5f5; }
    </style>
</head>
<body>`)

	// Header
	html.WriteString(fmt.Sprintf(`
    <div class="header">
        <h1>Infrastructure Drift Report</h1>
        <p><strong>Generated:</strong> %s</p>
        <p><strong>Baseline ID:</strong> %s</p>
        <p><strong>Current ID:</strong> %s</p>
        <p><strong>Overall Risk:</strong> <span class="risk-badge risk-%s">%s</span> (%.2f)</p>
    </div>`,
		time.Now().Format(time.RFC3339),
		report.BaselineID,
		report.CurrentID,
		strings.ToLower(string(report.Summary.OverallRisk)),
		strings.ToUpper(string(report.Summary.OverallRisk)),
		report.Summary.RiskScore))

	// Summary table
	html.WriteString(`
    <div class="summary">
        <h2>Summary</h2>
        <table>
            <tr><th>Metric</th><th>Count</th></tr>`)
	
	html.WriteString(fmt.Sprintf(`
            <tr><td>Total Resources</td><td>%d</td></tr>
            <tr><td>Changed Resources</td><td>%d</td></tr>
            <tr><td>Added Resources</td><td>%d</td></tr>
            <tr><td>Removed Resources</td><td>%d</td></tr>
            <tr><td>Modified Resources</td><td>%d</td></tr>`,
		report.Summary.TotalResources,
		report.Summary.ChangedResources,
		report.Summary.AddedResources,
		report.Summary.RemovedResources,
		report.Summary.ModifiedResources))
	
	html.WriteString(`
        </table>
    </div>`)

	// Changes
	if len(report.ResourceChanges) > 0 {
		html.WriteString(`
    <div class="changes">
        <h2>Detailed Changes</h2>`)

		for _, change := range report.ResourceChanges {
			riskClass := strings.ToLower(string(change.Severity))
			html.WriteString(fmt.Sprintf(`
        <div class="change-item %s">
            <h3>%s <span class="risk-badge risk-%s">%s</span></h3>
            <p><strong>Type:</strong> %s</p>
            <p><strong>Change:</strong> %s</p>
            <p><strong>Category:</strong> %s</p>
            <p><strong>Risk Score:</strong> %.2f</p>
            <p><strong>Description:</strong> %s</p>
        </div>`,
				riskClass,
				change.ResourceID,
				riskClass,
				strings.ToUpper(string(change.Severity)),
				change.ResourceType,
				strings.Title(string(change.DriftType)),
				strings.Title(string(change.Category)),
				change.RiskScore,
				change.Description))
		}

		html.WriteString(`
    </div>`)
	}

	html.WriteString(`
</body>
</html>`)

	return []byte(html.String()), nil
}

// exportSnapshotToJSON exports snapshot to JSON
func (e *ExportManager) exportSnapshotToJSON(snapshot *types.Snapshot, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(snapshot, "", "  ")
	}
	return json.Marshal(snapshot)
}

// exportSnapshotToYAML exports snapshot to YAML
func (e *ExportManager) exportSnapshotToYAML(snapshot *types.Snapshot) ([]byte, error) {
	return yaml.Marshal(snapshot)
}

// exportSnapshotToMarkdown exports snapshot to Markdown
func (e *ExportManager) exportSnapshotToMarkdown(snapshot *types.Snapshot) ([]byte, error) {
	var md strings.Builder

	md.WriteString("# Infrastructure Snapshot\n\n")
	md.WriteString(fmt.Sprintf("**ID:** %s\n", snapshot.ID))
	md.WriteString(fmt.Sprintf("**Provider:** %s\n", snapshot.Provider))
	md.WriteString(fmt.Sprintf("**Timestamp:** %s\n", snapshot.Timestamp.Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("**Resource Count:** %d\n\n", len(snapshot.Resources)))

	// Group resources by type
	byType := make(map[string][]types.Resource)
	for _, resource := range snapshot.Resources {
		byType[resource.Type] = append(byType[resource.Type], resource)
	}

	md.WriteString("## Resources by Type\n\n")
	for resourceType, resources := range byType {
		md.WriteString(fmt.Sprintf("### %s (%d)\n\n", resourceType, len(resources)))
		for _, resource := range resources {
			md.WriteString(fmt.Sprintf("- **%s** (`%s`)", resource.Name, resource.ID))
			if resource.Region != "" {
				md.WriteString(fmt.Sprintf(" - Region: %s", resource.Region))
			}
			md.WriteString("\n")
		}
		md.WriteString("\n")
	}

	return []byte(md.String()), nil
}

// exportSnapshotToCSV exports snapshot to CSV
func (e *ExportManager) exportSnapshotToCSV(snapshot *types.Snapshot) ([]byte, error) {
	var csv strings.Builder

	csv.WriteString("ID,Type,Name,Provider,Region,Namespace\n")
	for _, resource := range snapshot.Resources {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			e.escapeCSV(resource.ID),
			e.escapeCSV(resource.Type),
			e.escapeCSV(resource.Name),
			e.escapeCSV(resource.Provider),
			e.escapeCSV(resource.Region),
			e.escapeCSV(resource.Namespace)))
	}

	return []byte(csv.String()), nil
}

// writeOutput writes data to the specified output
func (e *ExportManager) writeOutput(data []byte, options ExportOptions) error {
	if options.OutputPath == "" || options.OutputPath == "-" {
		// Write to stdout
		_, err := os.Stdout.Write(data)
		return err
	}

	// Write to file
	return e.atomicWriter.WriteFile(options.OutputPath, data, 0644)
}

// Helper methods

func (e *ExportManager) parseRiskLevel(level string) differ.RiskLevel {
	switch strings.ToLower(level) {
	case "critical":
		return differ.RiskLevelCritical
	case "high":
		return differ.RiskLevelHigh
	case "medium":
		return differ.RiskLevelMedium
	case "low":
		return differ.RiskLevelLow
	default:
		return ""
	}
}

func (e *ExportManager) shouldIncludeChange(severity, minLevel differ.RiskLevel) bool {
	severityWeight := map[differ.RiskLevel]int{
		differ.RiskLevelLow:      1,
		differ.RiskLevelMedium:   2,
		differ.RiskLevelHigh:     3,
		differ.RiskLevelCritical: 4,
	}

	return severityWeight[severity] >= severityWeight[minLevel]
}

func (e *ExportManager) recalculateSummary(changes []differ.ResourceDiff) differ.DriftSummary {
	summary := differ.DriftSummary{
		ChangesBySeverity: make(map[differ.RiskLevel]int),
		ChangesByCategory: make(map[differ.DriftCategory]int),
	}

	for _, change := range changes {
		summary.ChangedResources++
		
		switch change.DriftType {
		case differ.ChangeTypeAdded:
			summary.AddedResources++
		case differ.ChangeTypeRemoved:
			summary.RemovedResources++
		case differ.ChangeTypeModified:
			summary.ModifiedResources++
		}

		summary.ChangesBySeverity[change.Severity]++
		summary.ChangesByCategory[change.Category]++
	}

	// Calculate overall risk (simplified)
	if summary.ChangesBySeverity[differ.RiskLevelCritical] > 0 {
		summary.OverallRisk = differ.RiskLevelCritical
		summary.RiskScore = 0.9
	} else if summary.ChangesBySeverity[differ.RiskLevelHigh] > 0 {
		summary.OverallRisk = differ.RiskLevelHigh
		summary.RiskScore = 0.7
	} else if summary.ChangesBySeverity[differ.RiskLevelMedium] > 0 {
		summary.OverallRisk = differ.RiskLevelMedium
		summary.RiskScore = 0.5
	} else {
		summary.OverallRisk = differ.RiskLevelLow
		summary.RiskScore = 0.2
	}

	return summary
}

func (e *ExportManager) getRiskEmoji(risk differ.RiskLevel) string {
	switch risk {
	case differ.RiskLevelCritical:
		return "🔴"
	case differ.RiskLevelHigh:
		return "🟡"
	case differ.RiskLevelMedium:
		return "🔵"
	case differ.RiskLevelLow:
		return "🟢"
	default:
		return "⚪"
	}
}

func (e *ExportManager) escapeCSV(s string) string {
	if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
		s = strings.ReplaceAll(s, "\"", "\"\"")
		return "\"" + s + "\""
	}
	return s
}