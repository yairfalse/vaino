package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/exports"
	"github.com/yairfalse/vaino/pkg/types"
	"gopkg.in/yaml.v3"
)

// CLIExportPlugin implements ExportPlugin for CLI output
type CLIExportPlugin struct {
	mu           sync.RWMutex
	config       exports.PluginConfig
	running      bool
	startTime    time.Time
	metrics      exports.PluginMetrics
	outputDir    string
	enableColors bool
	tableWriter  *TableWriter
}

// TableWriter handles formatted table output for CLI
type TableWriter struct {
	noColor   bool
	maxWidth  int
	separator string
}

// NewCLIExportPlugin creates a new CLI export plugin
func NewCLIExportPlugin() *CLIExportPlugin {
	return &CLIExportPlugin{
		config: exports.PluginConfig{
			Name:    "cli",
			Version: "1.0.0",
			Enabled: true,
			Settings: map[string]interface{}{
				"output_dir":       "./exports",
				"enable_colors":    true,
				"max_width":        120,
				"table_borders":    true,
				"timestamp_format": time.RFC3339,
			},
		},
		outputDir:    "./exports",
		enableColors: true,
		tableWriter: &TableWriter{
			maxWidth:  120,
			separator: " | ",
		},
		startTime: time.Now(),
	}
}

// Plugin metadata methods

func (p *CLIExportPlugin) Name() string {
	return "cli"
}

func (p *CLIExportPlugin) Version() string {
	return "1.0.0"
}

func (p *CLIExportPlugin) Description() string {
	return "Command-line interface export plugin for various output formats"
}

func (p *CLIExportPlugin) SupportedFormats() []exports.ExportFormat {
	return []exports.ExportFormat{
		exports.FormatJSON,
		exports.FormatYAML,
		exports.FormatMarkdown,
		exports.FormatCSV,
		exports.FormatCustom,
	}
}

// Plugin lifecycle methods

func (p *CLIExportPlugin) Initialize(ctx context.Context, config exports.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if config.Name != "" {
		p.config = config
	}

	// Extract settings
	if outputDir, ok := p.config.Settings["output_dir"].(string); ok {
		p.outputDir = outputDir
	}

	if enableColors, ok := p.config.Settings["enable_colors"].(bool); ok {
		p.enableColors = enableColors
		p.tableWriter.noColor = !enableColors
	}

	if maxWidth, ok := p.config.Settings["max_width"].(int); ok {
		p.tableWriter.maxWidth = maxWidth
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(p.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	p.metrics.MetricsUpdatedAt = time.Now()
	return nil
}

func (p *CLIExportPlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("CLI plugin already running")
	}

	p.running = true
	p.startTime = time.Now()

	return nil
}

func (p *CLIExportPlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.running = false
	return nil
}

func (p *CLIExportPlugin) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// Export methods

func (p *CLIExportPlugin) Export(ctx context.Context, request *exports.ExportRequest) (*exports.ExportResponse, error) {
	startTime := time.Now()
	defer func() {
		p.updateMetrics(time.Since(startTime))
	}()

	switch request.DataType {
	case exports.DataTypeDriftReport:
		if report, ok := request.Data.(*differ.DriftReport); ok {
			return p.exportDriftReport(ctx, report, request.Options)
		}
		return nil, fmt.Errorf("invalid data type for drift report export")

	case exports.DataTypeSnapshot:
		if snapshot, ok := request.Data.(*types.Snapshot); ok {
			return p.exportSnapshot(ctx, snapshot, request.Options)
		}
		return nil, fmt.Errorf("invalid data type for snapshot export")

	case exports.DataTypeCorrelation:
		if correlation, ok := request.Data.(*exports.CorrelationData); ok {
			return p.exportCorrelation(ctx, correlation, request.Options)
		}
		return nil, fmt.Errorf("invalid data type for correlation export")

	default:
		return nil, fmt.Errorf("unsupported data type: %s", request.DataType)
	}
}

func (p *CLIExportPlugin) ExportDriftReport(ctx context.Context, report *differ.DriftReport, options exports.ExportOptions) error {
	_, err := p.exportDriftReport(ctx, report, options)
	return err
}

func (p *CLIExportPlugin) ExportSnapshot(ctx context.Context, snapshot *types.Snapshot, options exports.ExportOptions) error {
	_, err := p.exportSnapshot(ctx, snapshot, options)
	return err
}

func (p *CLIExportPlugin) ExportCorrelation(ctx context.Context, correlation *exports.CorrelationData, options exports.ExportOptions) error {
	_, err := p.exportCorrelation(ctx, correlation, options)
	return err
}

// Internal export methods

func (p *CLIExportPlugin) exportDriftReport(ctx context.Context, report *differ.DriftReport, options exports.ExportOptions) (*exports.ExportResponse, error) {
	var data []byte
	var err error
	var contentType string

	switch options.Format {
	case exports.FormatJSON:
		data, err = p.formatDriftReportJSON(report, options.Pretty)
		contentType = "application/json"
	case exports.FormatYAML:
		data, err = p.formatDriftReportYAML(report)
		contentType = "application/yaml"
	case exports.FormatMarkdown:
		data, err = p.formatDriftReportMarkdown(report)
		contentType = "text/markdown"
	case exports.FormatCSV:
		data, err = p.formatDriftReportCSV(report)
		contentType = "text/csv"
	default:
		return nil, fmt.Errorf("unsupported format: %s", options.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to format drift report: %w", err)
	}

	// Write output
	outputPath, err := p.writeOutput(data, options, "drift-report", string(options.Format))
	if err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        data,
		ContentType: contentType,
		OutputPath:  outputPath,
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"report_id":     report.ID,
			"changes_count": len(report.ResourceChanges),
			"overall_risk":  report.Summary.OverallRisk,
		},
	}, nil
}

