package journald

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/pkg/types"
)

// Collector implements the EnhancedCollector interface for journald
type Collector struct {
	streamer   *LogStreamer
	parser     *LogParser
	patternLib *PatternLibrary
	filter     *LogFilter
	correlator *HistoricalCorrelator
	config     collectors.CollectorConfig
	mu         sync.RWMutex
	running    bool
	stats      CollectorStats
}

// CollectorStats tracks collector performance and statistics
type CollectorStats struct {
	StartTime          time.Time     `json:"start_time"`
	TotalEntries       int64         `json:"total_entries"`
	ProcessedEntries   int64         `json:"processed_entries"`
	FilteredEntries    int64         `json:"filtered_entries"`
	CriticalEvents     int64         `json:"critical_events"`
	OOMEvents          int64         `json:"oom_events"`
	MemoryUsageBytes   int64         `json:"memory_usage_bytes"`
	ProcessingLatency  time.Duration `json:"processing_latency"`
	LastProcessedEntry time.Time     `json:"last_processed_entry"`
	EventsPerSecond    float64       `json:"events_per_second"`
	FilterEfficiency   float64       `json:"filter_efficiency"`
}

// HistoricalCorrelator provides correlation analysis with historical data
type HistoricalCorrelator struct {
	eventHistory   []ParsedEvent
	maxHistorySize int
	correlationDb  map[string][]CorrelationPattern
	analysisWindow time.Duration
	mu             sync.RWMutex
}

// CorrelationPattern represents a discovered correlation pattern
type CorrelationPattern struct {
	EventSequence  []EventType   `json:"event_sequence"`
	TimeWindow     time.Duration `json:"time_window"`
	Confidence     float64       `json:"confidence"`
	Occurrences    int64         `json:"occurrences"`
	LastSeen       time.Time     `json:"last_seen"`
	Impact         string        `json:"impact"`
	Recommendation string        `json:"recommendation"`
}

// NewCollector creates a new journald collector
func NewCollector() (*Collector, error) {
	// Check if we're on a Linux system
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("journald collector only supported on Linux")
	}

	return &Collector{
		parser:     NewLogParser(),
		patternLib: NewPatternLibrary(),
		correlator: NewHistoricalCorrelator(),
		stats: CollectorStats{
			StartTime: time.Now(),
		},
	}, nil
}

// NewHistoricalCorrelator creates a new historical correlator
func NewHistoricalCorrelator() *HistoricalCorrelator {
	return &HistoricalCorrelator{
		eventHistory:   make([]ParsedEvent, 0),
		maxHistorySize: 10000,
		correlationDb:  make(map[string][]CorrelationPattern),
		analysisWindow: 24 * time.Hour,
	}
}

// Name returns the name of the collector
func (c *Collector) Name() string {
	return "journald"
}

// Status returns the current status of the collector
func (c *Collector) Status() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return "Not running"
	}

	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / 1024 / 1024

	// Calculate processing rate
	uptime := time.Since(c.stats.StartTime)
	eventsPerSec := float64(c.stats.ProcessedEntries) / uptime.Seconds()

	return fmt.Sprintf("Running - Memory: %.1fMB, Events/sec: %.1f, Critical: %d, OOM: %d",
		memoryMB, eventsPerSec, c.stats.CriticalEvents, c.stats.OOMEvents)
}

// Collect gathers journald log entries and returns a snapshot
func (c *Collector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	// Set up filtering based on config
	if err := c.setupFiltering(config); err != nil {
		return nil, fmt.Errorf("failed to setup filtering: %w", err)
	}

	// Set up streaming based on config
	if err := c.setupStreaming(config); err != nil {
		return nil, fmt.Errorf("failed to setup streaming: %w", err)
	}

	// Start collection
	collectionCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	events, err := c.collectEvents(collectionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect events: %w", err)
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        generateSnapshotID(),
		Timestamp: time.Now(),
		Provider:  "journald",
		Resources: make([]types.Resource, 0),
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			Tags:             config.Tags,
		},
	}

	startTime := time.Now()

	// Convert events to resources
	for _, event := range events {
		resource := c.eventToResource(event)
		snapshot.Resources = append(snapshot.Resources, resource)
	}

	// Update metadata
	snapshot.Metadata.CollectionTime = time.Since(startTime)
	snapshot.Metadata.ResourceCount = len(snapshot.Resources)

	// Add collection statistics
	snapshot.Metadata.AdditionalData = map[string]interface{}{
		"collector_stats": c.getCollectorStats(),
		"filter_stats":    c.getFilterStats(),
		"pattern_stats":   c.patternLib.GetStats(),
	}

	return snapshot, nil
}

