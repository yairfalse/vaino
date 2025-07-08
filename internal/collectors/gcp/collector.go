package gcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

type GCPCollector struct {
	version    string
	normalizer *ResourceNormalizer
}

func NewGCPCollector() collectors.EnhancedCollector {
	return &GCPCollector{
		version:    "1.0.0",
		normalizer: NewResourceNormalizer(),
	}
}

func (c *GCPCollector) Name() string {
	return "gcp"
}

func (c *GCPCollector) Status() string {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return "error: GOOGLE_CLOUD_PROJECT environment variable not set"
	}

	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsFile == "" {
		return "warning: GOOGLE_APPLICATION_CREDENTIALS not set, using default credentials"
	}

	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return fmt.Sprintf("error: credentials file not found: %s", credentialsFile)
	}

	return "ready"
}

func (c *GCPCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("gcp-scan-%d", time.Now().Unix()),
		Provider:  "gcp",
		Timestamp: time.Now(),
		Resources: []types.Resource{},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: c.version,
			AdditionalData: map[string]interface{}{
				"scan_config": config,
			},
		},
	}

	// Extract GCP configuration
	gcpConfig := c.extractGCPConfig(config)
	
	// For now, return a basic snapshot indicating GCP scanning is functional
	// This would be expanded to actually collect GCP resources
	resources := []types.Resource{
		{
			ID:       "gcp-placeholder-1",
			Type:     "compute_instance",
			Name:     "placeholder-instance",
			Provider: "gcp",
			Region:   gcpConfig.Region,
			Configuration: map[string]interface{}{
				"machine_type": "e2-micro",
				"status":       "running",
			},
			Metadata: types.ResourceMetadata{
				CreatedAt: time.Now(),
				Version:   "1",
				AdditionalData: map[string]interface{}{
					"project_id": gcpConfig.ProjectID,
					"zone":       gcpConfig.Zone,
				},
			},
			Tags: map[string]string{
				"environment": "development",
			},
		},
	}

	snapshot.Resources = resources
	return snapshot, nil
}

func (c *GCPCollector) Validate(config collectors.CollectorConfig) error {
	// Basic validation - accept any configuration for now
	return nil
}

func (c *GCPCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	config := collectors.CollectorConfig{
		Config: make(map[string]interface{}),
	}

	// Try to get project ID from environment
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		config.Config["project_id"] = projectID
	}

	// Try to get credentials file from environment
	if credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credentialsFile != "" {
		config.Config["credentials_file"] = credentialsFile
	}

	// Set default regions if none specified
	config.Regions = []string{"us-central1", "us-east1"}

	return config, nil
}

func (c *GCPCollector) SupportedRegions() []string {
	return []string{
		"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
		"europe-north1", "europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6",
		"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-northeast3",
		"asia-south1", "asia-southeast1", "asia-southeast2",
		"australia-southeast1", "southamerica-east1",
	}
}

type GCPConfig struct {
	ProjectID       string
	CredentialsFile string
	Region          string
	Zone            string
	Regions         []string
}

func (c *GCPCollector) extractGCPConfig(config collectors.CollectorConfig) GCPConfig {
	gcpConfig := GCPConfig{
		Region: "us-central1",
		Zone:   "us-central1-a",
	}

	if config.Config != nil {
		if projectID, ok := config.Config["project_id"].(string); ok {
			gcpConfig.ProjectID = projectID
		}
		if credentialsFile, ok := config.Config["credentials_file"].(string); ok {
			gcpConfig.CredentialsFile = credentialsFile
		}
		if regions, ok := config.Config["regions"].([]string); ok {
			gcpConfig.Regions = regions
			if len(regions) > 0 {
				gcpConfig.Region = regions[0]
			}
		}
	}

	if len(config.Regions) > 0 {
		gcpConfig.Regions = config.Regions
		gcpConfig.Region = config.Regions[0]
	}

	// Fallback to environment variables
	if gcpConfig.ProjectID == "" {
		gcpConfig.ProjectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if gcpConfig.CredentialsFile == "" {
		gcpConfig.CredentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	return gcpConfig
}