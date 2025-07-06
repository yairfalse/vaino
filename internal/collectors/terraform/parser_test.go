package terraform

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestExtractResourceID(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]interface{}
		expected   string
	}{
		{
			name: "id field present",
			attributes: map[string]interface{}{
				"id":   "i-1234567890abcdef0",
				"name": "test-instance",
			},
			expected: "i-1234567890abcdef0",
		},
		{
			name: "arn field present",
			attributes: map[string]interface{}{
				"arn":  "arn:aws:s3:::my-bucket",
				"name": "test-bucket",
			},
			expected: "arn:aws:s3:::my-bucket",
		},
		{
			name: "instance_id field present",
			attributes: map[string]interface{}{
				"instance_id": "i-abcdef1234567890",
				"name":        "test-instance",
			},
			expected: "i-abcdef1234567890",
		},
		{
			name: "no id fields present",
			attributes: map[string]interface{}{
				"name": "test-resource",
				"type": "aws_instance",
			},
			expected: "",
		},
		{
			name:       "empty attributes",
			attributes: map[string]interface{}{},
			expected:   "",
		},
		{
			name: "empty id field",
			attributes: map[string]interface{}{
				"id":   "",
				"name": "test-resource",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceID(tt.attributes)
			if result != tt.expected {
				t.Errorf("extractResourceID() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractRegion(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]interface{}
		expected   string
	}{
		{
			name: "region field present",
			attributes: map[string]interface{}{
				"region": "us-west-2",
			},
			expected: "us-west-2",
		},
		{
			name: "availability_zone field present",
			attributes: map[string]interface{}{
				"availability_zone": "us-east-1a",
			},
			expected: "us-east-1",
		},
		{
			name: "zone field present",
			attributes: map[string]interface{}{
				"zone": "europe-west1-b",
			},
			expected: "europe-west1",
		},
		{
			name: "location field present",
			attributes: map[string]interface{}{
				"location": "East US",
			},
			expected: "East US",
		},
		{
			name: "no region fields present",
			attributes: map[string]interface{}{
				"name": "test-resource",
			},
			expected: "",
		},
		{
			name:       "empty attributes",
			attributes: map[string]interface{}{},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRegion(tt.attributes)
			if result != tt.expected {
				t.Errorf("extractRegion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		expected string
	}{
		{
			name:     "registry format",
			provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
			expected: "aws",
		},
		{
			name:     "simple provider format",
			provider: "provider.aws",
			expected: "aws",
		},
		{
			name:     "azure provider",
			provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
			expected: "azurerm",
		},
		{
			name:     "google provider",
			provider: "provider[\"registry.terraform.io/hashicorp/google\"]",
			expected: "google",
		},
		{
			name:     "empty provider",
			provider: "",
			expected: "terraform",
		},
		{
			name:     "quoted provider",
			provider: "\"aws\"",
			expected: "aws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractProvider(tt.provider)
			if result != tt.expected {
				t.Errorf("extractProvider() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	tests := []struct {
		attributes map[string]interface{}
		expected   map[string]string
		name       string
	}{
		{
			name: "tags field present",
			attributes: map[string]interface{}{
				"tags": map[string]interface{}{
					"Name":        "test-instance",
					"Environment": "production",
					"Owner":       "team-alpha",
				},
			},
			expected: map[string]string{
				"Name":        "test-instance",
				"Environment": "production",
				"Owner":       "team-alpha",
			},
		},
		{
			name: "labels field present",
			attributes: map[string]interface{}{
				"labels": map[string]interface{}{
					"app":     "web-server",
					"version": "1.0.0",
				},
			},
			expected: map[string]string{
				"app":     "web-server",
				"version": "1.0.0",
			},
		},
		{
			name: "metadata field present",
			attributes: map[string]interface{}{
				"metadata": map[string]interface{}{
					"created_by": "terraform",
					"project":    "demo",
				},
			},
			expected: map[string]string{
				"created_by": "terraform",
				"project":    "demo",
			},
		},
		{
			name: "no tag fields present",
			attributes: map[string]interface{}{
				"name": "test-resource",
				"type": "aws_instance",
			},
			expected: map[string]string{},
		},
		{
			name:       "empty attributes",
			attributes: map[string]interface{}{},
			expected:   map[string]string{},
		},
		{
			name: "mixed tag types",
			attributes: map[string]interface{}{
				"tags": map[string]interface{}{
					"Name":    "test-instance",
					"Count":   123,  // non-string value should be ignored
					"Enabled": true, // non-string value should be ignored
					"Owner":   "team-beta",
				},
			},
			expected: map[string]string{
				"Name":  "test-instance",
				"Owner": "team-beta",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTags(tt.attributes)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractTags() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractState(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]interface{}
		expected   string
	}{
		{
			name: "state field present",
			attributes: map[string]interface{}{
				"state": "running",
			},
			expected: "running",
		},
		{
			name: "status field present",
			attributes: map[string]interface{}{
				"status": "active",
			},
			expected: "active",
		},
		{
			name: "lifecycle_state field present",
			attributes: map[string]interface{}{
				"lifecycle_state": "available",
			},
			expected: "available",
		},
		{
			name: "instance_state field present",
			attributes: map[string]interface{}{
				"instance_state": "stopped",
			},
			expected: "stopped",
		},
		{
			name: "no state fields present",
			attributes: map[string]interface{}{
				"name": "test-resource",
			},
			expected: "running",
		},
		{
			name:       "empty attributes",
			attributes: map[string]interface{}{},
			expected:   "running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractState(tt.attributes)
			if result != tt.expected {
				t.Errorf("extractState() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsStateFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "terraform.tfstate",
			filePath: "/path/to/terraform.tfstate",
			expected: true,
		},
		{
			name:     "custom.tfstate",
			filePath: "/path/to/custom.tfstate",
			expected: true,
		},
		{
			name:     "backup state file",
			filePath: "/path/to/terraform.tfstate.backup",
			expected: true,
		},
		{
			name:     "regular terraform file",
			filePath: "/path/to/main.tf",
			expected: false,
		},
		{
			name:     "json file",
			filePath: "/path/to/config.json",
			expected: false,
		},
		{
			name:     "empty path",
			filePath: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStateFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("isStateFile() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindStateFiles(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()

	// Create some test files
	stateFile1 := filepath.Join(tempDir, "terraform.tfstate")
	stateFile2 := filepath.Join(tempDir, "custom.tfstate")
	backupFile := filepath.Join(tempDir, "terraform.tfstate.backup")
	regularFile := filepath.Join(tempDir, "main.tf")

	// Create subdirectory with another state file
	subDir := filepath.Join(tempDir, "modules")
	err := os.Mkdir(subDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	stateFile3 := filepath.Join(subDir, "terraform.tfstate")

	// Write test files
	for _, file := range []string{stateFile1, stateFile2, backupFile, regularFile, stateFile3} {
		err := os.WriteFile(file, []byte("{}"), 0o600)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	tests := []struct {
		name     string
		paths    []string
		expected []string
	}{
		{
			name:  "single directory",
			paths: []string{tempDir},
			expected: []string{
				stateFile1,
				stateFile2,
				backupFile,
				stateFile3,
			},
		},
		{
			name:     "specific file",
			paths:    []string{stateFile1},
			expected: []string{stateFile1},
		},
		{
			name:     "non-existent path",
			paths:    []string{"/non/existent/path"},
			expected: []string{},
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: []string{},
		},
		{
			name:     "mixed paths",
			paths:    []string{stateFile1, subDir, "/non/existent"},
			expected: []string{stateFile1, stateFile3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindStateFiles(tt.paths)
			if err != nil {
				t.Errorf("FindStateFiles() error = %v", err)
				return
			}

			// Sort both slices for comparison
			if len(result) != len(tt.expected) {
				t.Errorf("FindStateFiles() found %d files, want %d", len(result), len(tt.expected))
				return
			}

			// Check that all expected files are found
			expectedMap := make(map[string]bool)
			for _, file := range tt.expected {
				expectedMap[file] = true
			}

			for _, file := range result {
				if !expectedMap[file] {
					t.Errorf("FindStateFiles() found unexpected file: %s", file)
				}
			}
		})
	}
}
