package gcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
	compute "google.golang.org/api/compute/v1"
	storage "google.golang.org/api/storage/v1"
)

type ResourceNormalizer struct{}

func NewResourceNormalizer() *ResourceNormalizer {
	return &ResourceNormalizer{}
}

// NormalizeComputeInstance converts a GCP compute instance to VAINO format
func (n *ResourceNormalizer) NormalizeComputeInstance(instance interface{}) types.Resource {
	computeInstance, ok := instance.(*compute.Instance)
	if !ok {
		return types.Resource{
			ID:       "unknown-instance",
			Type:     "compute_instance",
			Name:     "unknown",
			Provider: "gcp",
		}
	}

	// Extract creation timestamp
	var createdAt time.Time
	if computeInstance.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, computeInstance.CreationTimestamp); err == nil {
			createdAt = t
		}
	}

	// Extract labels (GCP equivalent of tags)
	labels := make(map[string]string)
	if computeInstance.Labels != nil {
		for k, v := range computeInstance.Labels {
			labels[k] = v
		}
	}

	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/zones/%s/instances/%s", extractProjectFromSelfLink(computeInstance.SelfLink), extractZoneFromSelfLink(computeInstance.SelfLink), computeInstance.Name),
		Type:     "compute_instance",
		Name:     computeInstance.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"machine_type":       extractMachineTypeFromURL(computeInstance.MachineType),
			"status":             computeInstance.Status,
			"zone":               extractZoneFromSelfLink(computeInstance.SelfLink),
			"creation_timestamp": computeInstance.CreationTimestamp,
			"self_link":          computeInstance.SelfLink,
			"id":                 computeInstance.Id,
			"cpu_platform":       computeInstance.CpuPlatform,
			"network_interfaces": computeInstance.NetworkInterfaces,
			"disks":              computeInstance.Disks,
			"service_accounts":   computeInstance.ServiceAccounts,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   fmt.Sprintf("%d", computeInstance.Id),
		},
		Tags: labels,
	}
}

// NormalizePersistentDisk converts a GCP persistent disk to VAINO format
func (n *ResourceNormalizer) NormalizePersistentDisk(disk interface{}) types.Resource {
	persistentDisk, ok := disk.(*compute.Disk)
	if !ok {
		return types.Resource{
			ID:       "unknown-disk",
			Type:     "persistent_disk",
			Name:     "unknown",
			Provider: "gcp",
		}
	}

	var createdAt time.Time
	if persistentDisk.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, persistentDisk.CreationTimestamp); err == nil {
			createdAt = t
		}
	}

	labels := make(map[string]string)
	if persistentDisk.Labels != nil {
		for k, v := range persistentDisk.Labels {
			labels[k] = v
		}
	}

	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/zones/%s/disks/%s", extractProjectFromSelfLink(persistentDisk.SelfLink), extractZoneFromSelfLink(persistentDisk.SelfLink), persistentDisk.Name),
		Type:     "persistent_disk",
		Name:     persistentDisk.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"size_gb":            persistentDisk.SizeGb,
			"type":               persistentDisk.Type,
			"status":             persistentDisk.Status,
			"zone":               extractZoneFromSelfLink(persistentDisk.SelfLink),
			"creation_timestamp": persistentDisk.CreationTimestamp,
			"self_link":          persistentDisk.SelfLink,
			"id":                 persistentDisk.Id,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   fmt.Sprintf("%d", persistentDisk.Id),
		},
		Tags: labels,
	}
}

// NormalizeStorageBucket converts a GCP storage bucket to VAINO format
func (n *ResourceNormalizer) NormalizeStorageBucket(bucket interface{}) types.Resource {
	storageBucket, ok := bucket.(*storage.Bucket)
	if !ok {
		return types.Resource{
			ID:       "unknown-bucket",
			Type:     "storage_bucket",
			Name:     "unknown",
			Provider: "gcp",
		}
	}

	var createdAt time.Time
	if storageBucket.TimeCreated != "" {
		if t, err := time.Parse(time.RFC3339, storageBucket.TimeCreated); err == nil {
			createdAt = t
		}
	}

	labels := make(map[string]string)
	if storageBucket.Labels != nil {
		for k, v := range storageBucket.Labels {
			labels[k] = v
		}
	}

	return types.Resource{
		ID:       storageBucket.SelfLink,
		Type:     "storage_bucket",
		Name:     storageBucket.Name,
		Provider: "gcp",
		Region:   storageBucket.Location,
		Configuration: map[string]interface{}{
			"storage_class":  storageBucket.StorageClass,
			"location":       storageBucket.Location,
			"location_type":  storageBucket.LocationType,
			"versioning":     storageBucket.Versioning,
			"website":        storageBucket.Website,
			"cors":           storageBucket.Cors,
			"lifecycle":      storageBucket.Lifecycle,
			"time_created":   storageBucket.TimeCreated,
			"time_updated":   storageBucket.Updated,
			"metageneration": storageBucket.Metageneration,
			"self_link":      storageBucket.SelfLink,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   fmt.Sprintf("%d", storageBucket.Metageneration),
		},
		Tags: labels,
	}
}

