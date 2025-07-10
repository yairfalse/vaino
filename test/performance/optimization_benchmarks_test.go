package performance_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/scanner"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/internal/workers"
	"github.com/yairfalse/vaino/pkg/config"
	"github.com/yairfalse/vaino/pkg/types"
)

// BenchmarkEndToEndPerformance benchmarks complete scan-diff-store cycle
func BenchmarkEndToEndPerformance(b *testing.B) {
	tests := []struct {
		name                 string
		providerCount        int
		resourcesPerProvider int
		concurrency          string
	}{
		{"small-sequential", 2, 100, "sequential"},
		{"small-concurrent", 2, 100, "concurrent"},
		{"medium-sequential", 4, 1000, "sequential"},
		{"medium-concurrent", 4, 1000, "concurrent"},
		{"large-sequential", 8, 5000, "sequential"},
		{"large-concurrent", 8, 5000, "concurrent"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Setup
			tempDir := b.TempDir()
			cfg := &config.Config{
				Storage: config.StorageConfig{
					BasePath: tempDir,
				},
			}

			// Create mock providers
			providers := createMockProviders(tt.providerCount, tt.resourcesPerProvider)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var totalTime time.Duration

				// Scan phase
				scanStart := time.Now()
				var snapshot *types.Snapshot

				if tt.concurrency == "concurrent" {
					// Use concurrent scanner
					concurrentScanner := scanner.NewConcurrentScanner(cfg)
					ctx := context.Background()
					scanResult, err := concurrentScanner.ScanAll(ctx, providers)
					if err != nil {
						b.Fatal(err)
					}
					snapshot = mergeSnapshots(scanResult)
				} else {
					// Sequential scan
					resources := []types.Resource{}
					for _, provider := range providers {
						providerResources := provider.GetResources()
						resources = append(resources, providerResources...)
					}
					snapshot = &types.Snapshot{
						ID:        fmt.Sprintf("snapshot-%d", i),
						Timestamp: time.Now(),
						Provider:  "multi",
						Resources: resources,
					}
				}
				scanTime := time.Since(scanStart)

				// Storage phase
				storageStart := time.Now()
				if tt.concurrency == "concurrent" {
					concurrentStorage, _ := storage.NewConcurrentStorage(storage.Config{BaseDir: tempDir})
					ctx := context.Background()
					err := concurrentStorage.SaveSnapshotsConcurrent(ctx, []*types.Snapshot{snapshot})
					if err != nil {
						b.Fatal(err)
					}
				} else {
					localStorage := storage.NewLocal(tempDir)
					err := localStorage.SaveSnapshot(snapshot)
					if err != nil {
						b.Fatal(err)
					}
				}
				storageTime := time.Since(storageStart)

				// Diff phase (if we have a previous snapshot)
				var diffTime time.Duration
				if i > 0 {
					diffStart := time.Now()

					// Create a modified version
					currentSnapshot := generateModifiedSnapshot(snapshot, 0.2)

					if tt.concurrency == "concurrent" {
						diffWorker := workers.NewDiffWorker(
							workers.WithDiffWorkerCount(runtime.NumCPU()),
							workers.WithComparisonCache(5*time.Minute),
						)
						_, err := diffWorker.ComputeDiffsConcurrent(snapshot, currentSnapshot)
						if err != nil {
							b.Fatal(err)
						}
					} else {
						// Sequential diff
						compareSnapshots(snapshot, currentSnapshot)
					}

					diffTime = time.Since(diffStart)
				}

				totalTime = scanTime + storageTime + diffTime

				// Report detailed metrics
				if b.N == 1 || i == b.N-1 {
					b.ReportMetric(float64(scanTime.Milliseconds()), "scan-ms")
					b.ReportMetric(float64(storageTime.Milliseconds()), "storage-ms")
					b.ReportMetric(float64(diffTime.Milliseconds()), "diff-ms")
					b.ReportMetric(float64(totalTime.Milliseconds()), "total-ms")
					b.ReportMetric(float64(len(snapshot.Resources)), "resources")
				}
			}
		})
	}
}

