package workers

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// BenchmarkResourceProcessor tests resource processor performance
func BenchmarkResourceProcessor(b *testing.B) {
	// Test different worker counts
	workerCounts := []int{1, 2, 4, 8, 16, runtime.NumCPU()}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workerCount), func(b *testing.B) {
			benchmarkResourceProcessorWithWorkers(b, workerCount)
		})
	}
}

func benchmarkResourceProcessorWithWorkers(b *testing.B, workerCount int) {
	// Create processor with specified worker count
	processor := NewResourceProcessor(
		WithWorkerCount(workerCount),
		WithBufferSize(1000),
		WithTimeout(30*time.Second),
		WithProcessingFunction(func(raw RawResource) (*types.Resource, error) {
			// Simulate processing time
			time.Sleep(time.Microsecond * 100)

			return &types.Resource{
				ID:            raw.ID,
				Type:          raw.Type,
				Provider:      raw.Provider,
				Configuration: raw.Data,
			}, nil
		}),
	)

	ctx := context.Background()
	processor.Start(ctx)
	defer processor.Stop()

	// Generate test data
	resources := generateRawResources(b.N)

	b.ResetTimer()
	b.ReportAllocs()

	// Process resources
	results, errors := processor.ProcessResources(resources)

	b.StopTimer()

	// Verify results
	if len(errors) > 0 {
		b.Errorf("Processing errors: %v", errors)
	}

	if len(results) != b.N {
		b.Errorf("Expected %d results, got %d", b.N, len(results))
	}

	// Report metrics
	stats := processor.GetStats()
	b.ReportMetric(float64(stats.TotalProcessed), "processed")
	b.ReportMetric(float64(stats.TotalErrors), "errors")
	b.ReportMetric(float64(stats.CurrentMemory), "memory_bytes")
}

// BenchmarkTerraformParser tests Terraform parser performance
func BenchmarkTerraformParser(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workerCount), func(b *testing.B) {
			benchmarkTerraformParserWithWorkers(b, workerCount)
		})
	}
}

func benchmarkTerraformParserWithWorkers(b *testing.B, workerCount int) {
	// Create parser with specified worker count
	parser := NewConcurrentTerraformParser(
		WithTerraformWorkerCount(workerCount),
		WithTerraformBufferSize(100),
		WithTerraformTimeout(30*time.Second),
	)

	// Generate test state files
	statePaths := generateTestStateFiles(b.N)
	defer cleanupTestFiles(statePaths)

	b.ResetTimer()
	b.ReportAllocs()

	// Parse state files
	resources, err := parser.ParseStatesConcurrent(statePaths)

	b.StopTimer()

	if err != nil {
		b.Fatalf("Parsing failed: %v", err)
	}

	// Report metrics
	stats := parser.GetStats()
	b.ReportMetric(float64(len(resources)), "resources")
	b.ReportMetric(float64(len(statePaths)), "files")

	totalFiles := int64(0)
	totalResources := int64(0)
	for _, workerStat := range stats.WorkerStats {
		totalFiles += workerStat.FilesParsed
		totalResources += workerStat.ResourcesFound
	}

	b.ReportMetric(float64(totalFiles), "files_parsed")
	b.ReportMetric(float64(totalResources), "resources_found")
}

// BenchmarkDiffWorker tests diff worker performance
func BenchmarkDiffWorker(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workerCount), func(b *testing.B) {
			benchmarkDiffWorkerWithWorkers(b, workerCount)
		})
	}
}

func benchmarkDiffWorkerWithWorkers(b *testing.B, workerCount int) {
	// Create diff worker
	diffWorker := NewDiffWorker(
		WithDiffWorkerCount(workerCount),
		WithDiffBufferSize(100),
		WithDiffTimeout(30*time.Second),
	)

	// Generate test snapshots
	baseline := generateTestSnapshot("baseline", b.N)
	current := generateTestSnapshot("current", b.N)

	// Modify some resources to create differences
	modifySnapshotResources(current, 0.3) // 30% of resources changed

	b.ResetTimer()
	b.ReportAllocs()

	// Compute diffs
	report, err := diffWorker.ComputeDiffsConcurrent(baseline, current)

	b.StopTimer()

	if err != nil {
		b.Fatalf("Diff computation failed: %v", err)
	}

	// Report metrics
	stats := diffWorker.GetStats()
	b.ReportMetric(float64(stats.TotalCompared), "compared")
	b.ReportMetric(float64(stats.TotalChanges), "changes")
	b.ReportMetric(float64(stats.TotalErrors), "errors")
	b.ReportMetric(float64(len(report.Changes)), "report_changes")
}

