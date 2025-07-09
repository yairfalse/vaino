package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
)

// TestParallelProcessingPerformance tests parallel state file processing
func TestParallelProcessingPerformance(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Create multiple realistic state files
	stateFiles := make([]string, 5)
	for i := 0; i < 5; i++ {
		stateFile := filepath.Join(tmpDir, fmt.Sprintf("terraform-%d.tfstate", i))
		stateFiles[i] = stateFile

		// Create state with varying number of resources
		resourceCount := (i + 1) * 10 // 10, 20, 30, 40, 50 resources
		state := createLargeTestState(resourceCount)

		data, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal test state: %v", err)
		}

		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			t.Fatalf("Failed to write test state file: %v", err)
		}
	}

	// Test with enhanced collector
	collector := NewTerraformCollector()
	config := collectors.CollectorConfig{
		StatePaths: stateFiles,
		Tags:       map[string]string{"environment": "test", "type": "performance"},
	}

	ctx := context.Background()
	startTime := time.Now()

	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		t.Fatalf("Parallel collection failed: %v", err)
	}

	collectionTime := time.Since(startTime)

	// Verify results
	expectedResources := 10 + 20 + 30 + 40 + 50 // 150 total resources
	if len(snapshot.Resources) != expectedResources {
		t.Errorf("Expected %d resources, got %d", expectedResources, len(snapshot.Resources))
	}

	// Performance should be under 1 second for 150 resources across 5 files
	if collectionTime > time.Second {
		t.Errorf("Parallel processing took too long: %v (expected < 1s)", collectionTime)
	}

	// Check if metadata includes performance stats
	if snapshot.Metadata.AdditionalData != nil {
		if parseStats, exists := snapshot.Metadata.AdditionalData["parsing_stats"]; exists {
			stats := parseStats.(map[string]interface{})
			t.Logf("Parsing stats: %+v", stats)

			// Verify parsing was successful
			if successRate, ok := stats["success_rate"]; ok {
				if rate := successRate.(float64); rate < 100.0 {
					t.Errorf("Expected 100%% success rate, got %.1f%%", rate)
				}
			}
		}
	}

	t.Logf("Processed %d resources from %d files in %v",
		len(snapshot.Resources), len(stateFiles), collectionTime)
}

// TestStreamingParserPerformance tests streaming parser with large files
func TestStreamingParserPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping streaming parser performance test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a large state file (simulate 1000 resources)
	largeStateFile := filepath.Join(tmpDir, "large-terraform.tfstate")
	largeState := createLargeTestState(1000)

	data, err := json.MarshalIndent(largeState, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal large test state: %v", err)
	}

	if err := os.WriteFile(largeStateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write large test state file: %v", err)
	}

	// Get file size for reporting
	stat, _ := os.Stat(largeStateFile)
	fileSizeMB := float64(stat.Size()) / (1024 * 1024)

	t.Logf("Created large state file: %.2f MB with 1000 resources", fileSizeMB)

	// Test streaming parser directly
	streamParser := NewStreamingParser()

	startTime := time.Now()
	parsedState, err := streamParser.ParseStateFile(largeStateFile)
	if err != nil {
		t.Fatalf("Streaming parser failed: %v", err)
	}
	parseTime := time.Since(startTime)

	// Verify parsing results
	if len(parsedState.Resources) != 1000 {
		t.Errorf("Expected 1000 resources, got %d", len(parsedState.Resources))
	}

	// Performance should be reasonable (under 500ms for 1000 resources)
	if parseTime > 500*time.Millisecond {
		t.Errorf("Streaming parsing took too long: %v (expected < 500ms)", parseTime)
	}

	t.Logf("Streaming parser processed %.2f MB file with %d resources in %v",
		fileSizeMB, len(parsedState.Resources), parseTime)
}

// TestParallelParserConcurrency tests concurrent parsing of multiple files
func TestParallelParserConcurrency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 10 state files with different sizes
	stateFiles := make([]string, 10)
	for i := 0; i < 10; i++ {
		stateFile := filepath.Join(tmpDir, fmt.Sprintf("state-%d.tfstate", i))
		stateFiles[i] = stateFile

		// Varying resource counts from 5 to 50
		resourceCount := (i + 1) * 5
		state := createLargeTestState(resourceCount)

		data, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal test state %d: %v", i, err)
		}

		if err := os.WriteFile(stateFile, data, 0644); err != nil {
			t.Fatalf("Failed to write test state file %d: %v", i, err)
		}
	}

	// Test parallel parser
	parallelParser := NewParallelStateParser()
	ctx := context.Background()

	startTime := time.Now()
	parseResults, err := parallelParser.ParseMultipleStates(ctx, stateFiles)
	if err != nil {
		t.Fatalf("Parallel parsing failed: %v", err)
	}
	parseTime := time.Since(startTime)

	// Verify all files were parsed successfully
	successCount := 0
	totalResources := 0
	for _, result := range parseResults {
		if result.Error == nil {
			successCount++
			totalResources += len(result.State.Resources)
		} else {
			t.Errorf("Parse error for %s: %v", result.FilePath, result.Error)
		}
	}

	if successCount != 10 {
		t.Errorf("Expected 10 successful parses, got %d", successCount)
	}

	expectedResources := 5 + 10 + 15 + 20 + 25 + 30 + 35 + 40 + 45 + 50 // 275 total
	if totalResources != expectedResources {
		t.Errorf("Expected %d total resources, got %d", expectedResources, totalResources)
	}

	// Get performance statistics
	stats := parallelParser.GetParsingStats(parseResults)

	// Performance should be better than sequential (parallel should be faster)
	if parseTime > 2*time.Second {
		t.Errorf("Parallel parsing took too long: %v (expected < 2s for 10 files)", parseTime)
	}

	t.Logf("Parallel parsed %d files with %d total resources in %v",
		len(stateFiles), totalResources, parseTime)
	t.Logf("Performance stats: %+v", stats)
}

// createLargeTestState creates a test state with specified number of resources
func createLargeTestState(resourceCount int) *TerraformState {
	resources := make([]TerraformResource, resourceCount)

	for i := 0; i < resourceCount; i++ {
		// Create diverse resource types
		resourceTypes := []string{"aws_instance", "aws_s3_bucket", "aws_vpc", "aws_subnet", "kubernetes_deployment"}
		resourceType := resourceTypes[i%len(resourceTypes)]

		resources[i] = TerraformResource{
			Mode:     "managed",
			Type:     resourceType,
			Name:     fmt.Sprintf("resource_%d", i),
			Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
			Instances: []TerraformInstance{
				{
					SchemaVersion: 1,
					Attributes: map[string]interface{}{
						"id":         fmt.Sprintf("resource-id-%d", i),
						"name":       fmt.Sprintf("resource-name-%d", i),
						"region":     "us-west-2",
						"created_at": "2023-01-01T00:00:00Z",
						"tags": map[string]interface{}{
							"Environment": "test",
							"Resource":    fmt.Sprintf("%d", i),
						},
					},
				},
			},
		}
	}

	return &TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1,
		Lineage:          "test-lineage",
		Resources:        resources,
		Outputs:          make(map[string]interface{}),
	}
}
