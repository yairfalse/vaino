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

// Cloud SQL Normalizers

// NormalizeCloudSQLInstance converts a GCP Cloud SQL instance to VAINO format
func (n *ResourceNormalizer) NormalizeCloudSQLInstance(instance GCPCloudSQLInstance) types.Resource {
	var createdAt time.Time
	if instance.CreateTime != "" {
		if t, err := time.Parse(time.RFC3339, instance.CreateTime); err == nil {
			createdAt = t
		}
	}

	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/instances/%s", instance.ProjectID, instance.Name),
		Type:     "cloud_sql_instance",
		Name:     instance.Name,
		Provider: "gcp",
		Region:   instance.Region,
		Configuration: map[string]interface{}{
			"database_version":      instance.DatabaseVersion,
			"state":                 instance.State,
			"backend_type":          instance.BackendType,
			"instance_type":         instance.InstanceType,
			"connection_name":       instance.ConnectionName,
			"ip_addresses":          instance.IPAddresses,
			"settings":              instance.Settings,
			"current_disk_size":     instance.CurrentDiskSize,
			"max_disk_size":         instance.MaxDiskSize,
			"service_account_email": instance.ServiceAccountEmail,
		},
		Tags: instance.Settings.UserLabels,
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
		},
	}
}

// NormalizeCloudSQLDatabase converts a GCP Cloud SQL database to VAINO format
func (n *ResourceNormalizer) NormalizeCloudSQLDatabase(database GCPCloudSQLDatabase, instanceName string) types.Resource {
	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/instances/%s/databases/%s", database.Project, instanceName, database.Name),
		Type:     "cloud_sql_database",
		Name:     database.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"instance":  instanceName,
			"charset":   database.Charset,
			"collation": database.Collation,
		},
	}
}

// NormalizeCloudSQLUser converts a GCP Cloud SQL user to VAINO format
func (n *ResourceNormalizer) NormalizeCloudSQLUser(user GCPCloudSQLUser, instanceName string) types.Resource {
	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/instances/%s/users/%s", user.Project, instanceName, user.Name),
		Type:     "cloud_sql_user",
		Name:     user.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"instance": instanceName,
			"host":     user.Host,
			"type":     user.Type,
		},
	}
}

// IAM Normalizers

// NormalizeProjectIAMPolicy converts a GCP project IAM policy to VAINO format
func (n *ResourceNormalizer) NormalizeProjectIAMPolicy(policy GCPIAMPolicy, projectID string) types.Resource {
	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/iam-policy", projectID),
		Type:     "iam_policy",
		Name:     fmt.Sprintf("%s-iam-policy", projectID),
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"version":  policy.Version,
			"bindings": policy.Bindings,
			"etag":     policy.Etag,
		},
	}
}

// NormalizeServiceAccount converts a GCP service account to VAINO format
func (n *ResourceNormalizer) NormalizeServiceAccount(sa GCPServiceAccount) types.Resource {
	return types.Resource{
		ID:       fmt.Sprintf("projects/%s/serviceAccounts/%s", sa.ProjectID, sa.Email),
		Type:     "service_account",
		Name:     sa.DisplayName,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"email":            sa.Email,
			"unique_id":        sa.UniqueID,
			"display_name":     sa.DisplayName,
			"description":      sa.Description,
			"oauth2_client_id": sa.OAuth2ClientID,
			"disabled":         sa.Disabled,
			"etag":             sa.Etag,
		},
	}
}

// NormalizeServiceAccountKey converts a GCP service account key to VAINO format
func (n *ResourceNormalizer) NormalizeServiceAccountKey(key GCPServiceAccountKey) types.Resource {
	var validAfter, validBefore time.Time
	if key.ValidAfterTime != "" {
		if t, err := time.Parse(time.RFC3339, key.ValidAfterTime); err == nil {
			validAfter = t
		}
	}
	if key.ValidBeforeTime != "" {
		if t, err := time.Parse(time.RFC3339, key.ValidBeforeTime); err == nil {
			validBefore = t
		}
	}

	return types.Resource{
		ID:       key.Name,
		Type:     "service_account_key",
		Name:     extractKeyNameFromPath(key.Name),
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"private_key_type":      key.PrivateKeyType,
			"key_algorithm":         key.KeyAlgorithm,
			"key_origin":            key.KeyOrigin,
			"key_type":              key.KeyType,
			"service_account_email": key.ServiceAccountEmail,
			"valid_after_time":      key.ValidAfterTime,
			"valid_before_time":     key.ValidBeforeTime,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: validAfter,
			UpdatedAt: validBefore,
		},
	}
}

