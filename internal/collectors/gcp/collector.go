package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	vainoerrors "github.com/yairfalse/vaino/internal/errors"
	"github.com/yairfalse/vaino/pkg/types"
)

type GCPCollector struct {
	version    string
	normalizer *ResourceNormalizer
}

func NewGCPCollector() collectors.Collector {
	return &GCPCollector{
		version:    "1.0.0",
		normalizer: NewResourceNormalizer(),
	}
}

func (c *GCPCollector) Name() string {
	return "gcp"
}

func (c *GCPCollector) Status() string {
	// Check project ID availability
	projectID := c.getProjectID()
	if projectID == "" {
		return "error: no GCP project configured"
	}

	// Check credentials
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsFile != "" {
		// Validate credentials file
		if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
			return fmt.Sprintf("error: credentials file not found: %s", credentialsFile)
		}

		// Try to validate credentials file format
		if err := c.validateCredentialsFile(credentialsFile); err != nil {
			return fmt.Sprintf("error: invalid credentials file: %v", err)
		}

		return fmt.Sprintf("ready (project: %s, service account)", projectID)
	}

	// Check for application default credentials
	if c.hasDefaultCredentials() {
		return fmt.Sprintf("ready (project: %s, application default)", projectID)
	}

	// Check for gcloud user credentials
	if c.hasGcloudCredentials() {
		return fmt.Sprintf("ready (project: %s, gcloud user)", projectID)
	}

	return "error: no GCP credentials configured"
}

