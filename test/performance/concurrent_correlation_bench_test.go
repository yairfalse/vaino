package performance

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/analyzer"
)

// BenchmarkOriginalVsConcurrentCorrelation compares original vs concurrent correlation
func BenchmarkOriginalVsConcurrentCorrelation(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000, 5000}

	for _, size := range sizes {
		changes := generateRandomChanges(size)

		// Original implementation
		b.Run(fmt.Sprintf("Original_%d", size), func(b *testing.B) {
			correlator := analyzer.NewCorrelator()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = correlator.GroupChanges(changes)
			}
		})

		// Concurrent implementation
		b.Run(fmt.Sprintf("Concurrent_%d", size), func(b *testing.B) {
			correlator := analyzer.NewConcurrentCorrelator()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = correlator.AnalyzeChangesConcurrent(changes)
			}
		})
	}
}

// BenchmarkConcurrentCorrelationScaling tests scaling with different worker counts
func BenchmarkConcurrentCorrelationScaling(b *testing.B) {
	changes := generateRandomChanges(2000)
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			correlator := analyzer.NewConcurrentCorrelator()
			correlator.SetWorkerCount(workers)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = correlator.AnalyzeChangesConcurrent(changes)
			}
		})
	}
}

// BenchmarkTimelineProcessing benchmarks timeline processing performance
func BenchmarkTimelineProcessing(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}

	for _, size := range sizes {
		changes := generateRandomChanges(size)

		b.Run(fmt.Sprintf("Timeline_%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = analyzer.BuildTimelineFromChanges(changes, 1*time.Minute)
			}
		})
	}
}

// BenchmarkMemoryUsageConcurrent benchmarks memory usage of concurrent correlation
func BenchmarkMemoryUsageConcurrent(b *testing.B) {
	changes := generateRandomChanges(1000)
	correlator := analyzer.NewConcurrentCorrelator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result := correlator.AnalyzeChangesConcurrent(changes)
		_ = result // Prevent optimization
	}
}

// BenchmarkPatternMatching benchmarks individual pattern matchers
func BenchmarkPatternMatching(b *testing.B) {
	changes := generateKnownPatternChanges(1000)

	patterns := []struct {
		name    string
		matcher analyzer.PatternMatcher
	}{
		{"Scaling", &analyzer.ScalingPatternMatcher{}},
		{"ConfigUpdate", &analyzer.ConfigUpdatePatternMatcher{}},
		{"ServiceDeployment", &analyzer.ServiceDeploymentPatternMatcher{}},
		{"Network", &analyzer.NetworkPatternMatcher{}},
		{"Storage", &analyzer.StoragePatternMatcher{}},
		{"Security", &analyzer.SecurityPatternMatcher{}},
	}

	for _, pattern := range patterns {
		b.Run(pattern.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pattern.matcher.Match(changes)
			}
		})
	}
}

