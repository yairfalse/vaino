package collectors

import (
	"context"
	"encoding/json"

	"github.com/yairfalse/vaino/pkg/types"
)

// CollectorConfig holds configuration for a collector
type CollectorConfig struct {
	// Provider-specific configuration - use typed configs instead of interface{}
	AWSConfig        *types.AWSConfiguration        `json:"aws_config,omitempty"`
	GCPConfig        *types.GCPConfiguration        `json:"gcp_config,omitempty"`
	KubernetesConfig *types.KubernetesConfiguration `json:"kubernetes_config,omitempty"`
	TerraformConfig  *types.TerraformConfiguration  `json:"terraform_config,omitempty"`

	// Legacy support for backward compatibility - will be deprecated
	Config map[string]interface{} `json:"config,omitempty"`

	// Common options
	Regions    []string          `json:"regions,omitempty"`
	Namespaces []string          `json:"namespaces,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`

	// File paths for file-based collectors
	StatePaths []string `json:"state_paths,omitempty"`

	// Timeout settings
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// Collector defines the simple interface for all infrastructure collectors
type Collector interface {
	// Basic interface methods
	Name() string
	Status() string

	// Collection methods
	Collect(ctx context.Context, config CollectorConfig) (*types.Snapshot, error)
	Validate(config CollectorConfig) error

	// Discovery methods
	AutoDiscover() (CollectorConfig, error)
	SupportedRegions() []string

	// Optional: Separate collection (e.g., per Terraform codebase)
	// Only implemented by collectors that support it (like Terraform)
	CollectSeparate(ctx context.Context, config CollectorConfig) ([]*types.Snapshot, error)
}

// CollectorInfo provides metadata about a collector
type CollectorInfo struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Status      string   `json:"status"`
}

// GetAWSConfig returns AWS configuration, creating from legacy config if needed
func (c *CollectorConfig) GetAWSConfig() *types.AWSConfiguration {
	if c.AWSConfig != nil {
		return c.AWSConfig
	}

	// Try to convert from legacy config
	if c.Config != nil {
		var awsConfig types.AWSConfiguration
		if data, err := json.Marshal(c.Config); err == nil {
			if err := json.Unmarshal(data, &awsConfig); err == nil {
				return &awsConfig
			}
		}
	}

	return &types.AWSConfiguration{}
}

// SetAWSConfig sets AWS configuration
func (c *CollectorConfig) SetAWSConfig(config *types.AWSConfiguration) {
	c.AWSConfig = config
}

// GetGCPConfig returns GCP configuration, creating from legacy config if needed
func (c *CollectorConfig) GetGCPConfig() *types.GCPConfiguration {
	if c.GCPConfig != nil {
		return c.GCPConfig
	}

	// Try to convert from legacy config
	if c.Config != nil {
		var gcpConfig types.GCPConfiguration
		if data, err := json.Marshal(c.Config); err == nil {
			if err := json.Unmarshal(data, &gcpConfig); err == nil {
				return &gcpConfig
			}
		}
	}

	return &types.GCPConfiguration{}
}

// SetGCPConfig sets GCP configuration
func (c *CollectorConfig) SetGCPConfig(config *types.GCPConfiguration) {
	c.GCPConfig = config
}

// GetKubernetesConfig returns Kubernetes configuration, creating from legacy config if needed
func (c *CollectorConfig) GetKubernetesConfig() *types.KubernetesConfiguration {
	if c.KubernetesConfig != nil {
		return c.KubernetesConfig
	}

	// Try to convert from legacy config
	if c.Config != nil {
		var k8sConfig types.KubernetesConfiguration
		if data, err := json.Marshal(c.Config); err == nil {
			if err := json.Unmarshal(data, &k8sConfig); err == nil {
				return &k8sConfig
			}
		}
	}

	return &types.KubernetesConfiguration{}
}

// SetKubernetesConfig sets Kubernetes configuration
func (c *CollectorConfig) SetKubernetesConfig(config *types.KubernetesConfiguration) {
	c.KubernetesConfig = config
}

// GetTerraformConfig returns Terraform configuration, creating from legacy config if needed
func (c *CollectorConfig) GetTerraformConfig() *types.TerraformConfiguration {
	if c.TerraformConfig != nil {
		return c.TerraformConfig
	}

	// Try to convert from legacy config
	if c.Config != nil {
		var tfConfig types.TerraformConfiguration
		if data, err := json.Marshal(c.Config); err == nil {
			if err := json.Unmarshal(data, &tfConfig); err == nil {
				return &tfConfig
			}
		}
	}

	return &types.TerraformConfiguration{}
}

// SetTerraformConfig sets Terraform configuration
func (c *CollectorConfig) SetTerraformConfig(config *types.TerraformConfiguration) {
	c.TerraformConfig = config
}

// GetConfigValue gets a configuration value from the appropriate typed config or legacy config
func (c *CollectorConfig) GetConfigValue(key string) interface{} {
	// First try legacy config for backward compatibility
	if c.Config != nil {
		if value, exists := c.Config[key]; exists {
			return value
		}
	}

	// Could add logic to search in typed configs if needed
	return nil
}

// SetConfigValue sets a configuration value in the legacy config
func (c *CollectorConfig) SetConfigValue(key string, value interface{}) {
	if c.Config == nil {
		c.Config = make(map[string]interface{})
	}
	c.Config[key] = value
}
