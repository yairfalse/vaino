package workers_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/workers"
	"github.com/yairfalse/vaino/pkg/types"
)

// BenchmarkDiffWorkerComparison benchmarks diff worker performance
func BenchmarkDiffWorkerComparison(b *testing.B) {
	tests := []struct {
		name          string
		resourceCount int
		workerCount   int
		changePercent float64
	}{
		{"100-resources-4-workers", 100, 4, 0.3},
		{"1000-resources-8-workers", 1000, 8, 0.3},
		{"10000-resources-16-workers", 10000, 16, 0.3},
		{"10000-resources-cpu-workers", 10000, runtime.NumCPU(), 0.3},
	}

	for _, tt := range tests {
		b.Run(tt.name+"-sequential", func(b *testing.B) {
			baseline := generateSnapshot("baseline", tt.resourceCount)
			current := generateModifiedSnapshot(baseline, tt.changePercent)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				report := compareSequential(baseline, current)
				if report == nil {
					b.Fatal("nil report")
				}
			}
		})

		b.Run(tt.name+"-concurrent", func(b *testing.B) {
			baseline := generateSnapshot("baseline", tt.resourceCount)
			current := generateModifiedSnapshot(baseline, tt.changePercent)

			diffWorker := workers.NewDiffWorker(
				workers.WithDiffWorkerCount(tt.workerCount),
				workers.WithDiffBufferSize(100),
				workers.WithDiffTimeout(30*time.Second),
				workers.WithComparisonCache(5*time.Minute),
			)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				report, err := diffWorker.ComputeDiffsConcurrent(baseline, current)
				if err != nil {
					b.Fatal(err)
				}
				if report == nil {
					b.Fatal("nil report")
				}
			}
		})
	}
}

// BenchmarkDiffWorkerScaling benchmarks worker scaling efficiency
func BenchmarkDiffWorkerScaling(b *testing.B) {
	resourceCount := 10000
	baseline := generateSnapshot("baseline", resourceCount)
	current := generateModifiedSnapshot(baseline, 0.3)

	workerCounts := []int{1, 2, 4, 8, 16, 32}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("%d-workers", workerCount), func(b *testing.B) {
			diffWorker := workers.NewDiffWorker(
				workers.WithDiffWorkerCount(workerCount),
				workers.WithDiffBufferSize(100),
				workers.WithComparisonCache(5*time.Minute),
			)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				report, err := diffWorker.ComputeDiffsConcurrent(baseline, current)
				if err != nil {
					b.Fatal(err)
				}
				if report == nil {
					b.Fatal("nil report")
				}
			}
		})
	}
}

// BenchmarkDiffWorkerCache benchmarks cache effectiveness
func BenchmarkDiffWorkerCache(b *testing.B) {
	tests := []struct {
		name      string
		cacheSize time.Duration
	}{
		{"no-cache", 0},
		{"5min-cache", 5 * time.Minute},
		{"10min-cache", 10 * time.Minute},
	}

	resourceCount := 5000
	baseline := generateSnapshot("baseline", resourceCount)
	current := generateModifiedSnapshot(baseline, 0.1) // Only 10% changes

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			var diffWorker *workers.DiffWorker

			if tt.cacheSize > 0 {
				diffWorker = workers.NewDiffWorker(
					workers.WithDiffWorkerCount(8),
					workers.WithComparisonCache(tt.cacheSize),
				)
			} else {
				diffWorker = workers.NewDiffWorker(
					workers.WithDiffWorkerCount(8),
				)
			}

			b.ResetTimer()

			// Run multiple times to test cache hits
			for i := 0; i < b.N; i++ {
				// First comparison (cache miss)
				report1, err := diffWorker.ComputeDiffsConcurrent(baseline, current)
				if err != nil {
					b.Fatal(err)
				}

				// Second comparison (potential cache hit)
				report2, err := diffWorker.ComputeDiffsConcurrent(baseline, current)
				if err != nil {
					b.Fatal(err)
				}

				// Verify results are consistent
				if len(report1.Changes) != len(report2.Changes) {
					b.Fatal("inconsistent results")
				}
			}

			// Report cache statistics
			stats := diffWorker.GetStats()
			b.ReportMetric(float64(stats.TotalCompared), "comparisons")
			b.ReportMetric(float64(stats.TotalChanges), "changes-found")
		})
	}
}