func (c *GCPCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	startTime := time.Now()

	// Extract GCP configuration
	gcpConfig := c.extractGCPConfig(config)

	// Validate credentials
	if err := c.validateCredentials(gcpConfig); err != nil {
		return nil, err
	}

	// Initialize GCP clients
	clientPool, err := NewGCPClientPool(ctx, GCPClientConfig{
		ProjectID:       gcpConfig.ProjectID,
		CredentialsFile: gcpConfig.CredentialsFile,
		Regions:         gcpConfig.Regions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCP clients: %w", err)
	}

	var allResources []types.Resource

	// Collect resources from different services
	services := []struct {
		name      string
		collector func(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error)
	}{
		{"Compute Engine", c.collectComputeResources},
		{"Storage", c.collectStorageResources},
		{"Network", c.collectNetworkResources},
		{"Cloud SQL", c.collectCloudSQLResources},
		{"IAM", c.collectIAMResources},
		{"Container Engine", c.collectContainerResources},
	}

	for _, service := range services {
		resources, err := service.collector(ctx, clientPool, gcpConfig.ProjectID, gcpConfig.Regions)
		if err != nil {
			// Check for authentication errors
			if isGCPAuthError(err) {
				return nil, vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
					fmt.Sprintf("Authentication failed for %s service", service.name)).
					WithCause(err.Error()).
					WithSolutions(
						"Verify GCP credentials are configured correctly",
						"Check GOOGLE_APPLICATION_CREDENTIALS environment variable",
						"Ensure service account has required permissions",
						"Verify project ID is correct",
					).
					WithVerify("gcloud auth application-default print-access-token").
					WithHelp("vaino validate gcp")
			}

			// For other errors, log and continue
			fmt.Printf("Warning: Failed to collect %s resources: %v\n", service.name, err)
			continue
		}
		allResources = append(allResources, resources...)
	}

	collectionTime := time.Since(startTime)

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("gcp-scan-%d", time.Now().Unix()),
		Provider:  "gcp",
		Timestamp: time.Now(),
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: c.version,
			CollectionTime:   collectionTime,
			ResourceCount:    len(allResources),
			Regions:          gcpConfig.Regions,
			AdditionalData: map[string]interface{}{
				"project_id":    gcpConfig.ProjectID,
				"scan_config":   config,
				"regions_count": len(gcpConfig.Regions),
			},
		},
	}

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
	} else if gcloudProject := c.getGcloudProject(); gcloudProject != "" {
		config.Config["project_id"] = gcloudProject
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

	// Try gcloud config if still no project ID
	if gcpConfig.ProjectID == "" {
		gcpConfig.ProjectID = c.getGcloudProject()
	}

	if gcpConfig.CredentialsFile == "" {
		gcpConfig.CredentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	return gcpConfig
}

// collectComputeResources collects GCP Compute Engine resources
func (c *GCPCollector) collectComputeResources(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error) {
	var resources []types.Resource

	for _, region := range regions {
		// Get compute instances
		instances, err := clientPool.GetComputeInstances(ctx, projectID, region)
		if err != nil {
			return nil, fmt.Errorf("failed to get compute instances in region %s: %w", region, err)
		}

		for _, instance := range instances {
			resource := c.normalizer.NormalizeComputeInstance(instance)
			resource.Region = region
			resources = append(resources, resource)
		}

		// Get persistent disks
		disks, err := clientPool.GetPersistentDisks(ctx, projectID, region)
		if err != nil {
			return nil, fmt.Errorf("failed to get persistent disks in region %s: %w", region, err)
		}

		for _, disk := range disks {
			resource := c.normalizer.NormalizePersistentDisk(disk)
			resource.Region = region
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// collectStorageResources collects GCP Storage resources
func (c *GCPCollector) collectStorageResources(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error) {
	var resources []types.Resource

	// Get storage buckets (global)
	buckets, err := clientPool.GetStorageBuckets(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage buckets: %w", err)
	}

	for _, bucket := range buckets {
		resource := c.normalizer.NormalizeStorageBucket(bucket)
		resources = append(resources, resource)
	}

	return resources, nil
}

// collectNetworkResources collects GCP Network resources
func (c *GCPCollector) collectNetworkResources(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error) {
	var resources []types.Resource

	// Get VPC networks (global)
	networks, err := clientPool.GetVPCNetworks(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPC networks: %w", err)
	}

	for _, network := range networks {
		resource := c.normalizer.NormalizeVPCNetwork(network)
		resources = append(resources, resource)
	}

	for _, region := range regions {
		// Get subnets
		subnets, err := clientPool.GetSubnets(ctx, projectID, region)
		if err != nil {
			return nil, fmt.Errorf("failed to get subnets in region %s: %w", region, err)
		}

		for _, subnet := range subnets {
			resource := c.normalizer.NormalizeSubnet(subnet)
			resource.Region = region
			resources = append(resources, resource)
		}

		// Get firewall rules
		firewalls, err := clientPool.GetFirewallRules(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get firewall rules: %w", err)
		}

		for _, firewall := range firewalls {
			resource := c.normalizer.NormalizeFirewallRule(firewall)
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// isGCPAuthError checks if an error is related to GCP authentication
func isGCPAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	authErrorPatterns := []string{
		"permission denied",
		"unauthorized",
		"invalid authentication",
		"unauthenticated",
		"credentials",
		"oauth",
		"token",
		"service account",
		"application default credentials",
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// validateCredentials validates GCP credentials
func (c *GCPCollector) validateCredentials(config GCPConfig) error {
	// Check project ID first
	if config.ProjectID == "" {
		return vainoerrors.New(vainoerrors.ErrorTypeConfiguration, vainoerrors.ProviderGCP,
			"GCP project ID not specified").
			WithCause("No project_id found in configuration, environment, or gcloud config").
			WithSolutions(
				"Set GOOGLE_CLOUD_PROJECT environment variable",
				"Specify project_id with --project flag",
				"Run 'gcloud config set project YOUR-PROJECT-ID'",
			).
			WithVerify("gcloud config get-value project").
			WithHelp("vaino validate gcp")
	}

	// Check if credentials file is specified and exists
	if config.CredentialsFile != "" {
		if _, err := os.Stat(config.CredentialsFile); os.IsNotExist(err) {
			return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
				"GCP service account credentials file not found").
				WithCause(fmt.Sprintf("File does not exist: %s", config.CredentialsFile)).
				WithSolutions(
					"Check the path to your service account key file",
					"Set GOOGLE_APPLICATION_CREDENTIALS to the correct path",
					"Download a new service account key from GCP Console",
				).
				WithVerify("ls -la \"$GOOGLE_APPLICATION_CREDENTIALS\"").
				WithHelp("vaino validate gcp")
		}

		// Try to read and parse the credentials file
		content, err := os.ReadFile(config.CredentialsFile)
		if err != nil {
			return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
				"Failed to read GCP credentials file").
				WithCause(err.Error()).
				WithSolutions(
					"Check file permissions for the credentials file",
					"Ensure the file is readable by the current user",
				).
				WithVerify("cat \"$GOOGLE_APPLICATION_CREDENTIALS\"").
				WithHelp("vaino validate gcp")
		}

		// Try to parse as JSON
		var creds map[string]interface{}
		if err := json.Unmarshal(content, &creds); err != nil {
			return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
				"Invalid GCP service account credentials file format").
				WithCause("Failed to parse JSON: "+err.Error()).
				WithSolutions(
					"Verify the credentials file is valid JSON",
					"Download a fresh service account key from GCP Console",
					"Check for file corruption or incomplete download",
				).
				WithVerify("python -m json.tool \"$GOOGLE_APPLICATION_CREDENTIALS\"").
				WithHelp("vaino validate gcp")
		}

		// Check credential type and validate accordingly
		credType, hasType := creds["type"].(string)
		if !hasType {
			return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
				"GCP credentials file missing required field: type").
				WithCause("Incomplete credentials file").
				WithSolutions(
					"Download a complete credentials file from GCP Console",
					"Ensure the credentials file was not truncated during download",
				).
				WithVerify("python -c \"import json; print(json.load(open('$GOOGLE_APPLICATION_CREDENTIALS')).keys())\"").
				WithHelp("vaino validate gcp")
		}

		// Validate fields based on credential type
		if credType == "service_account" {
			// Service account key - validate all required fields
			requiredFields := []string{"type", "project_id", "private_key_id", "private_key"}
			for _, field := range requiredFields {
				if _, exists := creds[field]; !exists {
					return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
						fmt.Sprintf("GCP service account credentials missing required field: %s", field)).
						WithCause("Incomplete service account key file").
						WithSolutions(
							"Download a complete service account key from GCP Console",
							"Ensure the key file was not truncated during download",
						).
						WithVerify("python -c \"import json; print(json.load(open('$GOOGLE_APPLICATION_CREDENTIALS')).keys())\"").
						WithHelp("vaino validate gcp")
				}
			}
		} else if credType == "authorized_user" {
			// Application Default Credentials - validate required fields
			requiredFields := []string{"type", "client_id", "client_secret", "refresh_token"}
			for _, field := range requiredFields {
				if _, exists := creds[field]; !exists {
					return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
						fmt.Sprintf("GCP application default credentials missing required field: %s", field)).
						WithCause("Incomplete application default credentials").
						WithSolutions(
							"Run 'gcloud auth application-default login' to refresh credentials",
							"Ensure gcloud is properly authenticated",
						).
						WithVerify("gcloud auth application-default print-access-token").
						WithHelp("vaino validate gcp")
				}
			}
		} else {
			return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
				fmt.Sprintf("Unsupported GCP credential type: %s", credType)).
				WithCause("Unknown credential type in credentials file").
				WithSolutions(
					"Use a service account key for production environments",
					"Use 'gcloud auth application-default login' for development",
				).
				WithHelp("vaino validate gcp")
		}

		// Check if the private key looks valid (only for service accounts)
		if credType == "service_account" {
			if privateKey, ok := creds["private_key"].(string); ok {
				if !strings.Contains(privateKey, "BEGIN PRIVATE KEY") {
					return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderGCP,
						"GCP service account private key appears to be invalid").
						WithCause("Private key does not contain expected PEM format").
						WithSolutions(
							"Download a new service account key from GCP Console",
							"Ensure the key file was not modified or corrupted",
						).
						WithHelp("vaino validate gcp")
				}
			}
		}
	} else {
		// No explicit credentials file specified, try to use default authentication
		// Check if default application credentials exist
		if !c.hasDefaultCredentials() {
			return vainoerrors.New(vainoerrors.ErrorTypeConfiguration, vainoerrors.ProviderGCP,
				"No GCP credentials configured").
				WithCause("GOOGLE_APPLICATION_CREDENTIALS not set and no default credentials found").
				WithSolutions(
					"Run 'gcloud auth application-default login' to set up default credentials",
					"Set GOOGLE_APPLICATION_CREDENTIALS environment variable",
					"Download a service account key from GCP Console",
				).
				WithVerify("gcloud auth application-default print-access-token").
				WithHelp("vaino validate gcp")
		}
		// If we have default credentials, proceed without explicit file
	}

	return nil
}

