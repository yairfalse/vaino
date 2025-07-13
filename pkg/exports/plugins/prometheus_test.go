package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/exports"
	"github.com/yairfalse/vaino/pkg/types"
)

func TestPrometheusExportPlugin(t *testing.T) {
	plugin := NewPrometheusExportPlugin()

	// Test basic plugin properties
	if plugin.Name() != "prometheus" {
		t.Errorf("Expected plugin name 'prometheus', got %s", plugin.Name())
	}

	if plugin.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", plugin.Version())
	}

	formats := plugin.SupportedFormats()
	if len(formats) != 1 || formats[0] != exports.FormatPrometheus {
		t.Errorf("Expected FormatPrometheus, got %v", formats)
	}
}

func TestPrometheusPluginLifecycle(t *testing.T) {
	plugin := NewPrometheusExportPlugin()
	ctx := context.Background()

	// Test initialization
	err := plugin.Initialize(ctx, exports.PluginConfig{})
	if err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}

	// Test start
	err = plugin.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start plugin: %v", err)
	}

	if !plugin.IsRunning() {
		t.Error("Plugin should be running after start")
	}

	// Test stop
	err = plugin.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop plugin: %v", err)
	}

	if plugin.IsRunning() {
		t.Error("Plugin should not be running after stop")
	}
}

func TestPrometheusDriftReportExport(t *testing.T) {
	plugin := NewPrometheusExportPlugin()
	ctx := context.Background()

	// Initialize and start plugin
	plugin.Initialize(ctx, exports.PluginConfig{})
	plugin.Start(ctx)
	defer plugin.Stop(ctx)

	// Create test drift report
	report := &differ.DriftReport{
		ID:         "test-report",
		BaselineID: "baseline-1",
		CurrentID:  "current-1",
		Timestamp:  time.Now(),
		Summary: differ.DriftSummary{
			TotalResources:    100,
			ChangedResources:  5,
			AddedResources:    2,
			RemovedResources:  1,
			ModifiedResources: 2,
			RiskScore:         7.5,
			OverallRisk:       differ.RiskLevelHigh,
			ChangesBySeverity: map[differ.RiskLevel]int{
				differ.RiskLevelHigh:   2,
				differ.RiskLevelMedium: 2,
				differ.RiskLevelLow:    1,
			},
		},
		ResourceChanges: []differ.ResourceDiff{
			{
				ResourceID:   "resource-1",
				ResourceType: "aws_instance",
				DriftType:    differ.ChangeTypeModified,
				Severity:     differ.RiskLevelHigh,
				Category:     differ.DriftCategoryConfig,
				RiskScore:    8.0,
				Description:  "Instance type changed",
			},
		},
	}

	// Test export
	request := &exports.ExportRequest{
		ID:       "test-export",
		DataType: exports.DataTypeDriftReport,
		Data:     report,
		Options:  exports.ExportOptions{Format: exports.FormatPrometheus},
	}

	response, err := plugin.Export(ctx, request)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if response.Status != exports.StatusCompleted {
		t.Errorf("Expected status completed, got %s", response.Status)
	}

	if response.ContentType != "text/plain; version=0.0.4" {
		t.Errorf("Expected Prometheus content type, got %s", response.ContentType)
	}

	if len(response.Data) == 0 {
		t.Error("Expected non-empty response data")
	}

	// Verify that the response contains Prometheus metrics
	data := string(response.Data)
	if !contains(data, "vaino_drift_report_risk_score") {
		t.Error("Expected drift report risk score metric in output")
	}

	if !contains(data, "vaino_drift_report_total_resources") {
		t.Error("Expected total resources metric in output")
	}

	if !contains(data, "vaino_drift_resource_changes_total") {
		t.Error("Expected resource changes counter in output")
	}
}

