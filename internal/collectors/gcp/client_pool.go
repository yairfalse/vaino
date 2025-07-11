package gcp

import (
	"context"
	"fmt"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
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
	options        []option.ClientOption
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

	return &GCPServicePool{
		computeService: computeService,
		storageService: storageService,
		options:        options,
	}, nil
}

// GetComputeInstances retrieves compute instances from a specific region
func (pool *GCPServicePool) GetComputeInstances(ctx context.Context, projectID, region string) ([]*compute.Instance, error) {
	var allInstances []*compute.Instance

	// Get zones for the region
	zones, err := pool.getZonesForRegion(ctx, projectID, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones for region %s: %w", region, err)
	}

	// Collect instances from each zone
	for _, zone := range zones {
		instancesCall := pool.computeService.Instances.List(projectID, zone)
		instanceList, err := instancesCall.Context(ctx).Do()
		if err != nil {
			// Continue with other zones if one fails
			continue
		}

		allInstances = append(allInstances, instanceList.Items...)
	}

	return allInstances, nil
}

// GetPersistentDisks retrieves persistent disks from a specific region
func (pool *GCPServicePool) GetPersistentDisks(ctx context.Context, projectID, region string) ([]*compute.Disk, error) {
	var allDisks []*compute.Disk

	// Get zones for the region
	zones, err := pool.getZonesForRegion(ctx, projectID, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get zones for region %s: %w", region, err)
	}

	// Collect disks from each zone
	for _, zone := range zones {
		disksCall := pool.computeService.Disks.List(projectID, zone)
		diskList, err := disksCall.Context(ctx).Do()
		if err != nil {
			// Continue with other zones if one fails
			continue
		}

		allDisks = append(allDisks, diskList.Items...)
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
