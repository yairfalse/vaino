package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// EventType represents different types of watch events
type EventType string

const (
	EventTypeResourceCreated  EventType = "created"
	EventTypeResourceDeleted  EventType = "deleted"
	EventTypeResourceModified EventType = "modified"
	EventTypeResourceMigrated EventType = "migrated"
)

// WatchEvent represents a single watch event
type WatchEvent struct {
	ID           string                 `json:"id"`
	Type         EventType              `json:"type"`
	Timestamp    time.Time              `json:"timestamp"`
	Provider     string                 `json:"provider"`
	Resource     types.Resource         `json:"resource"`
	Changes      []types.Change         `json:"changes,omitempty"`
	PreviousHash string                 `json:"previous_hash,omitempty"`
	CurrentHash  string                 `json:"current_hash,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// WatchConfig holds configuration for watching infrastructure changes
type WatchConfig struct {
	// Providers to watch
	Providers []string `json:"providers"`

	// Polling intervals per provider
	PollingIntervals map[string]time.Duration `json:"polling_intervals"`

	// Regions to watch
	Regions []string `json:"regions"`

	// Namespaces to watch (for Kubernetes)
	Namespaces []string `json:"namespaces"`

	// Resource types to watch
	ResourceTypes []string `json:"resource_types"`

	// Minimum severity to report
	MinSeverity types.DriftSeverity `json:"min_severity"`

	// Buffer size for event channels
	BufferSize int `json:"buffer_size"`

	// Enable incremental scanning
	IncrementalScanning bool `json:"incremental_scanning"`

	// Memory optimization settings
	MemoryOptimization bool `json:"memory_optimization"`
}

// DefaultWatchConfig returns a default watch configuration
func DefaultWatchConfig() WatchConfig {
	return WatchConfig{
		Providers: []string{"terraform", "aws", "gcp", "kubernetes"},
		PollingIntervals: map[string]time.Duration{
			"terraform":  30 * time.Second,
			"aws":        60 * time.Second,
			"gcp":        60 * time.Second,
			"kubernetes": 15 * time.Second,
		},
		Regions:             []string{},
		Namespaces:          []string{},
		ResourceTypes:       []string{},
		MinSeverity:         types.DriftSeverityLow,
		BufferSize:          1000,
		IncrementalScanning: true,
		MemoryOptimization:  true,
	}
}

// ProviderWatcher watches changes for a specific provider
type ProviderWatcher struct {
	mu              sync.RWMutex
	provider        string
	collector       collectors.EnhancedCollector
	config          collectors.CollectorConfig
	pollingInterval time.Duration
	resourceCache   map[string]types.Resource
	resourceHashes  map[string]string
	eventChannel    chan WatchEvent
	ctx             context.Context
	cancel          context.CancelFunc
	running         bool
	stats           WatcherStats
	lastScan        time.Time
	incrementalMode bool
	memoryOptimized bool
}

// WatcherStats holds statistics for a provider watcher
type WatcherStats struct {
	TotalEvents     int64         `json:"total_events"`
	EventsPerMinute float64       `json:"events_per_minute"`
	LastEvent       time.Time     `json:"last_event"`
	ScanCount       int64         `json:"scan_count"`
	AverageScanTime time.Duration `json:"average_scan_time"`
	ErrorCount      int64         `json:"error_count"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	MemoryUsage     int64         `json:"memory_usage"`
}

// NewProviderWatcher creates a new provider watcher
func NewProviderWatcher(provider string, collector collectors.EnhancedCollector, config collectors.CollectorConfig, pollingInterval time.Duration, bufferSize int) *ProviderWatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &ProviderWatcher{
		provider:        provider,
		collector:       collector,
		config:          config,
		pollingInterval: pollingInterval,
		resourceCache:   make(map[string]types.Resource),
		resourceHashes:  make(map[string]string),
		eventChannel:    make(chan WatchEvent, bufferSize),
		ctx:             ctx,
		cancel:          cancel,
		running:         false,
		stats:           WatcherStats{},
		incrementalMode: true,
		memoryOptimized: true,
	}
}

// Start begins watching for changes
func (pw *ProviderWatcher) Start() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.running {
		return fmt.Errorf("provider watcher for %s is already running", pw.provider)
	}

	pw.running = true
	go pw.watchLoop()

	return nil
}

