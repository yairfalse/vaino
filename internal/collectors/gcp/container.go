package gcp

import (
	"context"
	"fmt"

	"github.com/yairfalse/vaino/pkg/types"
)

// GCP GKE Cluster
type GCPGKECluster struct {
	Name                           string                            `json:"name"`
	Description                    string                            `json:"description"`
	InitialNodeCount               int32                             `json:"initialNodeCount"`
	NodeConfig                     GCPNodeConfig                     `json:"nodeConfig"`
	MasterAuth                     GCPMasterAuth                     `json:"masterAuth"`
	LoggingService                 string                            `json:"loggingService"`
	MonitoringService              string                            `json:"monitoringService"`
	Network                        string                            `json:"network"`
	Subnetwork                     string                            `json:"subnetwork"`
	ClusterIpv4Cidr                string                            `json:"clusterIpv4Cidr"`
	AddonsConfig                   GCPAddonsConfig                   `json:"addonsConfig"`
	NodePools                      []GCPNodePool                     `json:"nodePools"`
	Locations                      []string                          `json:"locations"`
	EnableKubernetesAlpha          bool                              `json:"enableKubernetesAlpha"`
	ResourceLabels                 map[string]string                 `json:"resourceLabels"`
	LabelFingerprint               string                            `json:"labelFingerprint"`
	Status                         string                            `json:"status"`
	StatusMessage                  string                            `json:"statusMessage"`
	NodeIpv4CidrSize               int32                             `json:"nodeIpv4CidrSize"`
	ServicesIpv4Cidr               string                            `json:"servicesIpv4Cidr"`
	CurrentMasterVersion           string                            `json:"currentMasterVersion"`
	CurrentNodeVersion             string                            `json:"currentNodeVersion"`
	CreateTime                     string                            `json:"createTime"`
	ExpireTime                     string                            `json:"expireTime"`
	Location                       string                            `json:"location"`
	EnableTpu                      bool                              `json:"enableTpu"`
	TpuIpv4CidrBlock               string                            `json:"tpuIpv4CidrBlock"`
	SelfLink                       string                            `json:"selfLink"`
	Zone                           string                            `json:"zone"`
	Endpoint                       string                            `json:"endpoint"`
	InitialClusterVersion          string                            `json:"initialClusterVersion"`
	MasterAuthorizedNetworksConfig GCPMasterAuthorizedNetworksConfig `json:"masterAuthorizedNetworksConfig"`
	MaintenancePolicy              GCPMaintenancePolicy              `json:"maintenancePolicy"`
	BinaryAuthorization            GCPBinaryAuthorization            `json:"binaryAuthorization"`
	DatabaseEncryption             GCPDatabaseEncryption             `json:"databaseEncryption"`
	ShieldedNodes                  GCPShieldedNodes                  `json:"shieldedNodes"`
	ReleaseChannel                 GCPReleaseChannel                 `json:"releaseChannel"`
	WorkloadIdentityConfig         GCPWorkloadIdentityConfig         `json:"workloadIdentityConfig"`
	NetworkConfig                  GCPClusterNetworkConfig           `json:"networkConfig"`
	PrivateClusterConfig           GCPPrivateClusterConfig           `json:"privateClusterConfig"`
	IpAllocationPolicy             GCPIpAllocationPolicy             `json:"ipAllocationPolicy"`
	DefaultMaxPodsConstraint       GCPMaxPodsConstraint              `json:"defaultMaxPodsConstraint"`
	Conditions                     []GCPStatusCondition              `json:"conditions"`
}