// Validate checks if the collector configuration is valid
func (c *Collector) Validate(config collectors.CollectorConfig) error {
	// Check OS
	if runtime.GOOS != "linux" {
		return fmt.Errorf("journald collector requires Linux")
	}

	// Validate rate limits
	if rateLimit, ok := config.Config["rate_limit"].(int); ok {
		if rateLimit < 100 || rateLimit > 100000 {
			return fmt.Errorf("rate_limit must be between 100 and 100000")
		}
	}

	// Validate memory limits
	if memLimit, ok := config.Config["memory_limit_mb"].(int); ok {
		if memLimit < 10 || memLimit > 1000 {
			return fmt.Errorf("memory_limit_mb must be between 10 and 1000")
		}
	}

	// Validate priority filters
	if minPriority, ok := config.Config["min_priority"].(int); ok {
		if minPriority < 0 || minPriority > 7 {
			return fmt.Errorf("min_priority must be between 0 and 7")
		}
	}

	return nil
}

// AutoDiscover attempts to automatically discover journald configuration
func (c *Collector) AutoDiscover() (collectors.CollectorConfig, error) {
	config := collectors.CollectorConfig{
		Config: make(map[string]interface{}),
		Tags:   make(map[string]string),
	}

	// Check if journald is available
	if runtime.GOOS != "linux" {
		return config, fmt.Errorf("journald not available on %s", runtime.GOOS)
	}

	// Test journalctl access
	testStreamer := NewLogStreamer(StreamConfig{
		Follow: false,
		Lines:  1,
	})

	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testStreamer.Start(testCtx, StreamConfig{Lines: 1}); err != nil {
		return config, fmt.Errorf("journalctl not accessible: %w", err)
	}
	testStreamer.Stop()

	// Set optimal default configuration
	config.Config["rate_limit"] = 10000
	config.Config["memory_limit_mb"] = 30
	config.Config["min_priority"] = 3 // Error and above
	config.Config["enable_oom_detection"] = true
	config.Config["enable_pattern_matching"] = true
	config.Config["enable_correlation"] = true
	config.Config["filter_noise"] = true
	config.Config["sample_rate"] = 1.0

	// Detect system characteristics
	config.Tags["os"] = "linux"
	config.Tags["collector"] = "journald"

	return config, nil
}

// SupportedRegions returns supported regions (not applicable for journald)
func (c *Collector) SupportedRegions() []string {
	return []string{"local"}
}

// Close closes the collector and releases resources
func (c *Collector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.running = false

	if c.streamer != nil {
		return c.streamer.Stop()
	}
	return nil
}

// Helper methods

func (c *Collector) setupFiltering(config collectors.CollectorConfig) error {
	// Default filter configuration
	filterConfig := FilterConfig{
		MaxEntriesPerSec:    10000,
		MaxEntriesPerMin:    100000,
		MaxEntriesPerHour:   1000000,
		EnableDeduplication: true,
		MinPriority:         3, // Error and above
		EnableSampling:      false,
		SampleRate:          1.0,
	}

	// Override with user configuration
	if rateLimit, ok := config.Config["rate_limit"].(int); ok {
		filterConfig.MaxEntriesPerSec = rateLimit
	}

	if minPriority, ok := config.Config["min_priority"].(int); ok {
		filterConfig.MinPriority = minPriority
	}

	if excludePatterns, ok := config.Config["exclude_patterns"].([]string); ok {
		filterConfig.ExcludePatterns = excludePatterns
	}

	if includePatterns, ok := config.Config["include_patterns"].([]string); ok {
		filterConfig.IncludePatterns = includePatterns
	}

	if sampleRate, ok := config.Config["sample_rate"].(float64); ok {
		if sampleRate < 1.0 {
			filterConfig.EnableSampling = true
			filterConfig.SampleRate = sampleRate
		}
	}

	c.filter = NewLogFilter(filterConfig)
	return nil
}

