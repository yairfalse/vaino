package exports

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Manager implements the ExportManager interface for server-side plugin management
type Manager struct {
	mu        sync.RWMutex
	plugins   map[string]ExportPlugin
	router    ExportRouter
	queue     *ExportQueue
	workers   *WorkerPool
	health    *HealthMonitor
	metrics   *MetricsCollector
	config    ManagerConfig
	shutdown  chan struct{}
	running   bool
	startTime time.Time

	// Hot reload state
	configHash    [32]byte
	lastConfigMod time.Time
	reloadCount   int64
}

// ManagerConfig configures the export manager
type ManagerConfig struct {
	// Worker configuration
	MaxWorkers    int           `json:"max_workers"`
	WorkerTimeout time.Duration `json:"worker_timeout"`
	QueueSize     int           `json:"queue_size"`

	// Health monitoring
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthTimeout       time.Duration `json:"health_timeout"`

	// Metrics collection
	MetricsInterval  time.Duration `json:"metrics_interval"`
	MetricsRetention time.Duration `json:"metrics_retention"`

	// Plugin management
	PluginLoadTimeout  time.Duration `json:"plugin_load_timeout"`
	PluginStartTimeout time.Duration `json:"plugin_start_timeout"`
	HotReloadEnabled   bool          `json:"hot_reload_enabled"`
	ConfigWatchPath    string        `json:"config_watch_path"`
	ReloadInterval     time.Duration `json:"reload_interval"`

	// Performance tuning
	BatchSize      int           `json:"batch_size"`
	FlushInterval  time.Duration `json:"flush_interval"`
	MaxMemoryUsage int64         `json:"max_memory_usage_bytes"`

	// Security
	RequireAuth       bool `json:"require_auth"`
	EncryptionEnabled bool `json:"encryption_enabled"`
}

// ExportConfig represents the configuration file structure for hot-reloading
type ExportConfig struct {
	Plugins []PluginConfig `yaml:"plugins" json:"plugins"`
	Routes  []RouteConfig  `yaml:"routes" json:"routes"`
	Manager ManagerConfig  `yaml:"manager" json:"manager"`
}

// RouteConfig represents a routing configuration
type RouteConfig struct {
	Pattern    RoutePattern `yaml:"pattern" json:"pattern"`
	PluginName string       `yaml:"plugin_name" json:"plugin_name"`
	Priority   int          `yaml:"priority" json:"priority"`
	Enabled    bool         `yaml:"enabled" json:"enabled"`
}

// NewManager creates a new export manager with the specified configuration
func NewManager(config ManagerConfig) *Manager {
	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 10
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 1000
	}
	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.MetricsInterval <= 0 {
		config.MetricsInterval = 60 * time.Second
	}
	if config.WorkerTimeout <= 0 {
		config.WorkerTimeout = 5 * time.Minute
	}
	if config.ReloadInterval <= 0 {
		config.ReloadInterval = 30 * time.Second
	}
	if config.ConfigWatchPath == "" {
		config.ConfigWatchPath = "config/exports.yaml"
	}

	m := &Manager{
		plugins:   make(map[string]ExportPlugin),
		router:    NewDefaultRouter(),
		queue:     NewExportQueue(config.QueueSize),
		workers:   NewWorkerPool(config.MaxWorkers, config.WorkerTimeout),
		health:    NewHealthMonitor(config.HealthCheckInterval, config.HealthTimeout),
		metrics:   NewMetricsCollector(config.MetricsInterval, config.MetricsRetention),
		config:    config,
		shutdown:  make(chan struct{}),
		startTime: time.Now(),
	}

	// Set up worker result callback to update queue status
	m.workers.SetResultCallback(m.handleWorkerResult)

	return m
}

// Start starts the export manager and all background services
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("manager already running")
	}

	// Start background services
	if err := m.queue.Start(ctx); err != nil {
		return fmt.Errorf("failed to start export queue: %w", err)
	}

	if err := m.workers.Start(ctx); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	if err := m.health.Start(ctx); err != nil {
		return fmt.Errorf("failed to start health monitor: %w", err)
	}

	if err := m.metrics.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics collector: %w", err)
	}

	// Start plugins
	for name, plugin := range m.plugins {
		if err := plugin.Start(ctx); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}

	m.running = true
	go m.backgroundProcessor(ctx)

	return nil
}