type GCPNodeConfig struct {
	MachineType            string                    `json:"machineType"`
	DiskSizeGb             int32                     `json:"diskSizeGb"`
	OauthScopes            []string                  `json:"oauthScopes"`
	ServiceAccount         string                    `json:"serviceAccount"`
	Metadata               map[string]string         `json:"metadata"`
	ImageType              string                    `json:"imageType"`
	Labels                 map[string]string         `json:"labels"`
	LocalSsdCount          int32                     `json:"localSsdCount"`
	Tags                   []string                  `json:"tags"`
	Preemptible            bool                      `json:"preemptible"`
	Accelerators           []GCPAcceleratorConfig    `json:"accelerators"`
	DiskType               string                    `json:"diskType"`
	MinCpuPlatform         string                    `json:"minCpuPlatform"`
	WorkloadMetadataConfig GCPWorkloadMetadataConfig `json:"workloadMetadataConfig"`
	Taints                 []GCPNodeTaint            `json:"taints"`
	SandboxConfig          GCPSandboxConfig          `json:"sandboxConfig"`
	NodeGroup              string                    `json:"nodeGroup"`
	ReservationAffinity    GCPReservationAffinity    `json:"reservationAffinity"`
	ShieldedInstanceConfig GCPShieldedInstanceConfig `json:"shieldedInstanceConfig"`
	LinuxNodeConfig        GCPLinuxNodeConfig        `json:"linuxNodeConfig"`
	KubeletConfig          GCPNodeKubeletConfig      `json:"kubeletConfig"`
	BootDiskKmsKey         string                    `json:"bootDiskKmsKey"`
}

type GCPMasterAuth struct {
	Username                string                     `json:"username"`
	Password                string                     `json:"password"`
	ClusterCaCertificate    string                     `json:"clusterCaCertificate"`
	ClientCertificate       string                     `json:"clientCertificate"`
	ClientKey               string                     `json:"clientKey"`
	ClientCertificateConfig GCPClientCertificateConfig `json:"clientCertificateConfig"`
}

type GCPClientCertificateConfig struct {
	IssueClientCertificate bool `json:"issueClientCertificate"`
}

type GCPAddonsConfig struct {
	HttpLoadBalancing                GCPHttpLoadBalancing                `json:"httpLoadBalancing"`
	HorizontalPodAutoscaling         GCPHorizontalPodAutoscaling         `json:"horizontalPodAutoscaling"`
	KubernetesDashboard              GCPKubernetesDashboard              `json:"kubernetesDashboard"`
	NetworkPolicyConfig              GCPNetworkPolicyConfig              `json:"networkPolicyConfig"`
	CloudRunConfig                   GCPCloudRunConfig                   `json:"cloudRunConfig"`
	DnsCacheConfig                   GCPDnsCacheConfig                   `json:"dnsCacheConfig"`
	ConfigConnectorConfig            GCPConfigConnectorConfig            `json:"configConnectorConfig"`
	GcePersistentDiskCsiDriverConfig GCPGcePersistentDiskCsiDriverConfig `json:"gcePersistentDiskCsiDriverConfig"`
}

type GCPHttpLoadBalancing struct {
	Disabled bool `json:"disabled"`
}

type GCPHorizontalPodAutoscaling struct {
	Disabled bool `json:"disabled"`
}

type GCPKubernetesDashboard struct {
	Disabled bool `json:"disabled"`
}

type GCPNetworkPolicyConfig struct {
	Disabled bool `json:"disabled"`
}

type GCPCloudRunConfig struct {
	Disabled         bool   `json:"disabled"`
	LoadBalancerType string `json:"loadBalancerType"`
}

type GCPDnsCacheConfig struct {
	Enabled bool `json:"enabled"`
}

type GCPConfigConnectorConfig struct {
	Enabled bool `json:"enabled"`
}

type GCPGcePersistentDiskCsiDriverConfig struct {
	Enabled bool `json:"enabled"`
}