func (p *CLIExportPlugin) exportSnapshot(ctx context.Context, snapshot *types.Snapshot, options exports.ExportOptions) (*exports.ExportResponse, error) {
	var data []byte
	var err error
	var contentType string

	switch options.Format {
	case exports.FormatJSON:
		data, err = p.formatSnapshotJSON(snapshot, options.Pretty)
		contentType = "application/json"
	case exports.FormatYAML:
		data, err = p.formatSnapshotYAML(snapshot)
		contentType = "application/yaml"
	case exports.FormatMarkdown:
		data, err = p.formatSnapshotMarkdown(snapshot)
		contentType = "text/markdown"
	case exports.FormatCSV:
		data, err = p.formatSnapshotCSV(snapshot)
		contentType = "text/csv"
	default:
		return nil, fmt.Errorf("unsupported format: %s", options.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to format snapshot: %w", err)
	}

	// Write output
	outputPath, err := p.writeOutput(data, options, "snapshot", string(options.Format))
	if err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        data,
		ContentType: contentType,
		OutputPath:  outputPath,
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"snapshot_id":    snapshot.ID,
			"provider":       snapshot.Provider,
			"resource_count": len(snapshot.Resources),
		},
	}, nil
}

func (p *CLIExportPlugin) exportCorrelation(ctx context.Context, correlation *exports.CorrelationData, options exports.ExportOptions) (*exports.ExportResponse, error) {
	var data []byte
	var err error
	var contentType string

	switch options.Format {
	case exports.FormatJSON:
		data, err = p.formatCorrelationJSON(correlation, options.Pretty)
		contentType = "application/json"
	case exports.FormatYAML:
		data, err = p.formatCorrelationYAML(correlation)
		contentType = "application/yaml"
	case exports.FormatMarkdown:
		data, err = p.formatCorrelationMarkdown(correlation)
		contentType = "text/markdown"
	default:
		return nil, fmt.Errorf("unsupported format: %s", options.Format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to format correlation: %w", err)
	}

	// Write output
	outputPath, err := p.writeOutput(data, options, "correlation", string(options.Format))
	if err != nil {
		return nil, fmt.Errorf("failed to write output: %w", err)
	}

	return &exports.ExportResponse{
		Status:      exports.StatusCompleted,
		Data:        data,
		ContentType: contentType,
		OutputPath:  outputPath,
		ProcessedAt: time.Now(),
		Metadata: map[string]interface{}{
			"correlation_id":     correlation.ID,
			"correlations_count": len(correlation.Correlations),
			"risk_score":         correlation.Summary.RiskScore,
		},
	}, nil
}

// Format methods for different data types

func (p *CLIExportPlugin) formatDriftReportJSON(report *differ.DriftReport, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(report, "", "  ")
	}
	return json.Marshal(report)
}

func (p *CLIExportPlugin) formatDriftReportYAML(report *differ.DriftReport) ([]byte, error) {
	return yaml.Marshal(report)
}

func (p *CLIExportPlugin) formatDriftReportMarkdown(report *differ.DriftReport) ([]byte, error) {
	var md strings.Builder

	md.WriteString("# Infrastructure Drift Report\n\n")
	md.WriteString(fmt.Sprintf("**Report ID:** %s\n", report.ID))
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

	// Changes by severity
	if len(report.Summary.ChangesBySeverity) > 0 {
		md.WriteString("### Changes by Severity\n\n")
		severities := []differ.RiskLevel{differ.RiskLevelCritical, differ.RiskLevelHigh, differ.RiskLevelMedium, differ.RiskLevelLow}
		for _, severity := range severities {
			if count, exists := report.Summary.ChangesBySeverity[severity]; exists && count > 0 {
				emoji := p.getRiskEmoji(severity)
				md.WriteString(fmt.Sprintf("- %s **%s:** %d\n", emoji, strings.Title(string(severity)), count))
			}
		}
		md.WriteString("\n")
	}

	return []byte(md.String()), nil
}

