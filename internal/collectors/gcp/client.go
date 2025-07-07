package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

// Client wraps GCP API clients with authentication
type Client struct {
	ctx                context.Context
	projectID          string
	computeService     *compute.Service
	storageService     *storage.Service
	iamService         *iam.Service
	resourceManager    *cloudresourcemanager.Service
	credentialsFile    string
	regions            []string
}

// ClientConfig holds configuration for GCP client
type ClientConfig struct {
	ProjectID       string
	CredentialsFile string
	Regions         []string
}

// NewClient creates a new GCP client with authentication
func NewClient(ctx context.Context, config ClientConfig) (*Client, error) {
	client := &Client{
		ctx:             ctx,
		projectID:       config.ProjectID,
		credentialsFile: config.CredentialsFile,
		regions:         config.Regions,
	}

	// Set up authentication options
	var opts []option.ClientOption
	if config.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.CredentialsFile))
	} else {
		// Use Application Default Credentials
		creds, err := google.FindDefaultCredentials(ctx, 
			compute.ComputeScope,
			storage.CloudPlatformScope,
			iam.CloudPlatformScope,
			cloudresourcemanager.CloudPlatformScope,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	// Initialize services
	var err error

	client.computeService, err = compute.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	client.storageService, err = storage.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	client.iamService, err = iam.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM service: %w", err)
	}

	client.resourceManager, err = cloudresourcemanager.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager service: %w", err)
	}

	// If no project ID provided, try to determine it
	if client.projectID == "" {
		client.projectID, err = client.detectProjectID()
		if err != nil {
			return nil, fmt.Errorf("failed to detect project ID: %w", err)
		}
	}

	// If no regions specified, use default regions
	if len(client.regions) == 0 {
		client.regions = []string{"us-central1", "us-east1", "us-west1", "europe-west1"}
	}

	return client, nil
}

// detectProjectID attempts to detect the current project ID
func (c *Client) detectProjectID() (string, error) {
	// Try to get project ID from metadata service (when running on GCP)
	creds, err := google.FindDefaultCredentials(c.ctx)
	if err != nil {
		return "", err
	}

	if creds.ProjectID != "" {
		return creds.ProjectID, nil
	}

	// List projects and use the first one (not ideal, but fallback)
	projects, err := c.resourceManager.Projects.List().Context(c.ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects.Projects) == 0 {
		return "", fmt.Errorf("no accessible projects found")
	}

	return projects.Projects[0].ProjectId, nil
}

// GetProjectID returns the current project ID
func (c *Client) GetProjectID() string {
	return c.projectID
}

// GetRegions returns the configured regions
func (c *Client) GetRegions() []string {
	return c.regions
}

// GetComputeService returns the compute service
func (c *Client) GetComputeService() *compute.Service {
	return c.computeService
}

// GetStorageService returns the storage service
func (c *Client) GetStorageService() *storage.Service {
	return c.storageService
}

// GetIAMService returns the IAM service
func (c *Client) GetIAMService() *iam.Service {
	return c.iamService
}

// GetResourceManager returns the resource manager service
func (c *Client) GetResourceManager() *cloudresourcemanager.Service {
	return c.resourceManager
}

// ValidateAccess verifies that the client has access to the project
func (c *Client) ValidateAccess() error {
	// Try to get project information
	project, err := c.resourceManager.Projects.Get(c.projectID).Context(c.ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to access project %s: %w", c.projectID, err)
	}

	if project.LifecycleState != "ACTIVE" {
		return fmt.Errorf("project %s is not active (state: %s)", c.projectID, project.LifecycleState)
	}

	return nil
}

// ListAvailableRegions returns all available compute regions for the project
func (c *Client) ListAvailableRegions() ([]string, error) {
	regionList, err := c.computeService.Regions.List(c.projectID).Context(c.ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	regions := make([]string, 0, len(regionList.Items))
	for _, region := range regionList.Items {
		if region.Status == "UP" {
			regions = append(regions, region.Name)
		}
	}

	return regions, nil
}

// ListAvailableZones returns all available zones for the specified region
func (c *Client) ListAvailableZones(region string) ([]string, error) {
	zoneList, err := c.computeService.Zones.List(c.projectID).Filter(fmt.Sprintf("region eq %s.*", region)).Context(c.ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones for region %s: %w", region, err)
	}

	zones := make([]string, 0, len(zoneList.Items))
	for _, zone := range zoneList.Items {
		if zone.Status == "UP" {
			zones = append(zones, zone.Name)
		}
	}

	return zones, nil
}