type GCPNodePool struct {
	Name              string                 `json:"name"`
	Config            GCPNodeConfig          `json:"config"`
	InitialNodeCount  int32                  `json:"initialNodeCount"`
	Locations         []string               `json:"locations"`
	SelfLink          string                 `json:"selfLink"`
	Version           string                 `json:"version"`
	InstanceGroupUrls []string               `json:"instanceGroupUrls"`
	Status            string                 `json:"status"`
	StatusMessage     string                 `json:"statusMessage"`
	Autoscaling       GCPNodePoolAutoscaling `json:"autoscaling"`
	Management        GCPNodeManagement      `json:"management"`
	MaxPodsConstraint GCPMaxPodsConstraint   `json:"maxPodsConstraint"`
	Conditions        []GCPStatusCondition   `json:"conditions"`
	PodIpv4CidrSize   int32                  `json:"podIpv4CidrSize"`
	UpgradeSettings   GCPUpgradeSettings     `json:"upgradeSettings"`
}

type GCPNodePoolAutoscaling struct {
	Enabled         bool  `json:"enabled"`
	MinNodeCount    int32 `json:"minNodeCount"`
	MaxNodeCount    int32 `json:"maxNodeCount"`
	Autoprovisioned bool  `json:"autoprovisioned"`
}

type GCPNodeManagement struct {
	AutoUpgrade    bool                  `json:"autoUpgrade"`
	AutoRepair     bool                  `json:"autoRepair"`
	UpgradeOptions GCPAutoUpgradeOptions `json:"upgradeOptions"`
}

type GCPAutoUpgradeOptions struct {
	AutoUpgradeStartTime string `json:"autoUpgradeStartTime"`
	Description          string `json:"description"`
}

type GCPMaxPodsConstraint struct {
	MaxPodsPerNode int64 `json:"maxPodsPerNode"`
}

type GCPStatusCondition struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CanonicalCode string `json:"canonicalCode"`
}

type GCPUpgradeSettings struct {
	MaxSurge       int32 `json:"maxSurge"`
	MaxUnavailable int32 `json:"maxUnavailable"`
}

type GCPAcceleratorConfig struct {
	AcceleratorCount int64  `json:"acceleratorCount"`
	AcceleratorType  string `json:"acceleratorType"`
}

type GCPWorkloadMetadataConfig struct {
	Mode string `json:"mode"`
}

type GCPNodeTaint struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

type GCPSandboxConfig struct {
	Type string `json:"type"`
}

type GCPReservationAffinity struct {
	ConsumeReservationType string   `json:"consumeReservationType"`
	Key                    string   `json:"key"`
	Values                 []string `json:"values"`
}

type GCPShieldedInstanceConfig struct {
	EnableSecureBoot          bool `json:"enableSecureBoot"`
	EnableIntegrityMonitoring bool `json:"enableIntegrityMonitoring"`
}

type GCPLinuxNodeConfig struct {
	Sysctls map[string]string `json:"sysctls"`
}

type GCPNodeKubeletConfig struct {
	CpuManagerPolicy  string `json:"cpuManagerPolicy"`
	CpuCfsQuota       bool   `json:"cpuCfsQuota"`
	CpuCfsQuotaPeriod string `json:"cpuCfsQuotaPeriod"`
}

type GCPMasterAuthorizedNetworksConfig struct {
	Enabled    bool           `json:"enabled"`
	CidrBlocks []GCPCidrBlock `json:"cidrBlocks"`
}

type GCPCidrBlock struct {
	DisplayName string `json:"displayName"`
	CidrBlock   string `json:"cidrBlock"`
}

type GCPMaintenancePolicy struct {
	Window GCPMaintenanceWindow `json:"window"`
}

type GCPMaintenanceWindow struct {
	DailyMaintenanceWindow GCPDailyMaintenanceWindow `json:"dailyMaintenanceWindow"`
	RecurringWindow        GCPRecurringTimeWindow    `json:"recurringWindow"`
	MaintenanceExclusions  map[string]GCPTimeWindow  `json:"maintenanceExclusions"`
}

type GCPDailyMaintenanceWindow struct {
	StartTime string `json:"startTime"`
	Duration  string `json:"duration"`
}

