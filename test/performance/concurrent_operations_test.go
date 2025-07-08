package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

// ConcurrencyResults tracks the results of concurrent operations
type ConcurrencyResults struct {
	TotalOperations   int
	SuccessfulOps     int64
	FailedOps         int64
	TotalDuration     time.Duration
	MaxMemoryUsage    uint64
	AvgOperationTime  time.Duration
	Throughput        float64 // operations per second
	ErrorRate         float64
}

// TestConcurrentScanning tests multiple simultaneous scans
func TestConcurrentScanning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent scanning test in short mode")
	}

	scenarios := []struct {
		name            string
		workers         int
		resourcesPerWorker int
		maxDuration     time.Duration
	}{
		{"low_concurrency", 4, 1000, 30 * time.Second},
		{"medium_concurrency", 8, 1000, 45 * time.Second},
		{"high_concurrency", 16, 1000, 60 * time.Second},
		{"extreme_concurrency", 32, 500, 90 * time.Second},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			
			t.Logf("Testing %d concurrent workers with %d resources each...", 
				scenario.workers, scenario.resourcesPerWorker)

			// Create separate state files for each worker
			stateFiles := make([]string, scenario.workers)
			for i := 0; i < scenario.workers; i++ {
				stateFile := filepath.Join(tmpDir, fmt.Sprintf("concurrent-state-%d.tfstate", i))
				stateFiles[i] = stateFile
				
				state := createMegaTestState(scenario.resourcesPerWorker)
				data, err := json.MarshalIndent(state, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal state for worker %d: %v", i, err)
				}
				
				if err := os.WriteFile(stateFile, data, 0644); err != nil {
					t.Fatalf("Failed to write state file for worker %d: %v", i, err)
				}
			}

			// Track results
			var results ConcurrencyResults
			results.TotalOperations = scenario.workers
			
			var wg sync.WaitGroup
			var successOps, failedOps int64
			operationTimes := make([]time.Duration, scenario.workers)
			var maxMemory uint64

			// Monitor memory usage
			memMonitorCtx, memCancel := context.WithCancel(context.Background())
			go func() {
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()
				
				for {
					select {
					case <-memMonitorCtx.Done():
						return
					case <-ticker.C:
						var m runtime.MemStats
						runtime.ReadMemStats(&m)
						atomic.StoreUint64(&maxMemory, m.HeapAlloc)
					}
				}
			}()

			startTime := time.Now()

			// Launch concurrent workers
			for i := 0; i < scenario.workers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					
					workerStart := time.Now()
					
					collector := terraform.NewTerraformCollector()
					config := collectors.CollectorConfig{
						StatePaths: []string{stateFiles[workerID]},
						Tags: map[string]string{
							"worker":     fmt.Sprintf("%d", workerID),
							"concurrent": "true",
						},
					}

					snapshot, err := collector.Collect(context.Background(), config)
					operationTimes[workerID] = time.Since(workerStart)
					
					if err != nil {
						t.Logf("Worker %d failed: %v", workerID, err)
						atomic.AddInt64(&failedOps, 1)
						return
					}

					if len(snapshot.Resources) != scenario.resourcesPerWorker {
						t.Logf("Worker %d: expected %d resources, got %d", 
							workerID, scenario.resourcesPerWorker, len(snapshot.Resources))
					}

					atomic.AddInt64(&successOps, 1)
				}(i)
			}

			wg.Wait()
			totalDuration := time.Since(startTime)
			memCancel()

			// Calculate results
			results.SuccessfulOps = atomic.LoadInt64(&successOps)
			results.FailedOps = atomic.LoadInt64(&failedOps)
			results.TotalDuration = totalDuration
			results.MaxMemoryUsage = atomic.LoadUint64(&maxMemory)
			results.ErrorRate = float64(results.FailedOps) / float64(results.TotalOperations) * 100
			results.Throughput = float64(results.SuccessfulOps) / totalDuration.Seconds()

			// Calculate average operation time
			var totalOpTime time.Duration
			for _, opTime := range operationTimes {
				totalOpTime += opTime
			}
			results.AvgOperationTime = totalOpTime / time.Duration(len(operationTimes))

			// Validate results
			if results.ErrorRate > 5.0 {
				t.Errorf("Error rate too high: %.2f%% (max 5%%)", results.ErrorRate)
			}

			if totalDuration > scenario.maxDuration {
				t.Errorf("Total duration %v exceeded limit %v", totalDuration, scenario.maxDuration)
			}

			// Log detailed results
			t.Logf("Concurrent scanning results (%s):", scenario.name)
			t.Logf("  Workers: %d", scenario.workers)
			t.Logf("  Successful operations: %d/%d", results.SuccessfulOps, results.TotalOperations)
			t.Logf("  Error rate: %.2f%%", results.ErrorRate)
			t.Logf("  Total duration: %v", results.TotalDuration)
			t.Logf("  Average operation time: %v", results.AvgOperationTime)
			t.Logf("  Throughput: %.2f ops/sec", results.Throughput)
			t.Logf("  Peak memory usage: %.2f MB", float64(results.MaxMemoryUsage)/(1024*1024))
		})
	}
}

