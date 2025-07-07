package gcp

import (
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/storage/v1"
)

// extractLabels converts GCP labels to WGO tags format
func extractLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return make(map[string]string)
	}
	return labels
}

// parseGCPTimestamp parses GCP timestamp format to Go time
func parseGCPTimestamp(timestamp string) *time.Time {
	if timestamp == "" {
		return nil
	}
	
	// GCP uses RFC3339 format
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return nil
	}
	return &t
}

// buildInstanceProperties creates properties map for a Compute Engine instance
func buildInstanceProperties(instance *compute.Instance) map[string]interface{} {
	properties := map[string]interface{}{
		"machine_type":     extractMachineTypeFromURL(instance.MachineType),
		"status":          instance.Status,
		"can_ip_forward":  instance.CanIpForward,
		"deletion_protection": instance.DeletionProtection,
		"description":     instance.Description,
		"scheduling":      buildSchedulingProperties(instance.Scheduling),
		"network_interfaces": buildNetworkInterfacesProperties(instance.NetworkInterfaces),
		"disks":           buildDisksProperties(instance.Disks),
		"metadata":        buildMetadataProperties(instance.Metadata),
		"service_accounts": buildServiceAccountsProperties(instance.ServiceAccounts),
		"tags":            buildTagsProperties(instance.Tags),
	}

	if instance.MinCpuPlatform != "" {
		properties["min_cpu_platform"] = instance.MinCpuPlatform
	}

	return properties
}

// buildDiskProperties creates properties map for a persistent disk
func buildDiskProperties(disk *compute.Disk) map[string]interface{} {
	properties := map[string]interface{}{
		"size_gb":         disk.SizeGb,
		"type":           extractDiskTypeFromURL(disk.Type),
		"status":         disk.Status,
		"description":    disk.Description,
		"source_image":   disk.SourceImage,
		"source_snapshot": disk.SourceSnapshot,
		"zone":           extractZoneFromURL(disk.Zone),
		"options":        disk.Options,
		"provisioned_iops": disk.ProvisionedIops,
	}

	if len(disk.Users) > 0 {
		properties["attached_to"] = disk.Users
	}

	return properties
}

// buildInstanceGroupProperties creates properties for instance groups
func buildInstanceGroupProperties(ig *compute.InstanceGroup) map[string]interface{} {
	return map[string]interface{}{
		"description":  ig.Description,
		"fingerprint":  ig.Fingerprint,
		"network":      ig.Network,
		"subnetwork":   ig.Subnetwork,
		"size":        ig.Size,
		"zone":        extractZoneFromURL(ig.Zone),
		"named_ports": buildNamedPortsProperties(ig.NamedPorts),
	}
}

// buildBucketProperties creates properties map for a Cloud Storage bucket
func buildBucketProperties(bucket *storage.Bucket) map[string]interface{} {
	properties := map[string]interface{}{
		"storage_class":    bucket.StorageClass,
		"location":        bucket.Location,
		"location_type":   bucket.LocationType,
		"versioning":      buildVersioningProperties(bucket.Versioning),
		"lifecycle":       buildLifecycleProperties(bucket.Lifecycle),
		"encryption":      buildEncryptionProperties(bucket.Encryption),
		"logging":         buildLoggingProperties(bucket.Logging),
		"website":         buildWebsiteProperties(bucket.Website),
		"cors":           buildCORSProperties(bucket.Cors),
		"acl":            buildACLProperties(bucket.Acl),
		"default_object_acl": buildDefaultObjectACLProperties(bucket.DefaultObjectAcl),
		"iam_config":      buildIAMConfigProperties(bucket.IamConfiguration),
		"retention_policy": buildRetentionPolicyProperties(bucket.RetentionPolicy),
	}

	if bucket.Metageneration > 0 {
		properties["metageneration"] = bucket.Metageneration
	}

	return properties
}

