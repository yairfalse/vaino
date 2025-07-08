package analyzer

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
)

// Event represents a timeline event
type Event struct {
	ID          string
	Timestamp   time.Time
	EventType   string
	ResourceID  string
	ResourceType string
	Namespace   string
	Description string
	Severity    string
	Metadata    map[string]interface{}
}

// Timeline represents a processed timeline with events and analysis
type Timeline struct {
	Events        []Event
	TimeWindows   []TimeWindow
	ProcessingTime time.Duration
	Stats         TimelineStats
}

// TimeWindow represents a time segment with aggregated events
type TimeWindow struct {
	Start       time.Time
	End         time.Time
	Duration    time.Duration
	Events      []Event
	EventCount  int
	Severity    string
	Summary     string
}

// TimelineStats provides timeline analysis statistics
type TimelineStats struct {
	TotalEvents      int
	TimeWindows      int
	HighSeverity     int
	MediumSeverity   int
	LowSeverity      int
	EventsPerMinute  float64
	PeakActivity     time.Time
	QuietPeriods     int
	AverageWindowSize int
}

// ConcurrentTimelineProcessor processes timeline events concurrently
type ConcurrentTimelineProcessor struct {
	windowSize   time.Duration
	workerCount  int
	eventChan    chan Event
	resultChan   chan TimelineResult
	mutex        sync.RWMutex
}

// TimelineResult represents the result from timeline processing
type TimelineResult struct {
	TimeWindow  TimeWindow
	ProcessTime time.Duration
	Error       error
}

// NewConcurrentTimelineProcessor creates a new concurrent timeline processor
func NewConcurrentTimelineProcessor(windowSize time.Duration) *ConcurrentTimelineProcessor {
	workerCount := runtime.NumCPU()
	if workerCount < 2 {
		workerCount = 2
	}
	if workerCount > 6 {
		workerCount = 6 // Timeline processing doesn't need as many workers
	}

	return &ConcurrentTimelineProcessor{
		windowSize:  windowSize,
		workerCount: workerCount,
		eventChan:   make(chan Event, 100),
		resultChan:  make(chan TimelineResult, 100),
	}
}

// BuildTimelineConcurrent processes events into a timeline using concurrent workers
func (tp *ConcurrentTimelineProcessor) BuildTimelineConcurrent(events []Event) *Timeline {
	if len(events) == 0 {
		return &Timeline{
			Events:      []Event{},
			TimeWindows: []TimeWindow{},
			Stats:       TimelineStats{},
		}
	}

	startTime := time.Now()
	
	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Create time windows
	windows := tp.createTimeWindows(events)
	
	// Process windows concurrently
	processedWindows := tp.processWindowsConcurrent(windows, events)
	
	// Generate timeline stats
	stats := tp.generateTimelineStats(processedWindows, events)
	
	timeline := &Timeline{
		Events:        events,
		TimeWindows:   processedWindows,
		ProcessingTime: time.Since(startTime),
		Stats:         stats,
	}

	return timeline
}

// createTimeWindows creates time windows based on event timestamps
func (tp *ConcurrentTimelineProcessor) createTimeWindows(events []Event) []TimeWindow {
	if len(events) == 0 {
		return []TimeWindow{}
	}

	firstEvent := events[0].Timestamp
	lastEvent := events[len(events)-1].Timestamp
	
	// Round down to nearest window boundary
	start := firstEvent.Truncate(tp.windowSize)
	end := lastEvent.Add(tp.windowSize).Truncate(tp.windowSize)
	
	var windows []TimeWindow
	current := start
	
	for current.Before(end) {
		windowEnd := current.Add(tp.windowSize)
		windows = append(windows, TimeWindow{
			Start:    current,
			End:      windowEnd,
			Duration: tp.windowSize,
			Events:   []Event{},
		})
		current = windowEnd
	}
	
	return windows
}