// BenchmarkStorageManager tests storage manager performance
func BenchmarkStorageManager(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workerCount), func(b *testing.B) {
			benchmarkStorageManagerWithWorkers(b, workerCount)
		})
	}
}

func benchmarkStorageManagerWithWorkers(b *testing.B, workerCount int) {
	// Create storage manager
	storageManager := NewConcurrentStorageManager(
		WithStorageWorkerCount(workerCount),
		WithStorageBufferSize(100),
		WithStorageTimeout(30*time.Second),
	)

	// Generate test snapshots
	snapshots := make([]*types.Snapshot, b.N)
	for i := 0; i < b.N; i++ {
		snapshots[i] = generateTestSnapshot(fmt.Sprintf("snapshot_%d", i), 100)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Save snapshots
	for _, snapshot := range snapshots {
		err := storageManager.SaveSnapshotConcurrent(snapshot)
		if err != nil {
			b.Errorf("Save failed: %v", err)
		}
	}

	b.StopTimer()

	// Report metrics
	stats := storageManager.GetStats()
	b.ReportMetric(float64(stats.TotalOperations), "operations")
	b.ReportMetric(float64(stats.TotalBytes), "bytes")
	b.ReportMetric(float64(stats.TotalErrors), "errors")
}

// BenchmarkMemoryOptimizer tests memory optimizer performance
func BenchmarkMemoryOptimizer(b *testing.B) {
	config := DefaultMemoryOptimizationConfig()
	optimizer := NewMemoryOptimizer(config)

	ctx := context.Background()
	optimizer.Start(ctx)
	defer optimizer.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	// Test object pooling
	for i := 0; i < b.N; i++ {
		// Get resource from pool
		resource := optimizer.GetResource()

		// Simulate usage
		resource.ID = fmt.Sprintf("resource_%d", i)
		resource.Type = "test_type"
		resource.Provider = "test_provider"

		// Return to pool
		optimizer.PutResource(resource)
	}

	b.StopTimer()

	// Report metrics
	stats := optimizer.GetStats()
	b.ReportMetric(float64(stats.ObjectPoolHits), "pool_hits")
	b.ReportMetric(float64(stats.ObjectPoolMisses), "pool_misses")
	b.ReportMetric(float64(stats.TotalMemoryUsage), "memory_usage")
}

// BenchmarkWorkerPoolManager tests worker pool manager performance
func BenchmarkWorkerPoolManager(b *testing.B) {
	config := DefaultWorkerPoolConfig()
	manager := NewWorkerPoolManager(config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// Generate test data
	resources := generateRawResources(b.N)

	b.ResetTimer()
	b.ReportAllocs()

	// Process resources through manager
	results, errors := manager.ProcessResourcesConcurrently(resources)

	b.StopTimer()

	if len(errors) > 0 {
		b.Errorf("Processing errors: %v", errors)
	}

	if len(results) != b.N {
		b.Errorf("Expected %d results, got %d", b.N, len(results))
	}

	// Report metrics
	metrics := manager.GetMetrics()
	b.ReportMetric(float64(metrics.ResourceProcessor.TotalProcessed), "processed")
	b.ReportMetric(float64(metrics.SystemMetrics.MemoryUsage), "memory_usage")
	b.ReportMetric(float64(metrics.SystemMetrics.GoroutineCount), "goroutines")
}

// BenchmarkConcurrentLoad tests concurrent load performance
func BenchmarkConcurrentLoad(b *testing.B) {
	concurrencyLevels := []int{1, 10, 50, 100, 500}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			benchmarkConcurrentLoadWithLevel(b, concurrency)
		})
	}
}

