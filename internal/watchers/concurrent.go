package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/aws"
	"github.com/yairfalse/wgo/internal/collectors/gcp"
	"github.com/yairfalse/wgo/internal/collectors/kubernetes"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
)

// ConcurrentWatcher manages multiple provider watchers concurrently
type ConcurrentWatcher struct {
	mu               sync.RWMutex
	providerWatchers map[string]*ProviderWatcher
	eventMerger      *EventMerger
	correlator       *ConcurrentCorrelator
	config           WatchConfig
	globalEventChan  chan WatchEvent
	running          bool
	stats            ConcurrentWatcherStats
	ctx              context.Context
	cancel           context.CancelFunc
}

// ConcurrentWatcherStats holds statistics for the concurrent watcher
type ConcurrentWatcherStats struct {
	TotalProviders    int                     `json:"total_providers"`
	ActiveProviders   int                     `json:"active_providers"`
	TotalEvents       int64                   `json:"total_events"`
	EventsPerSecond   float64                 `json:"events_per_second"`
	ProviderStats     map[string]WatcherStats `json:"provider_stats"`
	CorrelatedEvents  int64                   `json:"correlated_events"`
	LastActivity      time.Time               `json:"last_activity"`
	MemoryUsage       int64                   `json:"memory_usage"`
	ProcessingLatency time.Duration           `json:"processing_latency"`
	ErrorRate         float64                 `json:"error_rate"`
}

// NewConcurrentWatcher creates a new concurrent watcher
func NewConcurrentWatcher(config WatchConfig) *ConcurrentWatcher {
	ctx, cancel := context.WithCancel(context.Background())

	cw := &ConcurrentWatcher{
		providerWatchers: make(map[string]*ProviderWatcher),
		eventMerger:      NewEventMerger(config.BufferSize),
		correlator:       NewConcurrentCorrelator(),
		config:           config,
		globalEventChan:  make(chan WatchEvent, config.BufferSize*len(config.Providers)),
		running:          false,
		stats: ConcurrentWatcherStats{
			ProviderStats: make(map[string]WatcherStats),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return cw
}

// Start begins concurrent watching across all configured providers
func (cw *ConcurrentWatcher) Start() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if cw.running {
		return fmt.Errorf("concurrent watcher is already running")
	}

	// Initialize provider watchers
	for _, provider := range cw.config.Providers {
		if err := cw.initializeProviderWatcher(provider); err != nil {
			return fmt.Errorf("failed to initialize %s watcher: %w", provider, err)
		}
	}

	// Start event merger
	if err := cw.eventMerger.Start(); err != nil {
		return fmt.Errorf("failed to start event merger: %w", err)
	}

	// Start correlator
	if err := cw.correlator.Start(); err != nil {
		return fmt.Errorf("failed to start correlator: %w", err)
	}

	// Start all provider watchers
	for provider, watcher := range cw.providerWatchers {
		if err := watcher.Start(); err != nil {
			return fmt.Errorf("failed to start %s watcher: %w", provider, err)
		}

		// Connect watcher to event merger
		cw.eventMerger.AddEventSource(provider, watcher.EventChannel())
	}

	cw.running = true
	cw.stats.TotalProviders = len(cw.providerWatchers)
	cw.stats.ActiveProviders = len(cw.providerWatchers)

	// Start event processing loop
	go cw.eventProcessingLoop()

	// Start statistics collection
	go cw.statsCollectionLoop()

	return nil
}

// Stop stops all concurrent watching
func (cw *ConcurrentWatcher) Stop() error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if !cw.running {
		return fmt.Errorf("concurrent watcher is not running")
	}

	// Stop all provider watchers
	var stopErrors []error
	for provider, watcher := range cw.providerWatchers {
		if err := watcher.Stop(); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("failed to stop %s watcher: %w", provider, err))
		}
	}

	// Stop correlator
	if err := cw.correlator.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("failed to stop correlator: %w", err))
	}

	// Stop event merger
	if err := cw.eventMerger.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("failed to stop event merger: %w", err))
	}

	// Cancel context
	cw.cancel()
	cw.running = false
	cw.stats.ActiveProviders = 0

	// Close global event channel
	close(cw.globalEventChan)

	// Return combined errors if any
	if len(stopErrors) > 0 {
		return fmt.Errorf("multiple stop errors: %v", stopErrors)
	}

	return nil
}