// getGcloudProject tries to get the project ID from gcloud config
func (c *GCPCollector) getGcloudProject() string {
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	projectID := strings.TrimSpace(string(output))
	if projectID == "(unset)" {
		return ""
	}
	return projectID
}

// hasDefaultCredentials checks if default application credentials are available
func (c *GCPCollector) hasDefaultCredentials() bool {
	// Check if gcloud auth application-default print-access-token works
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	err := cmd.Run()
	if err == nil {
		return true
	}

	// Check if default credentials file exists in standard location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	defaultCredPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
	if _, err := os.Stat(defaultCredPath); err == nil {
		return true
	}

	return false
}

// hasGcloudCredentials checks if gcloud user credentials are available
func (c *GCPCollector) hasGcloudCredentials() bool {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	err := cmd.Run()
	return err == nil
}

// getProjectID gets the project ID from multiple sources in priority order
func (c *GCPCollector) getProjectID() string {
	// 1. Environment variable
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return projectID
	}

	// 2. GCP_PROJECT environment variable (alternative)
	if projectID := os.Getenv("GCP_PROJECT"); projectID != "" {
		return projectID
	}

	// 3. Gcloud config
	if projectID := c.getGcloudProject(); projectID != "" {
		return projectID
	}

	// 4. Try to extract from credentials file
	if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
		if projectID := c.getProjectFromCredentialsFile(credFile); projectID != "" {
			return projectID
		}
	}

	// 5. Try default application credentials file
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		defaultCredPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
		if projectID := c.getProjectFromCredentialsFile(defaultCredPath); projectID != "" {
			return projectID
		}
	}

	return ""
}