// buildNetworkProperties creates properties map for VPC networks
func buildNetworkProperties(network *compute.Network) map[string]interface{} {
	properties := map[string]interface{}{
		"description":              network.Description,
		"auto_create_subnetworks":  network.AutoCreateSubnetworks,
		"routing_config":          buildRoutingConfigProperties(network.RoutingConfig),
		"ipv4_range":              network.IPv4Range,
		"gateway_ipv4":            network.GatewayIPv4,
		"subnetworks":             network.Subnetworks,
		"peerings":                buildPeeringsProperties(network.Peerings),
	}

	return properties
}

// buildSubnetProperties creates properties map for subnets
func buildSubnetProperties(subnet *compute.Subnetwork) map[string]interface{} {
	properties := map[string]interface{}{
		"description":              subnet.Description,
		"ip_cidr_range":           subnet.IpCidrRange,
		"gateway_address":         subnet.GatewayAddress,
		"network":                 subnet.Network,
		"enable_flow_logs":        subnet.EnableFlowLogs,
		"private_ip_google_access": subnet.PrivateIpGoogleAccess,
		"secondary_ip_ranges":     buildSecondaryIPRangesProperties(subnet.SecondaryIpRanges),
		"log_config":              buildLogConfigProperties(subnet.LogConfig),
	}

	return properties
}

// buildFirewallProperties creates properties map for firewall rules
func buildFirewallProperties(firewall *compute.Firewall) map[string]interface{} {
	properties := map[string]interface{}{
		"description":       firewall.Description,
		"direction":        firewall.Direction,
		"disabled":         firewall.Disabled,
		"network":          firewall.Network,
		"priority":         firewall.Priority,
		"source_ranges":    firewall.SourceRanges,
		"source_tags":      firewall.SourceTags,
		"target_tags":      firewall.TargetTags,
		"target_service_accounts": firewall.TargetServiceAccounts,
		"source_service_accounts": firewall.SourceServiceAccounts,
		"allowed":          buildAllowedProperties(firewall.Allowed),
		"denied":           buildDeniedProperties(firewall.Denied),
		"destination_ranges": firewall.DestinationRanges,
		"log_config":       buildLogConfigProperties(firewall.LogConfig),
	}

	return properties
}

// buildServiceAccountProperties creates properties map for service accounts
func buildServiceAccountProperties(sa *iam.ServiceAccount) map[string]interface{} {
	properties := map[string]interface{}{
		"email":         sa.Email,
		"display_name":  sa.DisplayName,
		"description":   sa.Description,
		"disabled":      sa.Disabled,
		"oauth2_client_id": sa.Oauth2ClientId,
		"project_id":    sa.ProjectId,
	}

	return properties
}

// Helper functions for building nested properties

func buildSchedulingProperties(scheduling *compute.Scheduling) map[string]interface{} {
	if scheduling == nil {
		return nil
	}
	return map[string]interface{}{
		"automatic_restart":   scheduling.AutomaticRestart,
		"on_host_maintenance": scheduling.OnHostMaintenance,
		"preemptible":        scheduling.Preemptible,
	}
}

func buildNetworkInterfacesProperties(interfaces []*compute.NetworkInterface) []map[string]interface{} {
	result := make([]map[string]interface{}, len(interfaces))
	for i, iface := range interfaces {
		result[i] = map[string]interface{}{
			"name":            iface.Name,
			"network":         iface.Network,
			"subnetwork":      iface.Subnetwork,
			"network_ip":      iface.NetworkIP,
			"access_configs":  buildAccessConfigsProperties(iface.AccessConfigs),
			"alias_ip_ranges": buildAliasIPRangesProperties(iface.AliasIpRanges),
		}
	}
	return result
}

