package systemd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ServiceMonitor monitors systemd services and tracks their states
type ServiceMonitor struct {
	dbus           *DBusConnection
	services       map[string]*MonitoredService
	mu             sync.RWMutex
	changeHandlers []ChangeHandler
	filters        []ServiceFilter
	rateLimit      *RateLimiter
}

// MonitoredService represents a service being monitored
type MonitoredService struct {
	State           ServiceState
	LastSeen        time.Time
	StateHistory    []StateTransition
	FailureCount    int
	RestartPattern  *RestartPattern
	Dependencies    map[string]bool
	Dependents      map[string]bool
	ResourceMetrics ResourceMetrics
}

// StateTransition represents a state change in a service
type StateTransition struct {
	Timestamp   time.Time
	FromState   string
	ToState     string
	SubState    string
	Reason      string
	Duration    time.Duration // Time spent in previous state
	FailureInfo *FailureInfo
}

// FailureInfo contains details about a service failure
type FailureInfo struct {
	ExitCode   int
	Signal     int
	CoreDumped bool
	Message    string
}

// ResourceMetrics tracks resource usage for a service
type ResourceMetrics struct {
	MemoryPeak    uint64
	MemoryAverage uint64
	CPUTotal      time.Duration
	CPURate       float64 // CPU usage percentage
	LastUpdated   time.Time
	SampleCount   int
}

// ChangeHandler is called when a service state changes
type ChangeHandler func(service *MonitoredService, change StateTransition)

// ServiceFilter determines which services to monitor
type ServiceFilter func(state ServiceState) bool

// RateLimiter limits the rate of events processed
type RateLimiter struct {
	mu           sync.Mutex
	events       map[string]time.Time
	maxPerMinute int
	window       time.Duration
}

// NewServiceMonitor creates a new service monitor
func NewServiceMonitor(dbus *DBusConnection) *ServiceMonitor {
	return &ServiceMonitor{
		dbus:      dbus,
		services:  make(map[string]*MonitoredService),
		filters:   make([]ServiceFilter, 0),
		rateLimit: NewRateLimiter(1000), // 1000 events per minute default
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	return &RateLimiter{
		events:       make(map[string]time.Time),
		maxPerMinute: maxPerMinute,
		window:       time.Minute,
	}
}

// AddFilter adds a service filter
func (sm *ServiceMonitor) AddFilter(filter ServiceFilter) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.filters = append(sm.filters, filter)
}

// AddChangeHandler adds a handler for state changes
func (sm *ServiceMonitor) AddChangeHandler(handler ChangeHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.changeHandlers = append(sm.changeHandlers, handler)
}

// Start starts monitoring services
func (sm *ServiceMonitor) Start(ctx context.Context) error {
	// Initial scan
	if err := sm.scanServices(ctx); err != nil {
		return fmt.Errorf("initial scan failed: %w", err)
	}

	// Build dependency graph
	sm.buildDependencyGraph()

	// Start watching for changes
	changes := make(chan StateChange, 100)
	if err := sm.dbus.WatchStateChanges(ctx, changes); err != nil {
		return fmt.Errorf("failed to start watching: %w", err)
	}

	// Start resource monitoring
	go sm.monitorResources(ctx)

	// Process state changes
	go sm.processChanges(ctx, changes)

	// Periodic rescan to catch missed events
	go sm.periodicRescan(ctx)

	return nil
}

