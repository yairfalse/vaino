package exports

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// HealthMonitor monitors the health of export plugins and the system
type HealthMonitor struct {
	mu               sync.RWMutex
	plugins          map[string]ExportPlugin
	pluginHealth     map[string]*HealthStatus
	systemHealth     *SystemHealth
	config           HealthConfig
	shutdown         chan struct{}
	running          bool
	startTime        time.Time
	checkInProgress  map[string]bool
	alertSubscribers []HealthAlertSubscriber
}

// HealthConfig configures health monitoring behavior
type HealthConfig struct {
	CheckInterval     time.Duration `json:"check_interval"`
	CheckTimeout      time.Duration `json:"check_timeout"`
	FailureThreshold  int           `json:"failure_threshold"`
	RecoveryThreshold int           `json:"recovery_threshold"`
	AlertEnabled      bool          `json:"alert_enabled"`
	MetricsEnabled    bool          `json:"metrics_enabled"`
	DetailedChecks    bool          `json:"detailed_checks"`
	RetryFailedChecks bool          `json:"retry_failed_checks"`
	MaxRetries        int           `json:"max_retries"`
	RetryDelay        time.Duration `json:"retry_delay"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status        string                 `json:"status"`
	LastCheck     time.Time              `json:"last_check"`
	PluginSummary PluginHealthSummary    `json:"plugin_summary"`
	ResourceUsage ResourceUsage          `json:"resource_usage"`
	Performance   PerformanceMetrics     `json:"performance"`
	Alerts        []HealthAlert          `json:"alerts"`
	Uptime        time.Duration          `json:"uptime"`
	Version       string                 `json:"version"`
	Environment   map[string]interface{} `json:"environment"`
}

// PluginHealthSummary summarizes plugin health across the system
type PluginHealthSummary struct {
	Total     int            `json:"total"`
	Healthy   int            `json:"healthy"`
	Degraded  int            `json:"degraded"`
	Unhealthy int            `json:"unhealthy"`
	Unknown   int            `json:"unknown"`
	ByStatus  map[string]int `json:"by_status"`
}

// ResourceUsage tracks system resource utilization
type ResourceUsage struct {
	Memory     MemoryUsage  `json:"memory"`
	CPU        CPUUsage     `json:"cpu"`
	Disk       DiskUsage    `json:"disk"`
	Network    NetworkUsage `json:"network"`
	Goroutines int          `json:"goroutines"`
	Timestamp  time.Time    `json:"timestamp"`
}

// MemoryUsage tracks memory utilization
type MemoryUsage struct {
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Available   uint64  `json:"available_bytes"`
	UsedPercent float64 `json:"used_percent"`
	HeapAlloc   uint64  `json:"heap_alloc_bytes"`
	HeapSys     uint64  `json:"heap_sys_bytes"`
	HeapIdle    uint64  `json:"heap_idle_bytes"`
	HeapInuse   uint64  `json:"heap_inuse_bytes"`
	GCRuns      uint32  `json:"gc_runs"`
}

// CPUUsage tracks CPU utilization
type CPUUsage struct {
	UsedPercent float64   `json:"used_percent"`
	LoadAverage []float64 `json:"load_average"`
	CoreCount   int       `json:"core_count"`
	ProcessCPU  float64   `json:"process_cpu_percent"`
}

// DiskUsage tracks disk utilization
type DiskUsage struct {
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Available   uint64  `json:"available_bytes"`
	UsedPercent float64 `json:"used_percent"`
	IOReads     uint64  `json:"io_reads"`
	IOWrites    uint64  `json:"io_writes"`
}

// NetworkUsage tracks network utilization
type NetworkUsage struct {
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Connections int    `json:"active_connections"`
	ErrorsSent  uint64 `json:"errors_sent"`
	ErrorsRecv  uint64 `json:"errors_recv"`
}

// PerformanceMetrics tracks system performance
type PerformanceMetrics struct {
	RequestsPerSecond float64       `json:"requests_per_second"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
	ThroughputMBps    float64       `json:"throughput_mbps"`
	QueueDepth        int           `json:"queue_depth"`
	ActiveWorkers     int           `json:"active_workers"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// HealthAlert represents a health alert
type HealthAlert struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt time.Time              `json:"resolved_at,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Count      int                    `json:"count"`
	FirstSeen  time.Time              `json:"first_seen"`
}