func (c *Collector) setupStreaming(config collectors.CollectorConfig) error {
	// Default stream configuration
	streamConfig := StreamConfig{
		Follow:     true,
		Lines:      1000, // Initial historical entries
		RateLimit:  10000,
		BufferSize: 10000,
		Since:      time.Now().Add(-1 * time.Hour), // Last hour by default
	}

	// Override with user configuration
	if follow, ok := config.Config["follow"].(bool); ok {
		streamConfig.Follow = follow
	}

	if lines, ok := config.Config["historical_lines"].(int); ok {
		streamConfig.Lines = lines
	}

	if units, ok := config.Config["units"].([]string); ok {
		streamConfig.Units = units
	}

	if priorities, ok := config.Config["priorities"].([]int); ok {
		streamConfig.Priorities = priorities
	}

	c.streamer = NewLogStreamer(streamConfig)
	return nil
}

func (c *Collector) collectEvents(ctx context.Context) ([]ParsedEvent, error) {
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	events := make([]ParsedEvent, 0)

	// Set up streaming configuration
	streamConfig := StreamConfig{
		Follow:     false, // For snapshot collection, don't follow
		Lines:      1000,  // Collect recent entries
		RateLimit:  10000,
		BufferSize: 10000,
	}

	if err := c.streamer.Start(ctx, streamConfig); err != nil {
		return nil, err
	}
	defer c.streamer.Stop()

	// Process entries with timeout
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()

	processingStart := time.Now()
	entriesProcessed := 0

	for {
		select {
		case <-ctx.Done():
			c.updateStats(entriesProcessed, time.Since(processingStart))
			return events, ctx.Err()

		case <-timeout.C:
			c.updateStats(entriesProcessed, time.Since(processingStart))
			return events, nil // Timeout reached, return what we have

		case entry, ok := <-c.streamer.Entries():
			if !ok {
				c.updateStats(entriesProcessed, time.Since(processingStart))
				return events, nil // Stream closed
			}

			entriesProcessed++
			c.stats.TotalEntries++

			// Apply filtering
			if !c.filter.ShouldProcess(entry) {
				c.stats.FilteredEntries++
				continue
			}

			c.stats.ProcessedEntries++

			// Parse events
			parsedEvents := c.parser.Parse(entry)
			for _, event := range parsedEvents {
				// Check patterns
				matches := c.patternLib.ProcessEntry(entry)
				if len(matches) > 0 {
					// Enhance event with pattern information
					event.Tags = append(event.Tags, "pattern_matched")
					for _, match := range matches {
						if match.EventType == EventOOMKill {
							c.stats.OOMEvents++
						}
						if event.Severity == SeverityCritical {
							c.stats.CriticalEvents++
						}
					}
				}

				events = append(events, event)

				// Add to correlation history
				c.correlator.AddEvent(event)
			}

			// Check for completion (no follow mode)
			if len(events) >= 1000 { // Limit for snapshot collection
				c.updateStats(entriesProcessed, time.Since(processingStart))
				return events, nil
			}

		case err := <-c.streamer.Errors():
			if err != nil {
				return events, fmt.Errorf("streaming error: %w", err)
			}
		}
	}
}

func (c *Collector) eventToResource(event ParsedEvent) types.Resource {
	resource := types.Resource{
		ID:       fmt.Sprintf("journald:event:%s:%d", event.Type, event.Timestamp.Unix()),
		Type:     "journald:event",
		Name:     string(event.Type),
		Provider: "journald",
		Region:   "local",
		Configuration: map[string]interface{}{
			"event_type": event.Type,
			"severity":   event.Severity,
			"message":    event.Message,
			"confidence": event.Confidence,
			"source":     event.Source,
			"process":    event.Process,
			"pid":        event.PID,
			"unit":       event.Unit,
			"details":    event.Details,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: event.Timestamp,
			AdditionalData: map[string]interface{}{
				"raw_entry": event.RawEntry,
			},
		},
		Tags: make(map[string]string),
	}

	// Add tags
	for _, tag := range event.Tags {
		resource.Tags[tag] = "true"
	}

	// Add severity and type tags
	resource.Tags["severity"] = string(event.Severity)
	resource.Tags["event_type"] = string(event.Type)

	// Add priority tags for critical events
	if event.Severity == SeverityCritical {
		resource.Tags["critical"] = "true"
		resource.Tags["alert"] = "true"
	}

	// Add OOM-specific tags
	if event.Type == EventOOMKill {
		resource.Tags["oom"] = "true"
		resource.Tags["memory_issue"] = "true"
	}

	return resource
}