// processWindowsConcurrent processes time windows using concurrent workers
func (tp *ConcurrentTimelineProcessor) processWindowsConcurrent(windows []TimeWindow, events []Event) []TimeWindow {
	if len(windows) <= 2 {
		// For small datasets, use sequential processing
		return tp.processWindowsSequential(windows, events)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	results := make([]TimelineResult, len(windows))
	
	// Process windows in chunks
	chunkSize := len(windows) / tp.workerCount
	if chunkSize < 1 {
		chunkSize = 1
	}
	
	for i := 0; i < len(windows); i += chunkSize {
		end := i + chunkSize
		if end > len(windows) {
			end = len(windows)
		}
		
		wg.Add(1)
		go func(startIdx, endIdx int) {
			defer wg.Done()
			
			for j := startIdx; j < endIdx; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					processStart := time.Now()
					processedWindow := tp.processTimeWindow(windows[j], events)
					results[j] = TimelineResult{
						TimeWindow:  processedWindow,
						ProcessTime: time.Since(processStart),
					}
				}
			}
		}(i, end)
	}
	
	wg.Wait()
	
	// Extract processed windows
	processedWindows := make([]TimeWindow, len(windows))
	for i, result := range results {
		processedWindows[i] = result.TimeWindow
	}
	
	return processedWindows
}

// processWindowsSequential processes time windows sequentially (fallback)
func (tp *ConcurrentTimelineProcessor) processWindowsSequential(windows []TimeWindow, events []Event) []TimeWindow {
	processedWindows := make([]TimeWindow, len(windows))
	
	for i, window := range windows {
		processedWindows[i] = tp.processTimeWindow(window, events)
	}
	
	return processedWindows
}

// processTimeWindow processes a single time window
func (tp *ConcurrentTimelineProcessor) processTimeWindow(window TimeWindow, events []Event) TimeWindow {
	// Find events within this window
	var windowEvents []Event
	for _, event := range events {
		if !event.Timestamp.Before(window.Start) && event.Timestamp.Before(window.End) {
			windowEvents = append(windowEvents, event)
		}
	}
	
	window.Events = windowEvents
	window.EventCount = len(windowEvents)
	
	// Calculate window severity
	window.Severity = tp.calculateWindowSeverity(windowEvents)
	
	// Generate window summary
	window.Summary = tp.generateWindowSummary(windowEvents)
	
	return window
}

