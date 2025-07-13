package systemd

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/pkg/types"
)

// Collector implements the EnhancedCollector interface for systemd
type Collector struct {
	dbus    *DBusConnection
	monitor *ServiceMonitor
	config  collectors.CollectorConfig
	mu      sync.RWMutex
}

// NewCollector creates a new systemd collector
func NewCollector() (*Collector, error) {
	// Check if we're on a Linux system with systemd
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("systemd collector only supported on Linux")
	}

	dbus, err := NewDBusConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to systemd: %w", err)
	}

	monitor := NewServiceMonitor(dbus)

	return &Collector{
		dbus:    dbus,
		monitor: monitor,
	}, nil
}

// Name returns the name of the collector
func (c *Collector) Name() string {
	return "systemd"
}

// Status returns the current status of the collector
func (c *Collector) Status() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.dbus == nil {
		return "Not connected"
	}

	// Try to ping systemd
	var version string
	err := c.dbus.systemd.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Manager", "Version").Store(&version)

	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	services := c.monitor.GetAllServices()
	failedCount := 0
	for _, svc := range services {
		if svc.State.ActiveState == "failed" {
			failedCount++
		}
	}

	return fmt.Sprintf("Connected (systemd %s) - Monitoring %d services, %d failed",
		version, len(services), failedCount)
}

// Collect gathers systemd service information and returns a snapshot
func (c *Collector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	// Set up filters based on config
	if err := c.setupFilters(config); err != nil {
		return nil, fmt.Errorf("failed to setup filters: %w", err)
	}

	// Start monitoring if not already started
	monitorCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.monitor.Start(monitorCtx); err != nil {
		return nil, fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Wait a bit for initial data collection
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        generateSnapshotID(),
		Timestamp: time.Now(),
		Provider:  "systemd",
		Resources: make([]types.Resource, 0),
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			Tags:             config.Tags,
		},
	}

	startTime := time.Now()

	// Collect service resources
	services := c.monitor.GetAllServices()
	for name, service := range services {
		resource := c.serviceToResource(name, service)
		snapshot.Resources = append(snapshot.Resources, resource)
	}

	// Update metadata
	snapshot.Metadata.CollectionTime = time.Since(startTime)
	snapshot.Metadata.ResourceCount = len(snapshot.Resources)

	return snapshot, nil
}

// Validate checks if the collector configuration is valid
func (c *Collector) Validate(config collectors.CollectorConfig) error {
	// Check OS
	if runtime.GOOS != "linux" {
		return fmt.Errorf("systemd collector requires Linux")
	}

	// Validate filters if provided
	if filters, ok := config.Config["filters"].([]interface{}); ok {
		for _, filter := range filters {
			filterStr, ok := filter.(string)
			if !ok {
				return fmt.Errorf("invalid filter type: expected string")
			}

			// Validate filter format
			if !isValidFilter(filterStr) {
				return fmt.Errorf("invalid filter format: %s", filterStr)
			}
		}
	}

	// Validate rate limit if provided
	if rateLimit, ok := config.Config["rate_limit"].(int); ok {
		if rateLimit < 100 || rateLimit > 10000 {
			return fmt.Errorf("rate_limit must be between 100 and 10000")
		}
	}

	return nil
}

// AutoDiscover attempts to automatically discover systemd configuration
func (c *Collector) AutoDiscover() (collectors.CollectorConfig, error) {
	config := collectors.CollectorConfig{
		Config: make(map[string]interface{}),
		Tags:   make(map[string]string),
	}

	// Check if systemd is available
	if runtime.GOOS != "linux" {
		return config, fmt.Errorf("systemd not available on %s", runtime.GOOS)
	}

	// Try to connect to systemd
	testConn, err := NewDBusConnection()
	if err != nil {
		return config, fmt.Errorf("systemd not accessible: %w", err)
	}
	defer testConn.Close()

	// Get systemd version
	var version string
	err = testConn.systemd.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.systemd1.Manager", "Version").Store(&version)
	if err == nil {
		config.Tags["systemd_version"] = version
	}

	// Set default configuration
	config.Config["filters"] = []string{
		"state:active,failed", // Monitor active and failed services
		"type:service",        // Only service units
		"exclude:user@*",      // Exclude user services
	}
	config.Config["rate_limit"] = 1000 // 1000 events per minute
	config.Config["monitor_restarts"] = true
	config.Config["monitor_resources"] = true
	config.Config["monitor_dependencies"] = true

	return config, nil
}

// SupportedRegions returns supported regions (not applicable for systemd)
func (c *Collector) SupportedRegions() []string {
	return []string{"local"}
}

// Close closes the collector and releases resources
func (c *Collector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dbus != nil {
		return c.dbus.Close()
	}
	return nil
}

// Helper methods

func (c *Collector) setupFilters(config collectors.CollectorConfig) error {
	// Clear existing filters
	c.monitor.filters = make([]ServiceFilter, 0)

	// Add filters from config
	if filters, ok := config.Config["filters"].([]interface{}); ok {
		for _, filter := range filters {
			filterStr, ok := filter.(string)
			if !ok {
				continue
			}

			serviceFilter := c.parseFilter(filterStr)
			if serviceFilter != nil {
				c.monitor.AddFilter(serviceFilter)
			}
		}
	}

	// If no filters specified, add default filters
	if len(c.monitor.filters) == 0 {
		// Monitor all non-user services by default
		c.monitor.AddFilter(SystemServicesFilter())
	}

	// Set rate limit
	if rateLimit, ok := config.Config["rate_limit"].(int); ok {
		c.monitor.rateLimit = NewRateLimiter(rateLimit)
	}

	return nil
}

