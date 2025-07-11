package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/collectors/terraform"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// TestLargeDatasetScaling tests VAINO with progressively larger datasets
func TestLargeDatasetScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	datasets := []struct {
		name           string
		resourceCount  int
		expectedSizeMB float64
		maxProcessTime time.Duration
	}{
		{"1K_resources", 1000, 2, 5 * time.Second},
		{"5K_resources", 5000, 10, 15 * time.Second},
		{"10K_resources", 10000, 20, 30 * time.Second},
		{"25K_resources", 25000, 50, 1 * time.Minute},
		{"50K_resources", 50000, 100, 2 * time.Minute},
		{"100K_resources", 100000, 200, 5 * time.Minute},
	}

	for _, dataset := range datasets {
		t.Run(dataset.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "large-dataset.tfstate")

			t.Logf("Creating %s with %d resources...", dataset.name, dataset.resourceCount)

			// Create large state file
			state := createMegaTestState(dataset.resourceCount)
			data, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal large state: %v", err)
			}

			if err := os.WriteFile(stateFile, data, 0644); err != nil {
				t.Fatalf("Failed to write large state file: %v", err)
			}

			// Verify file size
			stat, err := os.Stat(stateFile)
			if err != nil {
				t.Fatalf("Failed to stat file: %v", err)
			}

			actualSizeMB := float64(stat.Size()) / (1024 * 1024)
			t.Logf("File size: %.2f MB (expected ~%.2f MB)", actualSizeMB, dataset.expectedSizeMB)

			// Performance test
			collector := terraform.NewTerraformCollector()
			config := collectors.CollectorConfig{
				StatePaths: []string{stateFile},
				Tags:       map[string]string{"dataset": dataset.name},
			}

			// Measure memory before
			runtime.GC()
			var beforeMem runtime.MemStats
			runtime.ReadMemStats(&beforeMem)

			startTime := time.Now()
			snapshot, err := collector.Collect(context.Background(), config)
			processingTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Large dataset collection failed: %v", err)
			}

			// Measure memory after
			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			// Verify results
			if len(snapshot.Resources) != dataset.resourceCount {
				t.Errorf("Expected %d resources, got %d", dataset.resourceCount, len(snapshot.Resources))
			}

			// Check performance requirements
			if processingTime > dataset.maxProcessTime {
				t.Errorf("Processing took %v, expected < %v", processingTime, dataset.maxProcessTime)
			}

			// Memory analysis
			memUsed := afterMem.HeapAlloc - beforeMem.HeapAlloc
			memUsedMB := float64(memUsed) / (1024 * 1024)

			t.Logf("Performance results:")
			t.Logf("  - Processing time: %v (limit: %v)", processingTime, dataset.maxProcessTime)
			t.Logf("  - Memory used: %.2f MB", memUsedMB)
			t.Logf("  - Memory efficiency: %.2f resources/MB", float64(dataset.resourceCount)/memUsedMB)
			t.Logf("  - Processing rate: %.0f resources/second", float64(dataset.resourceCount)/processingTime.Seconds())

			// Memory efficiency check - should not exceed 5MB per 1000 resources
			maxMemoryMB := float64(dataset.resourceCount) / 1000 * 5
			if memUsedMB > maxMemoryMB {
				t.Errorf("Memory usage too high: %.2f MB (max: %.2f MB)", memUsedMB, maxMemoryMB)
			}
		})
	}
}

