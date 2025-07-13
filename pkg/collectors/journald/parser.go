package journald

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LogParser handles structured parsing of log entries
type LogParser struct {
	patterns      map[EventType]*PatternMatcher
	oomKillRegex  *regexp.Regexp
	segfaultRegex *regexp.Regexp
	networkRegex  *regexp.Regexp
	diskRegex     *regexp.Regexp
	memoryRegex   *regexp.Regexp
	cpuRegex      *regexp.Regexp
	authRegex     *regexp.Regexp
}

// EventType represents different types of log events
type EventType string

const (
	EventOOMKill        EventType = "oom_kill"
	EventSegfault       EventType = "segfault"
	EventNetworkError   EventType = "network_error"
	EventDiskError      EventType = "disk_error"
	EventMemoryPressure EventType = "memory_pressure"
	EventCPUThrottle    EventType = "cpu_throttle"
	EventAuthFailure    EventType = "auth_failure"
	EventServiceStart   EventType = "service_start"
	EventServiceStop    EventType = "service_stop"
	EventServiceFailed  EventType = "service_failed"
	EventKernelPanic    EventType = "kernel_panic"
	EventDiskFull       EventType = "disk_full"
	EventProcessKilled  EventType = "process_killed"
	EventSystemBoot     EventType = "system_boot"
	EventSystemShutdown EventType = "system_shutdown"
	EventGeneric        EventType = "generic"
)