// NormalizeVPCNetwork converts a GCP VPC network to VAINO format
func (n *ResourceNormalizer) NormalizeVPCNetwork(network interface{}) types.Resource {
	vpcNetwork, ok := network.(*compute.Network)
	if !ok {
		return types.Resource{
			ID:       "unknown-network",
			Type:     "vpc_network",
			Name:     "unknown",
			Provider: "gcp",
		}
	}

	var createdAt time.Time
	if vpcNetwork.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, vpcNetwork.CreationTimestamp); err == nil {
			createdAt = t
		}
	}

	return types.Resource{
		ID:       vpcNetwork.SelfLink,
		Type:     "vpc_network",
		Name:     vpcNetwork.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"auto_create_subnetworks": vpcNetwork.AutoCreateSubnetworks,
			"routing_config":          vpcNetwork.RoutingConfig,
			"creation_timestamp":      vpcNetwork.CreationTimestamp,
			"self_link":               vpcNetwork.SelfLink,
			"id":                      vpcNetwork.Id,
			"description":             vpcNetwork.Description,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   fmt.Sprintf("%d", vpcNetwork.Id),
		},
	}
}

// NormalizeSubnet converts a GCP subnet to VAINO format
func (n *ResourceNormalizer) NormalizeSubnet(subnet interface{}) types.Resource {
	gcpSubnet, ok := subnet.(*compute.Subnetwork)
	if !ok {
		return types.Resource{
			ID:       "unknown-subnet",
			Type:     "subnet",
			Name:     "unknown",
			Provider: "gcp",
		}
	}

	var createdAt time.Time
	if gcpSubnet.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, gcpSubnet.CreationTimestamp); err == nil {
			createdAt = t
		}
	}

	return types.Resource{
		ID:       gcpSubnet.SelfLink,
		Type:     "subnet",
		Name:     gcpSubnet.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"ip_cidr_range":            gcpSubnet.IpCidrRange,
			"network":                  gcpSubnet.Network,
			"gateway_address":          gcpSubnet.GatewayAddress,
			"region":                   gcpSubnet.Region,
			"creation_timestamp":       gcpSubnet.CreationTimestamp,
			"self_link":                gcpSubnet.SelfLink,
			"id":                       gcpSubnet.Id,
			"private_ip_google_access": gcpSubnet.PrivateIpGoogleAccess,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   fmt.Sprintf("%d", gcpSubnet.Id),
		},
	}
}

// NormalizeFirewallRule converts a GCP firewall rule to VAINO format
func (n *ResourceNormalizer) NormalizeFirewallRule(firewall interface{}) types.Resource {
	firewallRule, ok := firewall.(*compute.Firewall)
	if !ok {
		return types.Resource{
			ID:       "unknown-firewall",
			Type:     "firewall_rule",
			Name:     "unknown",
			Provider: "gcp",
		}
	}

	var createdAt time.Time
	if firewallRule.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, firewallRule.CreationTimestamp); err == nil {
			createdAt = t
		}
	}

	return types.Resource{
		ID:       firewallRule.SelfLink,
		Type:     "firewall_rule",
		Name:     firewallRule.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"direction":          firewallRule.Direction,
			"priority":           firewallRule.Priority,
			"network":            firewallRule.Network,
			"source_ranges":      firewallRule.SourceRanges,
			"destination_ranges": firewallRule.DestinationRanges,
			"source_tags":        firewallRule.SourceTags,
			"target_tags":        firewallRule.TargetTags,
			"allowed":            firewallRule.Allowed,
			"denied":             firewallRule.Denied,
			"creation_timestamp": firewallRule.CreationTimestamp,
			"self_link":          firewallRule.SelfLink,
			"id":                 firewallRule.Id,
			"description":        firewallRule.Description,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   fmt.Sprintf("%d", firewallRule.Id),
		},
	}
}

// Helper functions for parsing GCP resource URLs
func extractProjectFromSelfLink(selfLink string) string {
	// Extract project from URL like: https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instances/my-instance
	parts := strings.Split(selfLink, "/")
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

func extractZoneFromSelfLink(selfLink string) string {
	parts := strings.Split(selfLink, "/")
	for i, part := range parts {
		if part == "zones" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

func extractMachineTypeFromURL(machineTypeURL string) string {
	parts := strings.Split(machineTypeURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return machineTypeURL
}
