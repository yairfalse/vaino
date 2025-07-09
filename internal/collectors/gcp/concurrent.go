package gcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

// ConcurrentGCPCollector implements parallel resource collection for GCP
type ConcurrentGCPCollector struct {
	*GCPCollector
	maxWorkers int
	timeout    time.Duration
	clientPool *GCPClientPool
}

// GCPClientPool manages reusable GCP service clients
type GCPClientPool struct {
	computeService   *compute.Service
	storageService   *storage.Service
	containerService *container.Service
	iamService       *iam.Service
	options          []option.ClientOption
}

// ResourceCollectionResult holds the result of a resource collection operation
type ResourceCollectionResult struct {
	ResourceType string
	Resources    []types.Resource
	Error        error
	Duration     time.Duration
}

// NewConcurrentGCPCollector creates a new concurrent GCP collector
func NewConcurrentGCPCollector(maxWorkers int, timeout time.Duration) collectors.EnhancedCollector {
	if maxWorkers <= 0 {
		maxWorkers = 8 // Default to 8 concurrent operations
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	return &ConcurrentGCPCollector{
		GCPCollector: &GCPCollector{
			version:    "1.0.0",
			normalizer: NewResourceNormalizer(),
		},
		maxWorkers: maxWorkers,
		timeout:    timeout,
	}
}

// CollectConcurrent performs concurrent resource collection across all GCP services
func (c *ConcurrentGCPCollector) CollectConcurrent(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	// Extract GCP configuration
	gcpConfig := c.extractGCPConfig(config)

	// Validate credentials
	if err := c.validateCredentials(gcpConfig); err != nil {
		return nil, err
	}

	// Initialize client pool
	clientPool, err := c.initializeClientPool(ctx, gcpConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client pool: %w", err)
	}
	c.clientPool = clientPool

	// Create context with timeout
	collectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Results channel for concurrent operations
	results := make(chan ResourceCollectionResult, 4) // 4 main resource types

	// Wait group for tracking goroutines
	var wg sync.WaitGroup

	// Launch concurrent resource collection
	resourceTypes := []string{"compute", "storage", "container", "iam"}

	for _, resourceType := range resourceTypes {
		wg.Add(1)
		go c.collectResourceType(collectCtx, resourceType, gcpConfig, results, &wg)
	}

	// Close results channel when all collections complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var allResources []types.Resource
	collectionErrors := make([]error, 0)

	for result := range results {
		if result.Error != nil {
			collectionErrors = append(collectionErrors,
				fmt.Errorf("%s collection failed: %w", result.ResourceType, result.Error))
		} else {
			allResources = append(allResources, result.Resources...)
		}
	}

	// Check for critical errors
	if len(collectionErrors) > 0 && len(allResources) == 0 {
		return nil, fmt.Errorf("all resource collections failed: %v", collectionErrors)
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("gcp-concurrent-%d", time.Now().Unix()),
		Provider:  "gcp",
		Timestamp: time.Now(),
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: c.version,
			AdditionalData: map[string]interface{}{
				"scan_config":        config,
				"concurrent_enabled": true,
				"collection_errors":  len(collectionErrors),
			},
		},
	}

	return snapshot, nil
}

// collectResourceType collects resources for a specific GCP service type
func (c *ConcurrentGCPCollector) collectResourceType(
	ctx context.Context,
	resourceType string,
	config GCPConfig,
	results chan<- ResourceCollectionResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	startTime := time.Now()
	result := ResourceCollectionResult{
		ResourceType: resourceType,
		Resources:    make([]types.Resource, 0),
	}

	var err error
	switch resourceType {
	case "compute":
		result.Resources, err = c.collectCompute(ctx, config)
	case "storage":
		result.Resources, err = c.collectStorage(ctx, config)
	case "container":
		result.Resources, err = c.collectContainer(ctx, config)
	case "iam":
		result.Resources, err = c.collectIAM(ctx, config)
	default:
		err = fmt.Errorf("unknown resource type: %s", resourceType)
	}

	result.Error = err
	result.Duration = time.Since(startTime)
	results <- result
}