// TestConcurrentDiffOperations tests multiple simultaneous diff operations
func TestConcurrentDiffOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent diff test in short mode")
	}

	scenarios := []struct {
		name           string
		diffOperations int
		resourceCount  int
		changePercent  int
	}{
		{"concurrent_small_diffs", 8, 1000, 5},
		{"concurrent_medium_diffs", 4, 5000, 3},
		{"concurrent_large_diffs", 2, 15000, 2},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Testing %d concurrent diff operations with %d resources each...", 
				scenario.diffOperations, scenario.resourceCount)

			// Pre-generate snapshot pairs for diff operations
			snapshotPairs := make([][2]*types.Snapshot, scenario.diffOperations)
			for i := 0; i < scenario.diffOperations; i++ {
				baseline := createLargeSnapshot(fmt.Sprintf("baseline-%d", i), scenario.resourceCount)
				modified := createLargeSnapshotWithChanges(fmt.Sprintf("modified-%d", i), 
					scenario.resourceCount, scenario.changePercent)
				snapshotPairs[i] = [2]*types.Snapshot{baseline, modified}
			}

			var wg sync.WaitGroup
			var successOps, failedOps int64
			results := make([]differ.SimpleChangeReport, scenario.diffOperations)
			operationTimes := make([]time.Duration, scenario.diffOperations)

			startTime := time.Now()

			// Launch concurrent diff operations
			for i := 0; i < scenario.diffOperations; i++ {
				wg.Add(1)
				go func(opID int) {
					defer wg.Done()
					
					opStart := time.Now()
					
					differ := differ.NewSimpleDiffer()
					report, err := differ.Compare(snapshotPairs[opID][0], snapshotPairs[opID][1])
					
					operationTimes[opID] = time.Since(opStart)
					
					if err != nil {
						t.Logf("Diff operation %d failed: %v", opID, err)
						atomic.AddInt64(&failedOps, 1)
						return
					}

					results[opID] = *report
					atomic.AddInt64(&successOps, 1)
				}(i)
			}

			wg.Wait()
			totalDuration := time.Since(startTime)

			// Analyze results
			totalChanges := 0
			for _, report := range results {
				totalChanges += len(report.Changes)
			}

			avgOperationTime := totalDuration / time.Duration(scenario.diffOperations)
			throughput := float64(successOps) / totalDuration.Seconds()
			errorRate := float64(failedOps) / float64(scenario.diffOperations) * 100

			t.Logf("Concurrent diff results (%s):", scenario.name)
			t.Logf("  Operations: %d", scenario.diffOperations)
			t.Logf("  Successful operations: %d/%d", successOps, scenario.diffOperations)
			t.Logf("  Error rate: %.2f%%", errorRate)
			t.Logf("  Total duration: %v", totalDuration)
			t.Logf("  Average operation time: %v", avgOperationTime)
			t.Logf("  Throughput: %.2f diffs/sec", throughput)
			t.Logf("  Total changes detected: %d", totalChanges)
			t.Logf("  Average changes per diff: %.1f", float64(totalChanges)/float64(successOps))

			// Validate performance
			if errorRate > 2.0 {
				t.Errorf("Diff error rate too high: %.2f%% (max 2%%)", errorRate)
			}

			maxExpectedTime := 30 * time.Second
			if avgOperationTime > maxExpectedTime {
				t.Errorf("Average diff time too high: %v (max %v)", avgOperationTime, maxExpectedTime)
			}
		})
	}
}

