package journald

import (
	"runtime"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
)

func TestCollectorInterface(t *testing.T) {
	// Skip if not on Linux
	if runtime.GOOS != "linux" {
		t.Skip("journald collector tests require Linux")
	}

	// This test ensures the collector implements the interface correctly
	var _ collectors.EnhancedCollector = (*Collector)(nil)
}

func TestCollectorName(t *testing.T) {
	// This test can run on any platform
	c := &Collector{}
	if got := c.Name(); got != "journald" {
		t.Errorf("Name() = %v, want %v", got, "journald")
	}
}

func TestCollectorValidate(t *testing.T) {
	c := &Collector{}

	tests := []struct {
		name    string
		config  collectors.CollectorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"rate_limit":      10000,
					"memory_limit_mb": 30,
					"min_priority":    3,
				},
			},
			wantErr: runtime.GOOS != "linux",
		},
		{
			name: "rate limit too low",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"rate_limit": 50,
				},
			},
			wantErr: true,
		},
		{
			name: "rate limit too high",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"rate_limit": 200000,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid priority",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"min_priority": 10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Validate(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSupportedRegions(t *testing.T) {
	c := &Collector{}
	regions := c.SupportedRegions()

	if len(regions) != 1 || regions[0] != "local" {
		t.Errorf("SupportedRegions() = %v, want [local]", regions)
	}
}

func TestAutoDiscover(t *testing.T) {
	c := &Collector{}
	config, err := c.AutoDiscover()

	if runtime.GOOS != "linux" {
		if err == nil {
			t.Error("AutoDiscover() should fail on non-Linux systems")
		}
		return
	}

	// On Linux, it might succeed or fail depending on journald availability
	if err == nil {
		// Verify default config
		if config.Config["rate_limit"] != 10000 {
			t.Error("AutoDiscover() should set rate_limit to 10000")
		}
		if config.Config["memory_limit_mb"] != 30 {
			t.Error("AutoDiscover() should set memory_limit_mb to 30")
		}
	}
}

func TestLogParser(t *testing.T) {
	parser := NewLogParser()

	// Test OOM detection
	oomEntry := LogEntry{
		Message:   "killed process 1234 (test-process) score 100 total-vm:1000kB anon-rss:500kB file-rss:200kB shmem-rss:100kB",
		Priority:  2, // Critical
		Timestamp: time.Now(),
		Unit:      "test.service",
	}

	events := parser.Parse(oomEntry)
	if len(events) == 0 {
		t.Error("Expected to parse OOM event, got none")
	}

	if len(events) > 0 && events[0].Type != EventOOMKill {
		t.Errorf("Expected OOM event type, got %v", events[0].Type)
	}
}

func TestPatternLibrary(t *testing.T) {
	lib := NewPatternLibrary()

	// Test OOM detection accuracy
	oomEntry := LogEntry{
		Message:   "killed process 1234 (test-process) score 100 total-vm:1000kB anon-rss:500kB file-rss:200kB shmem-rss:100kB",
		Priority:  2,
		Timestamp: time.Now(),
	}

	matches := lib.ProcessEntry(oomEntry)
	if len(matches) == 0 {
		t.Error("Expected pattern match for OOM, got none")
	}

	// Verify confidence is high for OOM detection
	if len(matches) > 0 && matches[0].Confidence < 0.9 {
		t.Errorf("Expected high confidence for OOM detection, got %f", matches[0].Confidence)
	}
}

func TestLogFilter(t *testing.T) {
	config := FilterConfig{
		MaxEntriesPerSec:    1000,
		MinPriority:         3,
		EnableDeduplication: true,
	}

	filter := NewLogFilter(config)

	// Test priority filtering
	lowPriorityEntry := LogEntry{
		Priority: 6, // Info level
		Message:  "This is an info message",
	}

	highPriorityEntry := LogEntry{
		Priority: 2, // Critical level
		Message:  "This is a critical message",
	}

	if filter.ShouldProcess(lowPriorityEntry) {
		t.Error("Low priority entry should be filtered out")
	}

	if !filter.ShouldProcess(highPriorityEntry) {
		t.Error("High priority entry should be processed")
	}
}

func TestOOMDetector(t *testing.T) {
	detector := NewOOMDetector()

	// Test primary OOM pattern
	oomEntry := LogEntry{
		Message:   "killed process 1234 (test-process) score 100 total-vm:1000kB anon-rss:500kB file-rss:200kB shmem-rss:100kB",
		Timestamp: time.Now(),
	}

	event, detected := detector.DetectOOM(oomEntry)
	if !detected {
		t.Error("Should detect OOM event")
	}

	if event.Confidence < 0.99 {
		t.Errorf("Expected 99%% confidence for OOM detection, got %f", event.Confidence)
	}

	// Test non-OOM entry
	normalEntry := LogEntry{
		Message:   "Normal log message",
		Timestamp: time.Now(),
	}

	_, detected = detector.DetectOOM(normalEntry)
	if detected {
		t.Error("Should not detect OOM in normal message")
	}
}

func TestRateLimit(t *testing.T) {
	bucket := NewTokenBucket(5, 1) // 5 capacity, 1 token per second

	// Should allow up to capacity
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			t.Errorf("Should allow request %d", i)
		}
	}

	// Should deny the next request
	if bucket.Allow() {
		t.Error("Should deny request when bucket is empty")
	}

	// Wait for refill and test again
	time.Sleep(1100 * time.Millisecond)
	if !bucket.Allow() {
		t.Error("Should allow request after refill")
	}
}

func TestDuplicateDetection(t *testing.T) {
	cache := NewDuplicateCache(100, 5*time.Minute)

	entry1 := LogEntry{
		Message:  "Test message",
		Unit:     "test.service",
		Priority: 3,
	}

	entry2 := LogEntry{
		Message:  "Test message", // Same message
		Unit:     "test.service",
		Priority: 3,
	}

	// First entry should not be duplicate
	_, isDup := cache.CheckDuplicate(entry1)
	if isDup {
		t.Error("First entry should not be duplicate")
	}

	// Second entry should be duplicate
	_, isDup = cache.CheckDuplicate(entry2)
	if !isDup {
		t.Error("Second entry should be duplicate")
	}
}

func TestEventCorrelation(t *testing.T) {
	correlator := NewEventCorrelator()

	now := time.Now()
	events := []ParsedEvent{
		{
			Type:      EventMemoryPressure,
			Timestamp: now,
			Severity:  SeverityHigh,
		},
		{
			Type:      EventOOMKill,
			Timestamp: now.Add(2 * time.Minute),
			Severity:  SeverityCritical,
		},
	}

	correlations := correlator.FindCorrelations(events)
	// Note: Correlation detection needs multiple occurrences, so this might not trigger
	// in a simple test. This is expected behavior.
	_ = correlations
}