func (p *CLIExportPlugin) formatDriftReportCSV(report *differ.DriftReport) ([]byte, error) {
	var csv strings.Builder

	csv.WriteString("ResourceID,ResourceType,ChangeType,Severity,Category,RiskScore,Description\n")

	for _, change := range report.ResourceChanges {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%.2f,\"%s\"\n",
			p.escapeCSV(change.ResourceID),
			p.escapeCSV(change.ResourceType),
			p.escapeCSV(string(change.DriftType)),
			p.escapeCSV(string(change.Severity)),
			p.escapeCSV(string(change.Category)),
			change.RiskScore,
			p.escapeCSV(change.Description)))
	}

	return []byte(csv.String()), nil
}

func (p *CLIExportPlugin) formatSnapshotJSON(snapshot *types.Snapshot, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(snapshot, "", "  ")
	}
	return json.Marshal(snapshot)
}

func (p *CLIExportPlugin) formatSnapshotYAML(snapshot *types.Snapshot) ([]byte, error) {
	return yaml.Marshal(snapshot)
}

func (p *CLIExportPlugin) formatSnapshotMarkdown(snapshot *types.Snapshot) ([]byte, error) {
	var md strings.Builder

	md.WriteString("# Infrastructure Snapshot\n\n")
	md.WriteString(fmt.Sprintf("**Snapshot ID:** %s\n", snapshot.ID))
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

func (p *CLIExportPlugin) formatSnapshotCSV(snapshot *types.Snapshot) ([]byte, error) {
	var csv strings.Builder

	csv.WriteString("ID,Type,Name,Provider,Region,Namespace\n")
	for _, resource := range snapshot.Resources {
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			p.escapeCSV(resource.ID),
			p.escapeCSV(resource.Type),
			p.escapeCSV(resource.Name),
			p.escapeCSV(resource.Provider),
			p.escapeCSV(resource.Region),
			p.escapeCSV(resource.Namespace)))
	}

	return []byte(csv.String()), nil
}

func (p *CLIExportPlugin) formatCorrelationJSON(correlation *exports.CorrelationData, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(correlation, "", "  ")
	}
	return json.Marshal(correlation)
}

func (p *CLIExportPlugin) formatCorrelationYAML(correlation *exports.CorrelationData) ([]byte, error) {
	return yaml.Marshal(correlation)
}

