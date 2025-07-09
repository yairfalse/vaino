package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yairfalse/wgo/internal/collectors"
)

// BenchmarkTerraformCollector_ParseSmallState benchmarks parsing a small state file
func BenchmarkTerraformCollector_ParseSmallState(b *testing.B) {
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, "small.tfstate")

	// Create small state with 10 resources
	state := createTestState(10)
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)

	collector := NewTerraformCollector()
	config := collectors.CollectorConfig{
		StatePaths: []string{stateFile},
		Tags:       map[string]string{"benchmark": "small"},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx, config)
		if err != nil {
			b.Fatalf("Collection failed: %v", err)
		}
	}
}

// BenchmarkTerraformCollector_ParseMediumState benchmarks parsing a medium state file
func BenchmarkTerraformCollector_ParseMediumState(b *testing.B) {
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, "medium.tfstate")

	// Create medium state with 100 resources
	state := createTestState(100)
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)

	collector := NewTerraformCollector()
	config := collectors.CollectorConfig{
		StatePaths: []string{stateFile},
		Tags:       map[string]string{"benchmark": "medium"},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx, config)
		if err != nil {
			b.Fatalf("Collection failed: %v", err)
		}
	}
}

// BenchmarkTerraformCollector_ParseLargeState benchmarks parsing a large state file
func BenchmarkTerraformCollector_ParseLargeState(b *testing.B) {
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, "large.tfstate")

	// Create large state with 500 resources
	state := createTestState(500)
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)

	collector := NewTerraformCollector()
	config := collectors.CollectorConfig{
		StatePaths: []string{stateFile},
		Tags:       map[string]string{"benchmark": "large"},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx, config)
		if err != nil {
			b.Fatalf("Collection failed: %v", err)
		}
	}
}

// BenchmarkTerraformCollector_ParallelProcessing benchmarks parallel processing
func BenchmarkTerraformCollector_ParallelProcessing(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 5 state files with 50 resources each
	var stateFiles []string
	for i := 0; i < 5; i++ {
		stateFile := filepath.Join(tmpDir, fmt.Sprintf("state-%d.tfstate", i))
		stateFiles = append(stateFiles, stateFile)

		state := createTestState(50)
		data, _ := json.MarshalIndent(state, "", "  ")
		os.WriteFile(stateFile, data, 0644)
	}

	collector := NewTerraformCollector()
	config := collectors.CollectorConfig{
		StatePaths: stateFiles,
		Tags:       map[string]string{"benchmark": "parallel"},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx, config)
		if err != nil {
			b.Fatalf("Collection failed: %v", err)
		}
	}
}

// BenchmarkStreamingParser_LargeFile benchmarks the streaming parser with large files
func BenchmarkStreamingParser_LargeFile(b *testing.B) {
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, "streaming.tfstate")

	// Create very large state with 1000 resources
	state := createTestState(1000)
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(stateFile, data, 0644)

	parser := NewStreamingParser()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := parser.ParseStateFile(stateFile)
		if err != nil {
			b.Fatalf("Streaming parse failed: %v", err)
		}
	}
}

// BenchmarkParallelStateParser_Concurrency benchmarks the parallel state parser
func BenchmarkParallelStateParser_Concurrency(b *testing.B) {
	tmpDir := b.TempDir()

	// Create 10 state files with 25 resources each
	var stateFiles []string
	for i := 0; i < 10; i++ {
		stateFile := filepath.Join(tmpDir, fmt.Sprintf("concurrent-%d.tfstate", i))
		stateFiles = append(stateFiles, stateFile)

		state := createTestState(25)
		data, _ := json.MarshalIndent(state, "", "  ")
		os.WriteFile(stateFile, data, 0644)
	}

	parser := NewParallelStateParser()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := parser.ParseMultipleStates(ctx, stateFiles)
		if err != nil {
			b.Fatalf("Parallel parse failed: %v", err)
		}
	}
}

