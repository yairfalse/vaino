package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

// BenchmarkMegaFileProcessing tests WGO with 100MB+ files
func BenchmarkMegaFileProcessing(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping mega file benchmark in short mode")
	}

	sizes := []struct {
		name      string
		resources int
		fileSizeMB float64
	}{
		{"10MB_file", 5000, 10},
		{"50MB_file", 25000, 50}, 
		{"100MB_file", 50000, 100},
		{"200MB_file", 100000, 200},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			tmpDir := b.TempDir()
			stateFile := filepath.Join(tmpDir, "mega-state.tfstate")
			
			// Create massive state file
			state := createMegaTestState(size.resources)
			data, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				b.Fatalf("Failed to marshal mega state: %v", err)
			}

			if err := os.WriteFile(stateFile, data, 0644); err != nil {
				b.Fatalf("Failed to write mega state file: %v", err)
			}

			// Verify file size
			stat, _ := os.Stat(stateFile)
			actualSizeMB := float64(stat.Size()) / (1024 * 1024)
			b.Logf("Created file: %.2f MB with %d resources", actualSizeMB, size.resources)

			collector := terraform.NewTerraformCollector()
			config := collectors.CollectorConfig{
				StatePaths: []string{stateFile},
				Tags:       map[string]string{"environment": "mega-test"},
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				snapshot, err := collector.Collect(context.Background(), config)
				if err != nil {
					b.Fatalf("Mega file collection failed: %v", err)
				}
				
				if len(snapshot.Resources) != size.resources {
					b.Errorf("Expected %d resources, got %d", size.resources, len(snapshot.Resources))
				}
			}
		})
	}
}

// BenchmarkConcurrentOperations tests multiple simultaneous scans
func BenchmarkConcurrentOperations(b *testing.B) {
	concurrencyLevels := []int{2, 4, 8, 16}
	resourceCounts := []int{100, 500, 1000}

	for _, concurrency := range concurrencyLevels {
		for _, resourceCount := range resourceCounts {
			b.Run(fmt.Sprintf("concurrent_%d_resources_%d", concurrency, resourceCount), func(b *testing.B) {
				// Setup test files for each concurrent operation
				tmpDir := b.TempDir()
				stateFiles := make([]string, concurrency)
				
				for i := 0; i < concurrency; i++ {
					stateFile := filepath.Join(tmpDir, fmt.Sprintf("state-%d.tfstate", i))
					stateFiles[i] = stateFile
					
					state := createMegaTestState(resourceCount)
					data, err := json.MarshalIndent(state, "", "  ")
					if err != nil {
						b.Fatalf("Failed to marshal state %d: %v", i, err)
					}
					
					if err := os.WriteFile(stateFile, data, 0644); err != nil {
						b.Fatalf("Failed to write state file %d: %v", i, err)
					}
				}

				b.ResetTimer()
				b.ReportAllocs()

				for i := 0; i < b.N; i++ {
					var wg sync.WaitGroup
					errors := make(chan error, concurrency)
					results := make([]types.Snapshot, concurrency)

					startTime := time.Now()

					for j := 0; j < concurrency; j++ {
						wg.Add(1)
						go func(idx int) {
							defer wg.Done()
							
							collector := terraform.NewTerraformCollector()
							config := collectors.CollectorConfig{
								StatePaths: []string{stateFiles[idx]},
								Tags:       map[string]string{"worker": fmt.Sprintf("%d", idx)},
							}

							snapshot, err := collector.Collect(context.Background(), config)
							if err != nil {
								errors <- fmt.Errorf("worker %d failed: %v", idx, err)
								return
							}
							
							results[idx] = *snapshot
						}(j)
					}

					wg.Wait()
					close(errors)

					duration := time.Since(startTime)

					// Check for errors
					for err := range errors {
						b.Fatalf("Concurrent operation failed: %v", err)
					}

					// Verify all operations completed successfully
					totalResources := 0
					for _, result := range results {
						totalResources += len(result.Resources)
					}

					expectedTotal := concurrency * resourceCount
					if totalResources != expectedTotal {
						b.Errorf("Expected %d total resources, got %d", expectedTotal, totalResources)
					}

					b.Logf("Concurrent level %d: processed %d resources in %v", 
						concurrency, totalResources, duration)
				}
			})
		}
	}
}