// Stop stops the export manager and all plugins
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.shutdown)

	// Stop plugins
	var stopErrors []error
	for name, plugin := range m.plugins {
		if err := plugin.Stop(ctx); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("failed to stop plugin %s: %w", name, err))
		}
	}

	// Stop background services
	if err := m.workers.Stop(ctx); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("failed to stop worker pool: %w", err))
	}

	if err := m.queue.Stop(ctx); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("failed to stop export queue: %w", err))
	}

	if err := m.health.Stop(ctx); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("failed to stop health monitor: %w", err))
	}

	if err := m.metrics.Stop(ctx); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("failed to stop metrics collector: %w", err))
	}

	m.running = false

	if len(stopErrors) > 0 {
		return fmt.Errorf("errors stopping manager: %v", stopErrors)
	}

	return nil
}

// RegisterPlugin registers a new export plugin
func (m *Manager) RegisterPlugin(plugin ExportPlugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := plugin.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Initialize plugin
	ctx, cancel := context.WithTimeout(context.Background(), m.config.PluginLoadTimeout)
	defer cancel()

	if err := plugin.Initialize(ctx, PluginConfig{}); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Validate plugin
	if err := plugin.Validate(PluginConfig{}); err != nil {
		return fmt.Errorf("plugin %s validation failed: %w", name, err)
	}

	// Start plugin if manager is running
	if m.running {
		startCtx, startCancel := context.WithTimeout(context.Background(), m.config.PluginStartTimeout)
		defer startCancel()

		if err := plugin.Start(startCtx); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}

	m.plugins[name] = plugin

	// Register with health monitor
	m.health.RegisterPlugin(name, plugin)

	// Register with metrics collector
	m.metrics.RegisterPlugin(name, plugin)

	// Register default route with router for this plugin
	for _, format := range plugin.SupportedFormats() {
		pattern := RoutePattern{
			Format: format,
		}
		if err := m.router.RegisterRoute(pattern, plugin); err != nil {
			return fmt.Errorf("failed to register route for plugin %s: %w", name, err)
		}
	}

	return nil
}

// UnregisterPlugin unregisters an export plugin
func (m *Manager) UnregisterPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop plugin
	ctx, cancel := context.WithTimeout(context.Background(), m.config.PluginStartTimeout)
	defer cancel()

	if err := plugin.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	// Remove from maps
	delete(m.plugins, name)

	// Unregister from monitors
	m.health.UnregisterPlugin(name)
	m.metrics.UnregisterPlugin(name)

	// Unregister routes for this plugin
	for _, format := range plugin.SupportedFormats() {
		pattern := RoutePattern{
			Format: format,
		}
		m.router.UnregisterRoute(pattern)
	}

	return nil
}

// GetPlugin returns a plugin by name
func (m *Manager) GetPlugin(name string) (ExportPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// ListPlugins returns information about all registered plugins
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]PluginInfo, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		health := m.health.GetPluginHealth(plugin.Name())

		plugins = append(plugins, PluginInfo{
			Name:             plugin.Name(),
			Version:          plugin.Version(),
			Description:      plugin.Description(),
			SupportedFormats: plugin.SupportedFormats(),
			Status:           health.Status,
			Enabled:          plugin.IsRunning(),
			LoadedAt:         m.startTime, // Simplified - would track individual load times
			ConfigSchema:     plugin.Schema(),
			Capabilities:     getPluginCapabilities(plugin),
		})
	}

	return plugins
}