// TestConcurrentFileOperations tests concurrent file I/O operations
func TestConcurrentFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent file operations test in short mode")
	}

	scenarios := []struct {
		name        string
		workers     int
		filesPerWorker int
		duration    time.Duration
	}{
		{"multiple_file_workers", 4, 10, 5 * time.Second},
		{"high_concurrency_files", 8, 5, 3 * time.Second},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			
			t.Logf("Testing %d concurrent file workers with %d files each...", scenario.workers, scenario.filesPerWorker)

			var wg sync.WaitGroup
			var successOps, failedOps int64
			
			startTime := time.Now()

			for i := 0; i < scenario.workers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					
					for j := 0; j < scenario.filesPerWorker; j++ {
						stateFile := filepath.Join(tmpDir, fmt.Sprintf("worker-%d-file-%d.tfstate", workerID, j))
						
						// Create and write state file
						state := createMegaTestState(500) // Smaller states for concurrency
						data, err := json.MarshalIndent(state, "", "  ")
						if err != nil {
							atomic.AddInt64(&failedOps, 1)
							continue
						}
						
						if err := os.WriteFile(stateFile, data, 0644); err != nil {
							atomic.AddInt64(&failedOps, 1)
							continue
						}
						
						// Process the file
						collector := terraform.NewTerraformCollector()
						config := collectors.CollectorConfig{
							StatePaths: []string{stateFile},
							Tags:       map[string]string{"worker": fmt.Sprintf("%d", workerID)},
						}
						
						_, err = collector.Collect(context.Background(), config)
						if err != nil {
							atomic.AddInt64(&failedOps, 1)
						} else {
							atomic.AddInt64(&successOps, 1)
						}
					}
				}(i)
			}

			wg.Wait()
			totalDuration := time.Since(startTime)

			totalOps := scenario.workers * scenario.filesPerWorker
			errorRate := float64(failedOps) / float64(totalOps) * 100
			throughput := float64(successOps) / totalDuration.Seconds()

			t.Logf("Concurrent file operations results (%s):", scenario.name)
			t.Logf("  Workers: %d", scenario.workers)
			t.Logf("  Files per worker: %d", scenario.filesPerWorker)
			t.Logf("  Total operations: %d", totalOps)
			t.Logf("  Successful operations: %d", successOps)
			t.Logf("  Failed operations: %d", failedOps)
			t.Logf("  Error rate: %.2f%%", errorRate)
			t.Logf("  Duration: %v", totalDuration)
			t.Logf("  Throughput: %.2f ops/sec", throughput)
			
			// Validate concurrent file handling
			if errorRate > 15.0 {
				t.Errorf("Concurrent file operation error rate too high: %.2f%% (max 15%%)", errorRate)
			}
		})
	}
}

