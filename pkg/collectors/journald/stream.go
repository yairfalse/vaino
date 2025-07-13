package journald

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogStreamer handles real-time journald log streaming
type LogStreamer struct {
	cmd          *exec.Cmd
	scanner      *bufio.Scanner
	entries      chan LogEntry
	errors       chan error
	mu           sync.RWMutex
	running      bool
	rateLimit    *StreamRateLimit
	filters      []StreamFilter
	bufferSize   int
	maxEntrySize int
}

// LogEntry represents a structured journald log entry
type LogEntry struct {
	Timestamp      time.Time              `json:"timestamp"`
	Priority       int                    `json:"priority"`
	Facility       int                    `json:"facility"`
	Unit           string                 `json:"_SYSTEMD_UNIT,omitempty"`
	PID            int                    `json:"_PID,omitempty"`
	UID            int                    `json:"_UID,omitempty"`
	GID            int                    `json:"_GID,omitempty"`
	Comm           string                 `json:"_COMM,omitempty"`
	Exe            string                 `json:"_EXE,omitempty"`
	Cmdline        string                 `json:"_CMDLINE,omitempty"`
	Hostname       string                 `json:"_HOSTNAME,omitempty"`
	Message        string                 `json:"MESSAGE"`
	MessageID      string                 `json:"MESSAGE_ID,omitempty"`
	Transport      string                 `json:"_TRANSPORT,omitempty"`
	BootID         string                 `json:"_BOOT_ID,omitempty"`
	MachineID      string                 `json:"_MACHINE_ID,omitempty"`
	SystemdSlice   string                 `json:"_SYSTEMD_SLICE,omitempty"`
	SystemdCGroup  string                 `json:"_SYSTEMD_CGROUP,omitempty"`
	SourceRealtime int64                  `json:"__REALTIME_TIMESTAMP,omitempty"`
	Cursor         string                 `json:"__CURSOR,omitempty"`
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

// StreamRateLimit manages streaming rate limits
type StreamRateLimit struct {
	maxEntriesPerSec int
	window           time.Duration
	entries          []time.Time
	mu               sync.Mutex
}

// StreamFilter determines which log entries to process
type StreamFilter func(entry LogEntry) bool

// StreamConfig configures the log streamer
type StreamConfig struct {
	Units           []string  // Specific systemd units to monitor
	Priorities      []int     // Log priorities to include (0-7)
	Since           time.Time // Start streaming from this time
	Follow          bool      // Follow new entries (like tail -f)
	Lines           int       // Number of historical lines to read first
	RateLimit       int       // Max entries per second
	BufferSize      int       // Internal buffer size
	MaxEntrySize    int       // Max size of single log entry
	ExcludePatterns []string  // Exclude messages matching these patterns
	IncludePatterns []string  // Only include messages matching these patterns
}

// NewLogStreamer creates a new journald log streamer
func NewLogStreamer(config StreamConfig) *LogStreamer {
	if config.BufferSize == 0 {
		config.BufferSize = 10000
	}
	if config.MaxEntrySize == 0 {
		config.MaxEntrySize = 64 * 1024 // 64KB max per entry
	}
	if config.RateLimit == 0 {
		config.RateLimit = 10000 // 10k entries/sec default
	}

	return &LogStreamer{
		entries:      make(chan LogEntry, config.BufferSize),
		errors:       make(chan error, 100),
		bufferSize:   config.BufferSize,
		maxEntrySize: config.MaxEntrySize,
		rateLimit:    NewStreamRateLimit(config.RateLimit),
		filters:      make([]StreamFilter, 0),
	}
}

// NewStreamRateLimit creates a new rate limiter
func NewStreamRateLimit(maxPerSec int) *StreamRateLimit {
	return &StreamRateLimit{
		maxEntriesPerSec: maxPerSec,
		window:           time.Second,
		entries:          make([]time.Time, 0),
	}
}

// Allow checks if an entry should be allowed through the rate limit
func (rl *StreamRateLimit) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove entries outside the window
	filtered := make([]time.Time, 0, len(rl.entries))
	for _, t := range rl.entries {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	rl.entries = filtered

	// Check if we're at the limit
	if len(rl.entries) >= rl.maxEntriesPerSec {
		return false
	}

	// Add this entry
	rl.entries = append(rl.entries, now)
	return true
}

// AddFilter adds a stream filter
func (ls *LogStreamer) AddFilter(filter StreamFilter) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.filters = append(ls.filters, filter)
}

