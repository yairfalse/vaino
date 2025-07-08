package gcp

import (
	"context"
	"os"
	"testing"

	"github.com/yairfalse/wgo/internal/collectors"
)

func TestGCPCollector_Name(t *testing.T) {
	collector := NewGCPCollector()
	if collector.Name() != "gcp" {
		t.Errorf("Expected Name() to return 'gcp', got %s", collector.Name())
	}
}

func TestGCPCollector_Status(t *testing.T) {
	collector := NewGCPCollector()
	
	// Test without environment variables
	status := collector.Status()
	if status == "ready" {
		t.Error("Expected Status() to return error without GOOGLE_CLOUD_PROJECT")
	}
	
	// Test with project ID but no credentials
	os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	defer os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	
	status = collector.Status()
	if status != "warning: GOOGLE_APPLICATION_CREDENTIALS not set, using default credentials" {
		t.Errorf("Expected warning about credentials, got: %s", status)
	}
	
	// Test with both project and credentials file (that doesn't exist)
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/nonexistent/path.json")
	defer os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	
	status = collector.Status()
	if status != "error: credentials file not found: /nonexistent/path.json" {
		t.Errorf("Expected error about missing file, got: %s", status)
	}
}

func TestGCPCollector_Validate(t *testing.T) {
	collector := NewGCPCollector()
	
	tests := []struct {
		name    string
		config  collectors.CollectorConfig
		wantErr bool
	}{
		{
			name:    "empty config should validate",
			config:  collectors.CollectorConfig{},
			wantErr: false,
		},
		{
			name: "config with project should validate",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"project_id": "test-project",
				},
			},
			wantErr: false,
		},
		{
			name: "config with regions should validate",
			config: collectors.CollectorConfig{
				Regions: []string{"us-central1", "us-east1"},
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

func TestGCPCollector_SupportedRegions(t *testing.T) {
	collector := NewGCPCollector()
	regions := collector.SupportedRegions()
	
	if len(regions) == 0 {
		t.Error("Expected SupportedRegions() to return non-empty slice")
	}
	
	// Check for some expected regions
	expectedRegions := []string{"us-central1", "us-east1", "europe-west1", "asia-east1"}
	regionMap := make(map[string]bool)
	for _, region := range regions {
		regionMap[region] = true
	}
	
	for _, expected := range expectedRegions {
		if !regionMap[expected] {
			t.Errorf("Expected region %s not found in supported regions", expected)
		}
	}
}

func TestGCPCollector_AutoDiscover(t *testing.T) {
	collector := NewGCPCollector()
	
	// Test auto-discover without environment variables
	config, err := collector.AutoDiscover()
	if err != nil {
		t.Errorf("AutoDiscover() failed: %v", err)
	}
	
	if config.Config == nil {
		t.Error("Expected AutoDiscover() to return config with non-nil Config map")
	}
	
	if len(config.Regions) == 0 {
		t.Error("Expected AutoDiscover() to return default regions")
	}
	
	// Test with environment variables
	os.Setenv("GOOGLE_CLOUD_PROJECT", "test-project")
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/path/to/creds.json")
	defer func() {
		os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
	}()
	
	config, err = collector.AutoDiscover()
	if err != nil {
		t.Errorf("AutoDiscover() failed with env vars: %v", err)
	}
	
	if projectID, ok := config.Config["project_id"].(string); !ok || projectID != "test-project" {
		t.Errorf("Expected project_id to be 'test-project', got %v", config.Config["project_id"])
	}
	
	if credentialsFile, ok := config.Config["credentials_file"].(string); !ok || credentialsFile != "/path/to/creds.json" {
		t.Errorf("Expected credentials_file to be '/path/to/creds.json', got %v", config.Config["credentials_file"])
	}
}

func TestGCPCollector_Collect(t *testing.T) {
	collector := NewGCPCollector()
	
	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"project_id": "test-project",
		},
		Regions: []string{"us-central1"},
	}
	
	ctx := context.Background()
	snapshot, err := collector.Collect(ctx, config)
	
	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}
	
	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}
	
	if snapshot.Provider != "gcp" {
		t.Errorf("Expected Provider to be 'gcp', got %s", snapshot.Provider)
	}
	
	if len(snapshot.Resources) == 0 {
		t.Error("Expected at least one resource in snapshot")
	}
	
	// Check the placeholder resource
	if len(snapshot.Resources) > 0 {
		resource := snapshot.Resources[0]
		if resource.Provider != "gcp" {
			t.Errorf("Expected resource provider to be 'gcp', got %s", resource.Provider)
		}
		if resource.Type != "compute_instance" {
			t.Errorf("Expected resource type to be 'compute_instance', got %s", resource.Type)
		}
	}
}

func TestGCPCollector_ExtractGCPConfig(t *testing.T) {
	collector := &GCPCollector{}
	
	tests := []struct {
		name           string
		config         collectors.CollectorConfig
		expectedProj   string
		expectedRegion string
	}{
		{
			name:           "empty config uses defaults",
			config:         collectors.CollectorConfig{},
			expectedProj:   "",
			expectedRegion: "us-central1",
		},
		{
			name: "config with project_id",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"project_id": "my-project",
				},
			},
			expectedProj:   "my-project",
			expectedRegion: "us-central1",
		},
		{
			name: "config with regions",
			config: collectors.CollectorConfig{
				Regions: []string{"europe-west1", "us-east1"},
			},
			expectedProj:   "",
			expectedRegion: "europe-west1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gcpConfig := collector.extractGCPConfig(tt.config)
			
			if gcpConfig.ProjectID != tt.expectedProj {
				t.Errorf("Expected ProjectID %s, got %s", tt.expectedProj, gcpConfig.ProjectID)
			}
			
			if gcpConfig.Region != tt.expectedRegion {
				t.Errorf("Expected Region %s, got %s", tt.expectedRegion, gcpConfig.Region)
			}
		})
	}
}