// HealthAlertSubscriber defines interface for health alert subscribers
type HealthAlertSubscriber interface {
	OnHealthAlert(alert HealthAlert) error
}

// PluginHealthTracker tracks health for a specific plugin
type PluginHealthTracker struct {
	mu                 sync.RWMutex
	plugin             ExportPlugin
	currentHealth      *HealthStatus
	healthHistory      []HealthStatus
	consecutiveFails   int
	consecutiveSuccess int
	lastCheckTime      time.Time
	checkCount         int64
	totalCheckTime     time.Duration
	isRecovering       bool
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(checkInterval, checkTimeout time.Duration) *HealthMonitor {
	config := HealthConfig{
		CheckInterval:     checkInterval,
		CheckTimeout:      checkTimeout,
		FailureThreshold:  3,
		RecoveryThreshold: 2,
		AlertEnabled:      true,
		MetricsEnabled:    true,
		DetailedChecks:    true,
		RetryFailedChecks: true,
		MaxRetries:        2,
		RetryDelay:        5 * time.Second,
	}

	return &HealthMonitor{
		plugins:          make(map[string]ExportPlugin),
		pluginHealth:     make(map[string]*HealthStatus),
		systemHealth:     &SystemHealth{},
		config:           config,
		shutdown:         make(chan struct{}),
		startTime:        time.Now(),
		checkInProgress:  make(map[string]bool),
		alertSubscribers: make([]HealthAlertSubscriber, 0),
	}
}

// Start starts the health monitoring service
func (hm *HealthMonitor) Start(ctx context.Context) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		return fmt.Errorf("health monitor already running")
	}

	hm.running = true
	hm.startTime = time.Now()

	// Start background health checking
	go hm.healthCheckLoop(ctx)

	// Start system monitoring
	go hm.systemMonitorLoop(ctx)

	return nil
}

// Stop stops the health monitoring service
func (hm *HealthMonitor) Stop(ctx context.Context) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if !hm.running {
		return nil
	}

	close(hm.shutdown)
	hm.running = false

	return nil
}

// RegisterPlugin registers a plugin for health monitoring
func (hm *HealthMonitor) RegisterPlugin(name string, plugin ExportPlugin) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.plugins[name] = plugin

	// Perform initial health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initialHealth := plugin.HealthCheck(ctx)
	hm.pluginHealth[name] = &initialHealth
}

// UnregisterPlugin removes a plugin from health monitoring
func (hm *HealthMonitor) UnregisterPlugin(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.plugins, name)
	delete(hm.pluginHealth, name)
	delete(hm.checkInProgress, name)
}

// GetPluginHealth returns the health status of a specific plugin
func (hm *HealthMonitor) GetPluginHealth(pluginName string) HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if health, exists := hm.pluginHealth[pluginName]; exists {
		return *health
	}

	return HealthStatus{
		Status:    "unknown",
		LastCheck: time.Time{},
		Message:   "Plugin not found",
	}
}

// GetSystemHealth returns the overall system health
func (hm *HealthMonitor) GetSystemHealth() map[string]HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	result := make(map[string]HealthStatus)
	for name, health := range hm.pluginHealth {
		result[name] = *health
	}

	return result
}

// GetDetailedSystemHealth returns detailed system health information
func (hm *HealthMonitor) GetDetailedSystemHealth() SystemHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// Update system health
	hm.updateSystemHealth()

	return *hm.systemHealth
}

// SubscribeToAlerts subscribes to health alerts
func (hm *HealthMonitor) SubscribeToAlerts(subscriber HealthAlertSubscriber) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.alertSubscribers = append(hm.alertSubscribers, subscriber)
}

// UnsubscribeFromAlerts unsubscribes from health alerts
func (hm *HealthMonitor) UnsubscribeFromAlerts(subscriber HealthAlertSubscriber) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for i, sub := range hm.alertSubscribers {
		if sub == subscriber {
			hm.alertSubscribers = append(hm.alertSubscribers[:i], hm.alertSubscribers[i+1:]...)
			break
		}
	}
}