// Start begins streaming logs from journald
func (ls *LogStreamer) Start(ctx context.Context, config StreamConfig) error {
	ls.mu.Lock()
	if ls.running {
		ls.mu.Unlock()
		return fmt.Errorf("streamer already running")
	}
	ls.running = true
	ls.mu.Unlock()

	// Build journalctl command
	args := ls.buildJournalctlArgs(config)
	ls.cmd = exec.CommandContext(ctx, "journalctl", args...)

	// Set up stdout pipe
	stdout, err := ls.cmd.StdoutPipe()
	if err != nil {
		ls.mu.Lock()
		ls.running = false
		ls.mu.Unlock()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Set up stderr pipe for error handling
	stderr, err := ls.cmd.StderrPipe()
	if err != nil {
		ls.mu.Lock()
		ls.running = false
		ls.mu.Unlock()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := ls.cmd.Start(); err != nil {
		ls.mu.Lock()
		ls.running = false
		ls.mu.Unlock()
		return fmt.Errorf("failed to start journalctl: %w", err)
	}

	// Create scanner with size limit
	ls.scanner = bufio.NewScanner(stdout)
	buf := make([]byte, 0, ls.maxEntrySize)
	ls.scanner.Buffer(buf, ls.maxEntrySize)

	// Start processing goroutines
	go ls.processStdout(ctx)
	go ls.processStderr(ctx, stderr)
	go ls.waitForCompletion(ctx)

	return nil
}

// Stop stops the log streamer
func (ls *LogStreamer) Stop() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if !ls.running {
		return nil
	}

	ls.running = false

	if ls.cmd != nil && ls.cmd.Process != nil {
		// Send SIGTERM first
		if err := ls.cmd.Process.Signal(os.Interrupt); err != nil {
			// If SIGTERM fails, force kill
			ls.cmd.Process.Kill()
		}
	}

	// Close channels
	close(ls.entries)
	close(ls.errors)

	return nil
}

// Entries returns the channel for receiving log entries
func (ls *LogStreamer) Entries() <-chan LogEntry {
	return ls.entries
}

// Errors returns the channel for receiving errors
func (ls *LogStreamer) Errors() <-chan error {
	return ls.errors
}

// IsRunning returns whether the streamer is currently running
func (ls *LogStreamer) IsRunning() bool {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.running
}

// GetStats returns streaming statistics
func (ls *LogStreamer) GetStats() StreamStats {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return StreamStats{
		Running:       ls.running,
		BufferSize:    ls.bufferSize,
		MaxEntrySize:  ls.maxEntrySize,
		EntriesQueued: len(ls.entries),
		ErrorsQueued:  len(ls.errors),
	}
}

// StreamStats contains streaming statistics
type StreamStats struct {
	Running       bool
	BufferSize    int
	MaxEntrySize  int
	EntriesQueued int
	ErrorsQueued  int
	TotalEntries  int64
	TotalErrors   int64
	StartTime     time.Time
}

// buildJournalctlArgs constructs journalctl command arguments
func (ls *LogStreamer) buildJournalctlArgs(config StreamConfig) []string {
	args := []string{
		"--output=json", // JSON output for structured parsing
		"--no-pager",    // Disable paging
		"--quiet",       // Reduce noise
	}

	// Follow new entries
	if config.Follow {
		args = append(args, "--follow")
	}

	// Historical lines
	if config.Lines > 0 {
		args = append(args, fmt.Sprintf("--lines=%d", config.Lines))
	}

	// Start time
	if !config.Since.IsZero() {
		args = append(args, fmt.Sprintf("--since=%s", config.Since.Format("2006-01-02 15:04:05")))
	}

	// Priority filters
	if len(config.Priorities) > 0 {
		for _, priority := range config.Priorities {
			if priority >= 0 && priority <= 7 {
				args = append(args, fmt.Sprintf("--priority=%d", priority))
			}
		}
	}

	// Unit filters
	for _, unit := range config.Units {
		args = append(args, fmt.Sprintf("--unit=%s", unit))
	}

	return args
}

// processStdout processes stdout from journalctl
func (ls *LogStreamer) processStdout(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			ls.errors <- fmt.Errorf("stdout processor panic: %v", r)
		}
	}()

	for ls.scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Rate limiting
		if !ls.rateLimit.Allow() {
			continue
		}

		line := ls.scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON entry
		entry, err := ls.parseLogEntry(line)
		if err != nil {
			ls.errors <- fmt.Errorf("failed to parse log entry: %w", err)
			continue
		}

		// Apply filters
		if !ls.shouldProcess(entry) {
			continue
		}

		// Send to channel (non-blocking)
		select {
		case ls.entries <- entry:
		default:
			// Buffer full, drop entry
			ls.errors <- fmt.Errorf("entry buffer full, dropping entry")
		}
	}

	if err := ls.scanner.Err(); err != nil {
		ls.errors <- fmt.Errorf("scanner error: %w", err)
	}
}

// processStderr processes stderr from journalctl
func (ls *LogStreamer) processStderr(ctx context.Context, stderr io.ReadCloser) {
	defer stderr.Close()

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line != "" {
			ls.errors <- fmt.Errorf("journalctl stderr: %s", line)
		}
	}
}

