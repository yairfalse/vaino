package terraform

import (
	"testing"
)

func TestRemoteStateHandler_parseRemoteStateURL(t *testing.T) {
	handler := NewRemoteStateHandler()

	tests := []struct {
		name            string
		stateURL        string
		expectedBackend string
		expectedConfig  map[string]string
		expectError     bool
	}{
		{
			name:            "S3 URL with region",
			stateURL:        "s3://my-bucket/path/to/terraform.tfstate?region=us-east-1",
			expectedBackend: "s3",
			expectedConfig: map[string]string{
				"bucket": "my-bucket",
				"key":    "path/to/terraform.tfstate",
				"region": "us-east-1",
			},
			expectError: false,
		},
		{
			name:            "S3 URL without region",
			stateURL:        "s3://my-bucket/terraform.tfstate",
			expectedBackend: "s3",
			expectedConfig: map[string]string{
				"bucket": "my-bucket",
				"key":    "terraform.tfstate",
			},
			expectError: false,
		},
		{
			name:            "Azure URL",
			stateURL:        "azurerm://mystorageaccount/mycontainer/terraform.tfstate",
			expectedBackend: "azurerm",
			expectedConfig: map[string]string{
				"storage_account_name": "mystorageaccount",
				"container_name":       "mycontainer",
				"key":                  "terraform.tfstate",
			},
			expectError: false,
		},
		{
			name:            "GCS URL",
			stateURL:        "gcs://my-bucket/path/to/terraform.tfstate",
			expectedBackend: "gcs",
			expectedConfig: map[string]string{
				"bucket": "my-bucket",
				"prefix": "path/to/terraform.tfstate",
			},
			expectError: false,
		},
		{
			name:        "Unsupported scheme",
			stateURL:    "http://example.com/terraform.tfstate",
			expectError: true,
		},
		{
			name:        "Invalid URL",
			stateURL:    "not-a-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, config, err := handler.parseRemoteStateURL(tt.stateURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if backend != tt.expectedBackend {
				t.Errorf("Expected backend %s, got %s", tt.expectedBackend, backend)
			}

			for key, expectedValue := range tt.expectedConfig {
				if actualValue, exists := config[key]; !exists {
					t.Errorf("Expected config key %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected config[%s] = %s, got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestRemoteStateHandler_ValidateRemoteConfig(t *testing.T) {
	handler := NewRemoteStateHandler()

	tests := []struct {
		name        string
		backend     string
		config      map[string]string
		expectError bool
	}{
		{
			name:    "Valid S3 config",
			backend: "s3",
			config: map[string]string{
				"bucket": "my-bucket",
				"key":    "terraform.tfstate",
			},
			expectError: false,
		},
		{
			name:    "S3 config missing bucket",
			backend: "s3",
			config: map[string]string{
				"key": "terraform.tfstate",
			},
			expectError: true,
		},
		{
			name:    "Valid Azure config",
			backend: "azurerm",
			config: map[string]string{
				"storage_account_name": "mystorageaccount",
				"container_name":       "mycontainer",
				"key":                  "terraform.tfstate",
			},
			expectError: false,
		},
		{
			name:    "Azure config missing container",
			backend: "azurerm",
			config: map[string]string{
				"storage_account_name": "mystorageaccount",
				"key":                  "terraform.tfstate",
			},
			expectError: true,
		},
		{
			name:    "Valid GCS config",
			backend: "gcs",
			config: map[string]string{
				"bucket": "my-bucket",
				"prefix": "terraform.tfstate",
			},
			expectError: false,
		},
		{
			name:    "GCS config missing bucket",
			backend: "gcs",
			config: map[string]string{
				"prefix": "terraform.tfstate",
			},
			expectError: true,
		},
		{
			name:        "Unsupported backend",
			backend:     "unsupported",
			config:      map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.ValidateRemoteConfig(tt.backend, tt.config)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRemoteStateHandler_GetSupportedBackends(t *testing.T) {
	handler := NewRemoteStateHandler()
	backends := handler.GetSupportedBackends()

	expectedBackends := []string{"s3", "azurerm", "gcs"}

	if len(backends) != len(expectedBackends) {
		t.Errorf("Expected %d backends, got %d", len(expectedBackends), len(backends))
	}

	for _, expected := range expectedBackends {
		found := false
		for _, actual := range backends {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected backend %s not found in supported backends", expected)
		}
	}
}
