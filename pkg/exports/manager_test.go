package exports

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// MockExportPlugin is a mock implementation of ExportPlugin for testing
type MockExportPlugin struct {
	name             string
	version          string
	supportedFormats []ExportFormat
	running          bool
	initialized      bool
	exportCalls      int
	healthStatus     HealthStatus
	metrics          PluginMetrics
	config           PluginConfig
	shouldFail       bool
	exportDelay      time.Duration
}

func NewMockExportPlugin(name string) *MockExportPlugin {
	return &MockExportPlugin{
		name:    name,
		version: "1.0.0",
		supportedFormats: []ExportFormat{
			FormatJSON,
			FormatYAML,
		},
		healthStatus: HealthStatus{
			Status:  "healthy",
			Message: "Mock plugin operational",
			Version: "1.0.0",
		},
		metrics: PluginMetrics{
			MetricsUpdatedAt: time.Now(),
		},
		config: PluginConfig{
			Name:     name,
			Version:  "1.0.0",
			Enabled:  true,
			Settings: make(map[string]interface{}),
		},
	}
}

func (m *MockExportPlugin) Name() string                     { return m.name }
func (m *MockExportPlugin) Version() string                  { return m.version }
func (m *MockExportPlugin) Description() string              { return "Mock export plugin" }
func (m *MockExportPlugin) SupportedFormats() []ExportFormat { return m.supportedFormats }

func (m *MockExportPlugin) Initialize(ctx context.Context, config PluginConfig) error {
	if m.shouldFail {
		return &MockError{message: "initialization failed"}
	}
	m.initialized = true
	if config.Name != "" {
		m.config = config
	}
	return nil
}

func (m *MockExportPlugin) Start(ctx context.Context) error {
	if m.shouldFail {
		return &MockError{message: "start failed"}
	}
	m.running = true
	return nil
}

func (m *MockExportPlugin) Stop(ctx context.Context) error {
	m.running = false
	return nil
}

func (m *MockExportPlugin) IsRunning() bool { return m.running }

func (m *MockExportPlugin) Export(ctx context.Context, request *ExportRequest) (*ExportResponse, error) {
	if m.exportDelay > 0 {
		time.Sleep(m.exportDelay)
	}

	m.exportCalls++

	if m.shouldFail {
		return nil, &MockError{message: "export failed"}
	}

	return &ExportResponse{
		ID:          request.ID,
		PluginName:  m.name,
		Status:      StatusCompleted,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}, nil
}

func (m *MockExportPlugin) ExportDriftReport(ctx context.Context, report *differ.DriftReport, options ExportOptions) error {
	_, err := m.Export(ctx, &ExportRequest{
		DataType: DataTypeDriftReport,
		Data:     report,
		Options:  options,
	})
	return err
}

func (m *MockExportPlugin) ExportSnapshot(ctx context.Context, snapshot *types.Snapshot, options ExportOptions) error {
	_, err := m.Export(ctx, &ExportRequest{
		DataType: DataTypeSnapshot,
		Data:     snapshot,
		Options:  options,
	})
	return err
}

func (m *MockExportPlugin) ExportCorrelation(ctx context.Context, correlation *CorrelationData, options ExportOptions) error {
	_, err := m.Export(ctx, &ExportRequest{
		DataType: DataTypeCorrelation,
		Data:     correlation,
		Options:  options,
	})
	return err
}

func (m *MockExportPlugin) Validate(config PluginConfig) error {
	if m.shouldFail {
		return &MockError{message: "validation failed"}
	}
	return nil
}

func (m *MockExportPlugin) HealthCheck(ctx context.Context) HealthStatus {
	return m.healthStatus
}

func (m *MockExportPlugin) GetMetrics() PluginMetrics {
	return m.metrics
}

func (m *MockExportPlugin) Schema() PluginSchema {
	return PluginSchema{
		Name:        m.name,
		Version:     m.version,
		Description: "Mock plugin schema",
		Schema:      make(map[string]SchemaField),
	}
}

func (m *MockExportPlugin) GetConfig() PluginConfig {
	return m.config
}

func (m *MockExportPlugin) UpdateConfig(ctx context.Context, config PluginConfig) error {
	if m.shouldFail {
		return &MockError{message: "config update failed"}
	}
	m.config = config
	return nil
}

