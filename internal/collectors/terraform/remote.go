package terraform

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yairfalse/vaino/pkg/types"
	"google.golang.org/api/option"
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
	bucket := config["bucket"]
	key := config["key"]
	region := config["region"]

	if bucket == "" || key == "" {
		return nil, fmt.Errorf("S3 bucket and key are required")
	}

	// Load AWS configuration
	var awsConfig aws.Config
	var err error
	if region != "" {
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	} else {
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig)

	// Download the state file
	result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download state file from S3 s3://%s/%s: %w", bucket, key, err)
	}
	defer result.Body.Close()

	// Read the state file content
	stateContent, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 state file content: %w", err)
	}

	// Parse the state file
	tfState, err := h.parser.ParseStateFromBytes(stateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 state file: %w", err)
	}

	// Normalize resources
	resources, err := h.normalizer.NormalizeResources(tfState)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize S3 state resources: %w", err)
	}

	// Add remote state metadata to each resource
	for i := range resources {
		if resources[i].Metadata.AdditionalData == nil {
			resources[i].Metadata.AdditionalData = make(map[string]interface{})
		}
		resources[i].Metadata.AdditionalData["remote_backend"] = "s3"
		resources[i].Metadata.AdditionalData["remote_location"] = fmt.Sprintf("s3://%s/%s", bucket, key)
		resources[i].Metadata.StateFile = fmt.Sprintf("s3://%s/%s", bucket, key)
	}

	return resources, nil
}

// collectFromAzureState fetches state from Azure Storage
func (h *RemoteStateHandler) collectFromAzureState(ctx context.Context, config map[string]string) ([]types.Resource, error) {
	storageAccount := config["storage_account_name"]
	containerName := config["container_name"]
	blobKey := config["key"]

	if storageAccount == "" || containerName == "" || blobKey == "" {
		return nil, fmt.Errorf("Azure storage account, container, and key are required")
	}

	// Construct the Azure Storage URL
	// Azure Storage URL format: https://<account>.blob.core.windows.net/<container>/<blob>
	blobURLString := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccount, containerName, blobKey)

	// Create Azure blob URL (using anonymous access or SAS token from environment)
	// In a production environment, you would use proper authentication
	parsedURL, err := url.Parse(blobURLString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Azure blob URL: %w", err)
	}
	blobURL := azblob.NewBlobURL(*parsedURL, azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))

	// Download the blob
	response, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download state file from Azure Storage %s: %w", blobURLString, err)
	}
	defer response.Body(azblob.RetryReaderOptions{}).Close()

	// Read the state file content
	stateContent, err := io.ReadAll(response.Body(azblob.RetryReaderOptions{}))
	if err != nil {
		return nil, fmt.Errorf("failed to read Azure blob content: %w", err)
	}

	// Parse the state file
	tfState, err := h.parser.ParseStateFromBytes(stateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Azure state file: %w", err)
	}

	// Normalize resources
	resources, err := h.normalizer.NormalizeResources(tfState)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize Azure state resources: %w", err)
	}

	// Add remote state metadata to each resource
	for i := range resources {
		if resources[i].Metadata.AdditionalData == nil {
			resources[i].Metadata.AdditionalData = make(map[string]interface{})
		}
		resources[i].Metadata.AdditionalData["remote_backend"] = "azurerm"
		resources[i].Metadata.AdditionalData["remote_location"] = blobURLString
		resources[i].Metadata.StateFile = blobURLString
	}

	return resources, nil
}

// collectFromGCSState fetches state from Google Cloud Storage
func (h *RemoteStateHandler) collectFromGCSState(ctx context.Context, config map[string]string) ([]types.Resource, error) {
	bucket := config["bucket"]
	objectName := config["prefix"]

	if bucket == "" || objectName == "" {
		return nil, fmt.Errorf("GCS bucket and object name are required")
	}

	// Create GCS client with default authentication
	client, err := storage.NewClient(ctx, option.WithScopes(storage.ScopeReadOnly))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Get the object
	object := client.Bucket(bucket).Object(objectName)

	// Create a reader
	reader, err := object.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader for GCS object gs://%s/%s: %w", bucket, objectName, err)
	}
	defer reader.Close()

	// Read the state file content
	stateContent, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read GCS object content: %w", err)
	}

	// Parse the state file
	tfState, err := h.parser.ParseStateFromBytes(stateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GCS state file: %w", err)
	}

	// Normalize resources
	resources, err := h.normalizer.NormalizeResources(tfState)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize GCS state resources: %w", err)
	}

	// Add remote state metadata to each resource
	for i := range resources {
		if resources[i].Metadata.AdditionalData == nil {
			resources[i].Metadata.AdditionalData = make(map[string]interface{})
		}
		resources[i].Metadata.AdditionalData["remote_backend"] = "gcs"
		resources[i].Metadata.AdditionalData["remote_location"] = fmt.Sprintf("gs://%s/%s", bucket, objectName)
		resources[i].Metadata.StateFile = fmt.Sprintf("gs://%s/%s", bucket, objectName)
	}

	return resources, nil
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