// Export processes an export request synchronously
func (m *Manager) Export(ctx context.Context, request *ExportRequest) (*ExportResponse, error) {
	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	request.RequestedAt = time.Now()

	// Route the request to appropriate plugin
	var plugin ExportPlugin
	var err error

	// If plugin name is specified, use it directly
	if request.PluginName != "" {
		plugin, err = m.GetPlugin(request.PluginName)
		if err != nil {
			return nil, fmt.Errorf("specified plugin %s not found: %w", request.PluginName, err)
		}
	} else {
		// Otherwise use router
		plugin, err = m.router.Route(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("failed to route export request: %w", err)
		}
	}

	// Create response template
	response := &ExportResponse{
		ID:          request.ID,
		PluginName:  plugin.Name(),
		Status:      StatusProcessing,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Record start time for metrics
	startTime := time.Now()

	// Execute export
	_, err = plugin.Export(ctx, request)
	duration := time.Since(startTime)

	// Update response
	response.Duration = duration
	response.Metrics = ExportMetrics{
		ProcessingTime: duration,
		DataSize:       estimateDataSize(request.Data),
	}

	if err != nil {
		response.Status = StatusFailed
		response.Error = err.Error()

		// Update plugin metrics
		m.metrics.RecordError(plugin.Name(), err)
	} else {
		response.Status = StatusCompleted

		// Update plugin metrics
		m.metrics.RecordSuccess(plugin.Name(), duration)
	}

	return response, err
}

// ExportAsync processes an export request asynchronously
func (m *Manager) ExportAsync(ctx context.Context, request *ExportRequest) (string, error) {
	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	request.RequestedAt = time.Now()
	request.Async = true

	// Queue the request
	if err := m.queue.Enqueue(ctx, request); err != nil {
		return "", fmt.Errorf("failed to queue export request: %w", err)
	}

	return request.ID, nil
}

// GetExportStatus returns the status of an async export
func (m *Manager) GetExportStatus(exportID string) (*ExportResponse, error) {
	return m.queue.GetStatus(exportID)
}

// CancelExport cancels a pending or processing export
func (m *Manager) CancelExport(exportID string) error {
	return m.queue.Cancel(exportID)
}

// UpdatePluginConfig updates the configuration of a plugin
func (m *Manager) UpdatePluginConfig(pluginName string, config PluginConfig) error {
	plugin, err := m.GetPlugin(pluginName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return plugin.UpdateConfig(ctx, config)
}

// GetPluginHealth returns the health status of a specific plugin
func (m *Manager) GetPluginHealth(pluginName string) (HealthStatus, error) {
	health := m.health.GetPluginHealth(pluginName)
	if health.Status == "" {
		return HealthStatus{}, fmt.Errorf("plugin %s not found", pluginName)
	}
	return health, nil
}

// GetSystemHealth returns the health status of all plugins
func (m *Manager) GetSystemHealth() (map[string]HealthStatus, error) {
	return m.health.GetSystemHealth(), nil
}

// GetPluginMetrics returns metrics for a specific plugin
func (m *Manager) GetPluginMetrics(pluginName string) (PluginMetrics, error) {
	metrics := m.metrics.GetPluginMetrics(pluginName)
	if metrics.TotalRequests == 0 && time.Since(m.startTime) > time.Minute {
		// Plugin might not exist if it has no activity after startup
		if _, err := m.GetPlugin(pluginName); err != nil {
			return PluginMetrics{}, err
		}
	}
	return metrics, nil
}

// GetSystemMetrics returns metrics for all plugins
func (m *Manager) GetSystemMetrics() (map[string]PluginMetrics, error) {
	return m.metrics.GetSystemMetrics(), nil
}

// IsRunning returns whether the manager is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetUptime returns how long the manager has been running
func (m *Manager) GetUptime() time.Duration {
	return time.Since(m.startTime)
}

// backgroundProcessor handles background tasks
func (m *Manager) backgroundProcessor(ctx context.Context) {
	// Process queued requests immediately and frequently
	processTicker := time.NewTicker(100 * time.Millisecond)
	defer processTicker.Stop()

	// Cleanup less frequently
	cleanupTicker := time.NewTicker(60 * time.Second)
	defer cleanupTicker.Stop()

	if m.config.HotReloadEnabled {
		// Hot reload enabled - include reload ticker
		reloadTicker := time.NewTicker(m.config.ReloadInterval)
		defer reloadTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-m.shutdown:
				return
			case <-processTicker.C:
				// Process queued requests frequently
				m.processQueuedRequests(ctx)
			case <-cleanupTicker.C:
				// Cleanup expired data less frequently
				m.cleanup()
			case <-reloadTicker.C:
				// Check for configuration changes and hot reload
				m.checkAndReloadConfig(ctx)
			}
		}
	} else {
		// Hot reload disabled - simpler loop
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.shutdown:
				return
			case <-processTicker.C:
				// Process queued requests frequently
				m.processQueuedRequests(ctx)
			case <-cleanupTicker.C:
				// Cleanup expired data less frequently
				m.cleanup()
			}
		}
	}
}