// TestConcurrentStorageOperations tests concurrent storage read/write operations
func TestConcurrentStorageOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent storage test in short mode")
	}

	scenarios := []struct {
		name           string
		writers        int
		readers        int
		snapshotsPerWriter int
	}{
		{"balanced_load", 4, 4, 10},
		{"write_heavy", 8, 2, 5},
		{"read_heavy", 2, 8, 15},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			storageConfig := storage.Config{BaseDir: tmpDir}
			storageEngine, err := storage.NewLocalStorage(storageConfig)
			if err != nil {
				t.Fatalf("Failed to create storage engine: %v", err)
			}
			
			t.Logf("Testing %d writers and %d readers with %d snapshots per writer...", 
				scenario.writers, scenario.readers, scenario.snapshotsPerWriter)

			var wg sync.WaitGroup
			var writeOps, readOps, writeErrors, readErrors int64

			// Generate snapshots for writers
			allSnapshots := make([]*types.Snapshot, scenario.writers*scenario.snapshotsPerWriter)
			for i := 0; i < len(allSnapshots); i++ {
				allSnapshots[i] = createLargeSnapshot(fmt.Sprintf("concurrent-snapshot-%d", i), 1000)
			}

			startTime := time.Now()

			// Launch writers
			for w := 0; w < scenario.writers; w++ {
				wg.Add(1)
				go func(writerID int) {
					defer wg.Done()
					
					startIdx := writerID * scenario.snapshotsPerWriter
					endIdx := startIdx + scenario.snapshotsPerWriter
					
					for i := startIdx; i < endIdx; i++ {
						err := storageEngine.SaveSnapshot(allSnapshots[i])
						if err != nil {
							t.Logf("Writer %d failed to store snapshot %d: %v", writerID, i, err)
							atomic.AddInt64(&writeErrors, 1)
						} else {
							atomic.AddInt64(&writeOps, 1)
						}
						
						// Add small delay to simulate realistic load
						time.Sleep(10 * time.Millisecond)
					}
				}(w)
			}

			// Launch readers (start after a delay to ensure some data is written)
			time.Sleep(100 * time.Millisecond)
			
			for r := 0; r < scenario.readers; r++ {
				wg.Add(1)
				go func(readerID int) {
					defer wg.Done()
					
					// Readers try to read snapshots that should exist
					readsPerReader := scenario.snapshotsPerWriter
					
					for i := 0; i < readsPerReader; i++ {
						snapshotID := fmt.Sprintf("concurrent-snapshot-%d", i)
						_, err := storageEngine.LoadSnapshot(snapshotID)
						if err != nil {
							// It's ok if snapshot doesn't exist yet (race with writers)
							if !os.IsNotExist(err) {
								t.Logf("Reader %d failed to load snapshot %s: %v", readerID, snapshotID, err)
								atomic.AddInt64(&readErrors, 1)
							}
						} else {
							atomic.AddInt64(&readOps, 1)
						}
						
						time.Sleep(15 * time.Millisecond)
					}
				}(r)
			}

			wg.Wait()
			totalDuration := time.Since(startTime)

			// Calculate results
			totalOps := atomic.LoadInt64(&writeOps) + atomic.LoadInt64(&readOps)
			totalErrors := atomic.LoadInt64(&writeErrors) + atomic.LoadInt64(&readErrors)
			errorRate := float64(totalErrors) / float64(totalOps+totalErrors) * 100
			throughput := float64(totalOps) / totalDuration.Seconds()

			t.Logf("Concurrent storage results (%s):", scenario.name)
			t.Logf("  Write operations: %d (errors: %d)", writeOps, writeErrors)
			t.Logf("  Read operations: %d (errors: %d)", readOps, readErrors)
			t.Logf("  Total duration: %v", totalDuration)
			t.Logf("  Error rate: %.2f%%", errorRate)
			t.Logf("  Throughput: %.2f ops/sec", throughput)

			// Validate concurrent storage performance
			if errorRate > 5.0 {
				t.Errorf("Storage error rate too high: %.2f%% (max 5%%)", errorRate)
			}

			minThroughput := 10.0 // At least 10 ops/sec
			if throughput < minThroughput {
				t.Errorf("Storage throughput too low: %.2f ops/sec (min %.2f)", throughput, minThroughput)
			}
		})
	}
}

// TestResourceContention tests resource contention scenarios
func TestResourceContention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource contention test in short mode")
	}

	t.Log("Testing resource contention scenarios...")

	tmpDir := t.TempDir()
	sharedStateFile := filepath.Join(tmpDir, "shared-state.tfstate")
	
	// Create shared state file
	sharedState := createMegaTestState(5000)
	data, err := json.MarshalIndent(sharedState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal shared state: %v", err)
	}
	
	if err := os.WriteFile(sharedStateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write shared state file: %v", err)
	}

	scenarios := []struct {
		name         string
		concurrent   int
		operation    string
	}{
		{"concurrent_readers", 16, "read"},
		{"mixed_operations", 8, "mixed"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			var wg sync.WaitGroup
			var successOps, failedOps int64
			
			startTime := time.Now()
			
			for i := 0; i < scenario.concurrent; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					
					if scenario.operation == "read" || (scenario.operation == "mixed" && workerID%2 == 0) {
						// Read operation
						collector := terraform.NewTerraformCollector()
						config := collectors.CollectorConfig{
							StatePaths: []string{sharedStateFile},
							Tags:       map[string]string{"worker": fmt.Sprintf("%d", workerID)},
						}
						
						_, err := collector.Collect(context.Background(), config)
						if err != nil {
							t.Logf("Read worker %d failed: %v", workerID, err)
							atomic.AddInt64(&failedOps, 1)
						} else {
							atomic.AddInt64(&successOps, 1)
						}
					} else {
						// Write operation (for mixed scenario)
						modifiedState := createMegaTestStateWithChanges(5000, 1)
						modifiedData, err := json.MarshalIndent(modifiedState, "", "  ")
						if err != nil {
							atomic.AddInt64(&failedOps, 1)
							return
						}
						
						tempFile := filepath.Join(tmpDir, fmt.Sprintf("temp-state-%d.tfstate", workerID))
						if err := os.WriteFile(tempFile, modifiedData, 0644); err != nil {
							atomic.AddInt64(&failedOps, 1)
						} else {
							atomic.AddInt64(&successOps, 1)
						}
					}
				}(i)
			}
			
			wg.Wait()
			totalDuration := time.Since(startTime)
			
			errorRate := float64(failedOps) / float64(successOps+failedOps) * 100
			throughput := float64(successOps) / totalDuration.Seconds()
			
			t.Logf("Resource contention results (%s):", scenario.name)
			t.Logf("  Concurrent operations: %d", scenario.concurrent)
			t.Logf("  Successful operations: %d", successOps)
			t.Logf("  Failed operations: %d", failedOps)
			t.Logf("  Error rate: %.2f%%", errorRate)
			t.Logf("  Duration: %v", totalDuration)
			t.Logf("  Throughput: %.2f ops/sec", throughput)
			
			// Validate contention handling
			if errorRate > 10.0 {
				t.Errorf("Resource contention error rate too high: %.2f%% (max 10%%)", errorRate)
			}
		})
	}
}