func benchmarkConcurrentLoadWithLevel(b *testing.B, concurrency int) {
	config := DefaultWorkerPoolConfig()
	manager := NewWorkerPoolManager(config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// Generate test data
	resourcesPerGoroutine := b.N / concurrency
	if resourcesPerGoroutine == 0 {
		resourcesPerGoroutine = 1
	}

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	wg.Add(concurrency)

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()

			resources := generateRawResources(resourcesPerGoroutine)
			_, errors := manager.ProcessResourcesConcurrently(resources)

			if len(errors) > 0 {
				b.Errorf("Processing errors: %v", errors)
			}
		}()
	}

	wg.Wait()

	b.StopTimer()

	// Report metrics
	metrics := manager.GetMetrics()
	b.ReportMetric(float64(metrics.ResourceProcessor.TotalProcessed), "processed")
	b.ReportMetric(float64(metrics.SystemMetrics.GoroutineCount), "goroutines")
}

// BenchmarkMemoryUsage tests memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	resourceCounts := []int{100, 1000, 10000, 100000}

	for _, count := range resourceCounts {
		b.Run(fmt.Sprintf("Resources_%d", count), func(b *testing.B) {
			benchmarkMemoryUsageWithCount(b, count)
		})
	}
}

func benchmarkMemoryUsageWithCount(b *testing.B, resourceCount int) {
	config := DefaultWorkerPoolConfig()
	manager := NewWorkerPoolManager(config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// Measure initial memory
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Generate and process resources
		resources := generateRawResources(resourceCount)
		results, errors := manager.ProcessResourcesConcurrently(resources)

		if len(errors) > 0 {
			b.Errorf("Processing errors: %v", errors)
		}

		// Verify results
		if len(results) != resourceCount {
			b.Errorf("Expected %d results, got %d", resourceCount, len(results))
		}

		// Force GC to get accurate memory measurements
		runtime.GC()
	}

	b.StopTimer()

	// Measure final memory
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)

	// Report memory metrics
	memoryUsed := finalMem.Alloc - initialMem.Alloc
	b.ReportMetric(float64(memoryUsed), "memory_used")
	b.ReportMetric(float64(finalMem.Alloc), "final_memory")
	b.ReportMetric(float64(finalMem.TotalAlloc), "total_alloc")
	b.ReportMetric(float64(finalMem.NumGC), "gc_count")
}

// Helper functions

func generateRawResources(count int) []RawResource {
	resources := make([]RawResource, count)

	for i := 0; i < count; i++ {
		resources[i] = RawResource{
			ID:       fmt.Sprintf("resource_%d", i),
			Type:     fmt.Sprintf("type_%d", i%10),
			Provider: fmt.Sprintf("provider_%d", i%3),
			Data: map[string]interface{}{
				"name":  fmt.Sprintf("resource_%d", i),
				"value": rand.Intn(1000),
				"config": map[string]interface{}{
					"setting1": fmt.Sprintf("value_%d", i),
					"setting2": rand.Intn(100),
				},
			},
		}
	}

	return resources
}

func generateTestStateFiles(count int) []string {
	// In a real implementation, this would create temporary .tfstate files
	// For benchmarking, we'll just return file paths
	paths := make([]string, count)
	for i := 0; i < count; i++ {
		paths[i] = fmt.Sprintf("/tmp/test_state_%d.tfstate", i)
	}
	return paths
}

func cleanupTestFiles(paths []string) {
	// Clean up temporary files
	for _, path := range paths {
		// os.Remove(path) - would be used in real implementation
		_ = path
	}
}

func generateTestSnapshot(id string, resourceCount int) *types.Snapshot {
	resources := make([]types.Resource, resourceCount)

	for i := 0; i < resourceCount; i++ {
		resources[i] = types.Resource{
			ID:       fmt.Sprintf("%s_resource_%d", id, i),
			Type:     fmt.Sprintf("type_%d", i%10),
			Provider: fmt.Sprintf("provider_%d", i%3),
			Configuration: map[string]interface{}{
				"name":  fmt.Sprintf("resource_%d", i),
				"value": rand.Intn(1000),
				"config": map[string]interface{}{
					"setting1": fmt.Sprintf("value_%d", i),
					"setting2": rand.Intn(100),
				},
			},
			Tags: map[string]string{
				"environment": "test",
				"team":        fmt.Sprintf("team_%d", i%5),
			},
		}
	}

	return &types.Snapshot{
		ID:        id,
		Timestamp: time.Now(),
		Provider:  "test",
		Resources: resources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			CollectionTime:   time.Second,
			ResourceCount:    resourceCount,
		},
	}
}