// processQueuedRequests processes requests from the queue
func (m *Manager) processQueuedRequests(ctx context.Context) {
	for {
		request, err := m.queue.Dequeue(ctx)
		if err != nil || request == nil {
			break
		}

		// Route the request to appropriate plugin
		var plugin ExportPlugin
		var routeErr error

		if request.PluginName != "" {
			plugin, routeErr = m.GetPlugin(request.PluginName)
		} else {
			plugin, routeErr = m.router.Route(ctx, request)
		}

		if routeErr != nil {
			// Update status with error
			response := &ExportResponse{
				ID:          request.ID,
				Status:      StatusFailed,
				Error:       routeErr.Error(),
				ProcessedAt: time.Now(),
			}
			m.queue.UpdateStatus(request.ID, response)
			continue
		}

		// Create export job
		job := &ExportJob{
			ID:         request.ID,
			Request:    request,
			Plugin:     plugin,
			StartTime:  time.Now(),
			Timeout:    30 * time.Second, // Default timeout
			MaxRetries: 3,
		}

		// Submit to worker pool
		if err := m.workers.SubmitJob(job); err != nil {
			// Worker pool full, update status with error
			response := &ExportResponse{
				ID:          request.ID,
				Status:      StatusFailed,
				Error:       "worker pool full: " + err.Error(),
				ProcessedAt: time.Now(),
			}
			m.queue.UpdateStatus(request.ID, response)
		} else {
			// Update status to processing
			response := &ExportResponse{
				ID:          request.ID,
				PluginName:  plugin.Name(),
				Status:      StatusProcessing,
				ProcessedAt: time.Now(),
			}
			m.queue.UpdateStatus(request.ID, response)
		}
	}
}

// handleWorkerResult handles results from the worker pool and updates queue status
func (m *Manager) handleWorkerResult(result *ExportResult) {
	if result == nil || result.Job == nil {
		return
	}

	// Create response from result
	response := &ExportResponse{
		ID:          result.Job.ID,
		PluginName:  result.Job.Plugin.Name(),
		Duration:    result.Duration,
		ProcessedAt: result.EndTime,
		Metadata:    make(map[string]interface{}),
	}

	if result.Error != nil {
		response.Status = StatusFailed
		response.Error = result.Error.Error()
		// Record error metrics
		m.metrics.RecordError(result.Job.Plugin.Name(), result.Error)
	} else {
		response.Status = StatusCompleted
		if result.Response != nil {
			response.Data = result.Response.Data
			response.ContentType = result.Response.ContentType
			response.OutputPath = result.Response.OutputPath
			response.ExternalURL = result.Response.ExternalURL
			if result.Response.Metadata != nil {
				response.Metadata = result.Response.Metadata
			}
		}
		// Record success metrics
		m.metrics.RecordSuccess(result.Job.Plugin.Name(), result.Duration)
	}

	// Update queue status
	m.queue.UpdateStatus(result.Job.ID, response)
}

// checkAndReloadConfig checks if configuration has changed and reloads if needed
func (m *Manager) checkAndReloadConfig(ctx context.Context) {
	configPath := m.config.ConfigWatchPath

	// Check if config file exists
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		// Config file doesn't exist or can't be accessed
		return
	}

	// Check if file has been modified
	if !fileInfo.ModTime().After(m.lastConfigMod) {
		return
	}

	// Read and hash the config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Hot reload: Failed to read config file %s: %v\n", configPath, err)
		return
	}

	newHash := sha256.Sum256(configData)
	if newHash == m.configHash {
		// Content hasn't changed even though mod time has
		m.lastConfigMod = fileInfo.ModTime()
		return
	}

	// Parse the new configuration
	var newConfig ExportConfig
	if err := yaml.Unmarshal(configData, &newConfig); err != nil {
		fmt.Printf("Hot reload: Failed to parse config file %s: %v\n", configPath, err)
		return
	}

	// Apply the new configuration
	if err := m.reloadConfiguration(ctx, &newConfig); err != nil {
		fmt.Printf("Hot reload: Failed to apply new configuration: %v\n", err)
		return
	}

	// Update tracking variables
	m.configHash = newHash
	m.lastConfigMod = fileInfo.ModTime()
	atomic.AddInt64(&m.reloadCount, 1)

	fmt.Printf("Hot reload: Successfully reloaded configuration from %s (reload #%d)\n",
		configPath, atomic.LoadInt64(&m.reloadCount))
}