// ParsedEvent represents a structured parsed log event
type ParsedEvent struct {
	Type       EventType              `json:"type"`
	Severity   Severity               `json:"severity"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
	Message    string                 `json:"message"`
	Process    string                 `json:"process,omitempty"`
	PID        int                    `json:"pid,omitempty"`
	Unit       string                 `json:"unit,omitempty"`
	Details    map[string]interface{} `json:"details"`
	RawEntry   LogEntry               `json:"raw_entry"`
	Confidence float64                `json:"confidence"`
	Tags       []string               `json:"tags"`
}

// Severity represents event severity levels
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// PatternMatcher contains regex patterns and extraction logic for event types
type PatternMatcher struct {
	Regex      *regexp.Regexp
	Extractor  func(matches []string, entry LogEntry) ParsedEvent
	Confidence float64
	Severity   Severity
	Tags       []string
}

// NewLogParser creates a new structured log parser
func NewLogParser() *LogParser {
	parser := &LogParser{
		patterns: make(map[EventType]*PatternMatcher),
	}

	parser.initializePatterns()
	return parser
}

// Parse analyzes a log entry and returns structured events
func (lp *LogParser) Parse(entry LogEntry) []ParsedEvent {
	events := make([]ParsedEvent, 0)

	// Try each pattern matcher
	for eventType, matcher := range lp.patterns {
		if matches := matcher.Regex.FindStringSubmatch(entry.Message); matches != nil {
			event := matcher.Extractor(matches, entry)
			event.Type = eventType
			event.Confidence = matcher.Confidence
			event.Severity = matcher.Severity
			event.Tags = append(event.Tags, matcher.Tags...)
			event.RawEntry = entry
			event.Timestamp = entry.Timestamp
			event.Source = entry.Hostname
			event.Process = entry.Comm
			event.PID = entry.PID
			event.Unit = entry.Unit

			events = append(events, event)
		}
	}

	// If no specific patterns matched, create a generic event for high-priority messages
	if len(events) == 0 && entry.Priority <= 3 {
		events = append(events, ParsedEvent{
			Type:       EventGeneric,
			Severity:   priorityToSeverity(entry.Priority),
			Source:     entry.Hostname,
			Timestamp:  entry.Timestamp,
			Message:    entry.Message,
			Process:    entry.Comm,
			PID:        entry.PID,
			Unit:       entry.Unit,
			Details:    make(map[string]interface{}),
			RawEntry:   entry,
			Confidence: 0.5,
			Tags:       []string{"unstructured"},
		})
	}

	return events
}

// initializePatterns sets up all regex patterns and extractors
func (lp *LogParser) initializePatterns() {
	// OOM Kill patterns - flexible pattern that matches various formats
	lp.patterns[EventOOMKill] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)killed process (\d+) \(([^)]+)\).*score (\d+)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			pid, _ := strconv.Atoi(matches[1])
			processName := matches[2]
			score, _ := strconv.Atoi(matches[3])

			return ParsedEvent{
				Message: fmt.Sprintf("OOM killer terminated process %s (PID %d)", processName, pid),
				Details: map[string]interface{}{
					"killed_pid":     pid,
					"killed_process": processName,
					"oom_score":      score,
				},
			}
		},
		Confidence: 0.99,
		Severity:   SeverityCritical,
		Tags:       []string{"oom", "memory", "process_killed"},
	}

	// Segmentation fault patterns
	lp.patterns[EventSegfault] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)segfault at ([0-9a-f]+) ip ([0-9a-f]+) sp ([0-9a-f]+) error (\d+) in ([^\[]+)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			address := matches[1]
			ip := matches[2]
			sp := matches[3]
			errorCode := matches[4]
			binary := strings.TrimSpace(matches[5])

			return ParsedEvent{
				Message: fmt.Sprintf("Segmentation fault in %s", binary),
				Details: map[string]interface{}{
					"fault_address":   address,
					"instruction_ptr": ip,
					"stack_ptr":       sp,
					"error_code":      errorCode,
					"binary":          binary,
					"fault_type":      "segfault",
				},
			}
		},
		Confidence: 0.98,
		Severity:   SeverityHigh,
		Tags:       []string{"segfault", "crash", "memory_violation"},
	}

	// Network error patterns
	lp.patterns[EventNetworkError] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(network.*unreachable|connection.*refused|timeout.*connecting|no route to host|name resolution failed)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			errorType := strings.ToLower(matches[1])

			return ParsedEvent{
				Message: fmt.Sprintf("Network error: %s", errorType),
				Details: map[string]interface{}{
					"error_type": errorType,
					"category":   "network",
				},
			}
		},
		Confidence: 0.85,
		Severity:   SeverityMedium,
		Tags:       []string{"network", "connectivity"},
	}

	// Disk error patterns
	lp.patterns[EventDiskError] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(I/O error|disk.*error|filesystem.*error|bad block|EXT4-fs error|XFS.*error)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			errorType := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Disk error: %s", errorType),
				Details: map[string]interface{}{
					"error_type": errorType,
					"category":   "storage",
				},
			}
		},
		Confidence: 0.90,
		Severity:   SeverityHigh,
		Tags:       []string{"disk", "storage", "io_error"},
	}

	// Memory pressure patterns
	lp.patterns[EventMemoryPressure] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(memory pressure|low memory|memory.*critical|swap.*full)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			condition := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Memory pressure: %s", condition),
				Details: map[string]interface{}{
					"condition": condition,
					"category":  "memory",
				},
			}
		},
		Confidence: 0.88,
		Severity:   SeverityHigh,
		Tags:       []string{"memory", "pressure", "performance"},
	}

	// CPU throttling patterns
	lp.patterns[EventCPUThrottle] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(CPU.*throttl|thermal.*throttl|frequency.*reduc)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			throttleType := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("CPU throttling: %s", throttleType),
				Details: map[string]interface{}{
					"throttle_type": throttleType,
					"category":      "cpu",
				},
			}
		},
		Confidence: 0.85,
		Severity:   SeverityMedium,
		Tags:       []string{"cpu", "throttling", "performance"},
	}

	// Authentication failure patterns
	lp.patterns[EventAuthFailure] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(authentication failure|login.*failed|invalid.*password|permission denied|access denied)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			failureType := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Authentication failure: %s", failureType),
				Details: map[string]interface{}{
					"failure_type": failureType,
					"category":     "security",
				},
			}
		},
		Confidence: 0.90,
		Severity:   SeverityMedium,
		Tags:       []string{"auth", "security", "access_denied"},
	}

	// Service lifecycle patterns
	lp.patterns[EventServiceStart] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(started|starting).*service`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			action := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Service %s: %s", entry.Unit, action),
				Details: map[string]interface{}{
					"action":   action,
					"category": "service_lifecycle",
				},
			}
		},
		Confidence: 0.80,
		Severity:   SeverityInfo,
		Tags:       []string{"service", "lifecycle", "start"},
	}

	lp.patterns[EventServiceStop] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(stopped|stopping).*service`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			action := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Service %s: %s", entry.Unit, action),
				Details: map[string]interface{}{
					"action":   action,
					"category": "service_lifecycle",
				},
			}
		},
		Confidence: 0.80,
		Severity:   SeverityInfo,
		Tags:       []string{"service", "lifecycle", "stop"},
	}

	lp.patterns[EventServiceFailed] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(failed|error|crash).*service`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			action := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Service %s failed: %s", entry.Unit, action),
				Details: map[string]interface{}{
					"action":   action,
					"category": "service_lifecycle",
				},
			}
		},
		Confidence: 0.85,
		Severity:   SeverityHigh,
		Tags:       []string{"service", "lifecycle", "failed"},
	}

	// Kernel panic patterns
	lp.patterns[EventKernelPanic] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(kernel panic|oops|BUG:|general protection fault)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			panicType := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Kernel issue: %s", panicType),
				Details: map[string]interface{}{
					"panic_type": panicType,
					"category":   "kernel",
				},
			}
		},
		Confidence: 0.95,
		Severity:   SeverityCritical,
		Tags:       []string{"kernel", "panic", "system_crash"},
	}

	// Disk full patterns
	lp.patterns[EventDiskFull] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(no space left|disk.*full|filesystem.*full|\d+%.*full)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			condition := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("Disk space issue: %s", condition),
				Details: map[string]interface{}{
					"condition": condition,
					"category":  "storage",
				},
			}
		},
		Confidence: 0.92,
		Severity:   SeverityHigh,
		Tags:       []string{"disk", "storage", "full"},
	}

	// System boot/shutdown patterns
	lp.patterns[EventSystemBoot] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(system.*boot|kernel.*boot|startup.*complete)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			bootType := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("System boot: %s", bootType),
				Details: map[string]interface{}{
					"boot_type": bootType,
					"category":  "system_lifecycle",
				},
			}
		},
		Confidence: 0.85,
		Severity:   SeverityInfo,
		Tags:       []string{"system", "boot", "lifecycle"},
	}

	lp.patterns[EventSystemShutdown] = &PatternMatcher{
		Regex: regexp.MustCompile(`(?i)(system.*shutdown|shutdown.*request|halt.*system)`),
		Extractor: func(matches []string, entry LogEntry) ParsedEvent {
			shutdownType := matches[1]

			return ParsedEvent{
				Message: fmt.Sprintf("System shutdown: %s", shutdownType),
				Details: map[string]interface{}{
					"shutdown_type": shutdownType,
					"category":      "system_lifecycle",
				},
			}
		},
		Confidence: 0.85,
		Severity:   SeverityInfo,
		Tags:       []string{"system", "shutdown", "lifecycle"},
	}
}