// calculateWindowSeverity calculates the overall severity for a time window
func (tp *ConcurrentTimelineProcessor) calculateWindowSeverity(events []Event) string {
	if len(events) == 0 {
		return "none"
	}
	
	highCount := 0
	mediumCount := 0
	lowCount := 0
	
	for _, event := range events {
		switch event.Severity {
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}
	
	// Determine overall severity
	if highCount > 0 {
		return "high"
	} else if mediumCount > len(events)/2 {
		return "medium"
	} else if len(events) > 5 {
		return "medium"
	} else {
		return "low"
	}
}

// generateWindowSummary generates a summary for a time window
func (tp *ConcurrentTimelineProcessor) generateWindowSummary(events []Event) string {
	if len(events) == 0 {
		return "No activity"
	}
	
	// Count events by type
	eventTypes := make(map[string]int)
	resourceTypes := make(map[string]int)
	
	for _, event := range events {
		eventTypes[event.EventType]++
		resourceTypes[event.ResourceType]++
	}
	
	// Generate summary based on dominant patterns
	if len(eventTypes) == 1 {
		for eventType := range eventTypes {
			if len(resourceTypes) == 1 {
				for resourceType := range resourceTypes {
					return fmt.Sprintf("%d %s events on %s resources", 
						len(events), eventType, resourceType)
				}
			}
			return fmt.Sprintf("%d %s events", len(events), eventType)
		}
	}
	
	// Multiple event types
	summary := fmt.Sprintf("%d events", len(events))
	
	// Add most common event type
	maxCount := 0
	var dominantType string
	for eventType, count := range eventTypes {
		if count > maxCount {
			maxCount = count
			dominantType = eventType
		}
	}
	
	if maxCount > len(events)/2 {
		summary += fmt.Sprintf(" (mostly %s)", dominantType)
	}
	
	return summary
}

// generateTimelineStats generates comprehensive timeline statistics
func (tp *ConcurrentTimelineProcessor) generateTimelineStats(windows []TimeWindow, events []Event) TimelineStats {
	stats := TimelineStats{
		TotalEvents: len(events),
		TimeWindows: len(windows),
	}
	
	// Count severity levels
	for _, event := range events {
		switch event.Severity {
		case "high":
			stats.HighSeverity++
		case "medium":
			stats.MediumSeverity++
		case "low":
			stats.LowSeverity++
		}
	}
	
	// Calculate events per minute
	if len(events) > 0 && len(windows) > 0 {
		firstEvent := events[0].Timestamp
		lastEvent := events[len(events)-1].Timestamp
		duration := lastEvent.Sub(firstEvent)
		if duration > 0 {
			stats.EventsPerMinute = float64(len(events)) / duration.Minutes()
		}
	}
	
	// Find peak activity time
	maxEvents := 0
	for _, window := range windows {
		if window.EventCount > maxEvents {
			maxEvents = window.EventCount
			stats.PeakActivity = window.Start
		}
	}
	
	// Count quiet periods (windows with no events)
	totalEvents := 0
	for _, window := range windows {
		if window.EventCount == 0 {
			stats.QuietPeriods++
		}
		totalEvents += window.EventCount
	}
	
	// Calculate average window size
	if len(windows) > 0 {
		stats.AverageWindowSize = totalEvents / len(windows)
	}
	
	return stats
}

// ConvertChangesToEvents converts SimpleChange objects to Event objects
func ConvertChangesToEvents(changes []differ.SimpleChange) []Event {
	events := make([]Event, len(changes))
	
	for i, change := range changes {
		severity := "medium"
		if change.Type == "added" {
			severity = "low"
		} else if change.Type == "removed" {
			severity = "high"
		}
		
		events[i] = Event{
			ID:          change.ResourceID,
			Timestamp:   change.Timestamp,
			EventType:   change.Type,
			ResourceID:  change.ResourceID,
			ResourceType: change.ResourceType,
			Namespace:   change.Namespace,
			Description: fmt.Sprintf("%s %s %s", change.Type, change.ResourceType, change.ResourceName),
			Severity:    severity,
			Metadata: map[string]interface{}{
				"resource_name": change.ResourceName,
				"details":       change.Details,
			},
		}
	}
	
	return events
}

// BuildTimelineFromChanges builds a timeline directly from changes
func BuildTimelineFromChanges(changes []differ.SimpleChange, windowSize time.Duration) *Timeline {
	events := ConvertChangesToEvents(changes)
	processor := NewConcurrentTimelineProcessor(windowSize)
	return processor.BuildTimelineConcurrent(events)
}

// FormatTimelineToString formats a timeline for display
func FormatTimelineToString(timeline *Timeline) string {
	if timeline == nil || len(timeline.TimeWindows) == 0 {
		return "No timeline events found"
	}
	
	output := fmt.Sprintf("Timeline Analysis (%d events over %d windows)\n", 
		timeline.Stats.TotalEvents, timeline.Stats.TimeWindows)
	output += fmt.Sprintf("Processing time: %v\n", timeline.ProcessingTime)
	output += fmt.Sprintf("Events per minute: %.2f\n", timeline.Stats.EventsPerMinute)
	output += fmt.Sprintf("Peak activity: %s\n", timeline.Stats.PeakActivity.Format("15:04:05"))
	output += "\nTime Windows:\n"
	
	for _, window := range timeline.TimeWindows {
		if window.EventCount > 0 {
			output += fmt.Sprintf("  %s - %s (%s): %s [%d events]\n",
				window.Start.Format("15:04:05"),
				window.End.Format("15:04:05"),
				window.Severity,
				window.Summary,
				window.EventCount)
		}
	}
	
	return output
}

// GetTimelineMetrics returns timeline metrics for monitoring
func GetTimelineMetrics(timeline *Timeline) map[string]interface{} {
	if timeline == nil {
		return make(map[string]interface{})
	}
	
	return map[string]interface{}{
		"total_events":       timeline.Stats.TotalEvents,
		"time_windows":       timeline.Stats.TimeWindows,
		"high_severity":      timeline.Stats.HighSeverity,
		"medium_severity":    timeline.Stats.MediumSeverity,
		"low_severity":       timeline.Stats.LowSeverity,
		"events_per_minute":  timeline.Stats.EventsPerMinute,
		"peak_activity":      timeline.Stats.PeakActivity,
		"quiet_periods":      timeline.Stats.QuietPeriods,
		"average_window_size": timeline.Stats.AverageWindowSize,
		"processing_time_ms": timeline.ProcessingTime.Milliseconds(),
	}
}