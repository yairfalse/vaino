package terraform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yairfalse/wgo/internal/collectors"
)

// createTestStateFile creates a test Terraform state file
func createTestStateFile(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "terraform.tfstate")

	err := os.WriteFile(stateFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	return stateFile
}

// TestTerraformCollector_Basic tests basic functionality
func TestTerraformCollector_Basic(t *testing.T) {
	collector := NewCollector()

	// Test Name
	if collector.Name() != "terraform" {
		t.Errorf("Expected name 'terraform', got '%s'", collector.Name())
	}

	// Test validation with invalid config
	invalidConfig := collectors.Config{
		Provider: "aws", // Wrong provider
		Region:   "us-west-2",
		Paths:    []string{"/tmp"},
	}

	err := collector.Validate(invalidConfig)
	if err == nil {
		t.Error("Expected validation error for invalid provider")
	}

	// Test validation with valid config
	validConfig := collectors.Config{
		Provider: "terraform",
		Region:   "us-west-2",
		Paths:    []string{"/tmp"},
	}

	err = collector.Validate(validConfig)
	if err != nil {
		t.Errorf("Expected no validation error, got: %v", err)
	}
}

// TestParseStateFile tests parsing of a simple state file
func TestParseStateFile(t *testing.T) {
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test-lineage",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "example",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"schema_version": 1,
						"attributes": {
							"id": "i-1234567890abcdef0",
							"instance_type": "t2.micro",
							"region": "us-west-2",
							"state": "running",
							"tags": {
								"Name": "example-instance",
								"Environment": "test"
							}
						}
					}
				]
			}
		]
	}`

	stateFile := createTestStateFile(t, stateContent)

	resources, err := ParseStateFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to parse state file: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(resources))
	}

	resource := resources[0]
	if resource.ID != "i-1234567890abcdef0" {
		t.Errorf("Expected resource ID 'i-1234567890abcdef0', got '%s'", resource.ID)
	}

	if resource.Type != "aws_instance" {
		t.Errorf("Expected resource type 'aws_instance', got '%s'", resource.Type)
	}

	if resource.Provider != "aws" {
		t.Errorf("Expected provider 'aws', got '%s'", resource.Provider)
	}

	if resource.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", resource.Region)
	}

	if resource.State != "running" {
		t.Errorf("Expected state 'running', got '%s'", resource.State)
	}

	if resource.Tags["Name"] != "example-instance" {
		t.Errorf("Expected tag Name 'example-instance', got '%s'", resource.Tags["Name"])
	}

	if resource.Tags["Environment"] != "test" {
		t.Errorf("Expected tag Environment 'test', got '%s'", resource.Tags["Environment"])
	}
}

// TestCollect tests the full collection process
func TestCollect(t *testing.T) {
	stateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test-lineage",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_s3_bucket",
				"name": "example",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"schema_version": 1,
						"attributes": {
							"id": "my-test-bucket",
							"bucket": "my-test-bucket",
							"region": "us-east-1"
						}
					}
				]
			}
		]
	}`

	stateFile := createTestStateFile(t, stateContent)

	collector := NewCollector()
	config := collectors.Config{
		Provider: "terraform",
		Region:   "us-east-1",
		Paths:    []string{filepath.Dir(stateFile)},
	}

	ctx := context.Background()
	snapshot, err := collector.Collect(ctx, config)
	if err != nil {
		t.Fatalf("Failed to collect resources: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Expected snapshot, got nil")
	}

	if len(snapshot.Resources) != 1 {
		t.Fatalf("Expected 1 resource in snapshot, got %d", len(snapshot.Resources))
	}

	resource := snapshot.Resources[0]
	if resource.Type != "aws_s3_bucket" {
		t.Errorf("Expected resource type 'aws_s3_bucket', got '%s'", resource.Type)
	}

	if snapshot.Provider != "terraform" {
		t.Errorf("Expected snapshot provider 'terraform', got '%s'", snapshot.Provider)
	}

	if snapshot.Region != "us-east-1" {
		t.Errorf("Expected snapshot region 'us-east-1', got '%s'", snapshot.Region)
	}
}
