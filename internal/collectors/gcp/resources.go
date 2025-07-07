package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/storage/v1"

	"github.com/yairfalse/wgo/pkg/types"
)

// ResourceCollector handles collection of different GCP resource types
type ResourceCollector struct {
	client *Client
}

// NewResourceCollector creates a new resource collector
func NewResourceCollector(client *Client) *ResourceCollector {
	return &ResourceCollector{
		client: client,
	}
}

// CollectAll collects all supported GCP resources
func (rc *ResourceCollector) CollectAll(ctx context.Context) ([]types.Resource, error) {
	var allResources []types.Resource

	// Collect Compute Engine resources
	computeResources, err := rc.CollectComputeResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect compute resources: %w", err)
	}
	allResources = append(allResources, computeResources...)

	// Collect Cloud Storage resources
	storageResources, err := rc.CollectStorageResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect storage resources: %w", err)
	}
	allResources = append(allResources, storageResources...)

	// Collect Networking resources
	networkResources, err := rc.CollectNetworkResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect network resources: %w", err)
	}
	allResources = append(allResources, networkResources...)

	// Collect IAM resources
	iamResources, err := rc.CollectIAMResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect IAM resources: %w", err)
	}
	allResources = append(allResources, iamResources...)

	return allResources, nil
}

// CollectComputeResources collects Compute Engine resources
func (rc *ResourceCollector) CollectComputeResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect VM instances
	instances, err := rc.collectInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect instances: %w", err)
	}
	resources = append(resources, instances...)

	// Collect disks
	disks, err := rc.collectDisks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect disks: %w", err)
	}
	resources = append(resources, disks...)

	// Collect instance groups
	instanceGroups, err := rc.collectInstanceGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect instance groups: %w", err)
	}
	resources = append(resources, instanceGroups...)

	return resources, nil
}

// CollectStorageResources collects Cloud Storage resources
func (rc *ResourceCollector) CollectStorageResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect storage buckets
	buckets, err := rc.collectStorageBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect storage buckets: %w", err)
	}
	resources = append(resources, buckets...)

	return resources, nil
}

// CollectNetworkResources collects networking resources
func (rc *ResourceCollector) CollectNetworkResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect VPC networks
	networks, err := rc.collectNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect networks: %w", err)
	}
	resources = append(resources, networks...)

	// Collect subnets
	subnets, err := rc.collectSubnets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect subnets: %w", err)
	}
	resources = append(resources, subnets...)

	// Collect firewall rules
	firewalls, err := rc.collectFirewallRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect firewall rules: %w", err)
	}
	resources = append(resources, firewalls...)

	return resources, nil
}

// CollectIAMResources collects IAM resources
func (rc *ResourceCollector) CollectIAMResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect service accounts
	serviceAccounts, err := rc.collectServiceAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect service accounts: %w", err)
	}
	resources = append(resources, serviceAccounts...)

	return resources, nil
}