// MockError is a mock error type
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// Test Manager Creation and Lifecycle
func TestManagerCreation(t *testing.T) {
	config := ManagerConfig{
		MaxWorkers:    5,
		QueueSize:     100,
		WorkerTimeout: 30 * time.Second,
	}

	manager := NewManager(config)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if len(manager.plugins) != 0 {
		t.Errorf("Expected empty plugin map, got %d plugins", len(manager.plugins))
	}

	if manager.running {
		t.Error("Manager should not be running initially")
	}
}

func TestManagerStartStop(t *testing.T) {
	config := ManagerConfig{
		MaxWorkers: 2,
		QueueSize:  10,
	}

	manager := NewManager(config)
	ctx := context.Background()

	// Test start
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	if !manager.IsRunning() {
		t.Error("Manager should be running after start")
	}

	// Test double start
	err = manager.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running manager")
	}

	// Test stop
	err = manager.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop manager: %v", err)
	}

	if manager.IsRunning() {
		t.Error("Manager should not be running after stop")
	}
}

func TestPluginRegistration(t *testing.T) {
	manager := NewManager(ManagerConfig{})

	// Test plugin registration
	plugin := NewMockExportPlugin("test-plugin")
	err := manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Verify plugin is registered
	retrievedPlugin, err := manager.GetPlugin("test-plugin")
	if err != nil {
		t.Fatalf("Failed to get registered plugin: %v", err)
	}

	if retrievedPlugin.Name() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got %s", retrievedPlugin.Name())
	}

	// Test duplicate registration
	err = manager.RegisterPlugin(plugin)
	if err == nil {
		t.Error("Expected error when registering duplicate plugin")
	}

	// Test plugin unregistration
	err = manager.UnregisterPlugin("test-plugin")
	if err != nil {
		t.Fatalf("Failed to unregister plugin: %v", err)
	}

	// Verify plugin is unregistered
	_, err = manager.GetPlugin("test-plugin")
	if err == nil {
		t.Error("Expected error when getting unregistered plugin")
	}
}

func TestPluginRegistrationWithRunningManager(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	ctx := context.Background()

	// Start manager
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Register plugin with running manager
	plugin := NewMockExportPlugin("running-test")
	err = manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin with running manager: %v", err)
	}

	if !plugin.IsRunning() {
		t.Error("Plugin should be running when registered with running manager")
	}
}

func TestExportOperations(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	ctx := context.Background()

	// Register a mock plugin
	plugin := NewMockExportPlugin("export-test")
	err := manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Create test export request
	request := &ExportRequest{
		ID:         "test-export-1",
		PluginName: "export-test",
		DataType:   DataTypeDriftReport,
		Format:     FormatJSON,
		Data: &differ.DriftReport{
			ID: "test-report",
		},
		Options: ExportOptions{
			Format: FormatJSON,
		},
	}

	// Test synchronous export
	response, err := manager.Export(ctx, request)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if response.Status != StatusCompleted {
		t.Errorf("Expected status completed, got %s", response.Status)
	}

	if response.PluginName != "export-test" {
		t.Errorf("Expected plugin name 'export-test', got %s", response.PluginName)
	}

	if plugin.exportCalls != 1 {
		t.Errorf("Expected 1 export call, got %d", plugin.exportCalls)
	}
}

func TestAsyncExportOperations(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxWorkers: 2,
		QueueSize:  10,
	})
	ctx := context.Background()

	// Register a mock plugin with delay
	plugin := NewMockExportPlugin("async-test")
	plugin.exportDelay = 100 * time.Millisecond
	err := manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Create test export request
	request := &ExportRequest{
		ID:         "test-async-export",
		PluginName: "async-test",
		DataType:   DataTypeSnapshot,
		Format:     FormatJSON,
		Data: &types.Snapshot{
			ID: "test-snapshot",
		},
		Options: ExportOptions{
			Format: FormatJSON,
			Async:  true,
		},
	}

	// Test asynchronous export
	exportID, err := manager.ExportAsync(ctx, request)
	if err != nil {
		t.Fatalf("Async export failed: %v", err)
	}

	if exportID == "" {
		t.Error("Expected non-empty export ID")
	}

	// Wait for export to process
	time.Sleep(500 * time.Millisecond)

	// Check export status
	status, err := manager.GetExportStatus(exportID)
	if err != nil {
		t.Fatalf("Failed to get export status: %v", err)
	}

	if status.Status != StatusCompleted {
		t.Errorf("Expected status completed, got %s", status.Status)
	}
}