// reloadConfiguration applies a new configuration to the manager
func (m *Manager) reloadConfiguration(ctx context.Context, config *ExportConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track which plugins need to be updated/added/removed
	currentPlugins := make(map[string]ExportPlugin)
	for name, plugin := range m.plugins {
		currentPlugins[name] = plugin
	}

	newPlugins := make(map[string]PluginConfig)
	for _, pluginConfig := range config.Plugins {
		if pluginConfig.Enabled {
			newPlugins[pluginConfig.Name] = pluginConfig
		}
	}

	// Update existing plugins with new configurations
	for name, newConfig := range newPlugins {
		if plugin, exists := currentPlugins[name]; exists {
			// Update existing plugin configuration
			if err := plugin.UpdateConfig(ctx, newConfig); err != nil {
				fmt.Printf("Hot reload: Failed to update plugin %s config: %v\n", name, err)
				continue
			}
			fmt.Printf("Hot reload: Updated configuration for plugin %s\n", name)
			delete(currentPlugins, name) // Mark as processed
		} else {
			// This is a new plugin - would need plugin factory to create it
			fmt.Printf("Hot reload: New plugin %s detected but dynamic plugin loading not implemented\n", name)
		}
	}

	// Remove plugins that are no longer in configuration
	for name, plugin := range currentPlugins {
		fmt.Printf("Hot reload: Removing plugin %s\n", name)
		if err := m.unregisterPluginInternal(name, plugin); err != nil {
			fmt.Printf("Hot reload: Failed to remove plugin %s: %v\n", name, err)
		}
	}

	// Update routing rules
	if err := m.updateRoutes(config.Routes); err != nil {
		return fmt.Errorf("failed to update routes: %w", err)
	}

	return nil
}

// updateRoutes updates the routing configuration
func (m *Manager) updateRoutes(routes []RouteConfig) error {
	// Clear existing routes (simplified - in production would be more careful)
	m.router = NewDefaultRouter()

	// Re-register plugins with new routes
	for _, plugin := range m.plugins {
		for _, format := range plugin.SupportedFormats() {
			pattern := RoutePattern{
				Format: format,
			}
			if err := m.router.RegisterRoute(pattern, plugin); err != nil {
				return fmt.Errorf("failed to register route for plugin %s: %w", plugin.Name(), err)
			}
		}
	}

	// Add custom routes from configuration
	for _, routeConfig := range routes {
		if !routeConfig.Enabled {
			continue
		}

		if plugin, exists := m.plugins[routeConfig.PluginName]; exists {
			if err := m.router.RegisterRoute(routeConfig.Pattern, plugin); err != nil {
				fmt.Printf("Hot reload: Failed to register custom route for %s: %v\n", routeConfig.PluginName, err)
			}
		}
	}

	return nil
}

// unregisterPluginInternal is the internal version of UnregisterPlugin without locking
func (m *Manager) unregisterPluginInternal(name string, plugin ExportPlugin) error {
	// Stop plugin
	ctx, cancel := context.WithTimeout(context.Background(), m.config.PluginStartTimeout)
	defer cancel()

	if err := plugin.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	// Remove from maps
	delete(m.plugins, name)

	// Unregister from monitors
	m.health.UnregisterPlugin(name)
	m.metrics.UnregisterPlugin(name)

	// Unregister routes for this plugin
	for _, format := range plugin.SupportedFormats() {
		pattern := RoutePattern{
			Format: format,
		}
		m.router.UnregisterRoute(pattern)
	}

	return nil
}

// GetReloadCount returns the number of successful hot reloads
func (m *Manager) GetReloadCount() int64 {
	return atomic.LoadInt64(&m.reloadCount)
}

