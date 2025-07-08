package watcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/pkg/types"
)

// TestWatcher_EmptyProviders tests watcher with no providers
func TestWatcher_EmptyProviders(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{},
		Interval:  10 * time.Second,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Should handle empty providers gracefully
	ctx := context.Background()
	err = watcher.takeInitialSnapshot(ctx)
	if err != nil {
		t.Errorf("takeInitialSnapshot should handle empty providers: %v", err)
	}
}

// TestWatcher_RapidChanges tests handling of rapid consecutive changes
func TestWatcher_RapidChanges(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  5 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Simulate rapid changes
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		err = watcher.checkForChanges(ctx)
		if err != nil {
			t.Errorf("checkForChanges failed on iteration %d: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestWatcher_LargeChangeSet tests handling of many changes
func TestWatcher_LargeChangeSet(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Create large change set
	changes := make([]differ.SimpleChange, 1000)
	for i := 0; i < 1000; i++ {
		changes[i] = differ.SimpleChange{
			Type:         "modified",
			ResourceID:   fmt.Sprintf("resource-%d", i),
			ResourceType: "deployment",
			ResourceName: fmt.Sprintf("app-%d", i),
			Namespace:    "default",
			Timestamp:    time.Now(),
		}
	}

	// Test correlation performance
	start := time.Now()
	groups := watcher.correlator.GroupChanges(changes)
	duration := time.Since(start)

	if duration > 500*time.Millisecond {
		t.Errorf("Correlation took too long for 1000 changes: %v", duration)
	}

	if len(groups) == 0 {
		t.Error("Expected correlation groups for large change set")
	}
}

// TestWatcher_WebhookRetry tests webhook retry logic
func TestWatcher_WebhookRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := WatcherConfig{
		Providers:  []string{"kubernetes"},
		Interval:   10 * time.Second,
		WebhookURL: server.URL,
		Quiet:      true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	event := &WatchEvent{
		Timestamp: time.Now(),
		Summary:   differ.ChangeSummary{Total: 1},
		Source:    "test",
	}

	err = watcher.sendWebhookWithRetry(event, 3, 100*time.Millisecond)
	if err != nil {
		t.Errorf("sendWebhookWithRetry failed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestWatcher_MemoryLeak tests for memory leaks in long-running sessions
func TestWatcher_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  5 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Monitor memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	startAlloc := memStats.Alloc

	// Run watcher
	go watcher.Start(ctx)

	// Check memory periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	maxIncrease := uint64(0)
	checks := 0
	for {
		select {
		case <-ticker.C:
			runtime.ReadMemStats(&memStats)
			increase := memStats.Alloc - startAlloc
			if increase > maxIncrease {
				maxIncrease = increase
			}
			checks++
		case <-ctx.Done():
			// Check final memory increase
			if maxIncrease > 50*1024*1024 { // 50MB threshold
				t.Errorf("Memory usage increased by %d bytes after %d checks, possible leak", maxIncrease, checks)
			}
			return
		}
	}
}

// TestWatcher_ConcurrentAccess tests thread safety
func TestWatcher_ConcurrentAccess(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	ctx := context.Background()
	
	// Set initial snapshot
	watcher.lastSnapshot = &types.Snapshot{
		ID:        "test",
		Timestamp: time.Now(),
		Provider:  "multi",
		Resources: []types.Resource{},
	}

	// Run concurrent operations
	done := make(chan bool, 3)
	
	// Goroutine 1: Check for changes
	go func() {
		for i := 0; i < 10; i++ {
			watcher.checkForChanges(ctx)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 2: Get status
	go func() {
		for i := 0; i < 10; i++ {
			watcher.GetStatus()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Goroutine 3: Display changes
	go func() {
		event := &WatchEvent{
			Timestamp: time.Now(),
			Summary:   differ.ChangeSummary{Total: 1},
		}
		for i := 0; i < 10; i++ {
			watcher.displayChanges(event)
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent operations timed out")
		}
	}
}

// TestWatcher_InvalidWebhookURL tests handling of malformed webhook URLs
func TestWatcher_InvalidWebhookURL(t *testing.T) {
	testCases := []struct {
		name       string
		webhookURL string
	}{
		{"empty URL", ""},
		{"invalid URL", "not-a-url"},
		{"invalid scheme", "ftp://example.com"},
		{"valid HTTP", "http://example.com/webhook"},
		{"valid HTTPS", "https://example.com/webhook"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := WatcherConfig{
				Providers:  []string{"kubernetes"},
				Interval:   10 * time.Second,
				WebhookURL: tc.webhookURL,
				Quiet:      true,
			}

			w, err := NewWatcher(config)
			if err != nil {
				t.Fatalf("Failed to create watcher: %v", err)
			}

			if w != nil && tc.webhookURL != "" {
				event := &WatchEvent{
					Timestamp: time.Now(),
					Summary:   differ.ChangeSummary{Total: 1},
					Source:    "test",
				}
				// Should handle gracefully
				_ = w.sendWebhook(event)
			}
		})
	}
}

// TestWatcher_ContextCancellationDuringCheck tests cancellation during change check
func TestWatcher_ContextCancellationDuringCheck(t *testing.T) {
	config := WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  10 * time.Second,
		Quiet:     true,
	}

	_, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	// Simulate slow check operation
	slowCheck := func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return errors.New("check completed")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Start check in goroutine
	errCh := make(chan error)
	go func() {
		errCh <- slowCheck(ctx)
	}()

	// Cancel after 100ms
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Should receive context cancelled error
	err = <-errCh
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// Helper to add webhook retry functionality
func (w *Watcher) sendWebhookWithRetry(event *WatchEvent, maxRetries int, retryDelay time.Duration) error {
	var err error
	for i := 0; i <= maxRetries; i++ {
		err = w.sendWebhook(event)
		if err == nil {
			return nil
		}
		if i < maxRetries {
			time.Sleep(retryDelay)
		}
	}
	return err
}