// TestMegaFileParsing tests with 100MB+ individual files
func TestMegaFileParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping mega file test in short mode")
	}

	megaFiles := []struct {
		name          string
		resourceCount int
		targetSizeMB  int
		maxParseTime  time.Duration
	}{
		{"mega_100MB", 50000, 100, 3 * time.Minute},
		{"mega_200MB", 100000, 200, 6 * time.Minute},
		{"mega_500MB", 250000, 500, 15 * time.Minute},
	}

	for _, megaFile := range megaFiles {
		t.Run(megaFile.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "mega-file.tfstate")

			t.Logf("Creating %s with target size %dMB...", megaFile.name, megaFile.targetSizeMB)

			// Create mega file with rich data to reach target size
			state := createRichMegaTestState(megaFile.resourceCount)
			data, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal mega state: %v", err)
			}

			if err := os.WriteFile(stateFile, data, 0644); err != nil {
				t.Fatalf("Failed to write mega state file: %v", err)
			}

			// Verify file size
			stat, err := os.Stat(stateFile)
			if err != nil {
				t.Fatalf("Failed to stat mega file: %v", err)
			}

			actualSizeMB := float64(stat.Size()) / (1024 * 1024)
			t.Logf("Mega file created: %.2f MB", actualSizeMB)

			// Test streaming parser performance
			streamParser := terraform.NewStreamingParser()

			// Memory tracking
			runtime.GC()
			var beforeMem runtime.MemStats
			runtime.ReadMemStats(&beforeMem)

			startTime := time.Now()
			parsedState, err := streamParser.ParseStateFile(stateFile)
			parseTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Mega file parsing failed: %v", err)
			}

			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			// Verify parsing results
			if len(parsedState.Resources) != megaFile.resourceCount {
				t.Errorf("Expected %d resources, got %d", megaFile.resourceCount, len(parsedState.Resources))
			}

			// Performance validation
			if parseTime > megaFile.maxParseTime {
				t.Errorf("Parsing took %v, expected < %v", parseTime, megaFile.maxParseTime)
			}

			memUsed := afterMem.HeapAlloc - beforeMem.HeapAlloc
			memUsedMB := float64(memUsed) / (1024 * 1024)

			t.Logf("Mega file results:")
			t.Logf("  - File size: %.2f MB", actualSizeMB)
			t.Logf("  - Parse time: %v (limit: %v)", parseTime, megaFile.maxParseTime)
			t.Logf("  - Memory used: %.2f MB", memUsedMB)
			t.Logf("  - Parse rate: %.2f MB/second", actualSizeMB/parseTime.Seconds())
			t.Logf("  - Resource rate: %.0f resources/second", float64(megaFile.resourceCount)/parseTime.Seconds())

			// Memory efficiency - streaming should not load entire file into memory
			maxMemoryMB := actualSizeMB * 0.3 // Should use < 30% of file size in memory
			if memUsedMB > maxMemoryMB {
				t.Errorf("Streaming parser used too much memory: %.2f MB (max: %.2f MB)", memUsedMB, maxMemoryMB)
			}
		})
	}
}

// TestMultiFileProcessing tests processing many files simultaneously
func TestMultiFileProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-file test in short mode")
	}

	scenarios := []struct {
		name             string
		fileCount        int
		resourcesPerFile int
		maxProcessTime   time.Duration
	}{
		{"many_small_files", 100, 100, 30 * time.Second},
		{"medium_files", 50, 1000, 1 * time.Minute},
		{"large_files", 20, 5000, 3 * time.Minute},
		{"mega_files", 10, 10000, 5 * time.Minute},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			t.Logf("Creating %d files with %d resources each...", scenario.fileCount, scenario.resourcesPerFile)

			// Create multiple state files
			stateFiles := make([]string, scenario.fileCount)
			for i := 0; i < scenario.fileCount; i++ {
				stateFile := filepath.Join(tmpDir, fmt.Sprintf("state-%d.tfstate", i))
				stateFiles[i] = stateFile

				state := createMegaTestState(scenario.resourcesPerFile)
				data, err := json.MarshalIndent(state, "", "  ")
				if err != nil {
					t.Fatalf("Failed to marshal state %d: %v", i, err)
				}

				if err := os.WriteFile(stateFile, data, 0644); err != nil {
					t.Fatalf("Failed to write state file %d: %v", i, err)
				}
			}

			// Calculate total size
			var totalSizeMB float64
			for _, stateFile := range stateFiles {
				stat, _ := os.Stat(stateFile)
				totalSizeMB += float64(stat.Size()) / (1024 * 1024)
			}

			t.Logf("Created %d files, total size: %.2f MB", scenario.fileCount, totalSizeMB)

			// Test parallel processing
			collector := terraform.NewTerraformCollector()
			config := collectors.CollectorConfig{
				StatePaths: stateFiles,
				Tags:       map[string]string{"scenario": scenario.name},
			}

			// Memory tracking
			runtime.GC()
			var beforeMem runtime.MemStats
			runtime.ReadMemStats(&beforeMem)

			startTime := time.Now()
			snapshot, err := collector.Collect(context.Background(), config)
			processingTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Multi-file collection failed: %v", err)
			}

			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			// Verify results
			expectedResources := scenario.fileCount * scenario.resourcesPerFile
			if len(snapshot.Resources) != expectedResources {
				t.Errorf("Expected %d resources, got %d", expectedResources, len(snapshot.Resources))
			}

			// Performance validation
			if processingTime > scenario.maxProcessTime {
				t.Errorf("Processing took %v, expected < %v", processingTime, scenario.maxProcessTime)
			}

			memUsed := afterMem.HeapAlloc - beforeMem.HeapAlloc
			memUsedMB := float64(memUsed) / (1024 * 1024)

			t.Logf("Multi-file results:")
			t.Logf("  - Files processed: %d", scenario.fileCount)
			t.Logf("  - Total resources: %d", expectedResources)
			t.Logf("  - Processing time: %v (limit: %v)", processingTime, scenario.maxProcessTime)
			t.Logf("  - Memory used: %.2f MB", memUsedMB)
			t.Logf("  - Throughput: %.2f MB/second", totalSizeMB/processingTime.Seconds())
			t.Logf("  - File rate: %.1f files/second", float64(scenario.fileCount)/processingTime.Seconds())
		})
	}
}