func TestPluginHealth(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	ctx := context.Background()

	// Register a healthy plugin
	healthyPlugin := NewMockExportPlugin("healthy-plugin")
	err := manager.RegisterPlugin(healthyPlugin)
	if err != nil {
		t.Fatalf("Failed to register healthy plugin: %v", err)
	}

	// Register an unhealthy plugin
	unhealthyPlugin := NewMockExportPlugin("unhealthy-plugin")
	unhealthyPlugin.healthStatus.Status = "unhealthy"
	unhealthyPlugin.healthStatus.Message = "Plugin is experiencing issues"
	err = manager.RegisterPlugin(unhealthyPlugin)
	if err != nil {
		t.Fatalf("Failed to register unhealthy plugin: %v", err)
	}

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Test individual plugin health
	health, err := manager.GetPluginHealth("healthy-plugin")
	if err != nil {
		t.Fatalf("Failed to get plugin health: %v", err)
	}

	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	// Test system health
	systemHealth, err := manager.GetSystemHealth()
	if err != nil {
		t.Fatalf("Failed to get system health: %v", err)
	}

	if len(systemHealth) != 2 {
		t.Errorf("Expected 2 plugins in system health, got %d", len(systemHealth))
	}

	if systemHealth["healthy-plugin"].Status != "healthy" {
		t.Error("Healthy plugin should have healthy status")
	}

	if systemHealth["unhealthy-plugin"].Status != "unhealthy" {
		t.Error("Unhealthy plugin should have unhealthy status")
	}
}

func TestPluginMetrics(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	ctx := context.Background()

	// Register a plugin
	plugin := NewMockExportPlugin("metrics-test")
	plugin.metrics.TotalRequests = 100
	plugin.metrics.SuccessfulExports = 95
	plugin.metrics.FailedExports = 5
	plugin.metrics.AverageLatency = 50 * time.Millisecond
	err := manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Test individual plugin metrics
	metrics, err := manager.GetPluginMetrics("metrics-test")
	if err != nil {
		t.Fatalf("Failed to get plugin metrics: %v", err)
	}

	if metrics.TotalRequests != 100 {
		t.Errorf("Expected 100 total requests, got %d", metrics.TotalRequests)
	}

	if metrics.SuccessfulExports != 95 {
		t.Errorf("Expected 95 successful exports, got %d", metrics.SuccessfulExports)
	}

	if metrics.FailedExports != 5 {
		t.Errorf("Expected 5 failed exports, got %d", metrics.FailedExports)
	}

	// Test system metrics
	systemMetrics, err := manager.GetSystemMetrics()
	if err != nil {
		t.Fatalf("Failed to get system metrics: %v", err)
	}

	if len(systemMetrics) != 1 {
		t.Errorf("Expected 1 plugin in system metrics, got %d", len(systemMetrics))
	}

	if systemMetrics["metrics-test"].TotalRequests != 100 {
		t.Error("System metrics should match plugin metrics")
	}
}

func TestConfigurationUpdates(t *testing.T) {
	manager := NewManager(ManagerConfig{})

	// Register a plugin
	plugin := NewMockExportPlugin("config-test")
	err := manager.RegisterPlugin(plugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
	}

	// Test configuration update
	newConfig := PluginConfig{
		Name:    "config-test",
		Version: "2.0.0",
		Enabled: true,
		Settings: map[string]interface{}{
			"new_setting": "new_value",
		},
	}

	err = manager.UpdatePluginConfig("config-test", newConfig)
	if err != nil {
		t.Fatalf("Failed to update plugin config: %v", err)
	}

	// Verify configuration was updated
	updatedConfig := plugin.GetConfig()
	if updatedConfig.Version != "2.0.0" {
		t.Errorf("Expected version 2.0.0, got %s", updatedConfig.Version)
	}

	if value, ok := updatedConfig.Settings["new_setting"].(string); !ok || value != "new_value" {
		t.Error("New setting not found or incorrect value")
	}
}

