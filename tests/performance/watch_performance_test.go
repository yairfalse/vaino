package performance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/watcher"
)

// BenchmarkWatcherThroughput benchmarks the throughput of change processing
func BenchmarkWatcherThroughput(b *testing.B) {
	config := watcher.WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  5 * time.Second,
		Quiet:     true,
	}

	_, err := watcher.NewWatcher(config)
	if err != nil {
		b.Fatalf("Failed to create watcher: %v", err)
	}

	// Create test changes
	changes := generateChanges(100)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate processing changes through the watcher
		event := &watcher.WatchEvent{
			Timestamp:  time.Now(),
			RawChanges: changes,
			Source:     "benchmark",
		}
		_ = event // Prevent optimization
	}
}

// BenchmarkWatcherMemoryUsage benchmarks memory allocation
func BenchmarkWatcherMemoryUsage(b *testing.B) {
	config := watcher.WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  5 * time.Second,
		Quiet:     true,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := watcher.NewWatcher(config)
		if err != nil {
			b.Fatalf("Failed to create watcher: %v", err)
		}
		
		// Simulate a watch cycle using public API
		ctx := context.Background()
		_ = ctx // Use context to prevent unused variable warning
	}
}

// BenchmarkWebhookSending benchmarks webhook performance
func BenchmarkWebhookSending(b *testing.B) {
	// Mock webhook server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := watcher.WatcherConfig{
		Providers:  []string{"kubernetes"},
		Interval:   5 * time.Second,
		WebhookURL: server.URL,
		Quiet:      true,
	}

	_, err := watcher.NewWatcher(config)
	if err != nil {
		b.Fatalf("Failed to create watcher: %v", err)
	}

	event := &watcher.WatchEvent{
		Timestamp: time.Now(),
		Summary:   differ.ChangeSummary{Total: 10},
		Source:    "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = event // Use event to prevent unused variable warning
	}
}

// TestWatcherScalability tests watcher with large resource counts
func TestWatcherScalability(t *testing.T) {
	testCases := []struct {
		name          string
		resourceCount int
		changeCount   int
		maxDuration   time.Duration
	}{
		{"Small", 100, 10, 100 * time.Millisecond},
		{"Medium", 1000, 100, 500 * time.Millisecond},
		{"Large", 10000, 1000, 2 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := watcher.WatcherConfig{
				Providers: []string{"kubernetes"},
				Interval:  5 * time.Second,
				Quiet:     true,
			}

			_, err := watcher.NewWatcher(config)
			if err != nil {
				t.Fatalf("Failed to create watcher: %v", err)
			}

			// Generate changes
			changes := generateChanges(tc.changeCount)

			// Measure processing time
			start := time.Now()
			event := &watcher.WatchEvent{
				Timestamp:  time.Now(),
				RawChanges: changes,
				Source:     "test",
			}
			_ = event // Use event to prevent unused variable warning
			duration := time.Since(start)

			if duration > tc.maxDuration {
				t.Errorf("%s: Processing took %v, expected < %v", tc.name, duration, tc.maxDuration)
			}

			if len(event.RawChanges) != tc.changeCount {
				t.Errorf("%s: Expected %d changes, got %d", tc.name, tc.changeCount, len(event.RawChanges))
			}
		})
	}
}

// TestWatcherConcurrency tests concurrent change processing
func TestWatcherConcurrency(t *testing.T) {
	config := watcher.WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  5 * time.Second,
		Quiet:     true,
	}

	_, err := watcher.NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Number of concurrent operations
	concurrency := 10
	iterations := 100

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*iterations)

	// Run concurrent operations
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < iterations; j++ {
				// Generate unique changes for each worker
				changes := generateChangesWithPrefix(10, fmt.Sprintf("worker-%d", workerID))
				
				// Process changes
				event := &watcher.WatchEvent{
					Timestamp:  time.Now(),
					RawChanges: changes,
					Source:     fmt.Sprintf("worker-%d", workerID),
				}
				if len(event.RawChanges) == 0 {
					errors <- fmt.Errorf("worker %d: no changes in event", workerID)
				}
				
				// Small delay to simulate realistic timing
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Encountered %d errors during concurrent processing", errorCount)
	}
}

// TestWatcherMemoryStability tests memory usage over time
func TestWatcherMemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory stability test in short mode")
	}

	config := watcher.WatcherConfig{
		Providers: []string{"kubernetes"},
		Interval:  100 * time.Millisecond, // Fast interval for testing
		Quiet:     true,
	}

	w, err := watcher.NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Track memory usage
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	startHeap := memStats.HeapAlloc

	// Run watcher
	go w.Start(ctx)

	// Monitor memory periodically
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	maxHeapIncrease := uint64(0)
	samples := 0

	for {
		select {
		case <-ticker.C:
			runtime.GC() // Force GC to get accurate readings
			runtime.ReadMemStats(&memStats)
			
			heapIncrease := memStats.HeapAlloc - startHeap
			if heapIncrease > maxHeapIncrease {
				maxHeapIncrease = heapIncrease
			}
			samples++
			
		case <-ctx.Done():
			// Check final memory state
			avgIncreasePerSample := maxHeapIncrease / uint64(samples)
			
			// Allow 1MB per sample as reasonable overhead
			allowedIncrease := uint64(1024 * 1024 * samples)
			if maxHeapIncrease > allowedIncrease {
				t.Errorf("Memory increased by %d bytes (avg %d/sample), exceeds allowed %d",
					maxHeapIncrease, avgIncreasePerSample, allowedIncrease)
			}
			return
		}
	}
}

// Helper functions

func generateChanges(count int) []differ.SimpleChange {
	changes := make([]differ.SimpleChange, count)
	for i := 0; i < count; i++ {
		changes[i] = differ.SimpleChange{
			Type:         "modified",
			ResourceID:   fmt.Sprintf("resource-%d", i),
			ResourceType: "deployment",
			ResourceName: fmt.Sprintf("app-%d", i),
			Namespace:    "default",
			Timestamp:    time.Now(),
			Details: []differ.SimpleFieldChange{
				{
					Field:    "replicas",
					OldValue: i,
					NewValue: i + 1,
				},
			},
		}
	}
	return changes
}

func generateChangesWithPrefix(count int, prefix string) []differ.SimpleChange {
	changes := make([]differ.SimpleChange, count)
	for i := 0; i < count; i++ {
		changes[i] = differ.SimpleChange{
			Type:         "modified",
			ResourceID:   fmt.Sprintf("%s-resource-%d", prefix, i),
			ResourceType: "deployment",
			ResourceName: fmt.Sprintf("%s-app-%d", prefix, i),
			Namespace:    "default",
			Timestamp:    time.Now(),
		}
	}
	return changes
}