// BenchmarkDiffWorkerMemory benchmarks memory usage
func BenchmarkDiffWorkerMemory(b *testing.B) {
	tests := []struct {
		name          string
		resourceCount int
		workerCount   int
	}{
		{"1k-resources", 1000, 4},
		{"10k-resources", 10000, 8},
		{"50k-resources", 50000, 16},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			baseline := generateSnapshot("baseline", tt.resourceCount)
			current := generateModifiedSnapshot(baseline, 0.3)

			diffWorker := workers.NewDiffWorker(
				workers.WithDiffWorkerCount(tt.workerCount),
				workers.WithDiffBufferSize(100),
			)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				report, err := diffWorker.ComputeDiffsConcurrent(baseline, current)
				if err != nil {
					b.Fatal(err)
				}
				if report == nil {
					b.Fatal("nil report")
				}
			}
		})
	}
}

// BenchmarkDiffWorkerLatency benchmarks operation latency
func BenchmarkDiffWorkerLatency(b *testing.B) {
	resourceCount := 5000
	baseline := generateSnapshot("baseline", resourceCount)
	current := generateModifiedSnapshot(baseline, 0.3)

	diffWorker := workers.NewDiffWorker(
		workers.WithDiffWorkerCount(runtime.NumCPU()),
		workers.WithDiffBufferSize(100),
		workers.WithDiffTimeout(30*time.Second),
	)

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		report, err := diffWorker.ComputeDiffsConcurrent(baseline, current)
		if err != nil {
			b.Fatal(err)
		}
		if report == nil {
			b.Fatal("nil report")
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)
	}

	// Calculate percentiles
	if len(latencies) > 0 {
		p50 := percentile(latencies, 0.5)
		p95 := percentile(latencies, 0.95)
		p99 := percentile(latencies, 0.99)

		b.ReportMetric(float64(p50.Microseconds()), "p50-μs")
		b.ReportMetric(float64(p95.Microseconds()), "p95-μs")
		b.ReportMetric(float64(p99.Microseconds()), "p99-μs")
	}
}

// BenchmarkDiffWorkerParallel benchmarks parallel diff operations
func BenchmarkDiffWorkerParallel(b *testing.B) {
	resourceCount := 1000

	// Create multiple snapshot pairs
	pairs := make([]snapshotPair, 10)
	for i := 0; i < 10; i++ {
		baseline := generateSnapshot(fmt.Sprintf("baseline-%d", i), resourceCount)
		current := generateModifiedSnapshot(baseline, 0.3)
		pairs[i] = snapshotPair{baseline: baseline, current: current}
	}

	diffWorker := workers.NewDiffWorker(
		workers.WithDiffWorkerCount(runtime.NumCPU()),
		workers.WithDiffBufferSize(100),
	)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			pair := pairs[i%len(pairs)]
			i++

			report, err := diffWorker.ComputeDiffsConcurrent(pair.baseline, pair.current)
			if err != nil {
				b.Fatal(err)
			}
			if report == nil {
				b.Fatal("nil report")
			}
		}
	})
}

// BenchmarkResourceComparer benchmarks individual resource comparison
func BenchmarkResourceComparer(b *testing.B) {
	tests := []struct {
		name        string
		configSize  int
		tagCount    int
		changeCount int
	}{
		{"small-config", 10, 5, 3},
		{"medium-config", 50, 20, 10},
		{"large-config", 200, 50, 30},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			baseline := generateComplexResource("baseline", tt.configSize, tt.tagCount)
			current := modifyResource(baseline, tt.changeCount)

			comparer := workers.NewResourceComparer(workers.NewFieldComparer())
			ctx := context.Background()
			options := workers.DiffOptions{
				DeepComparison: true,
				IgnoreMetadata: false,
				CompareTimeout: 1 * time.Second,
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				changes, err := comparer.Compare(ctx, &baseline, &current, options)
				if err != nil {
					b.Fatal(err)
				}
				if len(changes) == 0 {
					b.Fatal("no changes detected")
				}
			}
		})
	}
}

// Helper types and functions

type snapshotPair struct {
	baseline *types.Snapshot
	current  *types.Snapshot
}