func TestPrometheusSnapshotExport(t *testing.T) {
	plugin := NewPrometheusExportPlugin()
	ctx := context.Background()

	// Initialize and start plugin
	plugin.Initialize(ctx, exports.PluginConfig{})
	plugin.Start(ctx)
	defer plugin.Stop(ctx)

	// Create test snapshot
	snapshot := &types.Snapshot{
		ID:        "test-snapshot",
		Provider:  "aws",
		Timestamp: time.Now(),
		Resources: []types.Resource{
			{
				ID:       "resource-1",
				Type:     "aws_instance",
				Name:     "web-server-1",
				Provider: "aws",
				Region:   "us-east-1",
			},
			{
				ID:       "resource-2",
				Type:     "aws_instance",
				Name:     "web-server-2",
				Provider: "aws",
				Region:   "us-east-1",
			},
			{
				ID:       "resource-3",
				Type:     "aws_rds_instance",
				Name:     "database-1",
				Provider: "aws",
				Region:   "us-east-1",
			},
		},
	}

	// Test export
	request := &exports.ExportRequest{
		ID:       "test-snapshot-export",
		DataType: exports.DataTypeSnapshot,
		Data:     snapshot,
		Options:  exports.ExportOptions{Format: exports.FormatPrometheus},
	}

	response, err := plugin.Export(ctx, request)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if response.Status != exports.StatusCompleted {
		t.Errorf("Expected status completed, got %s", response.Status)
	}

	// Verify that the response contains expected metrics
	data := string(response.Data)
	if !contains(data, "vaino_snapshot_resources_total") {
		t.Error("Expected snapshot total resources metric in output")
	}

	if !contains(data, "vaino_snapshot_resources_by_type") {
		t.Error("Expected resources by type metric in output")
	}
}

func TestPrometheusHealthCheck(t *testing.T) {
	plugin := NewPrometheusExportPlugin()
	ctx := context.Background()

	// Initialize and start plugin
	plugin.Initialize(ctx, exports.PluginConfig{})
	plugin.Start(ctx)
	defer plugin.Stop(ctx)

	// Test health check
	health := plugin.HealthCheck(ctx)
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", health.Version)
	}

	// Check health details
	if health.Details["push_gateway"] == nil {
		t.Error("Expected push_gateway in health details")
	}

	if health.Details["running"] != true {
		t.Error("Expected running=true in health details")
	}
}

func TestPrometheusConfigValidation(t *testing.T) {
	plugin := NewPrometheusExportPlugin()

	// Test valid config
	validConfig := exports.PluginConfig{
		Name:    "prometheus",
		Version: "1.0.0",
		Enabled: true,
		Settings: map[string]interface{}{
			"push_gateway": "http://localhost:9091",
			"job_name":     "test-job",
			"instance":     "test-instance",
			"timeout":      "30s",
		},
	}

	err := plugin.Validate(validConfig)
	if err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}

	// Test invalid config - empty push_gateway
	invalidConfig := validConfig
	invalidConfig.Settings["push_gateway"] = ""
	err = plugin.Validate(invalidConfig)
	if err == nil {
		t.Error("Empty push_gateway should fail validation")
	}

	// Test invalid config - bad timeout
	invalidConfig2 := validConfig
	invalidConfig2.Settings["timeout"] = "invalid"
	err = plugin.Validate(invalidConfig2)
	if err == nil {
		t.Error("Invalid timeout should fail validation")
	}
}

func TestPrometheusMetricGeneration(t *testing.T) {
	plugin := NewPrometheusExportPlugin()

	// Test metric name sanitization
	sanitized := plugin.sanitizeMetricName("test-metric.with_special-chars")
	expected := "test_metric_with_special_chars"
	if sanitized != expected {
		t.Errorf("Expected %s, got %s", expected, sanitized)
	}

	// Test float conversion
	val, ok := plugin.convertToFloat64(42)
	if !ok || val != 42.0 {
		t.Errorf("Expected 42.0, got %v (ok=%v)", val, ok)
	}

	val, ok = plugin.convertToFloat64("invalid")
	if ok {
		t.Error("String should not convert to float64")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			contains(s[1:], substr) ||
			(len(s) > 0 && s[:len(substr)] == substr))
}