// BenchmarkStorageOperations tests storage performance
func BenchmarkStorageOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping storage benchmark in short mode")
	}

	tmpDir := b.TempDir()
	storageConfig := storage.Config{BaseDir: tmpDir}
	storageEngine, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		b.Fatalf("Failed to create storage engine: %v", err)
	}

	// Create test snapshots
	snapshots := make([]*types.Snapshot, 10)
	for i := 0; i < 10; i++ {
		snapshots[i] = createLargeSnapshot(fmt.Sprintf("storage-test-%d", i), 1000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Store snapshots
		for j, snapshot := range snapshots {
			err := storageEngine.SaveSnapshot(snapshot)
			if err != nil {
				b.Fatalf("Failed to store snapshot %d: %v", j, err)
			}
		}

		// Load snapshots
		for j, snapshot := range snapshots {
			_, err := storageEngine.LoadSnapshot(snapshot.ID)
			if err != nil {
				b.Fatalf("Failed to load snapshot %d: %v", j, err)
			}
		}
	}
}

// BenchmarkMemoryIntensiveOperations tests memory usage patterns
func BenchmarkMemoryIntensiveOperations(b *testing.B) {
	resourceCounts := []int{1000, 5000, 10000, 25000}

	for _, resourceCount := range resourceCounts {
		b.Run(fmt.Sprintf("memory_test_%d_resources", resourceCount), func(b *testing.B) {
			tmpDir := b.TempDir()
			
			// Create large state files
			stateFiles := make([]string, 5) // 5 large files
			for i := 0; i < 5; i++ {
				stateFile := filepath.Join(tmpDir, fmt.Sprintf("memory-state-%d.tfstate", i))
				stateFiles[i] = stateFile
				
				state := createMegaTestState(resourceCount)
				data, err := json.MarshalIndent(state, "", "  ")
				if err != nil {
					b.Fatalf("Failed to marshal state %d: %v", i, err)
				}
				
				if err := os.WriteFile(stateFile, data, 0644); err != nil {
					b.Fatalf("Failed to write state file %d: %v", i, err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Force GC before measurement
				runtime.GC()
				runtime.GC()

				var beforeMem runtime.MemStats
				runtime.ReadMemStats(&beforeMem)

				// Collect all files
				collector := terraform.NewTerraformCollector()
				config := collectors.CollectorConfig{
					StatePaths: stateFiles,
					Tags:       map[string]string{"memory-test": "true"},
				}

				snapshots := make([]*types.Snapshot, len(stateFiles))
				for j, statePath := range stateFiles {
					singleConfig := collectors.CollectorConfig{
						StatePaths: []string{statePath},
						Tags:       config.Tags,
					}
					
					snapshot, err := collector.Collect(context.Background(), singleConfig)
					if err != nil {
						b.Fatalf("Collection failed for file %d: %v", j, err)
					}
					snapshots[j] = snapshot
				}

				// Perform diffs between snapshots
				differ := differ.NewSimpleDiffer()
				for j := 0; j < len(snapshots)-1; j++ {
					_, err := differ.Compare(snapshots[j], snapshots[j+1])
					if err != nil {
						b.Fatalf("Diff failed between snapshots %d and %d: %v", j, j+1, err)
					}
				}

				var afterMem runtime.MemStats
				runtime.ReadMemStats(&afterMem)

				memUsed := afterMem.HeapAlloc - beforeMem.HeapAlloc
				totalResources := resourceCount * 5

				b.Logf("Memory test %d resources: used %d bytes (%.2f MB)", 
					totalResources, memUsed, float64(memUsed)/(1024*1024))

				// Memory usage should be reasonable (< 100MB for 125k resources)
				maxMemoryMB := 100.0
				if totalResources > 100000 {
					maxMemoryMB = 200.0 // Allow more for very large datasets
				}

				memUsedMB := float64(memUsed) / (1024 * 1024)
				if memUsedMB > maxMemoryMB {
					b.Errorf("Memory usage too high: %.2f MB (max: %.2f MB)", memUsedMB, maxMemoryMB)
				}
			}
		})
	}
}

// BenchmarkEndToEndWorkflow tests complete WGO workflow performance
func BenchmarkEndToEndWorkflow(b *testing.B) {
	workflows := []struct {
		name          string
		resourceCount int
		changePercent int
	}{
		{"small_workflow", 100, 10},
		{"medium_workflow", 1000, 5},
		{"large_workflow", 5000, 2},
		{"mega_workflow", 10000, 1},
	}

	for _, workflow := range workflows {
		b.Run(workflow.name, func(b *testing.B) {
			tmpDir := b.TempDir()
			
			// Create baseline and current state files
			baselineFile := filepath.Join(tmpDir, "baseline.tfstate")
			currentFile := filepath.Join(tmpDir, "current.tfstate")
			
			baselineState := createMegaTestState(workflow.resourceCount)
			currentState := createMegaTestStateWithChanges(workflow.resourceCount, workflow.changePercent)
			
			baselineData, _ := json.MarshalIndent(baselineState, "", "  ")
			currentData, _ := json.MarshalIndent(currentState, "", "  ")
			
			os.WriteFile(baselineFile, baselineData, 0644)
			os.WriteFile(currentFile, currentData, 0644)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Complete workflow: collect -> store -> diff -> analyze
				
				// 1. Collect baseline
				collector := terraform.NewTerraformCollector()
				baselineConfig := collectors.CollectorConfig{
					StatePaths: []string{baselineFile},
					Tags:       map[string]string{"snapshot": "baseline"},
				}
				
				baselineSnapshot, err := collector.Collect(context.Background(), baselineConfig)
				if err != nil {
					b.Fatalf("Baseline collection failed: %v", err)
				}

				// 2. Collect current
				currentConfig := collectors.CollectorConfig{
					StatePaths: []string{currentFile},
					Tags:       map[string]string{"snapshot": "current"},
				}
				
				currentSnapshot, err := collector.Collect(context.Background(), currentConfig)
				if err != nil {
					b.Fatalf("Current collection failed: %v", err)
				}

				// 3. Store snapshots
				storageConfig := storage.Config{BaseDir: tmpDir}
	storageEngine, err := storage.NewLocalStorage(storageConfig)
	if err != nil {
		b.Fatalf("Failed to create storage engine: %v", err)
	}
				
				err = storageEngine.SaveSnapshot(baselineSnapshot)
				if err != nil {
					b.Fatalf("Baseline storage failed: %v", err)
				}
				
				err = storageEngine.SaveSnapshot(currentSnapshot)
				if err != nil {
					b.Fatalf("Current storage failed: %v", err)
				}

				// 4. Perform diff
				differ := differ.NewSimpleDiffer()
				report, err := differ.Compare(baselineSnapshot, currentSnapshot)
				if err != nil {
					b.Fatalf("Diff failed: %v", err)
				}

				// 5. Verify results
				expectedChanges := (workflow.resourceCount * workflow.changePercent) / 100
				changeCount := len(report.Changes)
				
				if changeCount < expectedChanges/2 || changeCount > expectedChanges*2 {
					b.Logf("Warning: Expected ~%d changes, got %d", expectedChanges, changeCount)
				}

				b.Logf("End-to-end %s: %d resources, %d changes", 
					workflow.name, workflow.resourceCount, changeCount)
			}
		})
	}
}

// TestPerformanceRequirements validates performance against requirements
func TestPerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance requirements test in short mode")
	}

	requirements := []struct {
		name         string
		resourceCount int
		maxDuration  time.Duration
		description  string
	}{
		{
			name:         "small_scale",
			resourceCount: 100,
			maxDuration:  1 * time.Second,
			description:  "Small infrastructures should be very fast",
		},
		{
			name:         "medium_scale", 
			resourceCount: 1000,
			maxDuration:  5 * time.Second,
			description:  "Medium infrastructures should be fast",
		},
		{
			name:         "large_scale",
			resourceCount: 5000,
			maxDuration:  30 * time.Second,
			description:  "Large infrastructures should complete within 30s",
		},
		{
			name:         "mega_scale",
			resourceCount: 25000,
			maxDuration:  2 * time.Minute,
			description:  "Mega infrastructures should complete within 2 minutes",
		},
	}

	for _, req := range requirements {
		t.Run(req.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateFile := filepath.Join(tmpDir, "requirement-test.tfstate")
			
			state := createMegaTestState(req.resourceCount)
			data, err := json.MarshalIndent(state, "", "  ")
			if err != nil {
				t.Fatalf("Failed to marshal state: %v", err)
			}
			
			if err := os.WriteFile(stateFile, data, 0644); err != nil {
				t.Fatalf("Failed to write state file: %v", err)
			}

			collector := terraform.NewTerraformCollector()
			config := collectors.CollectorConfig{
				StatePaths: []string{stateFile},
				Tags:       map[string]string{"test": "performance-requirement"},
			}

			start := time.Now()
			snapshot, err := collector.Collect(context.Background(), config)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Collection failed: %v", err)
			}

			if len(snapshot.Resources) != req.resourceCount {
				t.Errorf("Expected %d resources, got %d", req.resourceCount, len(snapshot.Resources))
			}

			if duration > req.maxDuration {
				t.Errorf("%s: took %v, requirement < %v", req.description, duration, req.maxDuration)
			} else {
				t.Logf("%s: ✓ processed %d resources in %v (< %v)", 
					req.description, req.resourceCount, duration, req.maxDuration)
			}
		})
	}
}