// priorityToSeverity converts journald priority to severity
func priorityToSeverity(priority int) Severity {
	switch priority {
	case 0, 1, 2: // Emergency, Alert, Critical
		return SeverityCritical
	case 3: // Error
		return SeverityHigh
	case 4: // Warning
		return SeverityMedium
	case 5: // Notice
		return SeverityLow
	case 6, 7: // Info, Debug
		return SeverityInfo
	default:
		return SeverityInfo
	}
}

// ExtractOOMDetails extracts detailed OOM information from log message
func (lp *LogParser) ExtractOOMDetails(message string) map[string]interface{} {
	details := make(map[string]interface{})

	// Extract memory usage details
	if matches := regexp.MustCompile(`anon-rss:(\d+)kB`).FindStringSubmatch(message); len(matches) > 1 {
		if size, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			details["anon_rss_kb"] = size
		}
	}

	if matches := regexp.MustCompile(`file-rss:(\d+)kB`).FindStringSubmatch(message); len(matches) > 1 {
		if size, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			details["file_rss_kb"] = size
		}
	}

	if matches := regexp.MustCompile(`shmem-rss:(\d+)kB`).FindStringSubmatch(message); len(matches) > 1 {
		if size, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			details["shmem_rss_kb"] = size
		}
	}

	if matches := regexp.MustCompile(`total-vm:(\d+)kB`).FindStringSubmatch(message); len(matches) > 1 {
		if size, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			details["total_vm_kb"] = size
		}
	}

	// Extract OOM score
	if matches := regexp.MustCompile(`score (\d+)`).FindStringSubmatch(message); len(matches) > 1 {
		if score, err := strconv.Atoi(matches[1]); err == nil {
			details["oom_score"] = score
		}
	}

	// Extract memory constraint information
	if strings.Contains(message, "limit_in_bytes") {
		details["constrained"] = true
		if matches := regexp.MustCompile(`limit_in_bytes (\d+)`).FindStringSubmatch(message); len(matches) > 1 {
			if limit, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				details["memory_limit_bytes"] = limit
			}
		}
	}

	return details
}

// GetEventSeverity returns the severity for a given event type
func (lp *LogParser) GetEventSeverity(eventType EventType) Severity {
	if pattern, exists := lp.patterns[eventType]; exists {
		return pattern.Severity
	}
	return SeverityInfo
}

// GetEventConfidence returns the confidence score for a given event type
func (lp *LogParser) GetEventConfidence(eventType EventType) float64 {
	if pattern, exists := lp.patterns[eventType]; exists {
		return pattern.Confidence
	}
	return 0.5
}

// ValidatePattern tests if a pattern correctly matches expected log messages
func (lp *LogParser) ValidatePattern(eventType EventType, testMessage string) bool {
	if pattern, exists := lp.patterns[eventType]; exists {
		return pattern.Regex.MatchString(testMessage)
	}
	return false
}

// GetSupportedEventTypes returns all supported event types
func (lp *LogParser) GetSupportedEventTypes() []EventType {
	types := make([]EventType, 0, len(lp.patterns))
	for eventType := range lp.patterns {
		types = append(types, eventType)
	}
	return types
}