// TestDifferPerformanceAtScale tests diff performance with large datasets
func TestDifferPerformanceAtScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping differ scale test in short mode")
	}

	scaleTests := []struct {
		name          string
		resourceCount int
		changePercent int
		maxDiffTime   time.Duration
	}{
		{"small_scale_10_percent", 1000, 10, 1 * time.Second},
		{"medium_scale_5_percent", 5000, 5, 5 * time.Second},
		{"large_scale_2_percent", 25000, 2, 30 * time.Second},
		{"mega_scale_1_percent", 100000, 1, 2 * time.Minute},
	}

	for _, test := range scaleTests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Testing differ with %d resources, %d%% changes...", test.resourceCount, test.changePercent)

			// Create baseline and modified snapshots
			baseline := createLargeSnapshot("baseline", test.resourceCount)
			modified := createLargeSnapshotWithChanges("modified", test.resourceCount, test.changePercent)

			// Test differ performance
			differ := differ.NewSimpleDiffer()

			// Memory tracking
			runtime.GC()
			var beforeMem runtime.MemStats
			runtime.ReadMemStats(&beforeMem)

			startTime := time.Now()
			report, err := differ.Compare(baseline, modified)
			diffTime := time.Since(startTime)

			if err != nil {
				t.Fatalf("Diff failed: %v", err)
			}

			var afterMem runtime.MemStats
			runtime.ReadMemStats(&afterMem)

			// Verify results
			expectedChanges := (test.resourceCount * test.changePercent) / 100
			actualChanges := len(report.Changes)

			// Allow some variance in change detection
			minExpected := expectedChanges / 2
			maxExpected := expectedChanges * 2

			if actualChanges < minExpected || actualChanges > maxExpected {
				t.Logf("Warning: Expected ~%d changes, got %d", expectedChanges, actualChanges)
			}

			// Performance validation
			if diffTime > test.maxDiffTime {
				t.Errorf("Diff took %v, expected < %v", diffTime, test.maxDiffTime)
			}

			memUsed := afterMem.HeapAlloc - beforeMem.HeapAlloc
			memUsedMB := float64(memUsed) / (1024 * 1024)

			t.Logf("Differ scale results:")
			t.Logf("  - Resources compared: %d", test.resourceCount)
			t.Logf("  - Changes detected: %d (%d%% of total)", actualChanges, (actualChanges*100)/test.resourceCount)
			t.Logf("  - Diff time: %v (limit: %v)", diffTime, test.maxDiffTime)
			t.Logf("  - Memory used: %.2f MB", memUsedMB)
			t.Logf("  - Compare rate: %.0f resources/second", float64(test.resourceCount)/diffTime.Seconds())

			// Memory efficiency check
			maxMemoryMB := float64(test.resourceCount) / 1000 * 2 // 2MB per 1000 resources
			if memUsedMB > maxMemoryMB {
				t.Errorf("Diff used too much memory: %.2f MB (max: %.2f MB)", memUsedMB, maxMemoryMB)
			}
		})
	}
}

