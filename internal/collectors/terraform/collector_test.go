package terraform

import (
	"context"
	"strings"
	"testing"

	"github.com/yairfalse/wgo/internal/collectors"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector()
	if collector == nil {
		t.Fatal("NewCollector() returned nil")
	}
	if collector.Name() != "terraform" {
		t.Errorf("NewCollector().Name() = %v, want %v", collector.Name(), "terraform")
	}
}

func TestCollector_Name(t *testing.T) {
	collector := &Collector{name: "test-collector"}
	if collector.Name() != "test-collector" {
		t.Errorf("Name() = %v, want %v", collector.Name(), "test-collector")
	}
}

func TestCollector_Validate(t *testing.T) {
	collector := NewCollector()

	tests := []struct {
		config  collectors.Config
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: collectors.Config{
				Provider: "terraform",
				Region:   "us-west-2",
				Paths:    []string{"/path/to/state"},
			},
			wantErr: false,
		},
		{
			name: "empty provider",
			config: collectors.Config{
				Provider: "",
				Region:   "us-west-2",
				Paths:    []string{"/path/to/state"},
			},
			wantErr: true,
		},
		{
			name: "wrong provider",
			config: collectors.Config{
				Provider: "aws",
				Region:   "us-west-2",
				Paths:    []string{"/path/to/state"},
			},
			wantErr: true,
		},
		{
			name: "empty region",
			config: collectors.Config{
				Provider: "terraform",
				Region:   "",
				Paths:    []string{"/path/to/state"},
			},
			wantErr: true,
		},
		{
			name: "whitespace region",
			config: collectors.Config{
				Provider: "terraform",
				Region:   "   ",
				Paths:    []string{"/path/to/state"},
			},
			wantErr: true,
		},
		{
			name: "no paths",
			config: collectors.Config{
				Provider: "terraform",
				Region:   "us-west-2",
				Paths:    []string{},
			},
			wantErr: true,
		},
		{
			name: "empty path in list",
			config: collectors.Config{
				Provider: "terraform",
				Region:   "us-west-2",
				Paths:    []string{"/valid/path", "", "/another/path"},
			},
			wantErr: true,
		},
		{
			name: "whitespace path in list",
			config: collectors.Config{
				Provider: "terraform",
				Region:   "us-west-2",
				Paths:    []string{"/valid/path", "   ", "/another/path"},
			},
			wantErr: true,
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

func TestCollector_ExtractWorkspace(t *testing.T) {
	collector := NewCollector()

	tests := []struct {
		name     string
		config   collectors.Config
		expected string
	}{
		{
			name: "workspace in options",
			config: collectors.Config{
				Options: map[string]interface{}{
					"workspace": "production",
				},
			},
			expected: "production",
		},
		{
			name: "no workspace in options",
			config: collectors.Config{
				Options: map[string]interface{}{
					"other_option": "value",
				},
			},
			expected: "",
		},
		{
			name: "nil options",
			config: collectors.Config{
				Options: nil,
			},
			expected: "",
		},
		{
			name: "workspace is not string",
			config: collectors.Config{
				Options: map[string]interface{}{
					"workspace": 123,
				},
			},
			expected: "",
		},
		{
			name: "empty workspace",
			config: collectors.Config{
				Options: map[string]interface{}{
					"workspace": "",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractWorkspace(tt.config)
			if result != tt.expected {
				t.Errorf("extractWorkspace() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCollector_BuildMetadata(t *testing.T) {
	collector := NewCollector()
	config := collectors.Config{
		Provider: "terraform",
		Region:   "us-west-2",
		Options: map[string]interface{}{
			"workspace":   "production",
			"environment": "prod",
			"numeric_opt": 123,
			"boolean_opt": true,
		},
	}
	stateFiles := []string{
		"/path/to/terraform.tfstate",
		"/path/to/modules/terraform.tfstate",
	}

	metadata := collector.buildMetadata(config, stateFiles)

	// Check required fields
	if metadata["collector"] != "terraform" {
		t.Errorf("Expected collector metadata to be 'terraform', got '%s'", metadata["collector"])
	}

	if metadata["state_files_count"] != "2" {
		t.Errorf("Expected state_files_count to be '2', got '%s'", metadata["state_files_count"])
	}

	if metadata["primary_state_file"] != "/path/to/terraform.tfstate" {
		t.Errorf("Expected primary_state_file to be '/path/to/terraform.tfstate', got '%s'", metadata["primary_state_file"])
	}

	if metadata["workspace"] != "production" {
		t.Errorf("Expected workspace to be 'production', got '%s'", metadata["workspace"])
	}

	// Check that collection_time is set
	if metadata["collection_time"] == "" {
		t.Error("Expected collection_time to be set")
	}

	// Check that string options are included
	if metadata["option_environment"] != "prod" {
		t.Errorf("Expected option_environment to be 'prod', got '%s'", metadata["option_environment"])
	}

	// Check that non-string options are not included
	if _, exists := metadata["option_numeric_opt"]; exists {
		t.Error("Expected numeric_opt to not be included in metadata")
	}

	if _, exists := metadata["option_boolean_opt"]; exists {
		t.Error("Expected boolean_opt to not be included in metadata")
	}
}

func TestCollector_SupportedResourceTypes(t *testing.T) {
	collector := NewCollector()
	resourceTypes := collector.SupportedResourceTypes()

	// Check that we have some expected resource types
	expectedTypes := []string{
		"aws_instance",
		"aws_s3_bucket",
		"azurerm_virtual_machine",
		"google_compute_instance",
	}

	for _, expectedType := range expectedTypes {
		found := false
		for _, resourceType := range resourceTypes {
			if resourceType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected resource type '%s' not found in supported types", expectedType)
		}
	}

	// Check length is reasonable
	if len(resourceTypes) < 10 {
		t.Errorf("Expected at least 10 supported resource types, got %d", len(resourceTypes))
	}
}

func TestCollector_IsResourceTypeSupported(t *testing.T) {
	collector := NewCollector()

	tests := []struct {
		name         string
		resourceType string
		expected     bool
	}{
		{
			name:         "supported AWS resource",
			resourceType: "aws_instance",
			expected:     true,
		},
		{
			name:         "supported Azure resource",
			resourceType: "azurerm_virtual_machine",
			expected:     true,
		},
		{
			name:         "supported Google Cloud resource",
			resourceType: "google_compute_instance",
			expected:     true,
		},
		{
			name:         "unsupported resource type",
			resourceType: "custom_resource_type",
			expected:     false,
		},
		{
			name:         "empty resource type",
			resourceType: "",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.IsResourceTypeSupported(tt.resourceType)
			if result != tt.expected {
				t.Errorf("IsResourceTypeSupported() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCollector_CollectFromPaths(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	// Test with non-existent paths (should return error)
	_, err := collector.CollectFromPaths(ctx, []string{"/non/existent/path"}, "us-west-2")
	if err == nil {
		t.Error("Expected error for non-existent paths, got nil")
	}
}

func TestCollector_CollectFromWorkspace(t *testing.T) {
	collector := NewCollector()
	ctx := context.Background()

	// Test with non-existent workspace path (should return error)
	_, err := collector.CollectFromWorkspace(ctx, "/non/existent/workspace", "production", "us-west-2")
	if err == nil {
		t.Error("Expected error for non-existent workspace path, got nil")
	}
}

func TestGenerateSnapshotID(t *testing.T) {
	id1 := generateSnapshotID()

	// Check that IDs are not empty
	if id1 == "" {
		t.Error("Expected non-empty snapshot ID")
	}

	// Check that IDs start with expected prefix
	expectedPrefix := "terraform-snapshot-"
	if !strings.HasPrefix(id1, expectedPrefix) {
		t.Errorf("Expected snapshot ID to start with '%s', got '%s'", expectedPrefix, id1)
	}

	// Check that the ID contains a timestamp
	if len(id1) <= len(expectedPrefix) {
		t.Error("Expected snapshot ID to contain timestamp after prefix")
	}
}