func generateSnapshot(id string, resourceCount int) *types.Snapshot {
	resources := make([]types.Resource, resourceCount)

	for i := 0; i < resourceCount; i++ {
		resources[i] = types.Resource{
			ID:       fmt.Sprintf("%s-resource-%d", id, i),
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

	return &types.Snapshot{
		ID:        id,
		Timestamp: time.Now(),
		Provider:  "test",
		Resources: resources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    count,
			Tags: map[string]string{
				"snapshot": id,
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

	// Copy most resources unchanged
	for i, resource := range baseline.Resources {
		if i < len(baseline.Resources)-changeCount {
			current.Resources = append(current.Resources, resource)
		} else {
			// Modify some resources
			modified := resource
			modified.Configuration = map[string]interface{}{
				"property1":   fmt.Sprintf("modified-%s", resource.Configuration["property1"]),
				"property2":   resource.Configuration["property2"].(int) + 100,
				"property3":   !resource.Configuration["property3"].(bool),
				"newProperty": "added",
			}
			current.Resources = append(current.Resources, modified)
		}
	}

	// Add some new resources
	for i := 0; i < changeCount/10; i++ {
		current.Resources = append(current.Resources, types.Resource{
			ID:       fmt.Sprintf("new-resource-%d", i),
			Type:     "new-type",
			Name:     fmt.Sprintf("New Resource %d", i),
			Provider: "test",
			Region:   "new-region",
			Configuration: map[string]interface{}{
				"property1": "new-value",
				"property2": i,
			},
			Tags: map[string]string{
				"new": "true",
			},
		})
	}

	return current
}

func generateComplexResource(id string, configSize, tagCount int) types.Resource {
	config := make(map[string]interface{})
	for i := 0; i < configSize; i++ {
		key := fmt.Sprintf("prop%d", i)
		switch i % 4 {
		case 0:
			config[key] = fmt.Sprintf("value-%d", i)
		case 1:
			config[key] = i
		case 2:
			config[key] = i%2 == 0
		case 3:
			config[key] = map[string]interface{}{
				"nested1": fmt.Sprintf("nested-%d", i),
				"nested2": i * 2,
			}
		}
	}

	tags := make(map[string]string)
	for i := 0; i < tagCount; i++ {
		tags[fmt.Sprintf("tag%d", i)] = fmt.Sprintf("value%d", i)
	}

	return types.Resource{
		ID:            id,
		Type:          "complex-type",
		Name:          "Complex Resource",
		Provider:      "test",
		Region:        "test-region",
		Configuration: config,
		Tags:          tags,
	}
}

func modifyResource(resource types.Resource, changeCount int) types.Resource {
	modified := resource
	modified.Configuration = make(map[string]interface{})

	// Copy most config unchanged
	i := 0
	for k, v := range resource.Configuration {
		if i < changeCount {
			// Modify value
			switch v.(type) {
			case string:
				modified.Configuration[k] = fmt.Sprintf("modified-%s", v)
			case int:
				modified.Configuration[k] = v.(int) + 100
			case bool:
				modified.Configuration[k] = !v.(bool)
			default:
				modified.Configuration[k] = v
			}
		} else {
			modified.Configuration[k] = v
		}
		i++
	}

	// Add new properties
	for j := 0; j < changeCount/2; j++ {
		modified.Configuration[fmt.Sprintf("new-prop%d", j)] = fmt.Sprintf("new-value%d", j)
	}

	return modified
}

func compareSequential(baseline, current *types.Snapshot) *types.DriftReport {
	report := &types.DriftReport{
		ID:         fmt.Sprintf("report-%d", time.Now().Unix()),
		Timestamp:  time.Now(),
		BaselineID: baseline.ID,
		CurrentID:  current.ID,
		Changes:    []types.Change{},
		Summary: types.DriftSummary{
			TotalChanges: 0,
		},
	}

	// Simple sequential comparison
	baselineMap := make(map[string]*types.Resource)
	for i := range baseline.Resources {
		baselineMap[baseline.Resources[i].ID] = &baseline.Resources[i]
	}

	currentMap := make(map[string]*types.Resource)
	for i := range current.Resources {
		currentMap[current.Resources[i].ID] = &current.Resources[i]
	}

	// Find changes
	for id, baselineResource := range baselineMap {
		if currentResource, exists := currentMap[id]; exists {
			// Compare resources
			if baselineResource.Name != currentResource.Name {
				report.Changes = append(report.Changes, types.Change{
					Field:    "name",
					OldValue: baselineResource.Name,
					NewValue: currentResource.Name,
				})
				report.Summary.TotalChanges++
			}
		} else {
			// Resource deleted
			report.Summary.DeletedResources++
			report.Summary.TotalChanges++
		}
	}

	// Find added resources
	for id := range currentMap {
		if _, exists := baselineMap[id]; !exists {
			report.Summary.AddedResources++
			report.Summary.TotalChanges++
		}
	}

	return report
}

func percentile(durations []time.Duration, p float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Simple bubble sort for benchmark
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}