// IsRunning returns whether the concurrent watcher is running
func (cw *ConcurrentWatcher) IsRunning() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return cw.running
}

// EventChannel returns the global event channel
func (cw *ConcurrentWatcher) EventChannel() <-chan WatchEvent {
	return cw.globalEventChan
}

// GetStats returns current statistics
func (cw *ConcurrentWatcher) GetStats() ConcurrentWatcherStats {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	// Update provider stats
	for provider, watcher := range cw.providerWatchers {
		cw.stats.ProviderStats[provider] = watcher.GetStats()
	}

	return cw.stats
}

// GetProviderWatcher returns a specific provider watcher
func (cw *ConcurrentWatcher) GetProviderWatcher(provider string) (*ProviderWatcher, bool) {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	watcher, exists := cw.providerWatchers[provider]
	return watcher, exists
}

// AddProvider adds a new provider to watch
func (cw *ConcurrentWatcher) AddProvider(provider string) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	if _, exists := cw.providerWatchers[provider]; exists {
		return fmt.Errorf("provider %s is already being watched", provider)
	}

	if err := cw.initializeProviderWatcher(provider); err != nil {
		return fmt.Errorf("failed to initialize %s watcher: %w", provider, err)
	}

	// If we're running, start the new watcher
	if cw.running {
		watcher := cw.providerWatchers[provider]
		if err := watcher.Start(); err != nil {
			delete(cw.providerWatchers, provider)
			return fmt.Errorf("failed to start %s watcher: %w", provider, err)
		}

		// Connect to event merger
		cw.eventMerger.AddEventSource(provider, watcher.EventChannel())
		cw.stats.TotalProviders++
		cw.stats.ActiveProviders++
	}

	return nil
}

// RemoveProvider removes a provider from watching
func (cw *ConcurrentWatcher) RemoveProvider(provider string) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	watcher, exists := cw.providerWatchers[provider]
	if !exists {
		return fmt.Errorf("provider %s is not being watched", provider)
	}

	// Stop the watcher
	if err := watcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop %s watcher: %w", provider, err)
	}

	// Remove from event merger
	cw.eventMerger.RemoveEventSource(provider)

	// Remove from our tracking
	delete(cw.providerWatchers, provider)
	delete(cw.stats.ProviderStats, provider)

	if cw.running {
		cw.stats.ActiveProviders--
	}

	return nil
}

// initializeProviderWatcher initializes a provider watcher
func (cw *ConcurrentWatcher) initializeProviderWatcher(provider string) error {
	// Get collector for provider
	collector, err := cw.getCollectorForProvider(provider)
	if err != nil {
		return fmt.Errorf("failed to get collector for %s: %w", provider, err)
	}

	// Create collector config
	config := collectors.CollectorConfig{
		Regions:    cw.config.Regions,
		Namespaces: cw.config.Namespaces,
		Config:     make(map[string]interface{}),
	}

	// Get polling interval
	pollingInterval := cw.config.PollingIntervals[provider]
	if pollingInterval == 0 {
		pollingInterval = 60 * time.Second // Default
	}

	// Create provider watcher
	watcher := NewProviderWatcher(provider, collector, config, pollingInterval, cw.config.BufferSize)

	// Configure watcher
	watcher.incrementalMode = cw.config.IncrementalScanning
	watcher.memoryOptimized = cw.config.MemoryOptimization

	cw.providerWatchers[provider] = watcher
	return nil
}

