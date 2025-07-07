package gcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// Collector implements the collectors.Collector interface for GCP
type Collector struct {
	client            *Client
	resourceCollector *ResourceCollector
	config            CollectorConfig
}

// CollectorConfig holds configuration for the GCP collector
type CollectorConfig struct {
	ProjectID       string
	CredentialsFile string
	Regions         []string
	ResourceTypes   []string
}

// NewGCPCollector creates a new GCP collector
func NewGCPCollector() *Collector {
	return &Collector{}
}

// Name returns the collector name
func (c *Collector) Name() string {
	return "gcp"
}

// Type returns the collector type
func (c *Collector) Type() string {
	return "cloud"
}

// Version returns the collector version
func (c *Collector) Version() string {
	return "1.0.0"
}

// Configure configures the collector with the given options
func (c *Collector) Configure(options map[string]interface{}) error {
	config := CollectorConfig{
		ResourceTypes: []string{"all"}, // Default to all resource types
	}

	// Extract configuration from options
	if projectID, ok := options["project_id"].(string); ok && projectID != "" {
		config.ProjectID = projectID
	} else if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		config.ProjectID = projectID
	}

	if credentialsFile, ok := options["credentials_file"].(string); ok && credentialsFile != "" {
		config.CredentialsFile = credentialsFile
	} else if credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credentialsFile != "" {
		config.CredentialsFile = credentialsFile
	}

	if regions, ok := options["regions"].([]string); ok && len(regions) > 0 {
		config.Regions = regions
	} else if regionsStr, ok := options["regions"].(string); ok && regionsStr != "" {
		config.Regions = strings.Split(regionsStr, ",")
		// Trim spaces
		for i, region := range config.Regions {
			config.Regions[i] = strings.TrimSpace(region)
		}
	}

	if resourceTypes, ok := options["resource_types"].([]string); ok && len(resourceTypes) > 0 {
		config.ResourceTypes = resourceTypes
	} else if resourceTypesStr, ok := options["resource_types"].(string); ok && resourceTypesStr != "" {
		config.ResourceTypes = strings.Split(resourceTypesStr, ",")
		// Trim spaces
		for i, rt := range config.ResourceTypes {
			config.ResourceTypes[i] = strings.TrimSpace(rt)
		}
	}

	c.config = config
	return nil
}

// Initialize initializes the GCP collector
func (c *Collector) Initialize(ctx context.Context) error {
	clientConfig := ClientConfig{
		ProjectID:       c.config.ProjectID,
		CredentialsFile: c.config.CredentialsFile,
		Regions:         c.config.Regions,
	}

	client, err := NewClient(ctx, clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	// Validate access to the project
	if err := client.ValidateAccess(); err != nil {
		return fmt.Errorf("failed to validate GCP access: %w", err)
	}

	c.client = client
	c.resourceCollector = NewResourceCollector(client)

	return nil
}

// IsAvailable checks if the GCP collector can be used
func (c *Collector) IsAvailable(ctx context.Context) bool {
	// Check if we have credentials available
	if c.config.CredentialsFile != "" {
		if _, err := os.Stat(c.config.CredentialsFile); err != nil {
			return false
		}
		return true
	}

	// Check for Application Default Credentials
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		return true
	}

	// Try to detect default credentials
	_, err := NewClient(ctx, ClientConfig{})
	return err == nil
}

// Collect collects resources from GCP
func (c *Collector) Collect(ctx context.Context) ([]types.Resource, error) {
	if c.client == nil {
		return nil, fmt.Errorf("collector not initialized")
	}

	var allResources []types.Resource

	// Determine which resource types to collect
	resourceTypes := c.resolveResourceTypes()

	for _, resourceType := range resourceTypes {
		var resources []types.Resource
		var err error

		switch resourceType {
		case "compute", "instances", "disks", "instance_groups":
			resources, err = c.resourceCollector.CollectComputeResources(ctx)
		case "storage", "buckets":
			resources, err = c.resourceCollector.CollectStorageResources(ctx)
		case "networking", "networks", "subnets", "firewalls":
			resources, err = c.resourceCollector.CollectNetworkResources(ctx)
		case "iam", "service_accounts":
			resources, err = c.resourceCollector.CollectIAMResources(ctx)
		case "all":
			resources, err = c.resourceCollector.CollectAll(ctx)
		default:
			continue // Skip unknown resource types
		}

		if err != nil {
			return nil, fmt.Errorf("failed to collect %s resources: %w", resourceType, err)
		}

		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// GetMetrics returns collector metrics
func (c *Collector) GetMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"collector_name":    c.Name(),
		"collector_version": c.Version(),
		"collector_type":    c.Type(),
	}

	if c.client != nil {
		metrics["project_id"] = c.client.GetProjectID()
		metrics["regions"] = c.client.GetRegions()
	}

	return metrics
}