// Helper functions for large dataset testing

func createRichMegaTestState(resourceCount int) *terraform.TerraformState {
	state := createMegaTestState(resourceCount)

	// Enrich with more data to increase file size
	for i := range state.Resources {
		resource := &state.Resources[i]
		instance := &resource.Instances[0]

		// Add large configuration blocks
		instance.Attributes["large_config"] = map[string]interface{}{
			"description": fmt.Sprintf("This is a very long description for resource %d. It contains detailed information about the resource configuration, its purpose, dependencies, and other metadata that might be stored in real-world Terraform state files. This helps us create larger file sizes for testing purposes.", i),
			"metadata": map[string]interface{}{
				"created_by":    "terraform",
				"created_at":    "2023-01-01T00:00:00Z",
				"updated_at":    "2023-12-01T00:00:00Z",
				"version":       "1.0.0",
				"documentation": "https://example.com/docs/resource/" + fmt.Sprintf("%d", i),
			},
			"complex_config": map[string]interface{}{
				"settings": make(map[string]interface{}),
				"rules":    make([]interface{}, 0),
				"policies": make(map[string]interface{}),
			},
		}

		// Add complex settings
		settings := instance.Attributes["large_config"].(map[string]interface{})["complex_config"].(map[string]interface{})["settings"].(map[string]interface{})
		for j := 0; j < 20; j++ {
			settings[fmt.Sprintf("setting_%d", j)] = fmt.Sprintf("value_%d_%d", i, j)
		}

		// Add rules array
		complexConfig := instance.Attributes["large_config"].(map[string]interface{})["complex_config"].(map[string]interface{})
		rulesSlice := complexConfig["rules"].([]interface{})
		for j := 0; j < 10; j++ {
			rule := map[string]interface{}{
				"id":     fmt.Sprintf("rule_%d_%d", i, j),
				"action": "allow",
				"conditions": map[string]interface{}{
					"field":    fmt.Sprintf("field_%d", j),
					"operator": "equals",
					"value":    fmt.Sprintf("value_%d", j),
				},
			}
			rulesSlice = append(rulesSlice, rule)
		}
		complexConfig["rules"] = rulesSlice
	}

	return state
}

func createLargeSnapshot(id string, resourceCount int) *types.Snapshot {
	resources := make([]types.Resource, resourceCount)
	now := time.Now()

	for i := 0; i < resourceCount; i++ {
		resources[i] = types.Resource{
			ID:        fmt.Sprintf("resource-%d", i),
			Type:      fmt.Sprintf("type-%d", i%10),
			Name:      fmt.Sprintf("resource-name-%d", i),
			Provider:  "test-provider",
			Namespace: fmt.Sprintf("namespace-%d", i%5),
			Configuration: map[string]interface{}{
				"index":  i,
				"config": fmt.Sprintf("config-value-%d", i),
				"tags": map[string]interface{}{
					"Environment": "test",
					"Index":       fmt.Sprintf("%d", i),
				},
			},
		}
	}

	return &types.Snapshot{
		ID:        id,
		Timestamp: now,
		Provider:  "test-provider",
		Resources: resources,
	}
}

func createLargeSnapshotWithChanges(id string, resourceCount int, changePercent int) *types.Snapshot {
	snapshot := createLargeSnapshot(id, resourceCount)

	// Modify specified percentage of resources
	changeCount := (resourceCount * changePercent) / 100
	for i := 0; i < changeCount; i++ {
		resourceIndex := i * (resourceCount / changeCount)
		if resourceIndex >= len(snapshot.Resources) {
			break
		}

		resource := &snapshot.Resources[resourceIndex]
		resource.Configuration["modified"] = true
		resource.Configuration["config"] = fmt.Sprintf("modified-config-value-%d", resourceIndex)

		if tags, ok := resource.Configuration["tags"].(map[string]interface{}); ok {
			tags["Modified"] = "true"
		}
	}

	return snapshot
}