// cleanup performs periodic cleanup tasks
func (m *Manager) cleanup() {
	// Cleanup old queue entries
	m.queue.Cleanup()

	// Cleanup old metrics
	m.metrics.Cleanup()
}

// Helper functions

func getPluginCapabilities(plugin ExportPlugin) []string {
	capabilities := []string{"export"}

	// Check for additional capabilities based on supported formats
	formats := plugin.SupportedFormats()
	for _, format := range formats {
		switch format {
		case FormatOTEL:
			capabilities = append(capabilities, "observability")
		case FormatPrometheus:
			capabilities = append(capabilities, "metrics")
		case FormatWebhook:
			capabilities = append(capabilities, "realtime")
		}
	}

	return capabilities
}

func estimateDataSize(data interface{}) int64 {
	// Simple size estimation - in production would use more sophisticated method
	switch v := data.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	default:
		// Rough estimate for complex objects
		return 1024 // 1KB default estimate
	}
}

// ExportQueue manages queued export requests
type ExportQueue struct {
	mu        sync.RWMutex
	queue     chan *ExportRequest
	status    map[string]*ExportResponse
	maxSize   int
	running   bool
	requestID int64
}

// NewExportQueue creates a new export queue
func NewExportQueue(maxSize int) *ExportQueue {
	return &ExportQueue{
		queue:   make(chan *ExportRequest, maxSize),
		status:  make(map[string]*ExportResponse),
		maxSize: maxSize,
	}
}

// Start starts the export queue
func (q *ExportQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.running {
		return fmt.Errorf("queue already running")
	}

	q.running = true
	return nil
}

// Stop stops the export queue
func (q *ExportQueue) Stop(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.running {
		return nil
	}

	close(q.queue)
	q.running = false
	return nil
}

// Enqueue adds a request to the queue
func (q *ExportQueue) Enqueue(ctx context.Context, request *ExportRequest) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.running {
		return fmt.Errorf("queue not running")
	}

	// Create initial status entry
	q.status[request.ID] = &ExportResponse{
		ID:          request.ID,
		PluginName:  request.PluginName,
		Status:      StatusPending,
		ProcessedAt: request.RequestedAt,
	}

	select {
	case q.queue <- request:
		atomic.AddInt64(&q.requestID, 1)
		return nil
	case <-ctx.Done():
		delete(q.status, request.ID)
		return ctx.Err()
	default:
		delete(q.status, request.ID)
		return fmt.Errorf("queue full")
	}
}

// Dequeue removes a request from the queue
func (q *ExportQueue) Dequeue(ctx context.Context) (*ExportRequest, error) {
	select {
	case request := <-q.queue:
		return request, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, nil // No requests available
	}
}

// GetStatus returns the status of a request
func (q *ExportQueue) GetStatus(exportID string) (*ExportResponse, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	response, exists := q.status[exportID]
	if !exists {
		return nil, fmt.Errorf("export %s not found", exportID)
	}

	return response, nil
}

// UpdateStatus updates the status of a request
func (q *ExportQueue) UpdateStatus(exportID string, response *ExportResponse) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.status[exportID] = response
}

// Cancel cancels a pending request
func (q *ExportQueue) Cancel(exportID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	response, exists := q.status[exportID]
	if !exists {
		return fmt.Errorf("export %s not found", exportID)
	}

	if response.Status == StatusProcessing {
		return fmt.Errorf("cannot cancel export %s: already processing", exportID)
	}

	response.Status = StatusCancelled
	return nil
}

// Cleanup removes old completed requests
func (q *ExportQueue) Cleanup() {
	q.mu.Lock()
	defer q.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)

	for id, response := range q.status {
		if response.ProcessedAt.Before(cutoff) &&
			(response.Status == StatusCompleted || response.Status == StatusFailed || response.Status == StatusCancelled) {
			delete(q.status, id)
		}
	}
}

// QueueSize returns the current queue size
func (q *ExportQueue) QueueSize() int {
	return len(q.queue)
}

// PendingCount returns the number of pending requests
func (q *ExportQueue) PendingCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	count := 0
	for _, response := range q.status {
		if response.Status == StatusPending {
			count++
		}
	}
	return count
}