// collectCompute collects Compute Engine resources concurrently
func (c *ConcurrentGCPCollector) collectCompute(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect different compute resources concurrently
	computeTypes := []string{"instances", "disks", "networks", "firewalls"}

	for _, computeType := range computeTypes {
		wg.Add(1)
		go func(cType string) {
			defer wg.Done()

			resources, err := c.collectComputeType(ctx, cType, config)
			if err != nil {
				// Log error but don't fail the entire collection
				return
			}

			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(computeType)
	}

	wg.Wait()
	return allResources, nil
}

// collectComputeType collects a specific compute resource type
func (c *ConcurrentGCPCollector) collectComputeType(ctx context.Context, computeType string, config GCPConfig) ([]types.Resource, error) {
	var resources []types.Resource

	switch computeType {
	case "instances":
		return c.collectInstances(ctx, config)
	case "disks":
		return c.collectDisks(ctx, config)
	case "networks":
		return c.collectNetworks(ctx, config)
	case "firewalls":
		return c.collectFirewalls(ctx, config)
	}

	return resources, nil
}

// collectInstances collects Compute Engine instances
func (c *ConcurrentGCPCollector) collectInstances(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	var allInstances []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect instances from all zones in parallel
	zones := c.getZonesForRegions(config.Regions)

	for _, zone := range zones {
		wg.Add(1)
		go func(z string) {
			defer wg.Done()

			instances, err := c.collectInstancesFromZone(ctx, config.ProjectID, z)
			if err != nil {
				return // Skip failed zones
			}

			mu.Lock()
			allInstances = append(allInstances, instances...)
			mu.Unlock()
		}(zone)
	}

	wg.Wait()
	return allInstances, nil
}

// collectInstancesFromZone collects instances from a specific zone
func (c *ConcurrentGCPCollector) collectInstancesFromZone(ctx context.Context, projectID, zone string) ([]types.Resource, error) {
	// Mock implementation - in real implementation, this would use the Compute API
	resources := []types.Resource{
		{
			ID:       fmt.Sprintf("instance-%s-%d", zone, time.Now().Unix()),
			Type:     "compute_instance",
			Name:     fmt.Sprintf("instance-%s", zone),
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"zone":         zone,
				"machine_type": "e2-micro",
				"status":       "running",
				"project_id":   projectID,
			},
			Metadata: types.ResourceMetadata{
				CreatedAt: time.Now(),
				Version:   "1",
			},
		},
	}

	return resources, nil
}

// collectDisks collects Compute Engine disks
func (c *ConcurrentGCPCollector) collectDisks(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("disk-%d", time.Now().Unix()),
			Type:     "compute_disk",
			Name:     "boot-disk",
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"size_gb":    10,
				"type":       "pd-standard",
				"project_id": config.ProjectID,
			},
		},
	}, nil
}

// collectNetworks collects VPC networks
func (c *ConcurrentGCPCollector) collectNetworks(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("network-%d", time.Now().Unix()),
			Type:     "compute_network",
			Name:     "default",
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"auto_create_subnetworks": true,
				"project_id":              config.ProjectID,
			},
		},
	}, nil
}

// collectFirewalls collects firewall rules
func (c *ConcurrentGCPCollector) collectFirewalls(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("firewall-%d", time.Now().Unix()),
			Type:     "compute_firewall",
			Name:     "allow-ssh",
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"direction":  "INGRESS",
				"priority":   1000,
				"project_id": config.ProjectID,
			},
		},
	}, nil
}

// collectStorage collects Cloud Storage resources
func (c *ConcurrentGCPCollector) collectStorage(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("bucket-%d", time.Now().Unix()),
			Type:     "storage_bucket",
			Name:     "my-bucket",
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"location":   "US",
				"project_id": config.ProjectID,
			},
		},
	}, nil
}

// collectContainer collects GKE cluster resources
func (c *ConcurrentGCPCollector) collectContainer(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("cluster-%d", time.Now().Unix()),
			Type:     "container_cluster",
			Name:     "my-cluster",
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"location":   config.Region,
				"project_id": config.ProjectID,
			},
		},
	}, nil
}

// collectIAM collects IAM resources
func (c *ConcurrentGCPCollector) collectIAM(ctx context.Context, config GCPConfig) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("service-account-%d", time.Now().Unix()),
			Type:     "iam_service_account",
			Name:     "my-service-account",
			Provider: "gcp",
			Configuration: map[string]interface{}{
				"display_name": "My Service Account",
				"project_id":   config.ProjectID,
			},
		},
	}, nil
}

// initializeClientPool creates and initializes the GCP client pool
func (c *ConcurrentGCPCollector) initializeClientPool(ctx context.Context, config GCPConfig) (*GCPClientPool, error) {
	// Create client options
	var options []option.ClientOption

	if config.CredentialsFile != "" {
		options = append(options, option.WithCredentialsFile(config.CredentialsFile))
	}

	// Initialize services (mock implementation)
	return &GCPClientPool{
		options: options,
	}, nil
}

// getZonesForRegions returns zones for the specified regions
func (c *ConcurrentGCPCollector) getZonesForRegions(regions []string) []string {
	zoneMap := map[string][]string{
		"us-central1": {"us-central1-a", "us-central1-b", "us-central1-c"},
		"us-east1":    {"us-east1-a", "us-east1-b", "us-east1-c"},
		"us-west1":    {"us-west1-a", "us-west1-b", "us-west1-c"},
		"us-west2":    {"us-west2-a", "us-west2-b", "us-west2-c"},
	}

	var allZones []string
	for _, region := range regions {
		if zones, exists := zoneMap[region]; exists {
			allZones = append(allZones, zones...)
		}
	}

	if len(allZones) == 0 {
		// Default zones if no regions specified
		allZones = []string{"us-central1-a", "us-central1-b"}
	}

	return allZones
}

// Override the Collect method to use concurrent collection
func (c *ConcurrentGCPCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	return c.CollectConcurrent(ctx, config)
}