func (c *Collector) parseFilter(filterStr string) ServiceFilter {
	parts := strings.Split(filterStr, ":")
	if len(parts) != 2 {
		return nil
	}

	filterType := strings.ToLower(parts[0])
	filterValue := parts[1]

	switch filterType {
	case "state":
		states := strings.Split(filterValue, ",")
		return func(state ServiceState) bool {
			for _, s := range states {
				if state.ActiveState == strings.TrimSpace(s) {
					return true
				}
			}
			return false
		}

	case "type":
		if filterValue == "service" {
			return func(state ServiceState) bool {
				return isServiceUnit(state.Name)
			}
		}

	case "name":
		return func(state ServiceState) bool {
			return strings.Contains(state.Name, filterValue)
		}

	case "exclude":
		pattern := filterValue
		return func(state ServiceState) bool {
			matched, _ := matchPattern(state.Name, pattern)
			return !matched
		}

	case "container":
		if filterValue == "true" {
			return ContainerServicesFilter()
		}
	}

	return nil
}

func (c *Collector) serviceToResource(name string, service *MonitoredService) types.Resource {
	resource := types.Resource{
		ID:       fmt.Sprintf("systemd:service:%s", name),
		Type:     "systemd:service",
		Name:     name,
		Provider: "systemd",
		Region:   "local",
		Configuration: map[string]interface{}{
			"active_state":  service.State.ActiveState,
			"sub_state":     service.State.SubState,
			"load_state":    service.State.LoadState,
			"description":   service.State.Description,
			"main_pid":      service.State.MainPID,
			"restart_count": service.State.RestartCount,
			"failure_count": service.FailureCount,
		},
		Metadata: types.ResourceMetadata{
			UpdatedAt: service.State.LastStateChange,
			AdditionalData: map[string]interface{}{
				"last_seen": service.LastSeen,
			},
		},
		Tags: make(map[string]string),
	}

	// Add resource metrics if available
	if service.ResourceMetrics.SampleCount > 0 {
		resource.Configuration["memory_current"] = service.State.MemoryCurrent
		resource.Configuration["memory_peak"] = service.ResourceMetrics.MemoryPeak
		resource.Configuration["memory_average"] = service.ResourceMetrics.MemoryAverage
		resource.Configuration["cpu_total_nsec"] = service.State.CPUUsageNSec
		resource.Configuration["cpu_rate_percent"] = service.ResourceMetrics.CPURate
	}

	// Add restart pattern analysis if available
	if service.RestartPattern != nil && len(service.RestartPattern.Restarts) > 0 {
		analysis := service.RestartPattern.GetAnalysis()
		resource.Configuration["restart_pattern"] = map[string]interface{}{
			"pattern":          analysis.Pattern,
			"frequency":        service.RestartPattern.Frequency,
			"trend":            analysis.Trend.Direction,
			"confidence":       analysis.Confidence,
			"average_interval": analysis.AverageInterval.String(),
			"anomalous_count":  len(analysis.AnomalousRestarts),
		}

		// Add recommendations as tags
		for i, rec := range analysis.RecommendedActions {
			if i < 3 { // Limit to 3 recommendations
				resource.Tags[fmt.Sprintf("recommendation_%d", i+1)] = rec
			}
		}
	}

	// Add dependencies if tracked
	if len(service.Dependencies) > 0 {
		deps := make([]string, 0, len(service.Dependencies))
		for dep := range service.Dependencies {
			deps = append(deps, dep)
		}
		resource.Metadata.Dependencies = deps
	}

	// Add state history summary
	if len(service.StateHistory) > 0 {
		recentHistory := make([]map[string]interface{}, 0, 5)
		start := len(service.StateHistory) - 5
		if start < 0 {
			start = 0
		}

		for _, transition := range service.StateHistory[start:] {
			histEntry := map[string]interface{}{
				"timestamp": transition.Timestamp,
				"from":      transition.FromState,
				"to":        transition.ToState,
				"duration":  transition.Duration.String(),
			}
			if transition.FailureInfo != nil {
				histEntry["failure"] = transition.FailureInfo.Message
			}
			recentHistory = append(recentHistory, histEntry)
		}

		resource.Configuration["recent_transitions"] = recentHistory
	}

	// Add tags based on state
	resource.Tags["state"] = service.State.ActiveState
	if service.State.ActiveState == "failed" {
		resource.Tags["health"] = "unhealthy"
		resource.Tags["alert"] = "true"
	} else if service.State.ActiveState == "active" {
		resource.Tags["health"] = "healthy"
	}

	// Add pattern-based tags
	if service.RestartPattern != nil {
		resource.Tags["restart_pattern"] = service.RestartPattern.Pattern
		if service.RestartPattern.Pattern == "flapping" || service.RestartPattern.Pattern == "degrading" {
			resource.Tags["stability"] = "unstable"
			resource.Tags["alert"] = "true"
		}
	}

	return resource
}

// Utility functions

func generateSnapshotID() string {
	return fmt.Sprintf("systemd-%d", time.Now().Unix())
}

func isValidFilter(filter string) bool {
	parts := strings.Split(filter, ":")
	if len(parts) != 2 {
		return false
	}

	validTypes := map[string]bool{
		"state":     true,
		"type":      true,
		"name":      true,
		"exclude":   true,
		"container": true,
	}

	return validTypes[strings.ToLower(parts[0])]
}

func matchPattern(name, pattern string) (bool, error) {
	// Simple glob matching for now
	// In production, use a proper glob library
	if strings.Contains(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix), nil
	}
	return name == pattern, nil
}