// Helper functions

func createMegaTestState(resourceCount int) *terraform.TerraformState {
	resources := make([]terraform.TerraformResource, resourceCount)
	
	resourceTypes := []string{
		"aws_instance", "aws_s3_bucket", "aws_vpc", "aws_subnet", "aws_security_group",
		"aws_rds_instance", "aws_lambda_function", "aws_iam_role", "aws_cloudfront_distribution",
		"google_compute_instance", "google_storage_bucket", "google_compute_network",
		"kubernetes_deployment", "kubernetes_service", "kubernetes_configmap",
	}
	
	regions := []string{"us-west-2", "us-east-1", "eu-west-1", "ap-southeast-1"}
	environments := []string{"production", "staging", "development", "test"}
	
	for i := 0; i < resourceCount; i++ {
		resourceType := resourceTypes[i%len(resourceTypes)]
		region := regions[i%len(regions)]
		environment := environments[i%len(environments)]
		
		// Create realistic attributes based on resource type
		attributes := map[string]interface{}{
			"id":   fmt.Sprintf("%s-id-%d", resourceType, i),
			"name": fmt.Sprintf("%s-name-%d", resourceType, i),
			"region": region,
			"tags": map[string]interface{}{
				"Environment": environment,
				"Index":       fmt.Sprintf("%d", i),
				"Type":        resourceType,
				"CreatedBy":   "terraform",
			},
		}

		// Add type-specific attributes
		switch resourceType {
		case "aws_instance":
			attributes["instance_type"] = "t3.medium"
			attributes["ami"] = fmt.Sprintf("ami-%d", 1000000+i)
		case "aws_s3_bucket":
			attributes["bucket"] = fmt.Sprintf("my-bucket-%d", i)
			attributes["versioning"] = map[string]interface{}{"enabled": true}
		case "kubernetes_deployment":
			attributes["replicas"] = 3
			attributes["image"] = fmt.Sprintf("app:v1.%d", i)
		}

		resources[i] = terraform.TerraformResource{
			Mode: "managed",
			Type: resourceType,
			Name: fmt.Sprintf("resource_%d", i),
			Provider: fmt.Sprintf("provider[\"registry.terraform.io/hashicorp/%s\"]", 
				getProviderName(resourceType)),
			Instances: []terraform.TerraformInstance{
				{
					SchemaVersion: 1,
					Attributes:    attributes,
				},
			},
		}
	}
	
	return &terraform.TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1,
		Lineage:          fmt.Sprintf("mega-test-lineage-%d", resourceCount),
		Resources:        resources,
		Outputs:          make(map[string]interface{}),
	}
}