func modifySnapshotResources(snapshot *types.Snapshot, changeRatio float64) {
	changeCount := int(float64(len(snapshot.Resources)) * changeRatio)

	for i := 0; i < changeCount; i++ {
		resource := &snapshot.Resources[i]

		// Modify some configuration
		if resource.Configuration == nil {
			resource.Configuration = make(map[string]interface{})
		}

		resource.Configuration["modified"] = true
		resource.Configuration["modification_time"] = time.Now()

		// Add or modify tags
		if resource.Tags == nil {
			resource.Tags = make(map[string]string)
		}

		resource.Tags["modified"] = "true"
	}
}

// Performance test utility functions

func runPerformanceTest(name string, iterations int, fn func()) {
	fmt.Printf("Running performance test: %s\n", name)

	start := time.Now()

	for i := 0; i < iterations; i++ {
		fn()
	}

	duration := time.Since(start)

	fmt.Printf("Test: %s\n", name)
	fmt.Printf("Iterations: %d\n", iterations)
	fmt.Printf("Total time: %v\n", duration)
	fmt.Printf("Average time per iteration: %v\n", duration/time.Duration(iterations))
	fmt.Printf("Operations per second: %.2f\n", float64(iterations)/duration.Seconds())
	fmt.Println()
}

// Memory profiling test
func TestMemoryProfile(t *testing.T) {
	config := DefaultWorkerPoolConfig()
	manager := NewWorkerPoolManager(config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// Generate large dataset
	resources := generateRawResources(10000)

	// Measure memory before processing
	var beforeMem runtime.MemStats
	runtime.ReadMemStats(&beforeMem)

	// Process resources
	results, errors := manager.ProcessResourcesConcurrently(resources)

	// Measure memory after processing
	var afterMem runtime.MemStats
	runtime.ReadMemStats(&afterMem)

	// Report memory usage
	t.Logf("Memory usage before: %d bytes", beforeMem.Alloc)
	t.Logf("Memory usage after: %d bytes", afterMem.Alloc)
	t.Logf("Memory increase: %d bytes", afterMem.Alloc-beforeMem.Alloc)
	t.Logf("Total allocations: %d bytes", afterMem.TotalAlloc)
	t.Logf("GC runs: %d", afterMem.NumGC)

	// Verify results
	if len(errors) > 0 {
		t.Errorf("Processing errors: %v", errors)
	}

	if len(results) != len(resources) {
		t.Errorf("Expected %d results, got %d", len(resources), len(results))
	}
}

// Stress test
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	config := DefaultWorkerPoolConfig()
	manager := NewWorkerPoolManager(config)

	ctx := context.Background()
	manager.Start(ctx)
	defer manager.Stop()

	// Run stress test for 30 seconds
	timeout := 30 * time.Second
	startTime := time.Now()

	var processedCount int64
	var errorCount int64

	for time.Since(startTime) < timeout {
		// Generate resources
		resources := generateRawResources(1000)

		// Process resources
		results, errors := manager.ProcessResourcesConcurrently(resources)

		processedCount += int64(len(results))
		errorCount += int64(len(errors))

		// Brief pause to avoid overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	duration := time.Since(startTime)

	t.Logf("Stress test completed:")
	t.Logf("Duration: %v", duration)
	t.Logf("Total processed: %d", processedCount)
	t.Logf("Total errors: %d", errorCount)
	t.Logf("Processing rate: %.2f ops/sec", float64(processedCount)/duration.Seconds())

	// Check for excessive errors
	errorRate := float64(errorCount) / float64(processedCount)
	if errorRate > 0.01 { // More than 1% error rate
		t.Errorf("High error rate: %.2f%%", errorRate*100)
	}
}