func (p *CLIExportPlugin) formatCorrelationMarkdown(correlation *exports.CorrelationData) ([]byte, error) {
	var md strings.Builder

	md.WriteString("# Correlation Analysis Report\n\n")
	md.WriteString(fmt.Sprintf("**Correlation ID:** %s\n", correlation.ID))
	md.WriteString(fmt.Sprintf("**Generated:** %s\n", correlation.Timestamp.Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("**Total Correlations:** %d\n", len(correlation.Correlations)))
	md.WriteString(fmt.Sprintf("**Risk Score:** %.2f\n\n", correlation.Summary.RiskScore))

	// Summary
	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **High Confidence:** %d\n", correlation.Summary.HighConfidence))
	md.WriteString(fmt.Sprintf("- **Medium Confidence:** %d\n", correlation.Summary.MediumConfidence))
	md.WriteString(fmt.Sprintf("- **Low Confidence:** %d\n", correlation.Summary.LowConfidence))
	md.WriteString(fmt.Sprintf("- **Critical Findings:** %d\n\n", correlation.Summary.CriticalFindings))

	// Top correlations
	if len(correlation.Summary.TopCorrelations) > 0 {
		md.WriteString("### Top Correlations\n\n")
		for i, corr := range correlation.Summary.TopCorrelations {
			md.WriteString(fmt.Sprintf("%d. %s\n", i+1, corr))
		}
		md.WriteString("\n")
	}

	return []byte(md.String()), nil
}

// Utility methods

func (p *CLIExportPlugin) writeOutput(data []byte, options exports.ExportOptions, prefix, extension string) (string, error) {
	// Determine output destination
	if options.Writer != nil {
		_, err := options.Writer.Write(data)
		return "", err
	}

	if options.OutputPath == "" || options.OutputPath == "-" {
		// Write to stdout
		_, err := os.Stdout.Write(data)
		return "", err
	}

	// Generate filename if not provided
	outputPath := options.OutputPath
	if outputPath == "" {
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("%s-%s.%s", prefix, timestamp, extension)
		outputPath = filepath.Join(p.outputDir, filename)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file
	err := os.WriteFile(outputPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return outputPath, nil
}

func (p *CLIExportPlugin) getRiskEmoji(risk differ.RiskLevel) string {
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

func (p *CLIExportPlugin) escapeCSV(s string) string {
	if strings.Contains(s, ",") || strings.Contains(s, "\"") || strings.Contains(s, "\n") {
		s = strings.ReplaceAll(s, "\"", "\"\"")
		return "\"" + s + "\""
	}
	return s
}

func (p *CLIExportPlugin) updateMetrics(duration time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.TotalRequests++
	p.metrics.SuccessfulExports++
	p.metrics.LastExport = time.Now()

	// Update latency metrics (simplified)
	if p.metrics.AverageLatency == 0 {
		p.metrics.AverageLatency = duration
	} else {
		p.metrics.AverageLatency = (p.metrics.AverageLatency + duration) / 2
	}

	p.metrics.MetricsUpdatedAt = time.Now()
}

// Plugin validation and health methods

func (p *CLIExportPlugin) Validate(config exports.PluginConfig) error {
	// Validate required settings
	if outputDir, ok := config.Settings["output_dir"].(string); ok {
		if outputDir == "" {
			return fmt.Errorf("output_dir cannot be empty")
		}

		// Check if directory is writable
		testFile := filepath.Join(outputDir, ".test")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("cannot create output directory: %w", err)
		}

		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("output directory not writable: %w", err)
		}
		os.Remove(testFile)
	}

	return nil
}

func (p *CLIExportPlugin) HealthCheck(ctx context.Context) exports.HealthStatus {
	status := exports.HealthStatus{
		Status:    "healthy",
		LastCheck: time.Now(),
		Message:   "CLI plugin operational",
		Uptime:    time.Since(p.startTime),
		Version:   p.Version(),
		Details:   make(map[string]interface{}),
	}

	// Check if output directory is accessible
	if _, err := os.Stat(p.outputDir); err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("Output directory not accessible: %v", err)
		return status
	}

	// Check if directory is writable
	testFile := filepath.Join(p.outputDir, ".health_check")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		status.Status = "degraded"
		status.Message = fmt.Sprintf("Output directory not writable: %v", err)
	} else {
		os.Remove(testFile)
	}

	status.Details["output_dir"] = p.outputDir
	status.Details["enable_colors"] = p.enableColors
	status.Details["running"] = p.running

	return status
}

func (p *CLIExportPlugin) GetMetrics() exports.PluginMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.metrics
}

// Configuration methods

func (p *CLIExportPlugin) Schema() exports.PluginSchema {
	return exports.PluginSchema{
		Name:        "cli",
		Version:     "1.0.0",
		Description: "Command-line interface export plugin",
		Schema: map[string]exports.SchemaField{
			"output_dir": {
				Type:        "string",
				Description: "Directory to write exported files",
				Default:     "./exports",
				Required:    false,
			},
			"enable_colors": {
				Type:        "bool",
				Description: "Enable colored output",
				Default:     true,
				Required:    false,
			},
			"max_width": {
				Type:        "int",
				Description: "Maximum width for table output",
				Default:     120,
				Minimum:     &[]float64{80}[0],
				Maximum:     &[]float64{200}[0],
				Required:    false,
			},
			"table_borders": {
				Type:        "bool",
				Description: "Enable table borders",
				Default:     true,
				Required:    false,
			},
			"timestamp_format": {
				Type:        "string",
				Description: "Format for timestamps",
				Default:     time.RFC3339,
				Required:    false,
			},
		},
		Required: []string{},
		Examples: []map[string]interface{}{
			{
				"output_dir":       "./my-exports",
				"enable_colors":    true,
				"max_width":        100,
				"table_borders":    false,
				"timestamp_format": "2006-01-02 15:04:05",
			},
		},
	}
}

func (p *CLIExportPlugin) GetConfig() exports.PluginConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

func (p *CLIExportPlugin) UpdateConfig(ctx context.Context, config exports.PluginConfig) error {
	if err := p.Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config

	// Update internal settings
	if outputDir, ok := config.Settings["output_dir"].(string); ok {
		p.outputDir = outputDir
	}

	if enableColors, ok := config.Settings["enable_colors"].(bool); ok {
		p.enableColors = enableColors
		p.tableWriter.noColor = !enableColors
	}

	if maxWidth, ok := config.Settings["max_width"].(int); ok {
		p.tableWriter.maxWidth = maxWidth
	}

	return nil
}