// TestConcurrentCorrelationPerformance tests performance requirements
func TestConcurrentCorrelationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent correlation performance test in short mode")
	}

	tests := []struct {
		name            string
		changeCount     int
		maxDuration     time.Duration
		expectedSpeedup float64
		description     string
	}{
		{
			name:            "small_concurrent",
			changeCount:     100,
			maxDuration:     5 * time.Millisecond,
			expectedSpeedup: 1.0, // No speedup expected for small datasets
			description:     "Small datasets should have minimal overhead",
		},
		{
			name:            "medium_concurrent",
			changeCount:     1000,
			maxDuration:     50 * time.Millisecond,
			expectedSpeedup: 1.5, // 50% speedup expected
			description:     "Medium datasets should show speedup",
		},
		{
			name:            "large_concurrent",
			changeCount:     5000,
			maxDuration:     200 * time.Millisecond,
			expectedSpeedup: 2.0, // 100% speedup expected
			description:     "Large datasets should show significant speedup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := generateRandomChanges(tt.changeCount)

			// Test original implementation
			originalCorrelator := analyzer.NewCorrelator()
			originalStart := time.Now()
			originalGroups := originalCorrelator.GroupChanges(changes)
			originalDuration := time.Since(originalStart)

			// Test concurrent implementation
			concurrentCorrelator := analyzer.NewConcurrentCorrelator()
			concurrentStart := time.Now()
			concurrentResult := concurrentCorrelator.AnalyzeChangesConcurrent(changes)
			concurrentDuration := time.Since(concurrentStart)

			// Check duration requirements
			if concurrentDuration > tt.maxDuration {
				t.Errorf("%s: took %v, expected < %v", tt.description, concurrentDuration, tt.maxDuration)
			}

			// Check speedup
			if originalDuration > 0 {
				actualSpeedup := float64(originalDuration) / float64(concurrentDuration)
				if actualSpeedup < tt.expectedSpeedup {
					t.Logf("Warning: %s: speedup %.2fx, expected %.2fx", tt.description, actualSpeedup, tt.expectedSpeedup)
				}
			}

			// Verify results are reasonable
			if len(concurrentResult.Groups) == 0 {
				t.Error("Concurrent correlation produced no groups")
			}

			// Compare group counts (should be similar)
			originalCount := len(originalGroups)
			concurrentCount := len(concurrentResult.Groups)
			diff := float64(abs(originalCount-concurrentCount)) / float64(originalCount)
			if diff > 0.1 { // Allow 10% difference
				t.Errorf("Group count difference too large: original=%d, concurrent=%d (%.1f%%)",
					originalCount, concurrentCount, diff*100)
			}

			t.Logf("Processed %d changes: original=%v, concurrent=%v (%.2fx speedup), groups: %d->%d",
				tt.changeCount, originalDuration, concurrentDuration,
				float64(originalDuration)/float64(concurrentDuration),
				originalCount, concurrentCount)
		})
	}
}

// TestConcurrentCorrelationAccuracy tests that concurrent correlation produces accurate results
func TestConcurrentCorrelationAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent correlation accuracy test in short mode")
	}

	testCases := []struct {
		name        string
		changeCount int
		iterations  int
	}{
		{"small_accuracy", 100, 10},
		{"medium_accuracy", 500, 5},
		{"large_accuracy", 1000, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var originalResults [][]analyzer.ChangeGroup
			var concurrentResults []*analyzer.CorrelatedChanges

			// Run multiple iterations to check consistency
			for i := 0; i < tc.iterations; i++ {
				changes := generateKnownPatternChanges(tc.changeCount)

				// Original implementation
				originalCorrelator := analyzer.NewCorrelator()
				originalGroups := originalCorrelator.GroupChanges(changes)
				originalResults = append(originalResults, originalGroups)

				// Concurrent implementation
				concurrentCorrelator := analyzer.NewConcurrentCorrelator()
				concurrentResult := concurrentCorrelator.AnalyzeChangesConcurrent(changes)
				concurrentResults = append(concurrentResults, concurrentResult)
			}

			// Check consistency between implementations
			for i := 0; i < tc.iterations; i++ {
				originalGroups := originalResults[i]
				concurrentGroups := concurrentResults[i].Groups

				// Count high confidence groups
				originalHigh := 0
				concurrentHigh := 0

				for _, group := range originalGroups {
					if group.Confidence == "high" {
						originalHigh++
					}
				}

				for _, group := range concurrentGroups {
					if group.Confidence == "high" {
						concurrentHigh++
					}
				}

				// Should find similar number of high confidence groups
				if abs(originalHigh-concurrentHigh) > 2 {
					t.Errorf("Iteration %d: high confidence groups differ significantly: original=%d, concurrent=%d",
						i, originalHigh, concurrentHigh)
				}
			}

			t.Logf("Accuracy test passed for %d changes over %d iterations", tc.changeCount, tc.iterations)
		})
	}
}

// TestConcurrentMemoryUsage tests memory usage doesn't grow excessively
func TestConcurrentMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent memory test in short mode")
	}

	sizes := []int{500, 1000, 2000, 4000}
	var prevMemory int64

	for _, size := range sizes {
		changes := generateRandomChanges(size)
		correlator := analyzer.NewConcurrentCorrelator()

		// Force GC before measurement
		runtime.GC()
		runtime.GC()

		var beforeMem runtime.MemStats
		runtime.ReadMemStats(&beforeMem)

		result := correlator.AnalyzeChangesConcurrent(changes)
		_ = result

		var afterMem runtime.MemStats
		runtime.ReadMemStats(&afterMem)

		memUsed := int64(afterMem.HeapAlloc - beforeMem.HeapAlloc)

		t.Logf("Size %d: used %d bytes (%d KB)", size, memUsed, memUsed/1024)

		// Memory should grow somewhat linearly with input size
		if prevMemory > 0 {
			growth := float64(memUsed) / float64(prevMemory)
			if growth > 5.0 { // More than 5x growth is suspicious
				t.Errorf("Memory usage grew by %0.1fx from %d to %d items",
					growth, sizes[len(sizes)-2], size)
			}
		}

		prevMemory = memUsed
	}
}