type GCPRecurringTimeWindow struct {
	Window     GCPTimeWindow `json:"window"`
	Recurrence string        `json:"recurrence"`
}

type GCPTimeWindow struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type GCPBinaryAuthorization struct {
	Enabled bool `json:"enabled"`
}

type GCPDatabaseEncryption struct {
	State   string `json:"state"`
	KeyName string `json:"keyName"`
}

type GCPShieldedNodes struct {
	Enabled bool `json:"enabled"`
}

type GCPReleaseChannel struct {
	Channel string `json:"channel"`
}

type GCPWorkloadIdentityConfig struct {
	WorkloadPool string `json:"workloadPool"`
}

type GCPClusterNetworkConfig struct {
	Network                   string               `json:"network"`
	Subnetwork                string               `json:"subnetwork"`
	EnableIntraNodeVisibility bool                 `json:"enableIntraNodeVisibility"`
	DefaultSnatStatus         GCPDefaultSnatStatus `json:"defaultSnatStatus"`
	EnableL4ilbSubsetting     bool                 `json:"enableL4ilbSubsetting"`
	DatapathProvider          string               `json:"datapathProvider"`
	PrivateIpv6GoogleAccess   string               `json:"privateIpv6GoogleAccess"`
}

type GCPDefaultSnatStatus struct {
	Disabled bool `json:"disabled"`
}

type GCPPrivateClusterConfig struct {
	EnablePrivateNodes       bool                                      `json:"enablePrivateNodes"`
	EnablePrivateEndpoint    bool                                      `json:"enablePrivateEndpoint"`
	MasterIpv4CidrBlock      string                                    `json:"masterIpv4CidrBlock"`
	PrivateEndpoint          string                                    `json:"privateEndpoint"`
	PublicEndpoint           string                                    `json:"publicEndpoint"`
	PeeringName              string                                    `json:"peeringName"`
	MasterGlobalAccessConfig GCPPrivateClusterMasterGlobalAccessConfig `json:"masterGlobalAccessConfig"`
}

type GCPPrivateClusterMasterGlobalAccessConfig struct {
	Enabled bool `json:"enabled"`
}

type GCPIpAllocationPolicy struct {
	UseIpAliases               bool   `json:"useIpAliases"`
	CreateSubnetwork           bool   `json:"createSubnetwork"`
	SubnetworkName             string `json:"subnetworkName"`
	ClusterSecondaryRangeName  string `json:"clusterSecondaryRangeName"`
	ServicesSecondaryRangeName string `json:"servicesSecondaryRangeName"`
	ClusterIpv4CidrBlock       string `json:"clusterIpv4CidrBlock"`
	NodeIpv4CidrBlock          string `json:"nodeIpv4CidrBlock"`
	ServicesIpv4CidrBlock      string `json:"servicesIpv4CidrBlock"`
	TpuIpv4CidrBlock           string `json:"tpuIpv4CidrBlock"`
	UseRoutes                  bool   `json:"useRoutes"`
}

// collectContainerResources collects GCP Container Engine (GKE) resources
func (c *GCPCollector) collectContainerResources(ctx context.Context, clientPool *GCPServicePool, projectID string, regions []string) ([]types.Resource, error) {
	var resources []types.Resource

	for _, region := range regions {
		// Get GKE clusters in this region
		clusters, err := clientPool.GetGKEClusters(ctx, projectID, region)
		if err != nil {
			return nil, fmt.Errorf("failed to get GKE clusters in region %s: %w", region, err)
		}

		for _, cluster := range clusters {
			resource := c.normalizer.NormalizeGKECluster(cluster)
			resource.Region = region
			resources = append(resources, resource)

			// Get node pools for this cluster
			for _, nodePool := range cluster.NodePools {
				nodePoolResource := c.normalizer.NormalizeGKENodePool(nodePool, cluster.Name)
				nodePoolResource.Region = region
				resources = append(resources, nodePoolResource)
			}
		}
	}

	return resources, nil
}