// Stop stops watching for changes
func (pw *ProviderWatcher) Stop() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if !pw.running {
		return fmt.Errorf("provider watcher for %s is not running", pw.provider)
	}

	pw.cancel()
	pw.running = false
	close(pw.eventChannel)

	return nil
}

// IsRunning returns whether the watcher is currently running
func (pw *ProviderWatcher) IsRunning() bool {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	return pw.running
}

// EventChannel returns the event channel for this watcher
func (pw *ProviderWatcher) EventChannel() <-chan WatchEvent {
	return pw.eventChannel
}

// GetStats returns current statistics
func (pw *ProviderWatcher) GetStats() WatcherStats {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	return pw.stats
}

// watchLoop is the main loop that performs periodic scans
func (pw *ProviderWatcher) watchLoop() {
	ticker := time.NewTicker(pw.pollingInterval)
	defer ticker.Stop()

	// Perform initial scan
	pw.performScan()

	for {
		select {
		case <-pw.ctx.Done():
			return
		case <-ticker.C:
			pw.performScan()
		}
	}
}

// performScan performs a scan and detects changes
func (pw *ProviderWatcher) performScan() {
	startTime := time.Now()

	// Create scan context with timeout
	scanCtx, cancel := context.WithTimeout(pw.ctx, 5*time.Minute)
	defer cancel()

	// Perform collection
	snapshot, err := pw.collector.Collect(scanCtx, pw.config)
	if err != nil {
		pw.mu.Lock()
		pw.stats.ErrorCount++
		pw.mu.Unlock()
		return
	}

	// Update stats
	pw.mu.Lock()
	pw.stats.ScanCount++
	pw.lastScan = time.Now()
	scanDuration := time.Since(startTime)
	pw.stats.AverageScanTime = time.Duration((int64(pw.stats.AverageScanTime) + int64(scanDuration)) / 2)
	pw.mu.Unlock()

	// Process resources and detect changes
	pw.processResources(snapshot.Resources)

	// Memory optimization
	if pw.memoryOptimized {
		pw.optimizeMemory()
	}
}

// processResources processes resources and detects changes
func (pw *ProviderWatcher) processResources(resources []types.Resource) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	currentResources := make(map[string]types.Resource)
	currentHashes := make(map[string]string)

	// Process each resource
	for _, resource := range resources {
		currentResources[resource.ID] = resource
		currentHashes[resource.ID] = resource.Hash()

		// Check for changes
		if cachedResource, exists := pw.resourceCache[resource.ID]; exists {
			// Resource exists - check for modifications
			if currentHashes[resource.ID] != pw.resourceHashes[resource.ID] {
				// Resource modified
				event := WatchEvent{
					ID:           fmt.Sprintf("%s-%s-%d", pw.provider, resource.ID, time.Now().UnixNano()),
					Type:         EventTypeResourceModified,
					Timestamp:    time.Now(),
					Provider:     pw.provider,
					Resource:     resource,
					PreviousHash: pw.resourceHashes[resource.ID],
					CurrentHash:  currentHashes[resource.ID],
					Metadata: map[string]interface{}{
						"scan_duration": pw.stats.AverageScanTime.String(),
					},
				}

				// Calculate changes
				event.Changes = pw.calculateChanges(cachedResource, resource)

				pw.sendEvent(event)
			}
		} else {
			// New resource
			event := WatchEvent{
				ID:          fmt.Sprintf("%s-%s-%d", pw.provider, resource.ID, time.Now().UnixNano()),
				Type:        EventTypeResourceCreated,
				Timestamp:   time.Now(),
				Provider:    pw.provider,
				Resource:    resource,
				CurrentHash: currentHashes[resource.ID],
				Metadata: map[string]interface{}{
					"scan_duration": pw.stats.AverageScanTime.String(),
				},
			}

			pw.sendEvent(event)
		}
	}

	// Check for deleted resources
	for resourceID := range pw.resourceCache {
		if _, exists := currentResources[resourceID]; !exists {
			// Resource deleted
			event := WatchEvent{
				ID:           fmt.Sprintf("%s-%s-%d", pw.provider, resourceID, time.Now().UnixNano()),
				Type:         EventTypeResourceDeleted,
				Timestamp:    time.Now(),
				Provider:     pw.provider,
				Resource:     pw.resourceCache[resourceID],
				PreviousHash: pw.resourceHashes[resourceID],
				Metadata: map[string]interface{}{
					"scan_duration": pw.stats.AverageScanTime.String(),
				},
			}

			pw.sendEvent(event)
		}
	}

	// Update cache
	pw.resourceCache = currentResources
	pw.resourceHashes = currentHashes
}