func createMegaTestStateWithChanges(resourceCount int, changePercentage int) *terraform.TerraformState {
	state := createMegaTestState(resourceCount)
	
	// Calculate number of resources to change
	changeCount := (resourceCount * changePercentage) / 100
	
	for i := 0; i < changeCount; i++ {
		// Modify every N-th resource where N = resourceCount/changeCount
		resourceIndex := i * (resourceCount / changeCount)
		if resourceIndex >= len(state.Resources) {
			break
		}
		
		resource := &state.Resources[resourceIndex]
		instance := &resource.Instances[0]
		
		// Make realistic changes based on resource type
		switch resource.Type {
		case "aws_instance":
			instance.Attributes["instance_type"] = "t3.large" // Scale up
		case "kubernetes_deployment":
			instance.Attributes["replicas"] = 5 // Scale up
			instance.Attributes["image"] = fmt.Sprintf("app:v2.%d", resourceIndex) // Update image
		case "aws_s3_bucket":
			if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
				tags["Updated"] = "true"
			}
		default:
			// Generic change - update tags
			if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
				tags["Modified"] = "true"
			}
		}
	}
	
	return state
}

func getProviderName(resourceType string) string {
	if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	}
	if strings.HasPrefix(resourceType, "google_") {
		return "google"
	}
	if strings.HasPrefix(resourceType, "kubernetes_") {
		return "kubernetes"
	}
	return "hashicorp"
}