// scanServices performs an initial scan of all services
func (sm *ServiceMonitor) scanServices(ctx context.Context) error {
	services, err := sm.dbus.ListUnits(ctx)
	if err != nil {
		return err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, state := range services {
		if sm.shouldMonitor(state) {
			monitored := &MonitoredService{
				State:          state,
				LastSeen:       time.Now(),
				StateHistory:   make([]StateTransition, 0),
				Dependencies:   make(map[string]bool),
				Dependents:     make(map[string]bool),
				RestartPattern: NewRestartPattern(),
			}
			sm.services[state.Name] = monitored
		}
	}

	return nil
}

// shouldMonitor checks if a service should be monitored based on filters
func (sm *ServiceMonitor) shouldMonitor(state ServiceState) bool {
	if len(sm.filters) == 0 {
		return true // Monitor all if no filters
	}

	for _, filter := range sm.filters {
		if filter(state) {
			return true
		}
	}
	return false
}

// buildDependencyGraph builds the service dependency graph
func (sm *ServiceMonitor) buildDependencyGraph() {
	for name, service := range sm.services {
		for _, dep := range service.State.Dependencies {
			service.Dependencies[dep] = true

			// Add reverse dependency
			if depService, exists := sm.services[dep]; exists {
				depService.Dependents[name] = true
			}
		}
	}
}

// processChanges processes state change events
func (sm *ServiceMonitor) processChanges(ctx context.Context, changes <-chan StateChange) {
	for {
		select {
		case <-ctx.Done():
			return
		case change := <-changes:
			sm.handleStateChange(change)
		}
	}
}

// handleStateChange handles a single state change
func (sm *ServiceMonitor) handleStateChange(change StateChange) {
	// Rate limiting
	if !sm.rateLimit.Allow(change.ServiceName) {
		return
	}

	sm.mu.Lock()
	service, exists := sm.services[change.ServiceName]
	if !exists {
		// New service appeared
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		state, err := sm.dbus.GetServiceState(ctx, change.ServiceName)
		cancel()

		if err != nil {
			sm.mu.Unlock()
			return
		}

		if sm.shouldMonitor(*state) {
			service = &MonitoredService{
				State:          *state,
				LastSeen:       time.Now(),
				StateHistory:   make([]StateTransition, 0),
				Dependencies:   make(map[string]bool),
				Dependents:     make(map[string]bool),
				RestartPattern: NewRestartPattern(),
			}
			sm.services[change.ServiceName] = service
		} else {
			sm.mu.Unlock()
			return
		}
	}

	// Calculate duration in previous state
	var duration time.Duration
	if len(service.StateHistory) > 0 {
		lastTransition := service.StateHistory[len(service.StateHistory)-1]
		duration = change.Timestamp.Sub(lastTransition.Timestamp)
	}

	// Create transition record
	transition := StateTransition{
		Timestamp: change.Timestamp,
		FromState: change.OldState,
		ToState:   change.NewState,
		SubState:  change.SubState,
		Reason:    change.Reason,
		Duration:  duration,
	}

	// Check for failure
	if change.NewState == "failed" {
		service.FailureCount++
		transition.FailureInfo = sm.getFailureInfo(change.ServiceName)
	}

	// Update service state
	service.State.ActiveState = change.NewState
	service.State.SubState = change.SubState
	service.State.LastStateChange = change.Timestamp
	service.LastSeen = time.Now()

	// Add to history (keep last 100 transitions)
	service.StateHistory = append(service.StateHistory, transition)
	if len(service.StateHistory) > 100 {
		service.StateHistory = service.StateHistory[1:]
	}

	// Update restart pattern
	if change.OldState == "failed" && change.NewState == "activating" {
		service.RestartPattern.RecordRestart(change.Timestamp)
		service.State.RestartCount++
	}

	// Copy handlers to avoid holding lock during callbacks
	handlers := make([]ChangeHandler, len(sm.changeHandlers))
	copy(handlers, sm.changeHandlers)
	sm.mu.Unlock()

	// Notify handlers
	for _, handler := range handlers {
		handler(service, transition)
	}
}

// getFailureInfo retrieves failure information for a service
func (sm *ServiceMonitor) getFailureInfo(serviceName string) *FailureInfo {
	// In a real implementation, this would query systemd for exit status
	// For now, return a placeholder
	return &FailureInfo{
		Message: "Service failed",
	}
}

// monitorResources periodically updates resource metrics
func (sm *ServiceMonitor) monitorResources(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.updateResourceMetrics(ctx)
		}
	}
}

// updateResourceMetrics updates resource usage for all monitored services
func (sm *ServiceMonitor) updateResourceMetrics(ctx context.Context) {
	sm.mu.RLock()
	serviceNames := make([]string, 0, len(sm.services))
	for name := range sm.services {
		serviceNames = append(serviceNames, name)
	}
	sm.mu.RUnlock()

	for _, name := range serviceNames {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		state, err := sm.dbus.GetServiceState(ctx, name)
		cancel()

		if err != nil {
			continue
		}

		sm.mu.Lock()
		if service, exists := sm.services[name]; exists {
			// Update metrics
			metrics := &service.ResourceMetrics

			// Update memory
			if state.MemoryCurrent > metrics.MemoryPeak {
				metrics.MemoryPeak = state.MemoryCurrent
			}

			// Calculate average
			if metrics.SampleCount == 0 {
				metrics.MemoryAverage = state.MemoryCurrent
			} else {
				metrics.MemoryAverage = (metrics.MemoryAverage*uint64(metrics.SampleCount) + state.MemoryCurrent) / uint64(metrics.SampleCount+1)
			}

			// Update CPU
			if state.CPUUsageNSec > 0 {
				cpuDuration := time.Duration(state.CPUUsageNSec)
				if metrics.LastUpdated.IsZero() {
					metrics.CPUTotal = cpuDuration
				} else {
					timeDiff := time.Since(metrics.LastUpdated)
					cpuDiff := cpuDuration - metrics.CPUTotal
					metrics.CPURate = float64(cpuDiff) / float64(timeDiff) * 100
					metrics.CPUTotal = cpuDuration
				}
			}

			metrics.LastUpdated = time.Now()
			metrics.SampleCount++

			// Update main state
			service.State = *state
			service.LastSeen = time.Now()
		}
		sm.mu.Unlock()
	}
}