// collectInstances collects VM instances from all regions
func (rc *ResourceCollector) collectInstances(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	for _, region := range rc.client.GetRegions() {
		zones, err := rc.client.ListAvailableZones(region)
		if err != nil {
			continue // Skip regions we can't access
		}

		for _, zone := range zones {
			instanceList, err := rc.client.GetComputeService().Instances.List(rc.client.GetProjectID(), zone).Context(ctx).Do()
			if err != nil {
				continue // Skip zones we can't access
			}

			for _, instance := range instanceList.Items {
				resource := types.Resource{
					ID:          fmt.Sprintf("%d", instance.Id),
					Name:        instance.Name,
					Type:        "instance",
					Provider:    "gcp",
					Region:      region,
					Zone:        zone,
					Tags:        extractLabels(instance.Labels),
					Properties:  buildInstanceProperties(instance),
					State:       strings.ToLower(instance.Status),
					CreatedTime: parseGCPTimestamp(instance.CreationTimestamp),
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// collectDisks collects persistent disks from all regions
func (rc *ResourceCollector) collectDisks(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	for _, region := range rc.client.GetRegions() {
		zones, err := rc.client.ListAvailableZones(region)
		if err != nil {
			continue
		}

		for _, zone := range zones {
			diskList, err := rc.client.GetComputeService().Disks.List(rc.client.GetProjectID(), zone).Context(ctx).Do()
			if err != nil {
				continue
			}

			for _, disk := range diskList.Items {
				resource := types.Resource{
					ID:          fmt.Sprintf("%d", disk.Id),
					Name:        disk.Name,
					Type:        "disk",
					Provider:    "gcp",
					Region:      region,
					Zone:        zone,
					Tags:        extractLabels(disk.Labels),
					Properties:  buildDiskProperties(disk),
					State:       strings.ToLower(disk.Status),
					CreatedTime: parseGCPTimestamp(disk.CreationTimestamp),
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// collectInstanceGroups collects managed instance groups
func (rc *ResourceCollector) collectInstanceGroups(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	for _, region := range rc.client.GetRegions() {
		zones, err := rc.client.ListAvailableZones(region)
		if err != nil {
			continue
		}

		for _, zone := range zones {
			igList, err := rc.client.GetComputeService().InstanceGroups.List(rc.client.GetProjectID(), zone).Context(ctx).Do()
			if err != nil {
				continue
			}

			for _, ig := range igList.Items {
				resource := types.Resource{
					ID:          fmt.Sprintf("%d", ig.Id),
					Name:        ig.Name,
					Type:        "instance_group",
					Provider:    "gcp",
					Region:      region,
					Zone:        zone,
					Properties:  buildInstanceGroupProperties(ig),
					CreatedTime: parseGCPTimestamp(ig.CreationTimestamp),
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// collectStorageBuckets collects Cloud Storage buckets
func (rc *ResourceCollector) collectStorageBuckets(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	bucketList, err := rc.client.GetStorageService().Buckets.List(rc.client.GetProjectID()).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	for _, bucket := range bucketList.Items {
		resource := types.Resource{
			ID:          bucket.Id,
			Name:        bucket.Name,
			Type:        "storage_bucket",
			Provider:    "gcp",
			Region:      bucket.Location,
			Tags:        extractLabels(bucket.Labels),
			Properties:  buildBucketProperties(bucket),
			CreatedTime: parseGCPTimestamp(bucket.TimeCreated),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// collectNetworks collects VPC networks
func (rc *ResourceCollector) collectNetworks(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	networkList, err := rc.client.GetComputeService().Networks.List(rc.client.GetProjectID()).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	for _, network := range networkList.Items {
		resource := types.Resource{
			ID:          fmt.Sprintf("%d", network.Id),
			Name:        network.Name,
			Type:        "network",
			Provider:    "gcp",
			Properties:  buildNetworkProperties(network),
			CreatedTime: parseGCPTimestamp(network.CreationTimestamp),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// collectSubnets collects subnets from all regions
func (rc *ResourceCollector) collectSubnets(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	for _, region := range rc.client.GetRegions() {
		subnetList, err := rc.client.GetComputeService().Subnetworks.List(rc.client.GetProjectID(), region).Context(ctx).Do()
		if err != nil {
			continue
		}

		for _, subnet := range subnetList.Items {
			resource := types.Resource{
				ID:          fmt.Sprintf("%d", subnet.Id),
				Name:        subnet.Name,
				Type:        "subnet",
				Provider:    "gcp",
				Region:      region,
				Properties:  buildSubnetProperties(subnet),
				CreatedTime: parseGCPTimestamp(subnet.CreationTimestamp),
			}
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// collectFirewallRules collects firewall rules
func (rc *ResourceCollector) collectFirewallRules(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	firewallList, err := rc.client.GetComputeService().Firewalls.List(rc.client.GetProjectID()).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	for _, firewall := range firewallList.Items {
		resource := types.Resource{
			ID:          fmt.Sprintf("%d", firewall.Id),
			Name:        firewall.Name,
			Type:        "firewall_rule",
			Provider:    "gcp",
			Properties:  buildFirewallProperties(firewall),
			CreatedTime: parseGCPTimestamp(firewall.CreationTimestamp),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// collectServiceAccounts collects IAM service accounts
func (rc *ResourceCollector) collectServiceAccounts(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	projectResourceName := fmt.Sprintf("projects/%s", rc.client.GetProjectID())
	saList, err := rc.client.GetIAMService().Projects.ServiceAccounts.List(projectResourceName).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	for _, sa := range saList.Accounts {
		resource := types.Resource{
			ID:         sa.UniqueId,
			Name:       sa.DisplayName,
			Type:       "service_account",
			Provider:   "gcp",
			Properties: buildServiceAccountProperties(sa),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}