// TestSystemLimits tests WGO behavior at system limits
func TestSystemLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system limits test in short mode")
	}

	t.Log("Testing system limits and resource exhaustion scenarios...")

	// Test 1: Maximum concurrent operations
	t.Run("max_concurrent_operations", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		// Create many small state files
		fileCount := 100
		stateFiles := make([]string, fileCount)
		
		for i := 0; i < fileCount; i++ {
			stateFile := filepath.Join(tmpDir, fmt.Sprintf("limit-state-%d.tfstate", i))
			stateFiles[i] = stateFile
			
			state := createMegaTestState(100) // Small states for maximum concurrency
			data, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal state %d: %v", i, err)
			}
			
			if err := os.WriteFile(stateFile, data, 0644); err != nil {
				t.Fatalf("Failed to write state file %d: %v", i, err)
			}
		}
		
		// Try to process all files concurrently
		var wg sync.WaitGroup
		var successOps, failedOps int64
		semaphore := make(chan struct{}, runtime.NumCPU()*4) // Limit based on CPU cores
		
		startTime := time.Now()
		
		for i := 0; i < fileCount; i++ {
			wg.Add(1)
			go func(fileIndex int) {
				defer wg.Done()
				
				semaphore <- struct{}{} // Acquire
				defer func() { <-semaphore }() // Release
				
				collector := terraform.NewTerraformCollector()
				config := collectors.CollectorConfig{
					StatePaths: []string{stateFiles[fileIndex]},
					Tags:       map[string]string{"limit-test": "true"},
				}
				
				_, err := collector.Collect(context.Background(), config)
				if err != nil {
					atomic.AddInt64(&failedOps, 1)
				} else {
					atomic.AddInt64(&successOps, 1)
				}
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(startTime)
		
		errorRate := float64(failedOps) / float64(fileCount) * 100
		
		t.Logf("System limits test results:")
		t.Logf("  Files processed: %d", fileCount)
		t.Logf("  Successful: %d", successOps)
		t.Logf("  Failed: %d", failedOps)
		t.Logf("  Error rate: %.2f%%", errorRate)
		t.Logf("  Duration: %v", duration)
		t.Logf("  CPU cores: %d", runtime.NumCPU())
		
		// Should handle high concurrency gracefully
		if errorRate > 20.0 {
			t.Errorf("System limits error rate too high: %.2f%% (max 20%%)", errorRate)
		}
	})
}

// BenchmarkConcurrentThroughput benchmarks maximum concurrent throughput
func BenchmarkConcurrentThroughput(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping concurrent throughput benchmark in short mode")
	}

	concurrencyLevels := []int{1, 2, 4, 8, 16}
	
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency_%d", concurrency), func(b *testing.B) {
			tmpDir := b.TempDir()
			
			// Create test data
			stateFiles := make([]string, concurrency)
			for i := 0; i < concurrency; i++ {
				stateFile := filepath.Join(tmpDir, fmt.Sprintf("bench-state-%d.tfstate", i))
				stateFiles[i] = stateFile
				
				state := createMegaTestState(1000)
				data, _ := json.MarshalIndent(state, "", "  ")
				os.WriteFile(stateFile, data, 0644)
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				
				for j := 0; j < concurrency; j++ {
					wg.Add(1)
					go func(workerID int) {
						defer wg.Done()
						
						collector := terraform.NewTerraformCollector()
						config := collectors.CollectorConfig{
							StatePaths: []string{stateFiles[workerID]},
							Tags:       map[string]string{"bench": "true"},
						}
						
						_, err := collector.Collect(context.Background(), config)
						if err != nil {
							b.Logf("Benchmark worker %d failed: %v", workerID, err)
						}
					}(j)
				}
				
				wg.Wait()
			}
		})
	}
}