// BenchmarkMemoryOptimization benchmarks memory optimization effectiveness
func BenchmarkMemoryOptimization(b *testing.B) {
	tests := []struct {
		name            string
		resourceCount   int
		enablePooling   bool
		enableStreaming bool
	}{
		{"10k-no-optimization", 10000, false, false},
		{"10k-with-pooling", 10000, true, false},
		{"10k-with-streaming", 10000, false, true},
		{"10k-full-optimization", 10000, true, true},
		{"50k-no-optimization", 50000, false, false},
		{"50k-full-optimization", 50000, true, true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Create memory optimizer
			memConfig := workers.DefaultMemoryOptimizationConfig()
			memConfig.EnableObjectPooling = tt.enablePooling
			memConfig.StreamingThreshold = 1024 * 1024 // 1MB

			memOptimizer := workers.NewMemoryOptimizer(memConfig)
			ctx := context.Background()
			memOptimizer.Start(ctx)
			defer memOptimizer.Stop()

			// Generate test data
			snapshot := generateLargeSnapshot(tt.resourceCount)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if tt.enablePooling {
					// Use pooled resources
					resources := make([]*types.Resource, tt.resourceCount)
					for j := 0; j < tt.resourceCount; j++ {
						resources[j] = memOptimizer.GetResource()
						*resources[j] = snapshot.Resources[j]
					}

					// Return to pool
					for _, r := range resources {
						memOptimizer.PutResource(r)
					}
				} else {
					// Regular allocation
					resources := make([]types.Resource, tt.resourceCount)
					copy(resources, snapshot.Resources)
				}

				// Get memory stats
				if i == b.N-1 {
					stats := memOptimizer.GetStats()
					b.ReportMetric(float64(stats.HeapMemoryUsage/1024/1024), "heap-mb")
					b.ReportMetric(float64(stats.GoroutineCount), "goroutines")
					b.ReportMetric(float64(stats.ObjectPoolHits), "pool-hits")
					b.ReportMetric(float64(stats.ObjectPoolMisses), "pool-misses")
				}
			}
		})
	}
}

// BenchmarkConcurrentCollectorScaling benchmarks collector scaling
func BenchmarkConcurrentCollectorScaling(b *testing.B) {
	workerCounts := []int{1, 2, 4, 8, 16, 32}
	resourceCount := 10000

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("%d-workers", workers), func(b *testing.B) {
			// Create mock provider with many resources
			provider := &mockProvider{
				name:      "test-provider",
				resources: generateResources(resourceCount),
			}

			cfg := &config.Config{
				Providers: config.ProvidersConfig{
					ConcurrentWorkers: workers,
				},
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				scanner := scanner.NewConcurrentScanner(cfg)

				snapshots, err := scanner.ScanProvider(ctx, provider, workers)
				if err != nil {
					b.Fatal(err)
				}

				if len(snapshots.Resources) != resourceCount {
					b.Fatalf("expected %d resources, got %d", resourceCount, len(snapshots.Resources))
				}
			}

			// Calculate efficiency
			if b.N > 0 {
				opsPerSecond := float64(b.N) / b.Elapsed().Seconds()
				resourcesPerSecond := opsPerSecond * float64(resourceCount)
				b.ReportMetric(resourcesPerSecond, "resources/sec")
				b.ReportMetric(resourcesPerSecond/float64(workers), "resources/sec/worker")
			}
		})
	}
}