// ForceHealthCheck triggers an immediate health check for all plugins
func (hm *HealthMonitor) ForceHealthCheck(ctx context.Context) error {
	hm.mu.RLock()
	plugins := make(map[string]ExportPlugin)
	for name, plugin := range hm.plugins {
		plugins[name] = plugin
	}
	hm.mu.RUnlock()

	var wg sync.WaitGroup
	errors := make(chan error, len(plugins))

	for name, plugin := range plugins {
		wg.Add(1)
		go func(pluginName string, p ExportPlugin) {
			defer wg.Done()
			if err := hm.checkPluginHealth(ctx, pluginName, p); err != nil {
				errors <- fmt.Errorf("health check failed for %s: %w", pluginName, err)
			}
		}(name, plugin)
	}

	wg.Wait()
	close(errors)

	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("health check errors: %v", allErrors)
	}

	return nil
}

// healthCheckLoop runs periodic health checks
func (hm *HealthMonitor) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(hm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.shutdown:
			return
		case <-ticker.C:
			hm.performHealthChecks(ctx)
		}
	}
}

// systemMonitorLoop monitors system resources
func (hm *HealthMonitor) systemMonitorLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Monitor system every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hm.shutdown:
			return
		case <-ticker.C:
			hm.updateSystemHealth()
		}
	}
}

// performHealthChecks runs health checks for all registered plugins
func (hm *HealthMonitor) performHealthChecks(ctx context.Context) {
	hm.mu.RLock()
	plugins := make(map[string]ExportPlugin)
	for name, plugin := range hm.plugins {
		// Skip if check is already in progress
		if !hm.checkInProgress[name] {
			plugins[name] = plugin
		}
	}
	hm.mu.RUnlock()

	var wg sync.WaitGroup
	for name, plugin := range plugins {
		wg.Add(1)
		go func(pluginName string, p ExportPlugin) {
			defer wg.Done()
			hm.checkPluginHealth(ctx, pluginName, p)
		}(name, plugin)
	}

	wg.Wait()
}

// checkPluginHealth performs a health check for a specific plugin
func (hm *HealthMonitor) checkPluginHealth(ctx context.Context, pluginName string, plugin ExportPlugin) error {
	// Mark check as in progress
	hm.mu.Lock()
	hm.checkInProgress[pluginName] = true
	hm.mu.Unlock()

	defer func() {
		hm.mu.Lock()
		delete(hm.checkInProgress, pluginName)
		hm.mu.Unlock()
	}()

	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, hm.config.CheckTimeout)
	defer cancel()

	// Perform health check
	health := plugin.HealthCheck(checkCtx)
	health.LastCheck = time.Now()

	// Update health status
	hm.mu.Lock()
	defer hm.mu.Unlock()

	previousHealth := hm.pluginHealth[pluginName]
	hm.pluginHealth[pluginName] = &health

	// Check for status changes and generate alerts
	if previousHealth != nil && previousHealth.Status != health.Status {
		hm.generateHealthAlert(pluginName, previousHealth.Status, health.Status, health.Message)
	}

	// Track plugin uptime
	if health.Status == "healthy" && plugin.IsRunning() {
		hm.pluginHealth[pluginName].Uptime = time.Since(hm.startTime)
	}

	return nil
}

// updateSystemHealth updates overall system health metrics
func (hm *HealthMonitor) updateSystemHealth() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Calculate plugin summary
	summary := PluginHealthSummary{
		Total:    len(hm.pluginHealth),
		ByStatus: make(map[string]int),
	}

	for _, health := range hm.pluginHealth {
		summary.ByStatus[health.Status]++
		switch health.Status {
		case "healthy":
			summary.Healthy++
		case "degraded":
			summary.Degraded++
		case "unhealthy":
			summary.Unhealthy++
		default:
			summary.Unknown++
		}
	}

	// Determine overall system status
	systemStatus := "healthy"
	if summary.Unhealthy > 0 {
		systemStatus = "unhealthy"
	} else if summary.Degraded > 0 {
		systemStatus = "degraded"
	}

	// Update system health
	hm.systemHealth.Status = systemStatus
	hm.systemHealth.LastCheck = time.Now()
	hm.systemHealth.PluginSummary = summary
	hm.systemHealth.ResourceUsage = hm.collectResourceUsage()
	hm.systemHealth.Uptime = time.Since(hm.startTime)
	hm.systemHealth.Version = "1.0.0" // Would be injected from build
}