// waitForCompletion waits for the journalctl process to complete
func (ls *LogStreamer) waitForCompletion(ctx context.Context) {
	if ls.cmd != nil {
		err := ls.cmd.Wait()
		if err != nil && ctx.Err() == nil {
			ls.errors <- fmt.Errorf("journalctl process error: %w", err)
		}
	}

	ls.mu.Lock()
	ls.running = false
	ls.mu.Unlock()
}

// parseLogEntry parses a JSON log entry from journalctl
func (ls *LogStreamer) parseLogEntry(line string) (LogEntry, error) {
	var rawEntry map[string]interface{}
	if err := json.Unmarshal([]byte(line), &rawEntry); err != nil {
		return LogEntry{}, err
	}

	entry := LogEntry{
		AdditionalData: make(map[string]interface{}),
	}

	// Parse timestamp
	if ts, ok := rawEntry["__REALTIME_TIMESTAMP"].(string); ok {
		if timestamp, err := parseJournaldTimestamp(ts); err == nil {
			entry.Timestamp = timestamp
			entry.SourceRealtime = timestamp.UnixNano() / 1000
		}
	}

	// Parse standard fields
	entry.Priority = getIntField(rawEntry, "PRIORITY")
	entry.Facility = getIntField(rawEntry, "SYSLOG_FACILITY")
	entry.Unit = getStringField(rawEntry, "_SYSTEMD_UNIT")
	entry.PID = getIntField(rawEntry, "_PID")
	entry.UID = getIntField(rawEntry, "_UID")
	entry.GID = getIntField(rawEntry, "_GID")
	entry.Comm = getStringField(rawEntry, "_COMM")
	entry.Exe = getStringField(rawEntry, "_EXE")
	entry.Cmdline = getStringField(rawEntry, "_CMDLINE")
	entry.Hostname = getStringField(rawEntry, "_HOSTNAME")
	entry.Message = getStringField(rawEntry, "MESSAGE")
	entry.MessageID = getStringField(rawEntry, "MESSAGE_ID")
	entry.Transport = getStringField(rawEntry, "_TRANSPORT")
	entry.BootID = getStringField(rawEntry, "_BOOT_ID")
	entry.MachineID = getStringField(rawEntry, "_MACHINE_ID")
	entry.SystemdSlice = getStringField(rawEntry, "_SYSTEMD_SLICE")
	entry.SystemdCGroup = getStringField(rawEntry, "_SYSTEMD_CGROUP")
	entry.Cursor = getStringField(rawEntry, "__CURSOR")

	// Store additional fields
	knownFields := map[string]bool{
		"__REALTIME_TIMESTAMP": true, "PRIORITY": true, "SYSLOG_FACILITY": true,
		"_SYSTEMD_UNIT": true, "_PID": true, "_UID": true, "_GID": true,
		"_COMM": true, "_EXE": true, "_CMDLINE": true, "_HOSTNAME": true,
		"MESSAGE": true, "MESSAGE_ID": true, "_TRANSPORT": true,
		"_BOOT_ID": true, "_MACHINE_ID": true, "_SYSTEMD_SLICE": true,
		"_SYSTEMD_CGROUP": true, "__CURSOR": true,
	}

	for key, value := range rawEntry {
		if !knownFields[key] {
			entry.AdditionalData[key] = value
		}
	}

	return entry, nil
}

// shouldProcess checks if an entry should be processed based on filters
func (ls *LogStreamer) shouldProcess(entry LogEntry) bool {
	ls.mu.RLock()
	filters := ls.filters
	ls.mu.RUnlock()

	for _, filter := range filters {
		if !filter(entry) {
			return false
		}
	}
	return true
}

// Helper functions

func parseJournaldTimestamp(ts string) (time.Time, error) {
	// journalctl timestamps are microseconds since epoch
	if usec, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.Unix(0, usec*1000), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp format: %s", ts)
}

func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntField(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return 0
}

// Common stream filters

// StreamPriorityFilter filters by log priority for streaming
func StreamPriorityFilter(priorities ...int) StreamFilter {
	priorityMap := make(map[int]bool)
	for _, p := range priorities {
		priorityMap[p] = true
	}

	return func(entry LogEntry) bool {
		return priorityMap[entry.Priority]
	}
}

// StreamUnitFilter filters by systemd unit for streaming
func StreamUnitFilter(units ...string) StreamFilter {
	unitMap := make(map[string]bool)
	for _, u := range units {
		unitMap[u] = true
	}

	return func(entry LogEntry) bool {
		return unitMap[entry.Unit]
	}
}

// MessagePatternFilter filters by message content patterns
func MessagePatternFilter(patterns []string, include bool) StreamFilter {
	return func(entry LogEntry) bool {
		for _, pattern := range patterns {
			if strings.Contains(strings.ToLower(entry.Message), strings.ToLower(pattern)) {
				return include
			}
		}
		return !include
	}
}

// EmergencyFilter filters only emergency/critical messages
func EmergencyFilter() StreamFilter {
	return StreamPriorityFilter(0, 1, 2) // Emergency, Alert, Critical
}