func TestPluginFailures(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	ctx := context.Background()

	// Register a failing plugin
	failingPlugin := NewMockExportPlugin("failing-plugin")
	failingPlugin.shouldFail = true

	// Test registration failure
	err := manager.RegisterPlugin(failingPlugin)
	if err == nil {
		t.Error("Expected error when registering failing plugin")
	}

	// Register a working plugin
	workingPlugin := NewMockExportPlugin("working-plugin")
	err = manager.RegisterPlugin(workingPlugin)
	if err != nil {
		t.Fatalf("Failed to register working plugin: %v", err)
	}

	// Start manager
	err = manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Make the plugin fail during export
	workingPlugin.shouldFail = true

	// Test export failure
	request := &ExportRequest{
		ID:         "test-failure",
		PluginName: "working-plugin",
		DataType:   DataTypeDriftReport,
		Data: &differ.DriftReport{
			ID: "test-report",
		},
		Options: ExportOptions{
			Format: FormatJSON,
		},
	}

	response, err := manager.Export(ctx, request)
	if err == nil {
		t.Error("Expected error from failing export")
	}

	if response != nil && response.Status != StatusFailed {
		t.Errorf("Expected failed status, got %s", response.Status)
	}
}

func TestConcurrentOperations(t *testing.T) {
	manager := NewManager(ManagerConfig{
		MaxWorkers: 5,
		QueueSize:  20,
	})
	ctx := context.Background()

	// Register multiple plugins
	for i := 0; i < 3; i++ {
		plugin := NewMockExportPlugin(fmt.Sprintf("concurrent-plugin-%d", i))
		plugin.exportDelay = 10 * time.Millisecond
		err := manager.RegisterPlugin(plugin)
		if err != nil {
			t.Fatalf("Failed to register plugin %d: %v", i, err)
		}
	}

	// Start manager
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.Stop(ctx)

	// Launch concurrent exports
	numExports := 10
	results := make(chan error, numExports)

	for i := 0; i < numExports; i++ {
		go func(exportNum int) {
			pluginName := fmt.Sprintf("concurrent-plugin-%d", exportNum%3)
			request := &ExportRequest{
				ID:         fmt.Sprintf("concurrent-export-%d", exportNum),
				PluginName: pluginName,
				DataType:   DataTypeSnapshot,
				Data: &types.Snapshot{
					ID: fmt.Sprintf("snapshot-%d", exportNum),
				},
				Options: ExportOptions{
					Format: FormatJSON,
				},
			}

			_, err := manager.Export(ctx, request)
			results <- err
		}(i)
	}

	// Wait for all exports to complete
	var errors []error
	for i := 0; i < numExports; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		t.Errorf("Got %d errors from concurrent exports: %v", len(errors), errors[0])
	}
}

func TestManagerUptime(t *testing.T) {
	manager := NewManager(ManagerConfig{})

	// Check initial uptime
	uptime := manager.GetUptime()
	if uptime <= 0 {
		t.Error("Uptime should be positive")
	}

	// Wait a bit and check again
	time.Sleep(10 * time.Millisecond)
	newUptime := manager.GetUptime()
	if newUptime <= uptime {
		t.Error("Uptime should increase over time")
	}
}

// Benchmark tests
func BenchmarkManagerExport(b *testing.B) {
	manager := NewManager(ManagerConfig{
		MaxWorkers: 10,
		QueueSize:  1000,
	})
	ctx := context.Background()

	// Register a fast plugin
	plugin := NewMockExportPlugin("benchmark-plugin")
	manager.RegisterPlugin(plugin)
	manager.Start(ctx)
	defer manager.Stop(ctx)

	// Prepare test data
	request := &ExportRequest{
		PluginName: "benchmark-plugin",
		DataType:   DataTypeSnapshot,
		Data: &types.Snapshot{
			ID: "benchmark-snapshot",
		},
		Options: ExportOptions{
			Format: FormatJSON,
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			request.ID = fmt.Sprintf("benchmark-%d", time.Now().UnixNano())
			_, err := manager.Export(ctx, request)
			if err != nil {
				b.Fatalf("Export failed: %v", err)
			}
		}
	})
}

func BenchmarkPluginRegistration(b *testing.B) {
	manager := NewManager(ManagerConfig{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin := NewMockExportPlugin(fmt.Sprintf("benchmark-plugin-%d", i))
		err := manager.RegisterPlugin(plugin)
		if err != nil {
			b.Fatalf("Plugin registration failed: %v", err)
		}
	}
}