// BenchmarkResourceNormalization benchmarks the resource normalization process
func BenchmarkResourceNormalization(b *testing.B) {
	// Create state with diverse resource types
	state := createDiverseTestState(100)
	normalizer := NewResourceNormalizer()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := normalizer.NormalizeResources(state)
		if err != nil {
			b.Fatalf("Normalization failed: %v", err)
		}
	}
}

// createTestState creates a test state with the specified number of resources
func createTestState(resourceCount int) *TerraformState {
	resources := make([]TerraformResource, resourceCount)

	for i := 0; i < resourceCount; i++ {
		// Alternate between different resource types for variety
		resourceTypes := []string{"aws_instance", "aws_s3_bucket", "aws_vpc", "aws_subnet"}
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
						"id":            fmt.Sprintf("resource-id-%d", i),
						"name":          fmt.Sprintf("resource-name-%d", i),
						"region":        "us-west-2",
						"created_at":    "2023-01-01T00:00:00Z",
						"instance_type": "t3.micro",
						"tags": map[string]interface{}{
							"Environment": "benchmark",
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
		Lineage:          "benchmark-lineage",
		Resources:        resources,
		Outputs:          make(map[string]interface{}),
	}
}

// createDiverseTestState creates a test state with diverse resource types
func createDiverseTestState(resourceCount int) *TerraformState {
	resources := make([]TerraformResource, resourceCount)

	// Wider variety of resource types for normalization benchmarking
	resourceTypes := []string{
		"aws_instance", "aws_s3_bucket", "aws_vpc", "aws_subnet", "aws_rds_instance",
		"google_compute_instance", "google_storage_bucket", "google_sql_instance",
		"azurerm_virtual_machine", "azurerm_storage_account", "azurerm_sql_server",
		"kubernetes_deployment", "kubernetes_service", "kubernetes_namespace",
	}

	for i := 0; i < resourceCount; i++ {
		resourceType := resourceTypes[i%len(resourceTypes)]

		resources[i] = TerraformResource{
			Mode:     "managed",
			Type:     resourceType,
			Name:     fmt.Sprintf("diverse_%d", i),
			Provider: getProviderForType(resourceType),
			Instances: []TerraformInstance{
				{
					SchemaVersion: 1,
					Attributes: map[string]interface{}{
						"id":     fmt.Sprintf("diverse-id-%d", i),
						"name":   fmt.Sprintf("diverse-name-%d", i),
						"region": getRegionForType(resourceType),
						"tags": map[string]interface{}{
							"Type":        resourceType,
							"Index":       fmt.Sprintf("%d", i),
							"Environment": "benchmark",
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
		Lineage:          "diverse-benchmark-lineage",
		Resources:        resources,
		Outputs:          make(map[string]interface{}),
	}
}

// getProviderForType returns the appropriate provider for a resource type
func getProviderForType(resourceType string) string {
	switch {
	case strings.HasPrefix(resourceType, "aws_"):
		return "provider[\"registry.terraform.io/hashicorp/aws\"]"
	case strings.HasPrefix(resourceType, "google_"):
		return "provider[\"registry.terraform.io/hashicorp/google\"]"
	case strings.HasPrefix(resourceType, "azurerm_"):
		return "provider[\"registry.terraform.io/hashicorp/azurerm\"]"
	case strings.HasPrefix(resourceType, "kubernetes_"):
		return "provider[\"registry.terraform.io/hashicorp/kubernetes\"]"
	default:
		return "provider[\"registry.terraform.io/hashicorp/aws\"]"
	}
}

// getRegionForType returns the appropriate region for a resource type
func getRegionForType(resourceType string) string {
	switch {
	case strings.HasPrefix(resourceType, "aws_"):
		return "us-west-2"
	case strings.HasPrefix(resourceType, "google_"):
		return "us-central1"
	case strings.HasPrefix(resourceType, "azurerm_"):
		return "eastus"
	case strings.HasPrefix(resourceType, "kubernetes_"):
		return "cluster-region"
	default:
		return "us-west-2"
	}
}