// BenchmarkBackpressure benchmarks backpressure handling
func BenchmarkBackpressure(b *testing.B) {
	tests := []struct {
		name              string
		resourceCount     int
		memoryLimit       int64
		backpressureDelay time.Duration
	}{
		{"no-backpressure", 1000, 1024 * 1024 * 1024, 0},                       // 1GB limit
		{"light-backpressure", 5000, 100 * 1024 * 1024, 10 * time.Millisecond}, // 100MB limit
		{"heavy-backpressure", 10000, 50 * 1024 * 1024, 50 * time.Millisecond}, // 50MB limit
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// Create memory optimizer with limits
			memConfig := workers.DefaultMemoryOptimizationConfig()
			memConfig.MaxMemoryUsage = tt.memoryLimit
			memConfig.BackpressureThreshold = tt.memoryLimit * 8 / 10

			memOptimizer := workers.NewMemoryOptimizer(memConfig)
			ctx := context.Background()
			memOptimizer.Start(ctx)
			defer memOptimizer.Stop()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				backpressureEvents := 0

				// Process resources with potential backpressure
				for j := 0; j < tt.resourceCount; j++ {
					// Check for backpressure
					if memOptimizer.IsBackpressureActive() {
						backpressureEvents++

						// Wait for backpressure to clear
						err := memOptimizer.WaitForBackpressure(ctx)
						if err != nil {
							b.Fatal(err)
						}
					}

					// Simulate resource processing
					resource := memOptimizer.GetResource()
					resource.ID = fmt.Sprintf("resource-%d", j)
					resource.Configuration = generateLargeConfig(100)

					// Simulate processing delay
					if tt.backpressureDelay > 0 {
						time.Sleep(tt.backpressureDelay)
					}

					memOptimizer.PutResource(resource)
				}

				// Report backpressure events
				if i == b.N-1 {
					b.ReportMetric(float64(backpressureEvents), "backpressure-events")
				}
			}
		})
	}
}

// BenchmarkStreamingVsFullLoad benchmarks streaming vs full-load processing
func BenchmarkStreamingVsFullLoad(b *testing.B) {
	tests := []struct {
		name          string
		resourceCount int
		fileSize      int64 // in MB
	}{
		{"small-1mb", 1000, 1},
		{"medium-50mb", 50000, 50},
		{"large-200mb", 200000, 200},
	}

	for _, tt := range tests {
		b.Run(tt.name+"-full-load", func(b *testing.B) {
			snapshot := generateLargeSnapshot(tt.resourceCount)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Process all resources at once
				totalSize := 0
				for _, resource := range snapshot.Resources {
					// Simulate processing
					totalSize += len(resource.ID) + len(resource.Name)
					for k, v := range resource.Configuration {
						totalSize += len(k) + len(fmt.Sprintf("%v", v))
					}
				}

				if totalSize == 0 {
					b.Fatal("no data processed")
				}
			}
		})

		b.Run(tt.name+"-streaming", func(b *testing.B) {
			memOptimizer := workers.NewMemoryOptimizer(workers.DefaultMemoryOptimizationConfig())

			// Check if should use streaming
			shouldStream := memOptimizer.ShouldUseStreaming(tt.fileSize * 1024 * 1024)
			if !shouldStream {
				b.Skip("streaming not triggered for this size")
			}

			snapshot := generateLargeSnapshot(tt.resourceCount)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Create streaming context
				streamCtx := memOptimizer.CreateStreamingContext(
					fmt.Sprintf("stream-%d", i),
					tt.fileSize*1024*1024,
				)

				// Process in chunks
				chunkSize := 1000
				totalSize := 0

				for j := 0; j < len(snapshot.Resources); j += chunkSize {
					end := j + chunkSize
					if end > len(snapshot.Resources) {
						end = len(snapshot.Resources)
					}

					chunk := snapshot.Resources[j:end]
					for _, resource := range chunk {
						// Simulate processing
						totalSize += len(resource.ID) + len(resource.Name)
						for k, v := range resource.Configuration {
							totalSize += len(k) + len(fmt.Sprintf("%v", v))
						}
					}

					// Update streaming progress
					streamCtx.ProcessedSize = int64(j * 1024) // Simulate bytes processed
				}

				if totalSize == 0 {
					b.Fatal("no data processed")
				}
			}
		})
	}
}

// Helper functions

type mockProvider struct {
	name      string
	resources []types.Resource
	mu        sync.Mutex
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) GetResources() []types.Resource {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resources
}

