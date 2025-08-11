package types

import "encoding/json"

// Configuration types for different providers
// This replaces the use of interface{} in configuration fields

// AWSConfiguration represents AWS-specific configuration
type AWSConfiguration struct {
	InstanceType     string              `json:"instance_type,omitempty"`
	State            string              `json:"state,omitempty"`
	VPCID            string              `json:"vpc_id,omitempty"`
	SubnetID         string              `json:"subnet_id,omitempty"`
	AvailabilityZone string              `json:"availability_zone,omitempty"`
	PrivateIPAddress string              `json:"private_ip_address,omitempty"`
	PublicIPAddress  string              `json:"public_ip_address,omitempty"`
	ImageID          string              `json:"image_id,omitempty"`
	KeyName          string              `json:"key_name,omitempty"`
	SecurityGroups   []string            `json:"security_groups,omitempty"`
	Monitoring       bool                `json:"monitoring,omitempty"`
	Size             int32               `json:"size,omitempty"`
	VolumeType       string              `json:"volume_type,omitempty"`
	Encrypted        bool                `json:"encrypted,omitempty"`
	SnapshotID       string              `json:"snapshot_id,omitempty"`
	IOPS             int32               `json:"iops,omitempty"`
	Throughput       int32               `json:"throughput,omitempty"`
	Ingress          []SecurityGroupRule `json:"ingress,omitempty"`
	Egress           []SecurityGroupRule `json:"egress,omitempty"`
	Additional       map[string]string   `json:"additional,omitempty"`
}

// SecurityGroupRule represents AWS security group rule
type SecurityGroupRule struct {
	IPProtocol     string   `json:"ip_protocol"`
	FromPort       int32    `json:"from_port"`
	ToPort         int32    `json:"to_port"`
	CIDRBlocks     []string `json:"cidr_blocks,omitempty"`
	SecurityGroups []string `json:"security_groups,omitempty"`
}

// GCPConfiguration represents GCP-specific configuration
type GCPConfiguration struct {
	MachineType       string               `json:"machine_type,omitempty"`
	Zone              string               `json:"zone,omitempty"`
	NetworkInterface  []NetworkInterface   `json:"network_interface,omitempty"`
	DiskConfiguration []DiskConfig         `json:"disks,omitempty"`
	ServiceAccount    ServiceAccountConfig `json:"service_account,omitempty"`
	Metadata          map[string]string    `json:"metadata,omitempty"`
	Labels            map[string]string    `json:"labels,omitempty"`
	Status            string               `json:"status,omitempty"`
}

// NetworkInterface represents GCP network interface
type NetworkInterface struct {
	Network    string `json:"network"`
	Subnetwork string `json:"subnetwork"`
	ExternalIP string `json:"external_ip,omitempty"`
	InternalIP string `json:"internal_ip,omitempty"`
}

// DiskConfig represents GCP disk configuration
type DiskConfig struct {
	DeviceName string `json:"device_name"`
	Source     string `json:"source"`
	Type       string `json:"type"`
	SizeGB     int64  `json:"size_gb"`
	Encrypted  bool   `json:"encrypted,omitempty"`
}

// ServiceAccountConfig represents GCP service account configuration
type ServiceAccountConfig struct {
	Email  string   `json:"email"`
	Scopes []string `json:"scopes"`
}

// KubernetesConfiguration represents Kubernetes-specific configuration
type KubernetesConfiguration struct {
	Replicas    int32              `json:"replicas,omitempty"`
	Image       string             `json:"image,omitempty"`
	Namespace   string             `json:"namespace"`
	Selector    map[string]string  `json:"selector,omitempty"`
	Template    PodTemplate        `json:"template,omitempty"`
	Strategy    DeploymentStrategy `json:"strategy,omitempty"`
	ServiceType string             `json:"service_type,omitempty"`
	Ports       []ServicePort      `json:"ports,omitempty"`
	ClusterIP   string             `json:"cluster_ip,omitempty"`
	ExternalIPs []string           `json:"external_ips,omitempty"`
	Data        map[string]string  `json:"data,omitempty"`
	StringData  map[string]string  `json:"string_data,omitempty"`
	Type        string             `json:"type,omitempty"`
}

// PodTemplate represents Kubernetes pod template
type PodTemplate struct {
	Metadata PodMetadata `json:"metadata"`
	Spec     PodSpec     `json:"spec"`
}

