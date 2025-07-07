package terraform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yairfalse/wgo/internal/collectors"
)

func TestTerraformCollector_Name(t *testing.T) {
	collector := NewTerraformCollector()
	if collector.Name() != "terraform" {
		t.Errorf("Expected name 'terraform', got %s", collector.Name())
	}
}

func TestTerraformCollector_Status(t *testing.T) {
	collector := NewTerraformCollector()
	status := collector.Status()
	// Status should be either "ready" or contain "not found"
	if status != "ready" && status != "terraform not found in PATH" {
		t.Logf("Status: %s", status)
	}
}

func TestTerraformCollector_Validate(t *testing.T) {
	collector := NewTerraformCollector()
	
	tests := []struct {
		name    string
		config  collectors.CollectorConfig
		wantErr bool
	}{
		{
			name: "valid config with state paths",
			config: collectors.CollectorConfig{
				StatePaths: []string{"../../../test/fixtures/terraform/simple.tfstate"},
			},
			wantErr: false,
		},
		{
			name: "empty state paths",
			config: collectors.CollectorConfig{
				StatePaths: []string{},
			},
			wantErr: true,
		},
		{
			name: "nonexistent state file",
			config: collectors.CollectorConfig{
				StatePaths: []string{"/nonexistent/path.tfstate"},
			},
			wantErr: true,
		},
		{
			name: "remote state path (should pass validation)",
			config: collectors.CollectorConfig{
				StatePaths: []string{"s3://bucket/terraform.tfstate"},
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTerraformCollector_Collect(t *testing.T) {
	collector := NewTerraformCollector()
	
	// Get the absolute path to test fixtures
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	
	fixturesDir := filepath.Join(wd, "../../../test/fixtures/terraform")
	simpleStatePath := filepath.Join(fixturesDir, "simple.tfstate")
	
	// Check if fixture exists
	if _, err := os.Stat(simpleStatePath); err != nil {
		t.Skip("Test fixture not found, skipping test")
	}
	
	config := collectors.CollectorConfig{
		StatePaths: []string{simpleStatePath},
		Tags: map[string]string{
			"test": "true",
		},
	}
	
	ctx := context.Background()
	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	
	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}
	
	if snapshot.Provider != "terraform" {
		t.Errorf("Expected provider 'terraform', got %s", snapshot.Provider)
	}
	
	if len(snapshot.Resources) == 0 {
		t.Error("Expected at least one resource in snapshot")
	}
	
	// Check that resources are properly normalized
	for _, resource := range snapshot.Resources {
		if resource.Provider != "terraform" {
			t.Errorf("Expected resource provider 'terraform', got %s", resource.Provider)
		}
		
		if resource.ID == "" {
			t.Error("Resource ID should not be empty")
		}
		
		if resource.Type == "" {
			t.Error("Resource type should not be empty")
		}
		
		// Validate the resource
		if err := resource.Validate(); err != nil {
			t.Errorf("Resource validation failed: %v", err)
		}
	}
	
	// Validate the snapshot
	if err := snapshot.Validate(); err != nil {
		t.Errorf("Snapshot validation failed: %v", err)
	}
}

func TestTerraformCollector_AutoDiscover(t *testing.T) {
	collector := NewTerraformCollector()
	
	config, err := collector.AutoDiscover()
	if err != nil {
		t.Fatalf("AutoDiscover() error = %v", err)
	}
	
	// Auto-discover might not find any state files in test environment
	// Just check that it returns a valid config structure
	if config.Config == nil {
		t.Error("Expected non-nil config map")
	}
	
	autoDiscovered, exists := config.Config["auto_discovered"]
	if !exists || autoDiscovered != true {
		t.Error("Expected auto_discovered flag to be true")
	}
}

func TestTerraformCollector_SupportedRegions(t *testing.T) {
	collector := NewTerraformCollector()
	
	regions := collector.SupportedRegions()
	if len(regions) == 0 {
		t.Error("Expected at least one supported region")
	}
	
	// Check for common AWS regions
	expectedRegions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	found := make(map[string]bool)
	for _, region := range regions {
		found[region] = true
	}
	
	for _, expected := range expectedRegions {
		if !found[expected] {
			t.Errorf("Expected region %s not found in supported regions", expected)
		}
	}
}

func TestTerraformCollector_CollectLegacyState(t *testing.T) {
	collector := NewTerraformCollector()
	
	// Get the absolute path to test fixtures
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	
	fixturesDir := filepath.Join(wd, "../../../test/fixtures/terraform")
	legacyStatePath := filepath.Join(fixturesDir, "legacy.tfstate")
	
	// Check if fixture exists
	if _, err := os.Stat(legacyStatePath); err != nil {
		t.Skip("Legacy test fixture not found, skipping test")
	}
	
	config := collectors.CollectorConfig{
		StatePaths: []string{legacyStatePath},
	}
	
	ctx := context.Background()
	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	
	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}
	
	// Legacy state should be converted properly
	if len(snapshot.Resources) == 0 {
		t.Error("Expected at least one resource from legacy state")
	}
	
	// Check that legacy resources are properly converted
	for _, resource := range snapshot.Resources {
		if resource.Provider != "terraform" {
			t.Errorf("Expected resource provider 'terraform', got %s", resource.Provider)
		}
		
		// Validate the resource
		if err := resource.Validate(); err != nil {
			t.Errorf("Legacy resource validation failed: %v", err)
		}
	}
}

func TestTerraformCollector_CollectComplexState(t *testing.T) {
	collector := NewTerraformCollector()
	
	// Get the absolute path to test fixtures
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	
	fixturesDir := filepath.Join(wd, "../../../test/fixtures/terraform")
	complexStatePath := filepath.Join(fixturesDir, "complex.tfstate")
	
	// Check if fixture exists
	if _, err := os.Stat(complexStatePath); err != nil {
		t.Skip("Complex test fixture not found, skipping test")
	}
	
	config := collectors.CollectorConfig{
		StatePaths: []string{complexStatePath},
	}
	
	ctx := context.Background()
	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	
	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}
	
	// Complex state should have multiple resources
	if len(snapshot.Resources) < 3 {
		t.Errorf("Expected at least 3 resources from complex state, got %d", len(snapshot.Resources))
	}
	
	// Check for specific resource types
	resourceTypes := make(map[string]int)
	for _, resource := range snapshot.Resources {
		resourceTypes[resource.Type]++
		
		// Validate the resource
		if err := resource.Validate(); err != nil {
			t.Errorf("Complex resource validation failed: %v", err)
		}
	}
	
	// Should have multiple instances of aws_instance (count = 2)
	if resourceTypes["aws_instance"] < 2 {
		t.Errorf("Expected at least 2 aws_instance resources, got %d", resourceTypes["aws_instance"])
	}
	
	// Should have RDS and Lambda resources
	expectedTypes := []string{"aws_rds_instance", "aws_lambda_function"}
	for _, expectedType := range expectedTypes {
		if resourceTypes[expectedType] == 0 {
			t.Errorf("Expected %s resource not found", expectedType)
		}
	}
}