// collectResourceUsage collects current system resource usage
func (hm *HealthMonitor) collectResourceUsage() ResourceUsage {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ResourceUsage{
		Memory: MemoryUsage{
			HeapAlloc: memStats.HeapAlloc,
			HeapSys:   memStats.HeapSys,
			HeapIdle:  memStats.HeapIdle,
			HeapInuse: memStats.HeapInuse,
			GCRuns:    memStats.NumGC,
		},
		CPU: CPUUsage{
			CoreCount: runtime.NumCPU(),
		},
		Goroutines: runtime.NumGoroutine(),
		Timestamp:  time.Now(),
	}
}

// generateHealthAlert creates and dispatches health alerts
func (hm *HealthMonitor) generateHealthAlert(pluginName, oldStatus, newStatus, message string) {
	if !hm.config.AlertEnabled {
		return
	}

	alert := HealthAlert{
		ID:        fmt.Sprintf("health_%s_%d", pluginName, time.Now().UnixNano()),
		Type:      "health_status_change",
		Source:    pluginName,
		Message:   fmt.Sprintf("Plugin %s status changed from %s to %s: %s", pluginName, oldStatus, newStatus, message),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"plugin":     pluginName,
			"old_status": oldStatus,
			"new_status": newStatus,
		},
		Count:     1,
		FirstSeen: time.Now(),
	}

	// Determine severity
	switch newStatus {
	case "unhealthy":
		alert.Severity = "critical"
	case "degraded":
		alert.Severity = "warning"
	case "healthy":
		if oldStatus == "unhealthy" || oldStatus == "degraded" {
			alert.Severity = "info"
			alert.Type = "health_recovery"
			alert.Message = fmt.Sprintf("Plugin %s recovered to healthy status", pluginName)
		}
	}

	// Dispatch alert to subscribers
	go hm.dispatchAlert(alert)
}

// dispatchAlert sends alerts to all subscribers
func (hm *HealthMonitor) dispatchAlert(alert HealthAlert) {
	hm.mu.RLock()
	subscribers := make([]HealthAlertSubscriber, len(hm.alertSubscribers))
	copy(subscribers, hm.alertSubscribers)
	hm.mu.RUnlock()

	for _, subscriber := range subscribers {
		go func(sub HealthAlertSubscriber) {
			if err := sub.OnHealthAlert(alert); err != nil {
				// Log error dispatching alert (would use proper logger in production)
				fmt.Printf("Failed to dispatch health alert: %v\n", err)
			}
		}(subscriber)
	}
}

// GetConfig returns the current health monitoring configuration
func (hm *HealthMonitor) GetConfig() HealthConfig {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.config
}

// UpdateConfig updates the health monitoring configuration
func (hm *HealthMonitor) UpdateConfig(config HealthConfig) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.config = config
}

// IsRunning returns whether the health monitor is currently running
func (hm *HealthMonitor) IsRunning() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.running
}

// GetHealthHistory returns historical health data for a plugin
func (hm *HealthMonitor) GetHealthHistory(pluginName string, duration time.Duration) []HealthStatus {
	// In a production system, this would return historical data from storage
	// For now, return current status
	current := hm.GetPluginHealth(pluginName)
	if current.Status == "" {
		return []HealthStatus{}
	}
	return []HealthStatus{current}
}

// GetAlerts returns recent health alerts
func (hm *HealthMonitor) GetAlerts(since time.Time) []HealthAlert {
	// In a production system, this would return alerts from storage
	// For now, return empty slice
	return []HealthAlert{}
}

// Simple health alert logger that implements HealthAlertSubscriber
type ConsoleAlertSubscriber struct{}

func (c *ConsoleAlertSubscriber) OnHealthAlert(alert HealthAlert) error {
	fmt.Printf("[HEALTH ALERT] %s - %s: %s\n", alert.Severity, alert.Source, alert.Message)
	return nil
}