// PodMetadata represents pod metadata
type PodMetadata struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PodSpec represents pod specification
type PodSpec struct {
	Containers    []Container       `json:"containers"`
	RestartPolicy string            `json:"restart_policy,omitempty"`
	NodeSelector  map[string]string `json:"node_selector,omitempty"`
}

// Container represents container configuration
type Container struct {
	Name            string               `json:"name"`
	Image           string               `json:"image"`
	Ports           []ContainerPort      `json:"ports,omitempty"`
	EnvironmentVars []EnvVar             `json:"env,omitempty"`
	Resources       ResourceRequirements `json:"resources,omitempty"`
}

// ContainerPort represents container port
type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
}

// EnvVar represents environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResourceRequirements represents resource requirements
type ResourceRequirements struct {
	Limits   ResourceList `json:"limits,omitempty"`
	Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList represents resource list
type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// DeploymentStrategy represents deployment strategy
type DeploymentStrategy struct {
	Type          string             `json:"type"`
	RollingUpdate *RollingUpdateSpec `json:"rolling_update,omitempty"`
}

// RollingUpdateSpec represents rolling update specification
type RollingUpdateSpec struct {
	MaxUnavailable string `json:"max_unavailable,omitempty"`
	MaxSurge       string `json:"max_surge,omitempty"`
}

// ServicePort represents service port
type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol"`
	Port       int32  `json:"port"`
	TargetPort string `json:"target_port,omitempty"`
	NodePort   int32  `json:"node_port,omitempty"`
}

// TerraformConfiguration represents Terraform-specific configuration
type TerraformConfiguration struct {
	SchemaVersion       int                    `json:"schema_version,omitempty"`
	Attributes          map[string]interface{} `json:"attributes"`
	Dependencies        []string               `json:"dependencies,omitempty"`
	CreateBeforeDestroy bool                   `json:"create_before_destroy,omitempty"`
	Tainted             bool                   `json:"tainted,omitempty"`
	ProviderConfig      ProviderConfig         `json:"provider_config,omitempty"`
}

// ProviderConfig represents provider configuration in Terraform
type ProviderConfig struct {
	Name    string            `json:"name"`
	Alias   string            `json:"alias,omitempty"`
	Version string            `json:"version,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
}

// GetConfigurationType returns the configuration type for a resource
func (r *Resource) GetConfigurationType() string {
	switch r.Provider {
	case "aws":
		return "aws"
	case "gcp", "google":
		return "gcp"
	case "kubernetes":
		return "kubernetes"
	case "terraform":
		return "terraform"
	default:
		return "generic"
	}
}

// SetAWSConfiguration sets AWS-specific configuration
func (r *Resource) SetAWSConfiguration(config AWSConfiguration) {
	// Convert struct to map for backward compatibility
	data, _ := structToMap(config)
	r.Configuration = data
}

// GetAWSConfiguration gets AWS-specific configuration
func (r *Resource) GetAWSConfiguration() (AWSConfiguration, error) {
	var config AWSConfiguration
	err := mapToStruct(r.Configuration, &config)
	return config, err
}

// SetGCPConfiguration sets GCP-specific configuration
func (r *Resource) SetGCPConfiguration(config GCPConfiguration) {
	data, _ := structToMap(config)
	r.Configuration = data
}

// GetGCPConfiguration gets GCP-specific configuration
func (r *Resource) GetGCPConfiguration() (GCPConfiguration, error) {
	var config GCPConfiguration
	err := mapToStruct(r.Configuration, &config)
	return config, err
}

// SetKubernetesConfiguration sets Kubernetes-specific configuration
func (r *Resource) SetKubernetesConfiguration(config KubernetesConfiguration) {
	data, _ := structToMap(config)
	r.Configuration = data
}

// GetKubernetesConfiguration gets Kubernetes-specific configuration
func (r *Resource) GetKubernetesConfiguration() (KubernetesConfiguration, error) {
	var config KubernetesConfiguration
	err := mapToStruct(r.Configuration, &config)
	return config, err
}

// SetTerraformConfiguration sets Terraform-specific configuration
func (r *Resource) SetTerraformConfiguration(config TerraformConfiguration) {
	data, _ := structToMap(config)
	r.Configuration = data
}

// GetTerraformConfiguration gets Terraform-specific configuration
func (r *Resource) GetTerraformConfiguration() (TerraformConfiguration, error) {
	var config TerraformConfiguration
	err := mapToStruct(r.Configuration, &config)
	return config, err
}

// Helper functions for conversion between structs and maps
func structToMap(s interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func mapToStruct(m map[string]interface{}, s interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, s)
}
