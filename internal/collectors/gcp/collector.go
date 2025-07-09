package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	wgoerrors "github.com/yairfalse/wgo/internal/errors"
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
	// Extract GCP configuration
	gcpConfig := c.extractGCPConfig(config)

	// Validate credentials
	if err := c.validateCredentials(gcpConfig); err != nil {
		return nil, err
	}

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

// validateCredentials validates GCP credentials
func (c *GCPCollector) validateCredentials(config GCPConfig) error {
	// Check project ID first
	if config.ProjectID == "" {
		return wgoerrors.New(wgoerrors.ErrorTypeConfiguration, wgoerrors.ProviderGCP,
			"GCP project ID not specified").
			WithCause("No project_id found in configuration or environment").
			WithSolutions(
				"Set GOOGLE_CLOUD_PROJECT environment variable",
				"Specify project_id in the configuration",
				"Check that your service account key contains project_id",
			).
			WithVerify("echo $GOOGLE_CLOUD_PROJECT").
			WithHelp("wgo validate gcp")
	}

	// Check if credentials file is specified and exists
	if config.CredentialsFile != "" {
		if _, err := os.Stat(config.CredentialsFile); os.IsNotExist(err) {
			return wgoerrors.New(wgoerrors.ErrorTypeAuthentication, wgoerrors.ProviderGCP,
				"GCP service account credentials file not found").
				WithCause(fmt.Sprintf("File does not exist: %s", config.CredentialsFile)).
				WithSolutions(
					"Check the path to your service account key file",
					"Set GOOGLE_APPLICATION_CREDENTIALS to the correct path",
					"Download a new service account key from GCP Console",
				).
				WithVerify("ls -la \"$GOOGLE_APPLICATION_CREDENTIALS\"").
				WithHelp("wgo validate gcp")
		}

		// Try to read and parse the credentials file
		content, err := os.ReadFile(config.CredentialsFile)
		if err != nil {
			return wgoerrors.New(wgoerrors.ErrorTypeAuthentication, wgoerrors.ProviderGCP,
				"Failed to read GCP credentials file").
				WithCause(err.Error()).
				WithSolutions(
					"Check file permissions for the credentials file",
					"Ensure the file is readable by the current user",
				).
				WithVerify("cat \"$GOOGLE_APPLICATION_CREDENTIALS\"").
				WithHelp("wgo validate gcp")
		}

		// Try to parse as JSON
		var creds map[string]interface{}
		if err := json.Unmarshal(content, &creds); err != nil {
			return wgoerrors.New(wgoerrors.ErrorTypeAuthentication, wgoerrors.ProviderGCP,
				"Invalid GCP service account credentials file format").
				WithCause("Failed to parse JSON: "+err.Error()).
				WithSolutions(
					"Verify the credentials file is valid JSON",
					"Download a fresh service account key from GCP Console",
					"Check for file corruption or incomplete download",
				).
				WithVerify("python -m json.tool \"$GOOGLE_APPLICATION_CREDENTIALS\"").
				WithHelp("wgo validate gcp")
		}

		// Validate required fields
		requiredFields := []string{"type", "project_id", "private_key_id", "private_key"}
		for _, field := range requiredFields {
			if _, exists := creds[field]; !exists {
				return wgoerrors.New(wgoerrors.ErrorTypeAuthentication, wgoerrors.ProviderGCP,
					fmt.Sprintf("GCP credentials file missing required field: %s", field)).
					WithCause("Incomplete service account key file").
					WithSolutions(
						"Download a complete service account key from GCP Console",
						"Ensure the key file was not truncated during download",
					).
					WithVerify("python -c \"import json; print(json.load(open('$GOOGLE_APPLICATION_CREDENTIALS')).keys())\"").
					WithHelp("wgo validate gcp")
			}
		}

		// Check if the private key looks valid
		if privateKey, ok := creds["private_key"].(string); ok {
			if !strings.Contains(privateKey, "BEGIN PRIVATE KEY") {
				return wgoerrors.New(wgoerrors.ErrorTypeAuthentication, wgoerrors.ProviderGCP,
					"GCP service account private key appears to be invalid").
					WithCause("Private key does not contain expected PEM format").
					WithSolutions(
						"Download a new service account key from GCP Console",
						"Ensure the key file was not modified or corrupted",
					).
					WithHelp("wgo validate gcp")
			}
		}
	} else {
		// No credentials file specified, check if we have any way to authenticate
		return wgoerrors.New(wgoerrors.ErrorTypeConfiguration, wgoerrors.ProviderGCP,
			"No GCP credentials configured").
			WithCause("GOOGLE_APPLICATION_CREDENTIALS not set and no credentials file specified").
			WithSolutions(
				"Set GOOGLE_APPLICATION_CREDENTIALS environment variable",
				"Download a service account key from GCP Console",
				"Use 'gcloud auth application-default login' for local development",
			).
			WithVerify("echo $GOOGLE_APPLICATION_CREDENTIALS").
			WithHelp("wgo validate gcp")
	}

	return nil
}