// Cleanup performs cleanup operations
func (c *Collector) Cleanup() error {
	// GCP API clients don't need explicit cleanup
	c.client = nil
	c.resourceCollector = nil
	return nil
}

// GetSupportedResourceTypes returns the supported resource types
func (c *Collector) GetSupportedResourceTypes() []string {
	return []string{
		"all",
		"compute",
		"instances",
		"disks",
		"instance_groups",
		"storage",
		"buckets",
		"networking",
		"networks",
		"subnets",
		"firewalls",
		"iam",
		"service_accounts",
	}
}

// GetConfiguration returns the current configuration
func (c *Collector) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"project_id":       c.config.ProjectID,
		"credentials_file": c.config.CredentialsFile,
		"regions":         c.config.Regions,
		"resource_types":  c.config.ResourceTypes,
	}
}

// ValidateConfiguration validates the collector configuration
func (c *Collector) ValidateConfiguration() error {
	if c.config.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}

	// Validate credential access if specified
	if c.config.CredentialsFile != "" {
		if _, err := os.Stat(c.config.CredentialsFile); err != nil {
			return fmt.Errorf("credentials file not accessible: %w", err)
		}
	}

	// Validate resource types
	supportedTypes := c.GetSupportedResourceTypes()
	for _, resourceType := range c.config.ResourceTypes {
		if !contains(supportedTypes, resourceType) {
			return fmt.Errorf("unsupported resource type: %s", resourceType)
		}
	}

	return nil
}

// resolveResourceTypes resolves the actual resource types to collect
func (c *Collector) resolveResourceTypes() []string {
	if len(c.config.ResourceTypes) == 0 {
		return []string{"all"}
	}

	// If "all" is specified, return just "all"
	for _, rt := range c.config.ResourceTypes {
		if rt == "all" {
			return []string{"all"}
		}
	}

	return c.config.ResourceTypes
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Enhanced collector interface methods

// GetDisplayName returns the display name for the collector
func (c *Collector) GetDisplayName() string {
	return "Google Cloud Platform"
}

// GetDescription returns the collector description
func (c *Collector) GetDescription() string {
	return "Collects resources from Google Cloud Platform including Compute Engine, Cloud Storage, Networking, and IAM resources"
}

// GetProvider returns the cloud provider name
func (c *Collector) GetProvider() string {
	return "gcp"
}

// GetCapabilities returns the collector capabilities
func (c *Collector) GetCapabilities() []string {
	return []string{
		"resource_discovery",
		"multi_region",
		"real_time",
		"metadata_collection",
		"tag_support",
		"state_tracking",
	}
}

// GetRequiredPermissions returns the required GCP permissions
func (c *Collector) GetRequiredPermissions() []string {
	return []string{
		"compute.instances.list",
		"compute.disks.list",
		"compute.instanceGroups.list",
		"compute.networks.list",
		"compute.subnetworks.list",
		"compute.firewalls.list",
		"compute.regions.list",
		"compute.zones.list",
		"storage.buckets.list",
		"storage.buckets.get",
		"iam.serviceAccounts.list",
		"resourcemanager.projects.get",
	}
}

// GetEstimatedCost returns the estimated cost per collection
func (c *Collector) GetEstimatedCost() map[string]interface{} {
	return map[string]interface{}{
		"currency": "USD",
		"cost_per_1000_api_calls": 0.0, // Most GCP read APIs are free
		"estimated_calls_per_scan": 50,
		"notes": "GCP read APIs are typically free, costs may apply for large-scale usage",
	}
}

// SupportsRealTime indicates if the collector supports real-time collection
func (c *Collector) SupportsRealTime() bool {
	return true
}

// SupportsIncremental indicates if the collector supports incremental collection
func (c *Collector) SupportsIncremental() bool {
	return false // Not implemented yet
}

// SupportsFiltering indicates if the collector supports resource filtering
func (c *Collector) SupportsFiltering() bool {
	return true
}

// GetDefaultConfiguration returns the default configuration
func (c *Collector) GetDefaultConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"project_id":       "",
		"credentials_file": "",
		"regions":         []string{"us-central1", "us-east1", "us-west1", "europe-west1"},
		"resource_types":  []string{"all"},
	}
}

// Ensure GCP collector implements both Collector interfaces
var _ collectors.Collector = (*Collector)(nil)
var _ collectors.EnhancedCollector = (*Collector)(nil)