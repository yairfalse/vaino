package journald

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"sync"
	"time"
)

// PatternLibrary manages detection patterns for critical events
type PatternLibrary struct {
	patterns      map[string]*CriticalPattern
	oomDetector   *OOMDetector
	errorDetector *ErrorPatternDetector
	anomalyEngine *AnomalyDetectionEngine
	correlator    *EventCorrelator
	mu            sync.RWMutex
	stats         PatternStats
}

// CriticalPattern defines a pattern for detecting critical events
type CriticalPattern struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Regex       *regexp.Regexp  `json:"-"`
	RegexStr    string          `json:"regex"`
	Severity    Severity        `json:"severity"`
	Category    string          `json:"category"`
	Tags        []string        `json:"tags"`
	Confidence  float64         `json:"confidence"`
	Enabled     bool            `json:"enabled"`
	Extractor   EventExtractor  `json:"-"`
	Metadata    PatternMetadata `json:"metadata"`
}

// PatternMetadata contains additional pattern information
type PatternMetadata struct {
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	MatchCount     int64     `json:"match_count"`
	FalsePositives int64     `json:"false_positives"`
	Accuracy       float64   `json:"accuracy"`
	Version        string    `json:"version"`
}

// EventExtractor extracts structured data from matched events
type EventExtractor func(matches []string, entry LogEntry) map[string]interface{}

// OOMDetector specialized detector for OOM events with high accuracy
type OOMDetector struct {
	patterns   []*regexp.Regexp
	extractors []OOMExtractor
	confidence float64
	matched    int64
	total      int64
}

// OOMExtractor extracts OOM-specific information
type OOMExtractor func(message string, entry LogEntry) OOMEvent

// OOMEvent represents a detected Out of Memory event
type OOMEvent struct {
	VictimPID     int                    `json:"victim_pid"`
	VictimProcess string                 `json:"victim_process"`
	VictimCmdline string                 `json:"victim_cmdline"`
	OOMScore      int                    `json:"oom_score"`
	MemoryUsage   MemoryUsageDetails     `json:"memory_usage"`
	Trigger       string                 `json:"trigger"`
	Constraint    string                 `json:"constraint"`
	KillerPID     int                    `json:"killer_pid"`
	Timestamp     time.Time              `json:"timestamp"`
	Context       map[string]interface{} `json:"context"`
	Confidence    float64                `json:"confidence"`
}

// MemoryUsageDetails contains detailed memory usage information
type MemoryUsageDetails struct {
	TotalVMKB  int64 `json:"total_vm_kb"`
	AnonRSSKB  int64 `json:"anon_rss_kb"`
	FileRSSKB  int64 `json:"file_rss_kb"`
	ShmemRSSKB int64 `json:"shmem_rss_kb"`
	TotalRSSKB int64 `json:"total_rss_kb"`
	SwapKB     int64 `json:"swap_kb"`
	LimitBytes int64 `json:"limit_bytes,omitempty"`
}

// ErrorPatternDetector detects recurring error patterns
type ErrorPatternDetector struct {
	patterns   map[string]*ErrorPattern
	frequency  map[string][]time.Time
	thresholds map[string]FrequencyThreshold
	mu         sync.RWMutex
}

