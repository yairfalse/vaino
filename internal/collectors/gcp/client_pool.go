package gcp

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
)

// GCPClientConfig holds configuration for GCP clients
type GCPClientConfig struct {
	ProjectID       string
	CredentialsFile string
	Regions         []string
}

// GCPServicePool manages GCP service clients
type GCPServicePool struct {
	computeService *compute.Service
	storageService *storage.Service
	sqlService     *sqladmin.Service
	options        []option.ClientOption
}

// GetGKEClusters returns empty slice - placeholder for GKE clusters
func (p *GCPServicePool) GetGKEClusters(ctx context.Context, projectID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// GetCloudSQLInstances returns Cloud SQL instances for the project
func (p *GCPServicePool) GetCloudSQLInstances(ctx context.Context, projectID string) ([]interface{}, error) {
	if p.sqlService == nil {
		return []interface{}{}, fmt.Errorf("SQL service not initialized")
	}

	instancesList, err := p.sqlService.Instances.List(projectID).Context(ctx).Do()
	if err != nil {
		return []interface{}{}, fmt.Errorf("failed to list Cloud SQL instances: %w", err)
	}

	var instances []interface{}
	for _, instance := range instancesList.Items {
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetCloudSQLDatabases returns empty slice - placeholder for Cloud SQL databases
func (p *GCPServicePool) GetCloudSQLDatabases(ctx context.Context, projectID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// GetCloudSQLUsers returns empty slice - placeholder for Cloud SQL users
func (p *GCPServicePool) GetCloudSQLUsers(ctx context.Context, projectID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// GetProjectIAMPolicy returns empty map - placeholder for project IAM policy
func (p *GCPServicePool) GetProjectIAMPolicy(ctx context.Context, projectID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// GetServiceAccounts returns empty slice - placeholder for service accounts
func (p *GCPServicePool) GetServiceAccounts(ctx context.Context, projectID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// GetServiceAccountKeys returns empty slice - placeholder for service account keys
func (p *GCPServicePool) GetServiceAccountKeys(ctx context.Context, projectID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// GetServiceAccountIAMPolicy returns empty map - placeholder for service account IAM policy
func (p *GCPServicePool) GetServiceAccountIAMPolicy(ctx context.Context, projectID string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// GetCustomRoles returns empty slice - placeholder for custom roles
func (p *GCPServicePool) GetCustomRoles(ctx context.Context, projectID string) ([]interface{}, error) {
	return []interface{}{}, nil
}

// NewGCPClientPool creates a new GCP client pool
func NewGCPClientPool(ctx context.Context, config GCPClientConfig) (*GCPServicePool, error) {
	var options []option.ClientOption
	if config.CredentialsFile != "" {
		options = append(options, option.WithCredentialsFile(config.CredentialsFile))
	}

	// Initialize compute service
	computeService, err := compute.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	// Initialize storage service
	storageService, err := storage.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	// Initialize SQL admin service
	sqlService, err := sqladmin.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SQL admin service: %w", err)
	}

	return &GCPServicePool{
		computeService: computeService,
		storageService: storageService,
		sqlService:     sqlService,
		options:        options,
	}, nil
}

// GetComputeInstances retrieves compute instances from a specific region in parallel
func (pool *GCPServicePool) GetComputeInstances(ctx context.Context, projectID, region string) ([]*compute.Instance, error) {
	// Get zones for the region
	zones, err := pool.getZonesForRegion(ctx, projectID, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones for region %s: %w", region, err)
	}

	if len(zones) == 0 {
		return []*compute.Instance{}, nil
	}

	// Result aggregation
	type zoneResult struct {
		zone      string
		instances []*compute.Instance
		err       error
	}

	resultsChan := make(chan zoneResult, len(zones))
	var wg sync.WaitGroup

	// Semaphore to limit concurrent API calls
	semaphore := make(chan struct{}, 4) // Allow up to 4 concurrent zone queries

	fmt.Printf("  GCP: Collecting instances from %d zones in parallel...\n", len(zones))

	// Collect instances from each zone in parallel
	for _, zone := range zones {
		wg.Add(1)
		go func(z string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			instancesCall := pool.computeService.Instances.List(projectID, z)
			instanceList, err := instancesCall.Context(ctx).Do()

			result := zoneResult{zone: z}
			if err != nil {
				result.err = err
			} else {
				result.instances = instanceList.Items
			}

			resultsChan <- result
		}(zone)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Aggregate results
	var allInstances []*compute.Instance
	processedZones := 0

	for result := range resultsChan {
		processedZones++
		if result.err != nil {
			// Log error but continue with other zones
			fmt.Printf("    Warning: Failed to get instances from zone %s: %v\n", result.zone, result.err)
			continue
		}
		allInstances = append(allInstances, result.instances...)
		fmt.Printf("    [%d/%d] Completed zone %s: %d instances\n", processedZones, len(zones), result.zone, len(result.instances))
	}

	return allInstances, nil
}

// GetPersistentDisks retrieves persistent disks from a specific region in parallel
func (pool *GCPServicePool) GetPersistentDisks(ctx context.Context, projectID, region string) ([]*compute.Disk, error) {
	// Get zones for the region
	zones, err := pool.getZonesForRegion(ctx, projectID, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones for region %s: %w", region, err)
	}

	if len(zones) == 0 {
		return []*compute.Disk{}, nil
	}

	// Result aggregation
	type zoneResult struct {
		zone  string
		disks []*compute.Disk
		err   error
	}

	resultsChan := make(chan zoneResult, len(zones))
	var wg sync.WaitGroup

	// Semaphore to limit concurrent API calls
	semaphore := make(chan struct{}, 4) // Allow up to 4 concurrent zone queries

	fmt.Printf("  GCP: Collecting disks from %d zones in parallel...\n", len(zones))

	// Collect disks from each zone in parallel
	for _, zone := range zones {
		wg.Add(1)
		go func(z string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			disksCall := pool.computeService.Disks.List(projectID, z)
			diskList, err := disksCall.Context(ctx).Do()

			result := zoneResult{zone: z}
			if err != nil {
				result.err = err
			} else {
				result.disks = diskList.Items
			}

			resultsChan <- result
		}(zone)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Aggregate results
	var allDisks []*compute.Disk
	processedZones := 0

	for result := range resultsChan {
		processedZones++
		if result.err != nil {
			// Log error but continue with other zones
			fmt.Printf("    Warning: Failed to get disks from zone %s: %v\n", result.zone, result.err)
			continue
		}
		allDisks = append(allDisks, result.disks...)
		fmt.Printf("    [%d/%d] Completed zone %s: %d disks\n", processedZones, len(zones), result.zone, len(result.disks))
	}

	return allDisks, nil
}

// GetStorageBuckets retrieves storage buckets for the project
func (pool *GCPServicePool) GetStorageBuckets(ctx context.Context, projectID string) ([]*storage.Bucket, error) {
	bucketsCall := pool.storageService.Buckets.List(projectID)
	bucketList, err := bucketsCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list storage buckets: %w", err)
	}

	return bucketList.Items, nil
}

// GetVPCNetworks retrieves VPC networks for the project
func (pool *GCPServicePool) GetVPCNetworks(ctx context.Context, projectID string) ([]*compute.Network, error) {
	networksCall := pool.computeService.Networks.List(projectID)
	networkList, err := networksCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list VPC networks: %w", err)
	}

	return networkList.Items, nil
}

// GetSubnets retrieves subnets from a specific region
func (pool *GCPServicePool) GetSubnets(ctx context.Context, projectID, region string) ([]*compute.Subnetwork, error) {
	subnetsCall := pool.computeService.Subnetworks.List(projectID, region)
	subnetList, err := subnetsCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list subnets in region %s: %w", region, err)
	}

	return subnetList.Items, nil
}

// GetFirewallRules retrieves firewall rules for the project
func (pool *GCPServicePool) GetFirewallRules(ctx context.Context, projectID string) ([]*compute.Firewall, error) {
	firewallsCall := pool.computeService.Firewalls.List(projectID)
	firewallList, err := firewallsCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list firewall rules: %w", err)
	}

	return firewallList.Items, nil
}

// getZonesForRegion gets zones for a specific region
func (pool *GCPServicePool) getZonesForRegion(ctx context.Context, projectID, region string) ([]string, error) {
	zonesCall := pool.computeService.Zones.List(projectID)
	zoneList, err := zonesCall.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var regionZones []string
	for _, zone := range zoneList.Items {
		// Check if zone belongs to the specified region
		if len(zone.Name) > len(region) && zone.Name[:len(region)] == region {
			regionZones = append(regionZones, zone.Name)
		}
	}

	return regionZones, nil
}