func buildDisksProperties(disks []*compute.AttachedDisk) []map[string]interface{} {
	result := make([]map[string]interface{}, len(disks))
	for i, disk := range disks {
		result[i] = map[string]interface{}{
			"auto_delete":  disk.AutoDelete,
			"boot":        disk.Boot,
			"device_name": disk.DeviceName,
			"disk_encryption_key": buildDiskEncryptionKeyProperties(disk.DiskEncryptionKey),
			"index":       disk.Index,
			"interface":   disk.Interface,
			"mode":        disk.Mode,
			"source":      disk.Source,
			"type":        disk.Type,
		}
	}
	return result
}

func buildMetadataProperties(metadata *compute.Metadata) map[string]interface{} {
	if metadata == nil {
		return nil
	}
	
	items := make(map[string]string)
	for _, item := range metadata.Items {
		if item.Value != nil {
			items[item.Key] = *item.Value
		}
	}
	
	return map[string]interface{}{
		"fingerprint": metadata.Fingerprint,
		"items":      items,
	}
}

func buildServiceAccountsProperties(accounts []*compute.ServiceAccount) []map[string]interface{} {
	result := make([]map[string]interface{}, len(accounts))
	for i, account := range accounts {
		result[i] = map[string]interface{}{
			"email":  account.Email,
			"scopes": account.Scopes,
		}
	}
	return result
}

func buildTagsProperties(tags *compute.Tags) map[string]interface{} {
	if tags == nil {
		return nil
	}
	return map[string]interface{}{
		"fingerprint": tags.Fingerprint,
		"items":      tags.Items,
	}
}

func buildNamedPortsProperties(ports []*compute.NamedPort) []map[string]interface{} {
	result := make([]map[string]interface{}, len(ports))
	for i, port := range ports {
		result[i] = map[string]interface{}{
			"name": port.Name,
			"port": port.Port,
		}
	}
	return result
}

func buildVersioningProperties(versioning *storage.BucketVersioning) map[string]interface{} {
	if versioning == nil {
		return nil
	}
	return map[string]interface{}{
		"enabled": versioning.Enabled,
	}
}

func buildLifecycleProperties(lifecycle *storage.BucketLifecycle) interface{} {
	if lifecycle == nil {
		return nil
	}
	// Simplified representation
	return map[string]interface{}{
		"rule_count": len(lifecycle.Rule),
	}
}

func buildEncryptionProperties(encryption *storage.BucketEncryption) map[string]interface{} {
	if encryption == nil {
		return nil
	}
	return map[string]interface{}{
		"default_kms_key_name": encryption.DefaultKmsKeyName,
	}
}

func buildLoggingProperties(logging *storage.BucketLogging) map[string]interface{} {
	if logging == nil {
		return nil
	}
	return map[string]interface{}{
		"log_bucket":        logging.LogBucket,
		"log_object_prefix": logging.LogObjectPrefix,
	}
}

func buildWebsiteProperties(website *storage.BucketWebsite) map[string]interface{} {
	if website == nil {
		return nil
	}
	return map[string]interface{}{
		"main_page_suffix": website.MainPageSuffix,
		"not_found_page":   website.NotFoundPage,
	}
}

func buildCORSProperties(cors []*storage.BucketCors) interface{} {
	if cors == nil {
		return nil
	}
	// Simplified representation
	return map[string]interface{}{
		"rule_count": len(cors),
	}
}

func buildACLProperties(acl []*storage.BucketAccessControl) interface{} {
	if acl == nil {
		return nil
	}
	// Simplified representation
	return map[string]interface{}{
		"rule_count": len(acl),
	}
}

func buildDefaultObjectACLProperties(acl []*storage.ObjectAccessControl) interface{} {
	if acl == nil {
		return nil
	}
	// Simplified representation
	return map[string]interface{}{
		"rule_count": len(acl),
	}
}

func buildIAMConfigProperties(config *storage.BucketIamConfiguration) map[string]interface{} {
	if config == nil {
		return nil
	}
	return map[string]interface{}{
		"uniform_bucket_level_access": buildUniformBucketLevelAccessProperties(config.UniformBucketLevelAccess),
		"public_access_prevention":   config.PublicAccessPrevention,
	}
}