// calculateChanges calculates the differences between two resources
func (pw *ProviderWatcher) calculateChanges(oldResource, newResource types.Resource) []types.Change {
	var changes []types.Change

	// Compare basic fields
	if oldResource.Name != newResource.Name {
		changes = append(changes, types.Change{
			Path:       "name",
			OldValue:   oldResource.Name,
			NewValue:   newResource.Name,
			ChangeType: types.ChangeTypeModified,
		})
	}

	if oldResource.Region != newResource.Region {
		changes = append(changes, types.Change{
			Path:       "region",
			OldValue:   oldResource.Region,
			NewValue:   newResource.Region,
			ChangeType: types.ChangeTypeModified,
		})
	}

	// Compare configuration
	for key, newValue := range newResource.Configuration {
		if oldValue, exists := oldResource.Configuration[key]; exists {
			if oldValue != newValue {
				changes = append(changes, types.Change{
					Path:       fmt.Sprintf("configuration.%s", key),
					OldValue:   oldValue,
					NewValue:   newValue,
					ChangeType: types.ChangeTypeModified,
				})
			}
		} else {
			changes = append(changes, types.Change{
				Path:       fmt.Sprintf("configuration.%s", key),
				OldValue:   nil,
				NewValue:   newValue,
				ChangeType: types.ChangeTypeAdded,
			})
		}
	}

	// Check for removed configuration
	for key, oldValue := range oldResource.Configuration {
		if _, exists := newResource.Configuration[key]; !exists {
			changes = append(changes, types.Change{
				Path:       fmt.Sprintf("configuration.%s", key),
				OldValue:   oldValue,
				NewValue:   nil,
				ChangeType: types.ChangeTypeRemoved,
			})
		}
	}

	// Compare tags
	for key, newValue := range newResource.Tags {
		if oldValue, exists := oldResource.Tags[key]; exists {
			if oldValue != newValue {
				changes = append(changes, types.Change{
					Path:       fmt.Sprintf("tags.%s", key),
					OldValue:   oldValue,
					NewValue:   newValue,
					ChangeType: types.ChangeTypeModified,
				})
			}
		} else {
			changes = append(changes, types.Change{
				Path:       fmt.Sprintf("tags.%s", key),
				OldValue:   nil,
				NewValue:   newValue,
				ChangeType: types.ChangeTypeAdded,
			})
		}
	}

	// Check for removed tags
	for key, oldValue := range oldResource.Tags {
		if _, exists := newResource.Tags[key]; !exists {
			changes = append(changes, types.Change{
				Path:       fmt.Sprintf("tags.%s", key),
				OldValue:   oldValue,
				NewValue:   nil,
				ChangeType: types.ChangeTypeRemoved,
			})
		}
	}

	return changes
}

// sendEvent sends an event to the channel
func (pw *ProviderWatcher) sendEvent(event WatchEvent) {
	select {
	case pw.eventChannel <- event:
		pw.stats.TotalEvents++
		pw.stats.LastEvent = time.Now()

		// Update events per minute
		if pw.stats.TotalEvents > 0 {
			duration := time.Since(time.Now().Add(-time.Minute))
			pw.stats.EventsPerMinute = float64(pw.stats.TotalEvents) / duration.Minutes()
		}
	default:
		// Channel is full, drop event
		pw.stats.ErrorCount++
	}
}

// optimizeMemory performs memory optimization
func (pw *ProviderWatcher) optimizeMemory() {
	// Implement memory optimization logic
	// This could include:
	// - Clearing old hashes
	// - Compacting resource cache
	// - Running garbage collection

	// For now, just clear old hashes if cache is too large
	if len(pw.resourceHashes) > 10000 {
		// Keep only recent hashes
		newHashes := make(map[string]string)
		for id, hash := range pw.resourceHashes {
			if _, exists := pw.resourceCache[id]; exists {
				newHashes[id] = hash
			}
		}
		pw.resourceHashes = newHashes
	}
}