// periodicRescan performs periodic full rescans to catch missed events
func (sm *ServiceMonitor) periodicRescan(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.scanServices(ctx)
		}
	}
}

// GetService returns a monitored service by name
func (sm *ServiceMonitor) GetService(name string) (*MonitoredService, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	service, exists := sm.services[name]
	return service, exists
}

// GetAllServices returns all monitored services
func (sm *ServiceMonitor) GetAllServices() map[string]*MonitoredService {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*MonitoredService, len(sm.services))
	for k, v := range sm.services {
		result[k] = v
	}
	return result
}

// GetFailedServices returns all services in failed state
func (sm *ServiceMonitor) GetFailedServices() []*MonitoredService {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	failed := make([]*MonitoredService, 0)
	for _, service := range sm.services {
		if service.State.ActiveState == "failed" {
			failed = append(failed, service)
		}
	}

	return failed
}

// GetServicesByState returns services in a specific state
func (sm *ServiceMonitor) GetServicesByState(state string) []*MonitoredService {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make([]*MonitoredService, 0)
	for _, service := range sm.services {
		if service.State.ActiveState == state {
			result = append(result, service)
		}
	}

	return result
}

// GetDependencyChain returns the dependency chain for a service
func (sm *ServiceMonitor) GetDependencyChain(serviceName string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	visited := make(map[string]bool)
	chain := make([]string, 0)

	sm.walkDependencies(serviceName, visited, &chain)

	return chain
}

// walkDependencies recursively walks service dependencies
func (sm *ServiceMonitor) walkDependencies(serviceName string, visited map[string]bool, chain *[]string) {
	if visited[serviceName] {
		return
	}

	visited[serviceName] = true
	*chain = append(*chain, serviceName)

	if service, exists := sm.services[serviceName]; exists {
		deps := make([]string, 0, len(service.Dependencies))
		for dep := range service.Dependencies {
			deps = append(deps, dep)
		}

		// Sort for consistent ordering
		sort.Strings(deps)

		for _, dep := range deps {
			sm.walkDependencies(dep, visited, chain)
		}
	}
}

// RateLimiter methods

// Allow checks if an event should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean old events
	cutoff := now.Add(-rl.window)
	for k, t := range rl.events {
		if t.Before(cutoff) {
			delete(rl.events, k)
		}
	}

	// Check rate limit
	count := 0
	prefix := key + ":"
	for k := range rl.events {
		if strings.HasPrefix(k, prefix) {
			count++
		}
	}

	if count >= rl.maxPerMinute {
		return false
	}

	// Record event
	rl.events[fmt.Sprintf("%s:%d", key, now.UnixNano())] = now
	return true
}

// Common filters

// ActiveServicesFilter filters only active services
func ActiveServicesFilter() ServiceFilter {
	return func(state ServiceState) bool {
		return state.ActiveState == "active"
	}
}

// FailedServicesFilter filters only failed services
func FailedServicesFilter() ServiceFilter {
	return func(state ServiceState) bool {
		return state.ActiveState == "failed"
	}
}

// SystemServicesFilter filters only system services
func SystemServicesFilter() ServiceFilter {
	return func(state ServiceState) bool {
		return !strings.Contains(state.Name, "@") && !strings.Contains(state.Name, "user@")
	}
}

// ContainerServicesFilter filters container-related services
func ContainerServicesFilter() ServiceFilter {
	return func(state ServiceState) bool {
		name := strings.ToLower(state.Name)
		return strings.Contains(name, "docker") ||
			strings.Contains(name, "containerd") ||
			strings.Contains(name, "podman") ||
			strings.Contains(name, "crio")
	}
}