func (c *Collector) updateStats(entriesProcessed int, processingTime time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.ProcessingLatency = processingTime
	c.stats.LastProcessedEntry = time.Now()

	if entriesProcessed > 0 {
		c.stats.EventsPerSecond = float64(entriesProcessed) / processingTime.Seconds()
	}

	if c.stats.TotalEntries > 0 {
		c.stats.FilterEfficiency = float64(c.stats.FilteredEntries) / float64(c.stats.TotalEntries)
	}

	// Update memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.stats.MemoryUsageBytes = int64(m.Alloc)
}

func (c *Collector) getCollectorStats() CollectorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *Collector) getFilterStats() FilterStats {
	if c.filter != nil {
		return c.filter.GetStats()
	}
	return FilterStats{}
}

// HistoricalCorrelator methods

func (hc *HistoricalCorrelator) AddEvent(event ParsedEvent) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Add to history
	hc.eventHistory = append(hc.eventHistory, event)

	// Maintain history size
	if len(hc.eventHistory) > hc.maxHistorySize {
		hc.eventHistory = hc.eventHistory[1:]
	}

	// Analyze for correlations
	hc.analyzeCorrelations()
}

func (hc *HistoricalCorrelator) analyzeCorrelations() {
	// Look for event sequences within time windows
	// This is a simplified implementation - production would use more sophisticated algorithms

	recentEvents := hc.getRecentEvents(time.Hour)
	if len(recentEvents) < 2 {
		return
	}

	// Find sequences of critical events
	for i := 0; i < len(recentEvents)-1; i++ {
		if recentEvents[i].Severity == SeverityCritical {
			sequence := []EventType{recentEvents[i].Type}

			// Look for following events within 5 minutes
			for j := i + 1; j < len(recentEvents); j++ {
				if recentEvents[j].Timestamp.Sub(recentEvents[i].Timestamp) > 5*time.Minute {
					break
				}
				sequence = append(sequence, recentEvents[j].Type)
			}

			if len(sequence) > 1 {
				hc.recordCorrelationPattern(sequence)
			}
		}
	}
}

func (hc *HistoricalCorrelator) getRecentEvents(window time.Duration) []ParsedEvent {
	cutoff := time.Now().Add(-window)
	recent := make([]ParsedEvent, 0)

	for _, event := range hc.eventHistory {
		if event.Timestamp.After(cutoff) {
			recent = append(recent, event)
		}
	}

	return recent
}

func (hc *HistoricalCorrelator) recordCorrelationPattern(sequence []EventType) {
	key := generateSequenceKey(sequence)

	// Update or create pattern
	patterns := hc.correlationDb[key]
	found := false

	for i := range patterns {
		if sequenceEqual(patterns[i].EventSequence, sequence) {
			patterns[i].Occurrences++
			patterns[i].LastSeen = time.Now()
			patterns[i].Confidence = min(float64(patterns[i].Occurrences)/10.0, 1.0)
			found = true
			break
		}
	}

	if !found {
		patterns = append(patterns, CorrelationPattern{
			EventSequence: sequence,
			TimeWindow:    5 * time.Minute,
			Confidence:    0.1,
			Occurrences:   1,
			LastSeen:      time.Now(),
		})
	}

	hc.correlationDb[key] = patterns
}

// Utility functions

func generateSnapshotID() string {
	return fmt.Sprintf("journald-%d", time.Now().Unix())
}

func generateSequenceKey(sequence []EventType) string {
	key := ""
	for _, eventType := range sequence {
		key += string(eventType) + "-"
	}
	return key
}

func sequenceEqual(a, b []EventType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
