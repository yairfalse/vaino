package terraform

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/yairfalse/vaino/pkg/types"
)

// RemoteStateHandler handles remote Terraform state backends
type RemoteStateHandler struct {
	parser     *StateParser
	normalizer *ResourceNormalizer
}

// NewRemoteStateHandler creates a new remote state handler
func NewRemoteStateHandler() *RemoteStateHandler {
	return &RemoteStateHandler{
		parser:     NewStateParser(),
		normalizer: NewResourceNormalizer(),
	}
}

// CollectFromRemoteState fetches and processes remote Terraform state
func (h *RemoteStateHandler) CollectFromRemoteState(ctx context.Context, stateURL string) ([]types.Resource, error) {
	// Parse the remote state URL
	backend, config, err := h.parseRemoteStateURL(stateURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote state URL: %w", err)
	}

	// Fetch state based on backend type
	switch backend {
	case "s3":
		return h.collectFromS3State(ctx, config)
	case "azurerm":
		return h.collectFromAzureState(ctx, config)
	case "gcs":
		return h.collectFromGCSState(ctx, config)
	default:
		return nil, fmt.Errorf("unsupported remote backend: %s", backend)
	}
}

// parseRemoteStateURL parses a remote state URL and extracts backend type and configuration
func (h *RemoteStateHandler) parseRemoteStateURL(stateURL string) (string, map[string]string, error) {
	u, err := url.Parse(stateURL)
	if err != nil {
		return "", nil, fmt.Errorf("invalid URL: %w", err)
	}

	config := make(map[string]string)

	switch u.Scheme {
	case "s3":
		// s3://bucket-name/path/to/terraform.tfstate
		config["bucket"] = u.Host
		config["key"] = strings.TrimPrefix(u.Path, "/")
		if region := u.Query().Get("region"); region != "" {
			config["region"] = region
		}
		return "s3", config, nil

	case "azurerm":
		// azurerm://storageaccount/container/terraform.tfstate
		config["storage_account_name"] = u.Host
		pathParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(pathParts) >= 1 {
			config["container_name"] = pathParts[0]
		}
		if len(pathParts) >= 2 {
			config["key"] = strings.Join(pathParts[1:], "/")
		}
		return "azurerm", config, nil

	case "gcs":
		// gcs://bucket-name/path/to/terraform.tfstate
		config["bucket"] = u.Host
		config["prefix"] = strings.TrimPrefix(u.Path, "/")
		return "gcs", config, nil

	default:
		return "", nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

// collectFromS3State fetches state from AWS S3
func (h *RemoteStateHandler) collectFromS3State(ctx context.Context, config map[string]string) ([]types.Resource, error) {
	// For now, return an error indicating that remote state collection requires additional setup
	// In a full implementation, this would:
	// 1. Set up AWS SDK client
	// 2. Download the state file from S3
	// 3. Parse and normalize the resources

	return nil, fmt.Errorf("S3 remote state collection requires AWS credentials and SDK setup - not implemented yet. Bucket: %s, Key: %s",
		config["bucket"], config["key"])
}

// collectFromAzureState fetches state from Azure Storage
func (h *RemoteStateHandler) collectFromAzureState(ctx context.Context, config map[string]string) ([]types.Resource, error) {
	// For now, return an error indicating that remote state collection requires additional setup
	// In a full implementation, this would:
	// 1. Set up Azure SDK client
	// 2. Download the state file from Azure Storage
	// 3. Parse and normalize the resources

	return nil, fmt.Errorf("Azure remote state collection requires Azure credentials and SDK setup - not implemented yet. Storage Account: %s, Container: %s",
		config["storage_account_name"], config["container_name"])
}

// collectFromGCSState fetches state from Google Cloud Storage
func (h *RemoteStateHandler) collectFromGCSState(ctx context.Context, config map[string]string) ([]types.Resource, error) {
	// For now, return an error indicating that remote state collection requires additional setup
	// In a full implementation, this would:
	// 1. Set up GCS SDK client
	// 2. Download the state file from GCS
	// 3. Parse and normalize the resources

	return nil, fmt.Errorf("GCS remote state collection requires GCP credentials and SDK setup - not implemented yet. Bucket: %s, Prefix: %s",
		config["bucket"], config["prefix"])
}

// GetSupportedBackends returns the list of supported remote backends
func (h *RemoteStateHandler) GetSupportedBackends() []string {
	return []string{"s3", "azurerm", "gcs"}
}

// ValidateRemoteConfig validates remote state configuration
func (h *RemoteStateHandler) ValidateRemoteConfig(backend string, config map[string]string) error {
	switch backend {
	case "s3":
		if config["bucket"] == "" {
			return fmt.Errorf("S3 bucket is required")
		}
		if config["key"] == "" {
			return fmt.Errorf("S3 key is required")
		}

	case "azurerm":
		if config["storage_account_name"] == "" {
			return fmt.Errorf("Azure storage account name is required")
		}
		if config["container_name"] == "" {
			return fmt.Errorf("Azure container name is required")
		}
		if config["key"] == "" {
			return fmt.Errorf("Azure blob key is required")
		}

	case "gcs":
		if config["bucket"] == "" {
			return fmt.Errorf("GCS bucket is required")
		}
		if config["prefix"] == "" {
			return fmt.Errorf("GCS object prefix is required")
		}

	default:
		return fmt.Errorf("unsupported backend: %s", backend)
	}

	return nil
}

// GetRemoteStateInfo returns information about a remote state configuration
func (h *RemoteStateHandler) GetRemoteStateInfo(stateURL string) (map[string]interface{}, error) {
	backend, config, err := h.parseRemoteStateURL(stateURL)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"backend": backend,
		"config":  config,
		"status":  "configured",
	}

	// Add backend-specific information
	switch backend {
	case "s3":
		info["description"] = fmt.Sprintf("AWS S3 bucket %s, key %s", config["bucket"], config["key"])
		if region, exists := config["region"]; exists {
			info["region"] = region
		}

	case "azurerm":
		info["description"] = fmt.Sprintf("Azure Storage %s, container %s", config["storage_account_name"], config["container_name"])

	case "gcs":
		info["description"] = fmt.Sprintf("Google Cloud Storage bucket %s", config["bucket"])
	}

	return info, nil
}