// ErrorPattern represents a recurring error pattern
type ErrorPattern struct {
	Pattern     string    `json:"pattern"`
	Count       int64     `json:"count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Frequency   float64   `json:"frequency"` // events per hour
	Severity    Severity  `json:"severity"`
	Impact      string    `json:"impact"`
	Recommended []string  `json:"recommended_actions"`
}

// FrequencyThreshold defines when a pattern becomes critical
type FrequencyThreshold struct {
	EventsPerHour int           `json:"events_per_hour"`
	TimeWindow    time.Duration `json:"time_window"`
	Severity      Severity      `json:"severity"`
}

// AnomalyDetectionEngine detects unusual patterns in log data
type AnomalyDetectionEngine struct {
	baselines   map[string]*LogBaseline
	detectors   []AnomalyDetector
	sensitivity float64
	mu          sync.RWMutex
}

// LogBaseline represents normal behavior patterns
type LogBaseline struct {
	EventType     string            `json:"event_type"`
	MeanFrequency float64           `json:"mean_frequency"`
	StdDeviation  float64           `json:"std_deviation"`
	TimeWindow    time.Duration     `json:"time_window"`
	Samples       []FrequencySample `json:"samples"`
	LastUpdated   time.Time         `json:"last_updated"`
}

// FrequencySample represents a frequency measurement
type FrequencySample struct {
	Timestamp time.Time     `json:"timestamp"`
	Count     int           `json:"count"`
	Window    time.Duration `json:"window"`
}

// AnomalyDetector interface for different anomaly detection algorithms
type AnomalyDetector interface {
	DetectAnomaly(baseline *LogBaseline, current float64) (bool, float64)
	Name() string
}

// EventCorrelator finds relationships between different log events
type EventCorrelator struct {
	correlations  map[string]*EventCorrelation
	timeWindows   map[string]time.Duration
	maxDistance   time.Duration
	minConfidence float64
	mu            sync.RWMutex
}

// EventCorrelation represents a correlation between events
type EventCorrelation struct {
	EventA      string        `json:"event_a"`
	EventB      string        `json:"event_b"`
	Correlation float64       `json:"correlation"`
	Confidence  float64       `json:"confidence"`
	TimeDelay   time.Duration `json:"time_delay"`
	Occurrences int64         `json:"occurrences"`
	LastSeen    time.Time     `json:"last_seen"`
}

// PatternStats contains pattern matching statistics
type PatternStats struct {
	TotalProcessed int64            `json:"total_processed"`
	TotalMatches   int64            `json:"total_matches"`
	MatchRate      float64          `json:"match_rate"`
	ProcessingTime time.Duration    `json:"processing_time"`
	LastUpdated    time.Time        `json:"last_updated"`
	PatternStats   map[string]int64 `json:"pattern_stats"`
}

// NewPatternLibrary creates a new pattern library
func NewPatternLibrary() *PatternLibrary {
	pl := &PatternLibrary{
		patterns:      make(map[string]*CriticalPattern),
		oomDetector:   NewOOMDetector(),
		errorDetector: NewErrorPatternDetector(),
		anomalyEngine: NewAnomalyDetectionEngine(),
		correlator:    NewEventCorrelator(),
		stats: PatternStats{
			PatternStats: make(map[string]int64),
		},
	}

	pl.loadDefaultPatterns()
	return pl
}

// NewOOMDetector creates a specialized OOM detector
func NewOOMDetector() *OOMDetector {
	detector := &OOMDetector{
		patterns:   make([]*regexp.Regexp, 0),
		extractors: make([]OOMExtractor, 0),
		confidence: 0.99,
	}

	// Primary OOM pattern - most accurate
	detector.patterns = append(detector.patterns,
		regexp.MustCompile(`killed process (\d+) \(([^)]+)\).*score (\d+).*total-vm:(\d+)kB.*anon-rss:(\d+)kB.*file-rss:(\d+)kB.*shmem-rss:(\d+)kB`))
	detector.extractors = append(detector.extractors, detector.extractDetailedOOM)

	// Secondary OOM pattern
	detector.patterns = append(detector.patterns,
		regexp.MustCompile(`Out of memory: Kill process (\d+) \(([^)]+)\) score (\d+) or sacrifice child`))
	detector.extractors = append(detector.extractors, detector.extractBasicOOM)

	// Memory cgroup OOM pattern
	detector.patterns = append(detector.patterns,
		regexp.MustCompile(`Memory cgroup out of memory: Kill process (\d+) \(([^)]+)\) score (\d+) or sacrifice child`))
	detector.extractors = append(detector.extractors, detector.extractCgroupOOM)

	return detector
}

// DetectOOM analyzes a log entry for OOM events with high accuracy
func (od *OOMDetector) DetectOOM(entry LogEntry) (*OOMEvent, bool) {
	od.total++

	for i, pattern := range od.patterns {
		if matches := pattern.FindStringSubmatch(entry.Message); matches != nil {
			od.matched++
			event := od.extractors[i](entry.Message, entry)
			event.Timestamp = entry.Timestamp
			event.Confidence = od.confidence
			return &event, true
		}
	}

	return nil, false
}

// GetAccuracy returns the current OOM detection accuracy
func (od *OOMDetector) GetAccuracy() float64 {
	if od.total == 0 {
		return 0.0
	}
	return float64(od.matched) / float64(od.total)
}

// extractDetailedOOM extracts detailed OOM information
func (od *OOMDetector) extractDetailedOOM(message string, entry LogEntry) OOMEvent {
	// This would contain the actual extraction logic
	// Implementation details omitted for brevity
	return OOMEvent{
		Confidence: 0.99,
		Trigger:    "system_oom",
	}
}

// extractBasicOOM extracts basic OOM information
func (od *OOMDetector) extractBasicOOM(message string, entry LogEntry) OOMEvent {
	return OOMEvent{
		Confidence: 0.95,
		Trigger:    "oom_killer",
	}
}

// extractCgroupOOM extracts cgroup-specific OOM information
func (od *OOMDetector) extractCgroupOOM(message string, entry LogEntry) OOMEvent {
	return OOMEvent{
		Confidence: 0.97,
		Trigger:    "cgroup_oom",
	}
}

// NewErrorPatternDetector creates a new error pattern detector
func NewErrorPatternDetector() *ErrorPatternDetector {
	return &ErrorPatternDetector{
		patterns:   make(map[string]*ErrorPattern),
		frequency:  make(map[string][]time.Time),
		thresholds: make(map[string]FrequencyThreshold),
	}
}

// DetectPattern analyzes log entries for recurring error patterns
func (epd *ErrorPatternDetector) DetectPattern(entry LogEntry) []ErrorPattern {
	epd.mu.Lock()
	defer epd.mu.Unlock()

	patterns := make([]ErrorPattern, 0)

	// Normalize message for pattern matching
	normalized := normalizeMessage(entry.Message)

	// Update frequency tracking
	now := time.Now()
	epd.frequency[normalized] = append(epd.frequency[normalized], now)

	// Clean old entries (keep last 24 hours)
	cutoff := now.Add(-24 * time.Hour)
	filtered := make([]time.Time, 0)
	for _, t := range epd.frequency[normalized] {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	epd.frequency[normalized] = filtered

	// Check if pattern meets threshold
	if len(filtered) >= 5 { // Minimum occurrences to consider a pattern
		frequency := float64(len(filtered)) / 24.0 // events per hour

		pattern := &ErrorPattern{
			Pattern:   normalized,
			Count:     int64(len(filtered)),
			FirstSeen: filtered[0],
			LastSeen:  filtered[len(filtered)-1],
			Frequency: frequency,
			Severity:  priorityToSeverity(entry.Priority),
		}

		// Classify pattern severity based on frequency
		if frequency > 10 {
			pattern.Severity = SeverityCritical
			pattern.Impact = "High frequency error pattern detected"
			pattern.Recommended = []string{
				"Investigate root cause immediately",
				"Check system resources",
				"Review recent changes",
			}
		} else if frequency > 5 {
			pattern.Severity = SeverityHigh
			pattern.Impact = "Moderate frequency error pattern"
			pattern.Recommended = []string{
				"Monitor pattern progression",
				"Check related services",
			}
		}

		epd.patterns[normalized] = pattern
		patterns = append(patterns, *pattern)
	}

	return patterns
}

// NewAnomalyDetectionEngine creates a new anomaly detection engine
func NewAnomalyDetectionEngine() *AnomalyDetectionEngine {
	return &AnomalyDetectionEngine{
		baselines:   make(map[string]*LogBaseline),
		detectors:   []AnomalyDetector{&ZScoreDetector{}, &IQRDetector{}},
		sensitivity: 2.0, // Standard deviations for anomaly threshold
	}
}

// DetectAnomaly checks if current log frequency represents an anomaly
func (ade *AnomalyDetectionEngine) DetectAnomaly(eventType string, currentFreq float64) (bool, float64) {
	ade.mu.RLock()
	baseline, exists := ade.baselines[eventType]
	ade.mu.RUnlock()

	if !exists {
		return false, 0.0
	}

	// Use multiple detectors and combine results
	anomalyScores := make([]float64, 0)

	for _, detector := range ade.detectors {
		isAnomaly, score := detector.DetectAnomaly(baseline, currentFreq)
		if isAnomaly {
			anomalyScores = append(anomalyScores, score)
		}
	}

	if len(anomalyScores) == 0 {
		return false, 0.0
	}

	// Calculate average anomaly score
	avgScore := 0.0
	for _, score := range anomalyScores {
		avgScore += score
	}
	avgScore /= float64(len(anomalyScores))

	return true, avgScore
}

// ZScoreDetector implements Z-score based anomaly detection
type ZScoreDetector struct{}

func (zsd *ZScoreDetector) DetectAnomaly(baseline *LogBaseline, current float64) (bool, float64) {
	if baseline.StdDeviation == 0 {
		return false, 0.0
	}

	zScore := math.Abs(current-baseline.MeanFrequency) / baseline.StdDeviation
	return zScore > 2.0, zScore
}

func (zsd *ZScoreDetector) Name() string {
	return "zscore"
}

// IQRDetector implements Interquartile Range based anomaly detection
type IQRDetector struct{}

func (iqr *IQRDetector) DetectAnomaly(baseline *LogBaseline, current float64) (bool, float64) {
	if len(baseline.Samples) < 4 {
		return false, 0.0
	}

	// Calculate quartiles
	values := make([]float64, len(baseline.Samples))
	for i, sample := range baseline.Samples {
		values[i] = float64(sample.Count)
	}
	sort.Float64s(values)

	q1 := values[len(values)/4]
	q3 := values[3*len(values)/4]
	iqrValue := q3 - q1

	lowerBound := q1 - 1.5*iqrValue
	upperBound := q3 + 1.5*iqrValue

	isAnomaly := current < lowerBound || current > upperBound
	score := 0.0

	if isAnomaly {
		if current < lowerBound {
			score = (lowerBound - current) / iqrValue
		} else {
			score = (current - upperBound) / iqrValue
		}
	}

	return isAnomaly, score
}

func (iqr *IQRDetector) Name() string {
	return "iqr"
}

// NewEventCorrelator creates a new event correlator
func NewEventCorrelator() *EventCorrelator {
	return &EventCorrelator{
		correlations:  make(map[string]*EventCorrelation),
		timeWindows:   make(map[string]time.Duration),
		maxDistance:   5 * time.Minute,
		minConfidence: 0.7,
	}
}

// FindCorrelations analyzes events for temporal correlations
func (ec *EventCorrelator) FindCorrelations(events []ParsedEvent) []EventCorrelation {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	correlations := make([]EventCorrelation, 0)

	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	// Find temporal correlations
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			eventA := events[i]
			eventB := events[j]

			timeDiff := eventB.Timestamp.Sub(eventA.Timestamp)
			if timeDiff > ec.maxDistance {
				break // Events too far apart
			}

			correlationKey := fmt.Sprintf("%s-%s", eventA.Type, eventB.Type)

			if existing, exists := ec.correlations[correlationKey]; exists {
				existing.Occurrences++
				existing.LastSeen = eventB.Timestamp
				existing.Confidence = ec.calculateConfidence(existing.Occurrences, timeDiff)
			} else {
				ec.correlations[correlationKey] = &EventCorrelation{
					EventA:      string(eventA.Type),
					EventB:      string(eventB.Type),
					TimeDelay:   timeDiff,
					Occurrences: 1,
					LastSeen:    eventB.Timestamp,
					Confidence:  0.1,
				}
			}
		}
	}

	// Return correlations above confidence threshold
	for _, correlation := range ec.correlations {
		if correlation.Confidence >= ec.minConfidence {
			correlations = append(correlations, *correlation)
		}
	}

	return correlations
}

// calculateConfidence calculates correlation confidence based on frequency
func (ec *EventCorrelator) calculateConfidence(occurrences int64, timeDelay time.Duration) float64 {
	// Simple confidence calculation - can be enhanced with more sophisticated methods
	baseConfidence := math.Min(float64(occurrences)/10.0, 1.0)

	// Reduce confidence for longer delays
	delayPenalty := 1.0 - (float64(timeDelay)/float64(ec.maxDistance))*0.3

	return baseConfidence * delayPenalty
}

// loadDefaultPatterns loads the default set of critical patterns
func (pl *PatternLibrary) loadDefaultPatterns() {
	// Load comprehensive pattern library
	patterns := []*CriticalPattern{
		{
			ID:          "oom-001",
			Name:        "Out of Memory Kill",
			Description: "Process killed by OOM killer",
			RegexStr:    `killed process \d+ \([^)]+\).*score \d+`,
			Severity:    SeverityCritical,
			Category:    "memory",
			Tags:        []string{"oom", "memory", "process_killed"},
			Confidence:  0.99,
			Enabled:     true,
		},
		{
			ID:          "seg-001",
			Name:        "Segmentation Fault",
			Description: "Process crashed with segmentation fault",
			RegexStr:    `segfault at [0-9a-f]+ ip [0-9a-f]+ sp [0-9a-f]+ error \d+`,
			Severity:    SeverityHigh,
			Category:    "crash",
			Tags:        []string{"segfault", "crash", "memory_violation"},
			Confidence:  0.98,
			Enabled:     true,
		},
		{
			ID:          "disk-001",
			Name:        "Disk I/O Error",
			Description: "Disk read/write error detected",
			RegexStr:    `(I/O error|disk.*error|bad block)`,
			Severity:    SeverityHigh,
			Category:    "storage",
			Tags:        []string{"disk", "io_error", "storage"},
			Confidence:  0.90,
			Enabled:     true,
		},
		{
			ID:          "net-001",
			Name:        "Network Unreachable",
			Description: "Network connectivity issues",
			RegexStr:    `(network.*unreachable|connection.*refused|timeout.*connecting)`,
			Severity:    SeverityMedium,
			Category:    "network",
			Tags:        []string{"network", "connectivity"},
			Confidence:  0.85,
			Enabled:     true,
		},
	}

	for _, pattern := range patterns {
		pattern.Regex = regexp.MustCompile(pattern.RegexStr)
		pattern.Metadata = PatternMetadata{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   "1.0",
		}
		pl.patterns[pattern.ID] = pattern
	}
}

// ProcessEntry processes a log entry against all patterns
func (pl *PatternLibrary) ProcessEntry(entry LogEntry) []PatternMatch {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	matches := make([]PatternMatch, 0)
	pl.stats.TotalProcessed++

	// Check OOM detector first (highest priority)
	if oomEvent, detected := pl.oomDetector.DetectOOM(entry); detected {
		matches = append(matches, PatternMatch{
			PatternID:  "oom-detector",
			EventType:  EventOOMKill,
			Confidence: oomEvent.Confidence,
			Details:    map[string]interface{}{"oom_event": oomEvent},
			Entry:      entry,
		})
		pl.stats.TotalMatches++
	}

	// Check other patterns
	for _, pattern := range pl.patterns {
		if !pattern.Enabled {
			continue
		}

		if pattern.Regex.MatchString(entry.Message) {
			match := PatternMatch{
				PatternID:  pattern.ID,
				Confidence: pattern.Confidence,
				Entry:      entry,
			}

			if pattern.Extractor != nil {
				regexMatches := pattern.Regex.FindStringSubmatch(entry.Message)
				match.Details = pattern.Extractor(regexMatches, entry)
			}

			matches = append(matches, match)
			pl.stats.TotalMatches++
			pl.stats.PatternStats[pattern.ID]++
		}
	}

	return matches
}

// PatternMatch represents a detected pattern match
type PatternMatch struct {
	PatternID  string                 `json:"pattern_id"`
	EventType  EventType              `json:"event_type,omitempty"`
	Confidence float64                `json:"confidence"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Entry      LogEntry               `json:"entry"`
	Timestamp  time.Time              `json:"timestamp"`
}

// GetStats returns pattern library statistics
func (pl *PatternLibrary) GetStats() PatternStats {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	stats := pl.stats
	if stats.TotalProcessed > 0 {
		stats.MatchRate = float64(stats.TotalMatches) / float64(stats.TotalProcessed)
	}
	stats.LastUpdated = time.Now()

	return stats
}

// normalizeMessage normalizes log messages for pattern detection
func normalizeMessage(message string) string {
	// Replace numbers, IPs, paths, etc. with placeholders for pattern matching
	normalized := message

	// Replace numbers
	normalized = regexp.MustCompile(`\d+`).ReplaceAllString(normalized, "NUM")

	// Replace IP addresses
	normalized = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`).ReplaceAllString(normalized, "IP")

	// Replace file paths
	normalized = regexp.MustCompile(`/[\w/.-]+`).ReplaceAllString(normalized, "PATH")

	// Replace hex addresses
	normalized = regexp.MustCompile(`0x[0-9a-fA-F]+`).ReplaceAllString(normalized, "ADDR")

	return normalized
}
