package watcher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/pkg/types"
)

func TestNewWatcher(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  30 * time.Second,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	if watcher == nil {
		t.Fatal("NewWatcher returned nil")
	}

	if watcher.interval != 30*time.Second {
		t.Errorf("Expected interval 30s, got %v", watcher.interval)
	}

	if len(watcher.providers) != 1 || watcher.providers[0] != "kubernetes" {
		t.Errorf("Expected providers [kubernetes], got %v", watcher.providers)
	}
}

func TestNewWatcher_MinimumInterval(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  2 * time.Second, // Below minimum
	}

	_, err := NewWatcher(config)
	if err == nil {
		t.Error("Expected error for interval below minimum")
	}

	if err.Error() != "minimum watch interval is 5 seconds" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestNewWatcher_DefaultOutputFormat(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		// OutputFormat not specified
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	if watcher.outputFormat != "table" {
		t.Errorf("Expected default output format 'table', got '%s'", watcher.outputFormat)
	}
}

func TestWatcher_GetStatus(t *testing.T) {
	config := WatcherConfig{
		Providers:    []string{"kubernetes", "terraform"},
		Interval:     15 * time.Second,
		OutputFormat: "json",
		Quiet:        true,
		OnlyHighConf: true,
		WebhookURL:   "https://hooks.slack.com/test",
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	status := watcher.GetStatus()

	// Check basic status fields
	if status["active"] != true {
		t.Error("Expected active to be true")
	}

	if status["interval"] != "15s" {
		t.Errorf("Expected interval '15s', got %v", status["interval"])
	}

	if status["output_format"] != "json" {
		t.Errorf("Expected output_format 'json', got %v", status["output_format"])
	}

	if status["quiet_mode"] != true {
		t.Error("Expected quiet_mode to be true")
	}

	if status["only_high_conf"] != true {
		t.Error("Expected only_high_conf to be true")
	}

	if status["webhook_url"] != true {
		t.Error("Expected webhook_url to be true")
	}

	// Check providers
	providers, ok := status["providers"].([]string)
	if !ok {
		t.Error("Expected providers to be []string")
	}

	if len(providers) != 2 || providers[0] != "kubernetes" || providers[1] != "terraform" {
		t.Errorf("Expected providers [kubernetes terraform], got %v", providers)
	}
}

func TestWatcher_TakeInitialSnapshot(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true, // Suppress output during test
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	ctx := context.Background()
	err = watcher.takeInitialSnapshot(ctx)
	if err != nil {
		t.Errorf("takeInitialSnapshot failed: %v", err)
	}

	if watcher.lastSnapshot == nil {
		t.Error("Expected lastSnapshot to be set after initial snapshot")
	}

	// Check snapshot properties
	if watcher.lastSnapshot.Provider != "multi" {
		t.Errorf("Expected provider 'multi', got '%s'", watcher.lastSnapshot.Provider)
	}

	if watcher.lastSnapshot.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestWatcher_CheckForChanges_NoChanges(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Set up initial snapshot
	watcher.lastSnapshot = &types.Snapshot{
		ID:        "test-snapshot",
		Timestamp: time.Now(),
		Provider:  "multi",
		Resources: []types.Resource{},
	}

	ctx := context.Background()
	err = watcher.checkForChanges(ctx)
	if err != nil {
		t.Errorf("checkForChanges failed: %v", err)
	}

	// Should complete without errors when no changes
}

func TestWatcher_ChangeCallback(t *testing.T) {
	callbackCalled := false
	var receivedEvent *WatchEvent

	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
		ChangeCallback: func(event *WatchEvent) {
			callbackCalled = true
			receivedEvent = event
		},
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Test that callback function is stored
	if watcher.changeCallback == nil {
		t.Error("Expected changeCallback to be set")
	}

	// Create a mock event
	event := &WatchEvent{
		Timestamp: time.Now(),
		Summary: differ.ChangeSummary{
			Total: 1,
			Added: 1,
		},
		Source: "test",
	}

	// Call the callback
	watcher.changeCallback(event)

	if !callbackCalled {
		t.Error("Expected callback to be called")
	}

	if receivedEvent == nil {
		t.Error("Expected receivedEvent to be set")
	}

	if receivedEvent.Source != "test" {
		t.Errorf("Expected source 'test', got '%s'", receivedEvent.Source)
	}
}

func TestWatcher_Start_ContextCancellation(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  5 * time.Second, // Minimum valid interval
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start should exit when context is cancelled
	err = watcher.Start(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestWatcher_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      WatcherConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: WatcherConfig{
				Providers: []string{"kubernetes"},
				Interval:  10 * time.Second,
			},
			expectError: false,
		},
		{
			name: "interval too short",
			config: WatcherConfig{
				Providers: []string{"kubernetes"},
				Interval:  3 * time.Second,
			},
			expectError: true,
			errorMsg:    "minimum watch interval is 5 seconds",
		},
		{
			name: "empty providers allowed",
			config: WatcherConfig{
				Providers: []string{},
				Interval:  10 * time.Second,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWatcher(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWatcher_IntegrationWithCorrelator(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Test that correlator is properly initialized
	if watcher.correlator == nil {
		t.Error("Expected correlator to be initialized")
	}

	// Test that differ is properly initialized
	if watcher.differ == nil {
		t.Error("Expected differ to be initialized")
	}

	// Test correlation with mock changes
	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "deployment/test",
			ResourceType: "deployment",
			ResourceName: "test",
			Namespace:    "default",
			Timestamp:    time.Now(),
			Details: []differ.SimpleFieldChange{
				{Field: "replicas", OldValue: 3, NewValue: 5},
			},
		},
	}

	groups := watcher.correlator.GroupChanges(changes)
	if len(groups) == 0 {
		t.Error("Expected correlator to return groups")
	}
}

// Benchmark tests
func BenchmarkWatcher_CheckForChanges(b *testing.B) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		b.Fatalf("NewWatcher failed: %v", err)
	}

	// Set up initial snapshot
	watcher.lastSnapshot = &types.Snapshot{
		ID:        "bench-snapshot",
		Timestamp: time.Now(),
		Provider:  "multi",
		Resources: []types.Resource{},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := watcher.checkForChanges(ctx)
		if err != nil {
			b.Errorf("checkForChanges failed: %v", err)
		}
	}
}

func BenchmarkWatcher_CorrelateChanges(b *testing.B) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		b.Fatalf("NewWatcher failed: %v", err)
	}

	// Create test changes
	changes := make([]differ.SimpleChange, 100)
	now := time.Now()
	for i := 0; i < 100; i++ {
		changes[i] = differ.SimpleChange{
			Type:         "modified",
			ResourceID:   fmt.Sprintf("deployment/test-%d", i),
			ResourceType: "deployment",
			ResourceName: fmt.Sprintf("test-%d", i),
			Namespace:    "default",
			Timestamp:    now,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		groups := watcher.correlator.GroupChanges(changes)
		_ = groups // Prevent optimization
	}
}