// TestWorkerScaling tests how performance scales with worker count
func TestWorkerScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping worker scaling test in short mode")
	}

	changes := generateRandomChanges(2000)
	workerCounts := []int{1, 2, 4, 8}

	var results []struct {
		workers  int
		duration time.Duration
		groups   int
	}

	for _, workers := range workerCounts {
		correlator := analyzer.NewConcurrentCorrelator()
		correlator.SetWorkerCount(workers)

		start := time.Now()
		result := correlator.AnalyzeChangesConcurrent(changes)
		duration := time.Since(start)

		results = append(results, struct {
			workers  int
			duration time.Duration
			groups   int
		}{workers, duration, len(result.Groups)})

		t.Logf("Workers %d: %v duration, %d groups", workers, duration, len(result.Groups))
	}

	// Check that we get reasonable speedup up to CPU count
	cpuCount := runtime.NumCPU()
	for i := 1; i < len(results) && results[i].workers <= cpuCount; i++ {
		speedup := float64(results[0].duration) / float64(results[i].duration)
		expectedSpeedup := float64(results[i].workers) * 0.6 // 60% efficiency

		if speedup < expectedSpeedup {
			t.Logf("Warning: Workers %d: speedup %.2fx, expected %.2fx",
				results[i].workers, speedup, expectedSpeedup)
		}
	}
}

// TestTimelinePerformance tests timeline processing performance
func TestTimelinePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeline performance test in short mode")
	}

	tests := []struct {
		name        string
		changeCount int
		windowSize  time.Duration
		maxDuration time.Duration
	}{
		{"small_timeline", 100, 30 * time.Second, 5 * time.Millisecond},
		{"medium_timeline", 1000, 1 * time.Minute, 50 * time.Millisecond},
		{"large_timeline", 5000, 5 * time.Minute, 200 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := generateRandomChanges(tt.changeCount)

			start := time.Now()
			timeline := analyzer.BuildTimelineFromChanges(changes, tt.windowSize)
			duration := time.Since(start)

			if duration > tt.maxDuration {
				t.Errorf("Timeline processing took %v, expected < %v", duration, tt.maxDuration)
			}

			// Verify timeline was built correctly
			if len(timeline.Events) != tt.changeCount {
				t.Errorf("Timeline has %d events, expected %d", len(timeline.Events), tt.changeCount)
			}

			if len(timeline.TimeWindows) == 0 {
				t.Error("Timeline has no time windows")
			}

			t.Logf("Processed %d changes in %v, created %d windows",
				tt.changeCount, duration, len(timeline.TimeWindows))
		})
	}
}

// Helper function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Stress test for concurrent correlation
func TestConcurrentCorrelationStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Run multiple concurrent correlations simultaneously
	changes := generateRandomChanges(1000)
	concurrentRoutines := 10

	var wg sync.WaitGroup
	results := make([]bool, concurrentRoutines)

	for i := 0; i < concurrentRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			correlator := analyzer.NewConcurrentCorrelator()
			result := correlator.AnalyzeChangesConcurrent(changes)

			// Basic validation
			if len(result.Groups) == 0 {
				t.Errorf("Goroutine %d: no groups produced", id)
				return
			}

			if result.CorrelationStats.TotalChanges != len(changes) {
				t.Errorf("Goroutine %d: wrong change count: got %d, expected %d",
					id, result.CorrelationStats.TotalChanges, len(changes))
				return
			}

			results[id] = true
		}(i)
	}

	wg.Wait()

	// Check all routines succeeded
	for i, success := range results {
		if !success {
			t.Errorf("Goroutine %d failed", i)
		}
	}

	t.Logf("Stress test passed: %d concurrent correlations completed successfully", concurrentRoutines)
}