// NormalizeServiceAccountIAMPolicy converts a GCP service account IAM policy to VAINO format
func (n *ResourceNormalizer) NormalizeServiceAccountIAMPolicy(policy GCPIAMPolicy, serviceAccountEmail string) types.Resource {
	return types.Resource{
		ID:       fmt.Sprintf("serviceAccounts/%s/iam-policy", serviceAccountEmail),
		Type:     "service_account_iam_policy",
		Name:     fmt.Sprintf("%s-iam-policy", serviceAccountEmail),
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"service_account": serviceAccountEmail,
			"version":         policy.Version,
			"bindings":        policy.Bindings,
			"etag":            policy.Etag,
		},
	}
}

// NormalizeCustomRole converts a GCP custom role to VAINO format
func (n *ResourceNormalizer) NormalizeCustomRole(role GCPCustomRole) types.Resource {
	return types.Resource{
		ID:       role.Name,
		Type:     "custom_role",
		Name:     role.Title,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"description":          role.Description,
			"included_permissions": role.IncludedPermissions,
			"stage":                role.Stage,
			"etag":                 role.Etag,
			"deleted":              role.Deleted,
		},
	}
}

// Container Engine (GKE) Normalizers

// NormalizeGKECluster converts a GCP GKE cluster to VAINO format
func (n *ResourceNormalizer) NormalizeGKECluster(cluster GCPGKECluster) types.Resource {
	var createdAt time.Time
	if cluster.CreateTime != "" {
		if t, err := time.Parse(time.RFC3339, cluster.CreateTime); err == nil {
			createdAt = t
		}
	}

	return types.Resource{
		ID:       cluster.SelfLink,
		Type:     "gke_cluster",
		Name:     cluster.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"description":                       cluster.Description,
			"initial_node_count":                cluster.InitialNodeCount,
			"node_config":                       cluster.NodeConfig,
			"master_auth":                       cluster.MasterAuth,
			"logging_service":                   cluster.LoggingService,
			"monitoring_service":                cluster.MonitoringService,
			"network":                           cluster.Network,
			"subnetwork":                        cluster.Subnetwork,
			"cluster_ipv4_cidr":                 cluster.ClusterIpv4Cidr,
			"addons_config":                     cluster.AddonsConfig,
			"locations":                         cluster.Locations,
			"enable_kubernetes_alpha":           cluster.EnableKubernetesAlpha,
			"status":                            cluster.Status,
			"status_message":                    cluster.StatusMessage,
			"node_ipv4_cidr_size":               cluster.NodeIpv4CidrSize,
			"services_ipv4_cidr":                cluster.ServicesIpv4Cidr,
			"current_master_version":            cluster.CurrentMasterVersion,
			"current_node_version":              cluster.CurrentNodeVersion,
			"endpoint":                          cluster.Endpoint,
			"initial_cluster_version":           cluster.InitialClusterVersion,
			"location":                          cluster.Location,
			"zone":                              cluster.Zone,
			"enable_tpu":                        cluster.EnableTpu,
			"tpu_ipv4_cidr_block":               cluster.TpuIpv4CidrBlock,
			"master_authorized_networks_config": cluster.MasterAuthorizedNetworksConfig,
			"maintenance_policy":                cluster.MaintenancePolicy,
			"binary_authorization":              cluster.BinaryAuthorization,
			"database_encryption":               cluster.DatabaseEncryption,
			"shielded_nodes":                    cluster.ShieldedNodes,
			"release_channel":                   cluster.ReleaseChannel,
			"workload_identity_config":          cluster.WorkloadIdentityConfig,
			"network_config":                    cluster.NetworkConfig,
			"private_cluster_config":            cluster.PrivateClusterConfig,
			"ip_allocation_policy":              cluster.IpAllocationPolicy,
			"default_max_pods_constraint":       cluster.DefaultMaxPodsConstraint,
		},
		Tags: cluster.ResourceLabels,
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
			Version:   cluster.CurrentMasterVersion,
		},
	}
}

// NormalizeGKENodePool converts a GCP GKE node pool to VAINO format
func (n *ResourceNormalizer) NormalizeGKENodePool(nodePool GCPNodePool, clusterName string) types.Resource {
	return types.Resource{
		ID:       nodePool.SelfLink,
		Type:     "gke_node_pool",
		Name:     nodePool.Name,
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"cluster":             clusterName,
			"config":              nodePool.Config,
			"initial_node_count":  nodePool.InitialNodeCount,
			"locations":           nodePool.Locations,
			"version":             nodePool.Version,
			"instance_group_urls": nodePool.InstanceGroupUrls,
			"status":              nodePool.Status,
			"status_message":      nodePool.StatusMessage,
			"autoscaling":         nodePool.Autoscaling,
			"management":          nodePool.Management,
			"max_pods_constraint": nodePool.MaxPodsConstraint,
			"pod_ipv4_cidr_size":  nodePool.PodIpv4CidrSize,
			"upgrade_settings":    nodePool.UpgradeSettings,
		},
		Metadata: types.ResourceMetadata{
			Version: nodePool.Version,
		},
	}
}

// Helper functions

func extractKeyNameFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}