// getCollectorForProvider returns the appropriate collector for a provider
func (cw *ConcurrentWatcher) getCollectorForProvider(provider string) (collectors.EnhancedCollector, error) {
	switch provider {
	case "terraform":
		return terraform.NewTerraformCollector(), nil
	case "aws":
		return aws.NewAWSCollector(), nil
	case "gcp":
		return gcp.NewGCPCollector(), nil
	case "kubernetes":
		return kubernetes.NewKubernetesCollector(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// eventProcessingLoop processes events from the event merger
func (cw *ConcurrentWatcher) eventProcessingLoop() {
	for {
		select {
		case <-cw.ctx.Done():
			return
		case event, ok := <-cw.eventMerger.EventChannel():
			if !ok {
				return
			}

			startTime := time.Now()

			// Process event through correlator
			correlatedEvents := cw.correlator.ProcessEvent(event)

			// Send original event
			select {
			case cw.globalEventChan <- event:
				cw.mu.Lock()
				cw.stats.TotalEvents++
				cw.stats.LastActivity = time.Now()
				cw.stats.ProcessingLatency = time.Since(startTime)
				cw.mu.Unlock()
			default:
				// Channel is full
				cw.mu.Lock()
				cw.stats.ErrorRate++
				cw.mu.Unlock()
			}

			// Send correlated events
			for _, correlatedEvent := range correlatedEvents {
				select {
				case cw.globalEventChan <- correlatedEvent:
					cw.mu.Lock()
					cw.stats.CorrelatedEvents++
					cw.mu.Unlock()
				default:
					// Channel is full
					cw.mu.Lock()
					cw.stats.ErrorRate++
					cw.mu.Unlock()
				}
			}
		}
	}
}

// statsCollectionLoop periodically collects and updates statistics
func (cw *ConcurrentWatcher) statsCollectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cw.ctx.Done():
			return
		case <-ticker.C:
			cw.updateStats()
		}
	}
}

// updateStats updates the statistics
func (cw *ConcurrentWatcher) updateStats() {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Calculate events per second
	if cw.stats.TotalEvents > 0 && !cw.stats.LastActivity.IsZero() {
		duration := time.Since(cw.stats.LastActivity)
		if duration > 0 {
			cw.stats.EventsPerSecond = float64(cw.stats.TotalEvents) / duration.Seconds()
		}
	}

	// Calculate error rate
	totalOperations := cw.stats.TotalEvents + cw.stats.CorrelatedEvents
	if totalOperations > 0 {
		cw.stats.ErrorRate = (cw.stats.ErrorRate / float64(totalOperations)) * 100
	}

	// Update active providers count
	activeCount := 0
	for _, watcher := range cw.providerWatchers {
		if watcher.IsRunning() {
			activeCount++
		}
	}
	cw.stats.ActiveProviders = activeCount

	// Update provider stats
	for provider, watcher := range cw.providerWatchers {
		cw.stats.ProviderStats[provider] = watcher.GetStats()
	}
}

// GetActiveProviders returns a list of currently active providers
func (cw *ConcurrentWatcher) GetActiveProviders() []string {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	var active []string
	for provider, watcher := range cw.providerWatchers {
		if watcher.IsRunning() {
			active = append(active, provider)
		}
	}

	return active
}

// RestartProvider restarts a specific provider watcher
func (cw *ConcurrentWatcher) RestartProvider(provider string) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	watcher, exists := cw.providerWatchers[provider]
	if !exists {
		return fmt.Errorf("provider %s is not being watched", provider)
	}

	// Stop the watcher
	if err := watcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop %s watcher: %w", provider, err)
	}

	// Start it again
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to restart %s watcher: %w", provider, err)
	}

	return nil
}

// UpdateConfig updates the watch configuration
func (cw *ConcurrentWatcher) UpdateConfig(newConfig WatchConfig) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Update internal config
	cw.config = newConfig

	// Update event merger buffer size
	cw.eventMerger.UpdateBufferSize(newConfig.BufferSize)

	// Update individual provider watchers
	for provider, watcher := range cw.providerWatchers {
		if interval, exists := newConfig.PollingIntervals[provider]; exists {
			watcher.mu.Lock()
			watcher.pollingInterval = interval
			watcher.incrementalMode = newConfig.IncrementalScanning
			watcher.memoryOptimized = newConfig.MemoryOptimization
			watcher.mu.Unlock()
		}
	}

	return nil
}
