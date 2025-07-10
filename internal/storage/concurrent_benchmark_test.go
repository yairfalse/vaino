package storage_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/types"
)

// BenchmarkConcurrentStorageList benchmarks concurrent vs sequential snapshot listing
func BenchmarkConcurrentStorageList(b *testing.B) {
	tests := []struct {
		name          string
		snapshotCount int
	}{
		{"10-snapshots", 10},
		{"100-snapshots", 100},
		{"1000-snapshots", 1000},
		{"10000-snapshots", 10000},
	}

	for _, tt := range tests {
		b.Run(tt.name+"-sequential", func(b *testing.B) {
			// Setup
			tempDir := b.TempDir()
			localStorage := storage.NewLocal(tempDir)

			// Create test snapshots
			createTestSnapshots(b, localStorage, tt.snapshotCount)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := localStorage.ListSnapshots()
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(tt.name+"-concurrent", func(b *testing.B) {
			// Setup
			tempDir := b.TempDir()
			config := storage.Config{BaseDir: tempDir}
			concurrentStorage, err := storage.NewConcurrentStorage(config)
			if err != nil {
				b.Fatal(err)
			}

			localStorage := storage.NewLocal(tempDir)
			createTestSnapshots(b, localStorage, tt.snapshotCount)

			ctx := context.Background()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := concurrentStorage.ListSnapshotsConcurrent(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkConcurrentStorageSave benchmarks concurrent vs sequential snapshot saving
func BenchmarkConcurrentStorageSave(b *testing.B) {
	tests := []struct {
		name          string
		snapshotCount int
		resourceCount int
	}{
		{"10-snapshots-100-resources", 10, 100},
		{"50-snapshots-500-resources", 50, 500},
		{"100-snapshots-1000-resources", 100, 1000},
	}

	for _, tt := range tests {
		b.Run(tt.name+"-sequential", func(b *testing.B) {
			tempDir := b.TempDir()
			localStorage := storage.NewLocal(tempDir)

			snapshots := generateTestSnapshots(tt.snapshotCount, tt.resourceCount)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for _, snapshot := range snapshots {
					if err := localStorage.SaveSnapshot(snapshot); err != nil {
						b.Fatal(err)
					}
				}
			}
		})

		b.Run(tt.name+"-concurrent", func(b *testing.B) {
			tempDir := b.TempDir()
			config := storage.Config{BaseDir: tempDir}
			concurrentStorage, err := storage.NewConcurrentStorage(config)
			if err != nil {
				b.Fatal(err)
			}

			snapshots := generateTestSnapshots(tt.snapshotCount, tt.resourceCount)
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := concurrentStorage.SaveSnapshotsConcurrent(ctx, snapshots); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkConcurrentStorageLoad benchmarks concurrent vs sequential snapshot loading
func BenchmarkConcurrentStorageLoad(b *testing.B) {
	tests := []struct {
		name          string
		snapshotCount int
		loadCount     int
	}{
		{"load-10-of-100", 100, 10},
		{"load-50-of-500", 500, 50},
		{"load-100-of-1000", 1000, 100},
	}

	for _, tt := range tests {
		b.Run(tt.name+"-sequential", func(b *testing.B) {
			tempDir := b.TempDir()
			localStorage := storage.NewLocal(tempDir)

			// Create test snapshots and collect IDs
			ids := createTestSnapshotsWithIDs(b, localStorage, tt.snapshotCount)
			loadIDs := ids[:tt.loadCount]

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				for _, id := range loadIDs {
					_, err := localStorage.LoadSnapshot(id)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})

		b.Run(tt.name+"-concurrent", func(b *testing.B) {
			tempDir := b.TempDir()
			config := storage.Config{BaseDir: tempDir}
			concurrentStorage, err := storage.NewConcurrentStorage(config)
			if err != nil {
				b.Fatal(err)
			}

			localStorage := storage.NewLocal(tempDir)
			ids := createTestSnapshotsWithIDs(b, localStorage, tt.snapshotCount)
			loadIDs := ids[:tt.loadCount]

			ctx := context.Background()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := concurrentStorage.LoadSnapshotsConcurrent(ctx, loadIDs)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkStreamingProcessor benchmarks streaming vs full-load processing
func BenchmarkStreamingProcessor(b *testing.B) {
	tests := []struct {
		name          string
		resourceCount int
		fileSize      string
	}{
		{"1k-resources", 1000, "small"},
		{"10k-resources", 10000, "medium"},
		{"100k-resources", 100000, "large"},
	}

	for _, tt := range tests {
		b.Run(tt.name+"-full-load", func(b *testing.B) {
			tempDir := b.TempDir()
			localStorage := storage.NewLocal(tempDir)

			// Create large snapshot
			snapshot := generateLargeSnapshot(tt.resourceCount)
			if err := localStorage.SaveSnapshot(snapshot); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Load full snapshot
				loaded, err := localStorage.LoadSnapshot(snapshot.ID)
				if err != nil {
					b.Fatal(err)
				}

				// Process all resources
				count := 0
				for range loaded.Resources {
					count++
				}

				if count != tt.resourceCount {
					b.Fatalf("expected %d resources, got %d", tt.resourceCount, count)
				}
			}
		})

		b.Run(tt.name+"-streaming", func(b *testing.B) {
			tempDir := b.TempDir()
			config := storage.Config{BaseDir: tempDir}
			concurrentStorage, err := storage.NewConcurrentStorage(config)
			if err != nil {
				b.Fatal(err)
			}

			localStorage := storage.NewLocal(tempDir)
			snapshot := generateLargeSnapshot(tt.resourceCount)
			if err := localStorage.SaveSnapshot(snapshot); err != nil {
				b.Fatal(err)
			}

			// Get snapshot file path
			infos, _ := localStorage.ListSnapshots()
			var filePath string
			for _, info := range infos {
				if info.ID == snapshot.ID {
					filePath = info.FilePath
					break
				}
			}

			processor := storage.NewStreamingProcessor(concurrentStorage)
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				count := 0
				err := processor.ProcessLargeSnapshot(ctx, filePath, func(r *types.Resource) error {
					count++
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}

				if count != tt.resourceCount {
					b.Fatalf("expected %d resources, got %d", tt.resourceCount, count)
				}
			}
		})
	}
}

// BenchmarkConcurrentStorageMemory benchmarks memory usage
func BenchmarkConcurrentStorageMemory(b *testing.B) {
	b.Run("memory-10k-resources", func(b *testing.B) {
		tempDir := b.TempDir()
		config := storage.Config{BaseDir: tempDir}
		concurrentStorage, err := storage.NewConcurrentStorage(config)
		if err != nil {
			b.Fatal(err)
		}

		// Generate 10 snapshots with 1000 resources each
		snapshots := generateTestSnapshots(10, 1000)
		ctx := context.Background()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Save snapshots
			if err := concurrentStorage.SaveSnapshotsConcurrent(ctx, snapshots); err != nil {
				b.Fatal(err)
			}

			// List snapshots
			infos, err := concurrentStorage.ListSnapshotsConcurrent(ctx)
			if err != nil {
				b.Fatal(err)
			}

			// Load random snapshots
			if len(infos) > 0 {
				ids := []string{infos[0].ID}
				if len(infos) > 1 {
					ids = append(ids, infos[len(infos)/2].ID)
				}
				if len(infos) > 2 {
					ids = append(ids, infos[len(infos)-1].ID)
				}

				_, err := concurrentStorage.LoadSnapshotsConcurrent(ctx, ids)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

// BenchmarkConcurrentStorageParallel benchmarks parallel operations
func BenchmarkConcurrentStorageParallel(b *testing.B) {
	tempDir := b.TempDir()
	config := storage.Config{BaseDir: tempDir}
	concurrentStorage, err := storage.NewConcurrentStorage(config)
	if err != nil {
		b.Fatal(err)
	}

	// Pre-create some snapshots
	localStorage := storage.NewLocal(tempDir)
	ids := createTestSnapshotsWithIDs(b, localStorage, 100)

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of operations
			switch rand.Intn(3) {
			case 0:
				// List operation
				_, err := concurrentStorage.ListSnapshotsConcurrent(ctx)
				if err != nil {
					b.Fatal(err)
				}
			case 1:
				// Load operation
				loadIDs := []string{ids[rand.Intn(len(ids))]}
				_, err := concurrentStorage.LoadSnapshotsConcurrent(ctx, loadIDs)
				if err != nil {
					b.Fatal(err)
				}
			case 2:
				// Save operation
				snapshots := generateTestSnapshots(1, 100)
				err := concurrentStorage.SaveSnapshotsConcurrent(ctx, snapshots)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

// Helper functions

func createTestSnapshots(b *testing.B, storage storage.Storage, count int) {
	b.Helper()

	for i := 0; i < count; i++ {
		snapshot := &types.Snapshot{
			ID:        fmt.Sprintf("snapshot-%d", i),
			Timestamp: time.Now().Add(time.Duration(-i) * time.Hour),
			Provider:  "test",
			Resources: generateResources(100),
			Metadata: types.SnapshotMetadata{
				Tags: map[string]string{
					"test":  "true",
					"index": fmt.Sprintf("%d", i),
				},
			},
		}

		if err := storage.SaveSnapshot(snapshot); err != nil {
			b.Fatal(err)
		}
	}
}

func createTestSnapshotsWithIDs(b *testing.B, storage storage.Storage, count int) []string {
	b.Helper()

	ids := make([]string, count)
	for i := 0; i < count; i++ {
		snapshot := &types.Snapshot{
			ID:        fmt.Sprintf("snapshot-%d-%d", time.Now().Unix(), i),
			Timestamp: time.Now().Add(time.Duration(-i) * time.Hour),
			Provider:  "test",
			Resources: generateResources(100),
			Metadata: types.SnapshotMetadata{
				Tags: map[string]string{
					"test":  "true",
					"index": fmt.Sprintf("%d", i),
				},
			},
		}

		ids[i] = snapshot.ID

		if err := storage.SaveSnapshot(snapshot); err != nil {
			b.Fatal(err)
		}
	}

	return ids
}

func generateTestSnapshots(count, resourceCount int) []*types.Snapshot {
	snapshots := make([]*types.Snapshot, count)

	for i := 0; i < count; i++ {
		snapshots[i] = &types.Snapshot{
			ID:        fmt.Sprintf("snapshot-%d-%d", time.Now().Unix(), i),
			Timestamp: time.Now().Add(time.Duration(-i) * time.Hour),
			Provider:  "test",
			Resources: generateResources(resourceCount),
			Metadata: types.SnapshotMetadata{
				Tags: map[string]string{
					"test":  "true",
					"index": fmt.Sprintf("%d", i),
				},
			},
		}
	}

	return snapshots
}

func generateLargeSnapshot(resourceCount int) *types.Snapshot {
	return &types.Snapshot{
		ID:        fmt.Sprintf("large-snapshot-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "test",
		Resources: generateResources(resourceCount),
		Metadata: types.SnapshotMetadata{
			Tags: map[string]string{
				"test": "true",
				"size": "large",
			},
		},
	}
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
				"nested": map[string]interface{}{
					"subprop1": fmt.Sprintf("sub-%d", i),
					"subprop2": float64(i) * 1.5,
				},
			},
			Tags: map[string]string{
				"env":   fmt.Sprintf("env-%d", i%3),
				"team":  fmt.Sprintf("team-%d", i%7),
				"index": fmt.Sprintf("%d", i),
			},
		}
	}

	return resources
}

// BenchmarkBufferPool benchmarks buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	b.Run("without-pool", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Allocate new buffer each time
			buf := make([]byte, 64*1024)
			// Simulate some work
			for j := 0; j < 100; j++ {
				buf[j] = byte(j)
			}
		}
	})

	b.Run("with-pool", func(b *testing.B) {
		pool := sync.Pool{
			New: func() interface{} {
				return make([]byte, 64*1024)
			},
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Get buffer from pool
			buf := pool.Get().([]byte)
			// Simulate some work
			for j := 0; j < 100; j++ {
				buf[j] = byte(j)
			}
			// Return to pool
			pool.Put(buf)
		}
	})
}

// BenchmarkMetrics tracks performance metrics
func BenchmarkMetrics(b *testing.B) {
	tempDir := b.TempDir()
	config := storage.Config{BaseDir: tempDir}
	concurrentStorage, err := storage.NewConcurrentStorage(config)
	if err != nil {
		b.Fatal(err)
	}

	// Create test data
	localStorage := storage.NewLocal(tempDir)
	createTestSnapshots(b, localStorage, 100)

	ctx := context.Background()

	// Get initial metrics
	initialMetrics := storage.GetMetrics()

	b.ResetTimer()

	// Run operations
	for i := 0; i < b.N; i++ {
		// List snapshots
		_, err := concurrentStorage.ListSnapshotsConcurrent(ctx)
		if err != nil {
			b.Fatal(err)
		}

		// Save new snapshot
		snapshots := generateTestSnapshots(1, 100)
		err = concurrentStorage.SaveSnapshotsConcurrent(ctx, snapshots)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Get final metrics
	finalMetrics := storage.GetMetrics()

	// Report metrics (in real implementation, these would be actual values)
	b.ReportMetric(float64(finalMetrics.ConcurrentReads.Load()-initialMetrics.ConcurrentReads.Load())/float64(b.N), "reads/op")
	b.ReportMetric(float64(finalMetrics.ConcurrentWrites.Load()-initialMetrics.ConcurrentWrites.Load())/float64(b.N), "writes/op")
}