func buildUniformBucketLevelAccessProperties(ubla *storage.BucketIamConfigurationUniformBucketLevelAccess) map[string]interface{} {
	if ubla == nil {
		return nil
	}
	return map[string]interface{}{
		"enabled":      ubla.Enabled,
		"locked_time":  ubla.LockedTime,
	}
}

func buildRetentionPolicyProperties(policy *storage.BucketRetentionPolicy) map[string]interface{} {
	if policy == nil {
		return nil
	}
	return map[string]interface{}{
		"effective_time":     policy.EffectiveTime,
		"is_locked":         policy.IsLocked,
		"retention_period":  policy.RetentionPeriod,
	}
}

func buildRoutingConfigProperties(config *compute.NetworkRoutingConfig) map[string]interface{} {
	if config == nil {
		return nil
	}
	return map[string]interface{}{
		"routing_mode": config.RoutingMode,
	}
}

func buildPeeringsProperties(peerings []*compute.NetworkPeering) interface{} {
	if peerings == nil {
		return nil
	}
	// Simplified representation
	return map[string]interface{}{
		"peering_count": len(peerings),
	}
}

func buildSecondaryIPRangesProperties(ranges []*compute.SubnetworkSecondaryRange) []map[string]interface{} {
	result := make([]map[string]interface{}, len(ranges))
	for i, r := range ranges {
		result[i] = map[string]interface{}{
			"range_name":   r.RangeName,
			"ip_cidr_range": r.IpCidrRange,
		}
	}
	return result
}

func buildLogConfigProperties(config *compute.FirewallLogConfig) map[string]interface{} {
	if config == nil {
		return nil
	}
	return map[string]interface{}{
		"enable": config.Enable,
		"metadata": config.Metadata,
	}
}

func buildAllowedProperties(allowed []*compute.FirewallAllowed) []map[string]interface{} {
	result := make([]map[string]interface{}, len(allowed))
	for i, rule := range allowed {
		result[i] = map[string]interface{}{
			"ip_protocol": rule.IPProtocol,
			"ports":      rule.Ports,
		}
	}
	return result
}

func buildDeniedProperties(denied []*compute.FirewallDenied) []map[string]interface{} {
	result := make([]map[string]interface{}, len(denied))
	for i, rule := range denied {
		result[i] = map[string]interface{}{
			"ip_protocol": rule.IPProtocol,
			"ports":      rule.Ports,
		}
	}
	return result
}

func buildAccessConfigsProperties(configs []*compute.AccessConfig) []map[string]interface{} {
	result := make([]map[string]interface{}, len(configs))
	for i, config := range configs {
		result[i] = map[string]interface{}{
			"name":              config.Name,
			"nat_ip":           config.NatIP,
			"network_tier":     config.NetworkTier,
			"public_ptr_domain_name": config.PublicPtrDomainName,
			"set_public_ptr":   config.SetPublicPtr,
			"type":            config.Type,
		}
	}
	return result
}

func buildAliasIPRangesProperties(ranges []*compute.AliasIpRange) []map[string]interface{} {
	result := make([]map[string]interface{}, len(ranges))
	for i, r := range ranges {
		result[i] = map[string]interface{}{
			"ip_cidr_range":           r.IpCidrRange,
			"subnetwork_range_name":   r.SubnetworkRangeName,
		}
	}
	return result
}

func buildDiskEncryptionKeyProperties(key *compute.CustomerEncryptionKey) map[string]interface{} {
	if key == nil {
		return nil
	}
	return map[string]interface{}{
		"kms_key_name":               key.KmsKeyName,
		"kms_key_service_account":    key.KmsKeyServiceAccount,
		"raw_key":                   key.RawKey,
		"rsa_encrypted_key":         key.RsaEncryptedKey,
		"sha256":                    key.Sha256,
	}
}

// URL parsing helper functions

func extractMachineTypeFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}

func extractDiskTypeFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}

func extractZoneFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}