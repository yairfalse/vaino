package watchers

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventMerger merges events from multiple provider watchers
type EventMerger struct {
	mu           sync.RWMutex
	eventSources map[string]<-chan WatchEvent
	mergedChan   chan WatchEvent
	bufferSize   int
	running      bool
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	stats        EventMergerStats
}

// EventMergerStats holds statistics for the event merger
type EventMergerStats struct {
	TotalProcessed    int64                    `json:"total_processed"`
	ProcessingRate    float64                  `json:"processing_rate"`
	BufferUtilization float64                  `json:"buffer_utilization"`
	DroppedEvents     int64                    `json:"dropped_events"`
	SourceStats       map[string]SourceStats   `json:"source_stats"`
	LastActivity      time.Time                `json:"last_activity"`
	AverageLatency    time.Duration            `json:"average_latency"`
}

// SourceStats holds statistics for a specific event source
type SourceStats struct {
	EventsProcessed int64         `json:"events_processed"`
	LastEvent       time.Time     `json:"last_event"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorCount      int64         `json:"error_count"`
}

// NewEventMerger creates a new event merger
func NewEventMerger(bufferSize int) *EventMerger {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EventMerger{
		eventSources: make(map[string]<-chan WatchEvent),
		mergedChan:   make(chan WatchEvent, bufferSize),
		bufferSize:   bufferSize,
		running:      false,
		ctx:          ctx,
		cancel:       cancel,
		stats: EventMergerStats{
			SourceStats: make(map[string]SourceStats),
		},
	}
}

// Start begins event merging
func (em *EventMerger) Start() error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if em.running {
		return fmt.Errorf("event merger is already running")
	}
	
	em.running = true
	
	// Start goroutine for each event source
	for provider, eventChan := range em.eventSources {
		em.wg.Add(1)
		go em.processEventSource(provider, eventChan)
	}
	
	// Start statistics collection
	go em.statsCollectionLoop()
	
	return nil
}

// Stop stops event merging
func (em *EventMerger) Stop() error {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if !em.running {
		return fmt.Errorf("event merger is not running")
	}
	
	em.cancel()
	em.running = false
	
	// Wait for all goroutines to finish
	em.wg.Wait()
	
	// Close merged channel
	close(em.mergedChan)
	
	return nil
}

// AddEventSource adds a new event source
func (em *EventMerger) AddEventSource(provider string, eventChan <-chan WatchEvent) {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	em.eventSources[provider] = eventChan
	em.stats.SourceStats[provider] = SourceStats{}
	
	// If we're already running, start processing this source
	if em.running {
		em.wg.Add(1)
		go em.processEventSource(provider, eventChan)
	}
}

// RemoveEventSource removes an event source
func (em *EventMerger) RemoveEventSource(provider string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	delete(em.eventSources, provider)
	delete(em.stats.SourceStats, provider)
}

// EventChannel returns the merged event channel
func (em *EventMerger) EventChannel() <-chan WatchEvent {
	return em.mergedChan
}

// GetStats returns current statistics
func (em *EventMerger) GetStats() EventMergerStats {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	// Update buffer utilization
	em.stats.BufferUtilization = float64(len(em.mergedChan)) / float64(em.bufferSize) * 100
	
	return em.stats
}

// UpdateBufferSize updates the buffer size
func (em *EventMerger) UpdateBufferSize(newSize int) {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if em.running {
		// Can't change buffer size while running
		return
	}
	
	em.bufferSize = newSize
	em.mergedChan = make(chan WatchEvent, newSize)
}

// IsRunning returns whether the event merger is running
func (em *EventMerger) IsRunning() bool {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.running
}

// processEventSource processes events from a specific source
func (em *EventMerger) processEventSource(provider string, eventChan <-chan WatchEvent) {
	defer em.wg.Done()
	
	for {
		select {
		case <-em.ctx.Done():
			return
		case event, ok := <-eventChan:
			if !ok {
				// Channel closed
				return
			}
			
			startTime := time.Now()
			
			// Try to send event to merged channel
			select {
			case em.mergedChan <- event:
				// Successfully sent
				em.updateSourceStats(provider, startTime)
			default:
				// Channel is full, drop event
				em.mu.Lock()
				em.stats.DroppedEvents++
				if stats, exists := em.stats.SourceStats[provider]; exists {
					stats.ErrorCount++
					em.stats.SourceStats[provider] = stats
				}
				em.mu.Unlock()
			}
		}
	}
}

// updateSourceStats updates statistics for a specific source
func (em *EventMerger) updateSourceStats(provider string, startTime time.Time) {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	latency := time.Since(startTime)
	
	stats := em.stats.SourceStats[provider]
	stats.EventsProcessed++
	stats.LastEvent = time.Now()
	
	// Update average latency
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = time.Duration((int64(stats.AverageLatency) + int64(latency)) / 2)
	}
	
	em.stats.SourceStats[provider] = stats
	em.stats.TotalProcessed++
	em.stats.LastActivity = time.Now()
	
	// Update overall average latency
	if em.stats.AverageLatency == 0 {
		em.stats.AverageLatency = latency
	} else {
		em.stats.AverageLatency = time.Duration((int64(em.stats.AverageLatency) + int64(latency)) / 2)
	}
}

// statsCollectionLoop periodically collects and updates statistics
func (em *EventMerger) statsCollectionLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-em.ctx.Done():
			return
		case <-ticker.C:
			em.updateProcessingRate()
		}
	}
}

// updateProcessingRate updates the processing rate
func (em *EventMerger) updateProcessingRate() {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	if em.stats.TotalProcessed > 0 && !em.stats.LastActivity.IsZero() {
		duration := time.Since(em.stats.LastActivity)
		if duration > 0 {
			em.stats.ProcessingRate = float64(em.stats.TotalProcessed) / duration.Seconds()
		}
	}
}

// GetBufferUtilization returns the current buffer utilization percentage
func (em *EventMerger) GetBufferUtilization() float64 {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	return float64(len(em.mergedChan)) / float64(em.bufferSize) * 100
}

// GetActiveSourceCount returns the number of active event sources
func (em *EventMerger) GetActiveSourceCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	return len(em.eventSources)
}

// FlushEvents flushes all pending events from the buffer
func (em *EventMerger) FlushEvents() []WatchEvent {
	em.mu.Lock()
	defer em.mu.Unlock()
	
	var events []WatchEvent
	
	// Drain the channel
	for len(em.mergedChan) > 0 {
		select {
		case event := <-em.mergedChan:
			events = append(events, event)
		default:
			break
		}
	}
	
	return events
}

// GetProviderEventCount returns the event count for a specific provider
func (em *EventMerger) GetProviderEventCount(provider string) int64 {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	if stats, exists := em.stats.SourceStats[provider]; exists {
		return stats.EventsProcessed
	}
	
	return 0
}

// GetHealthStatus returns the health status of the event merger
func (em *EventMerger) GetHealthStatus() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	health := map[string]interface{}{
		"running":            em.running,
		"buffer_utilization": em.GetBufferUtilization(),
		"active_sources":     len(em.eventSources),
		"total_processed":    em.stats.TotalProcessed,
		"dropped_events":     em.stats.DroppedEvents,
		"processing_rate":    em.stats.ProcessingRate,
		"average_latency":    em.stats.AverageLatency.String(),
	}
	
	// Add health status based on metrics
	if em.stats.DroppedEvents > 0 {
		health["status"] = "warning"
		health["message"] = fmt.Sprintf("Dropped %d events", em.stats.DroppedEvents)
	} else if em.GetBufferUtilization() > 80 {
		health["status"] = "warning"
		health["message"] = "Buffer utilization is high"
	} else if em.running {
		health["status"] = "healthy"
		health["message"] = "Event merger is running normally"
	} else {
		health["status"] = "stopped"
		health["message"] = "Event merger is not running"
	}
	
	return health
}

// PrioritizeEvents reorders events based on priority
func (em *EventMerger) PrioritizeEvents(events []WatchEvent) []WatchEvent {
	// Sort events by priority (critical first, then by timestamp)
	prioritized := make([]WatchEvent, len(events))
	copy(prioritized, events)
	
	// Simple priority sorting - can be enhanced with more sophisticated logic
	for i := 0; i < len(prioritized)-1; i++ {
		for j := i + 1; j < len(prioritized); j++ {
			if em.shouldPrioritize(prioritized[j], prioritized[i]) {
				prioritized[i], prioritized[j] = prioritized[j], prioritized[i]
			}
		}
	}
	
	return prioritized
}

// shouldPrioritize determines if event A should be prioritized over event B
func (em *EventMerger) shouldPrioritize(a, b WatchEvent) bool {
	// Priority order: Created > Deleted > Modified
	priorityOrder := map[EventType]int{
		EventTypeResourceCreated:  3,
		EventTypeResourceDeleted:  2,
		EventTypeResourceModified: 1,
		EventTypeResourceMigrated: 1,
	}
	
	priorityA := priorityOrder[a.Type]
	priorityB := priorityOrder[b.Type]
	
	if priorityA != priorityB {
		return priorityA > priorityB
	}
	
	// If same priority, order by timestamp (newer first)
	return a.Timestamp.After(b.Timestamp)
}

// GetEventsByProvider returns events grouped by provider
func (em *EventMerger) GetEventsByProvider(events []WatchEvent) map[string][]WatchEvent {
	grouped := make(map[string][]WatchEvent)
	
	for _, event := range events {
		grouped[event.Provider] = append(grouped[event.Provider], event)
	}
	
	return grouped
}

// GetEventsByType returns events grouped by type
func (em *EventMerger) GetEventsByType(events []WatchEvent) map[EventType][]WatchEvent {
	grouped := make(map[EventType][]WatchEvent)
	
	for _, event := range events {
		grouped[event.Type] = append(grouped[event.Type], event)
	}
	
	return grouped
}