func (m *mockProvider) Collect(ctx context.Context, cfg interface{}) (*types.Snapshot, error) {
	return &types.Snapshot{
		ID:        fmt.Sprintf("snapshot-%s-%d", m.name, time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  m.name,
		Resources: m.GetResources(),
	}, nil
}

func createMockProviders(count int, resourcesPerProvider int) []*mockProvider {
	providers := make([]*mockProvider, count)

	for i := 0; i < count; i++ {
		providers[i] = &mockProvider{
			name:      fmt.Sprintf("provider-%d", i),
			resources: generateResources(resourcesPerProvider),
		}
	}

	return providers
}

func generateResources(count int) []types.Resource {
	resources := make([]types.Resource, count)

	for i := 0; i < count; i++ {
		resources[i] = types.Resource{
			ID:       fmt.Sprintf("resource-%d", i),
			Type:     fmt.Sprintf("type-%d", i%10),
			Name:     fmt.Sprintf("Resource %d", i),
			Provider: "test",
			Region:   fmt.Sprintf("region-%d", i%5),
			Configuration: map[string]interface{}{
				"property1": fmt.Sprintf("value-%d", i),
				"property2": i,
				"property3": i%2 == 0,
			},
			Tags: map[string]string{
				"env":  fmt.Sprintf("env-%d", i%3),
				"team": fmt.Sprintf("team-%d", i%7),
			},
		}
	}

	return resources
}

func generateLargeSnapshot(resourceCount int) *types.Snapshot {
	return &types.Snapshot{
		ID:        fmt.Sprintf("large-snapshot-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "test",
		Resources: generateResources(resourceCount),
		Metadata: types.SnapshotMetadata{
			Version: "1.0.0",
			Tags: map[string]string{
				"size": "large",
			},
		},
	}
}

func generateModifiedSnapshot(baseline *types.Snapshot, changePercent float64) *types.Snapshot {
	current := &types.Snapshot{
		ID:        "current-" + baseline.ID,
		Timestamp: time.Now(),
		Provider:  baseline.Provider,
		Resources: make([]types.Resource, 0, len(baseline.Resources)),
		Metadata:  baseline.Metadata,
	}

	changeCount := int(float64(len(baseline.Resources)) * changePercent)

	// Copy and modify resources
	for i, resource := range baseline.Resources {
		if i < changeCount {
			// Modify resource
			modified := resource
			modified.Configuration = map[string]interface{}{
				"property1": fmt.Sprintf("modified-%s", resource.Configuration["property1"]),
				"property2": resource.Configuration["property2"].(int) + 100,
				"property3": !resource.Configuration["property3"].(bool),
			}
			current.Resources = append(current.Resources, modified)
		} else {
			current.Resources = append(current.Resources, resource)
		}
	}

	return current
}

func generateLargeConfig(size int) map[string]interface{} {
	config := make(map[string]interface{})

	for i := 0; i < size; i++ {
		key := fmt.Sprintf("property%d", i)
		config[key] = fmt.Sprintf("value-%d-with-some-extra-data-to-make-it-larger", i)
	}

	return config
}

func mergeSnapshots(snapshots []*types.Snapshot) *types.Snapshot {
	if len(snapshots) == 0 {
		return nil
	}

	merged := &types.Snapshot{
		ID:        fmt.Sprintf("merged-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "multi",
		Resources: []types.Resource{},
		Metadata: types.SnapshotMetadata{
			Tags: map[string]string{},
		},
	}

	for _, snapshot := range snapshots {
		merged.Resources = append(merged.Resources, snapshot.Resources...)
		for k, v := range snapshot.Metadata.Tags {
			merged.Metadata.Tags[k] = v
		}
	}

	return merged
}

func compareSnapshots(baseline, current *types.Snapshot) *types.DriftReport {
	// Simple sequential comparison for benchmark
	report := &types.DriftReport{
		ID:         fmt.Sprintf("report-%d", time.Now().Unix()),
		Timestamp:  time.Now(),
		BaselineID: baseline.ID,
		CurrentID:  current.ID,
		Changes:    []types.Change{},
		Summary:    types.DriftSummary{},
	}

	// Compare resources
	baselineMap := make(map[string]*types.Resource)
	for i := range baseline.Resources {
		baselineMap[baseline.Resources[i].ID] = &baseline.Resources[i]
	}

	for i := range current.Resources {
		if _, exists := baselineMap[current.Resources[i].ID]; !exists {
			report.Summary.AddedResources++
		}
	}

	return report
}