// getProjectFromCredentialsFile extracts project ID from a credentials file
func (c *GCPCollector) getProjectFromCredentialsFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	var creds map[string]interface{}
	if err := json.Unmarshal(content, &creds); err != nil {
		return ""
	}

	// For service account credentials
	if projectID, ok := creds["project_id"].(string); ok {
		return projectID
	}

	// For application default credentials, project might be in quota_project_id
	if projectID, ok := creds["quota_project_id"].(string); ok {
		return projectID
	}

	return ""
}

// validateCredentialsFile validates the format and content of a GCP credentials file
func (c *GCPCollector) validateCredentialsFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read credentials file: %w", err)
	}

	var creds map[string]interface{}
	if err := json.Unmarshal(content, &creds); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Check credential type
	credType, hasType := creds["type"].(string)
	if !hasType {
		return fmt.Errorf("missing 'type' field in credentials")
	}

	switch credType {
	case "service_account":
		return c.validateServiceAccountCredentials(creds)
	case "authorized_user":
		return c.validateAuthorizedUserCredentials(creds)
	default:
		return fmt.Errorf("unsupported credential type: %s", credType)
	}
}

// validateServiceAccountCredentials validates service account credentials
func (c *GCPCollector) validateServiceAccountCredentials(creds map[string]interface{}) error {
	requiredFields := []string{"type", "project_id", "private_key_id", "private_key", "client_email", "client_id", "auth_uri", "token_uri"}

	for _, field := range requiredFields {
		if _, exists := creds[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate private key format
	if privateKey, ok := creds["private_key"].(string); ok {
		if !strings.Contains(privateKey, "BEGIN PRIVATE KEY") {
			return fmt.Errorf("private key does not appear to be in PEM format")
		}
	}

	// Validate email format
	if email, ok := creds["client_email"].(string); ok {
		if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
			return fmt.Errorf("client_email appears to be invalid: %s", email)
		}
	}

	return nil
}

// validateAuthorizedUserCredentials validates authorized user credentials
func (c *GCPCollector) validateAuthorizedUserCredentials(creds map[string]interface{}) error {
	requiredFields := []string{"type", "client_id", "client_secret", "refresh_token"}

	for _, field := range requiredFields {
		if _, exists := creds[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}

func (c *GCPCollector) CollectSeparate(ctx context.Context, config collectors.CollectorConfig) ([]*types.Snapshot, error) {
	return nil, fmt.Errorf("separate collection not supported by %s collector", c.Name())
}
