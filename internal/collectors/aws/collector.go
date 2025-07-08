package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// AWSCollector implements the enhanced collector interface for AWS
type AWSCollector struct {
	clients    *AWSClients
	normalizer *Normalizer
	region     string
	profile    string
}

// NewAWSCollector creates a new AWS collector
func NewAWSCollector() *AWSCollector {
	return &AWSCollector{}
}

// Status returns the current status of the AWS collector
func (c *AWSCollector) Status() string {
	if c.clients == nil {
		return "not_configured"
	}
	return "ready"
}

// AutoDiscover automatically discovers AWS configuration
func (c *AWSCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	// For AWS, auto-discovery means using default credentials and region
	return collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region":  "", // Will use default region from AWS config
			"profile": "", // Will use default profile
		},
	}, nil
}

// Validate validates the collector configuration
func (c *AWSCollector) Validate(config collectors.CollectorConfig) error {
	// Extract configuration
	region := ""
	profile := ""
	
	if config.Config != nil {
		if r, ok := config.Config["region"].(string); ok {
			region = r
		}
		if p, ok := config.Config["profile"].(string); ok {
			profile = p
		}
	}
	
	// Test AWS client creation
	ctx := context.Background()
	clientConfig := ClientConfig{
		Region:  region,
		Profile: profile,
	}
	
	clients, err := NewAWSClients(ctx, clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create AWS clients: %w", err)
	}
	
	// Test credentials
	if err := clients.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("AWS credentials validation failed: %w", err)
	}
	
	return nil
}

// Collect collects AWS resources
func (c *AWSCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	// Extract configuration
	region := ""
	profile := ""
	
	if config.Config != nil {
		if r, ok := config.Config["region"].(string); ok {
			region = r
		}
		if p, ok := config.Config["profile"].(string); ok {
			profile = p
		}
	}
	
	// Create AWS clients
	clientConfig := ClientConfig{
		Region:  region,
		Profile: profile,
	}
	
	clients, err := NewAWSClients(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS clients: %w", err)
	}
	
	c.clients = clients
	c.region = clients.GetRegion()
	c.profile = profile
	c.normalizer = NewNormalizer(c.region)
	
	// Generate snapshot ID
	snapshotID := fmt.Sprintf("aws-%d", time.Now().Unix())
	
	var allResources []types.Resource
	
	// Collect resources from different services
	services := []struct {
		name      string
		collector func(ctx context.Context) ([]types.Resource, error)
	}{
		{"EC2", c.CollectEC2Resources},
		{"S3", c.CollectS3Resources},
		{"VPC", c.CollectVPCResources},
		{"RDS", c.CollectRDSResources},
		{"Lambda", c.CollectLambdaResources},
		{"IAM", c.CollectIAMResources},
	}
	
	for _, service := range services {
		resources, err := service.collector(ctx)
		if err != nil {
			// Log error but continue with other services
			fmt.Printf("Warning: Failed to collect %s resources: %v\n", service.name, err)
			continue
		}
		allResources = append(allResources, resources...)
	}
	
	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        snapshotID,
		Timestamp: time.Now(),
		Provider:  "aws",
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    len(allResources),
			AdditionalData: map[string]interface{}{
				"profile": c.profile,
				"region":  c.region,
			},
		},
	}
	
	return snapshot, nil
}

// Name returns the collector name
func (c *AWSCollector) Name() string {
	return "aws"
}

// GetDescription returns the collector description
func (c *AWSCollector) GetDescription() string {
	return "AWS resource collector for EC2, S3, VPC, RDS, Lambda, and IAM"
}

// GetVersion returns the collector version
func (c *AWSCollector) GetVersion() string {
	return "1.0.0"
}

// SupportsRegion returns whether the collector supports multiple regions
func (c *AWSCollector) SupportsRegion() bool {
	return true
}

// SupportedRegions returns the list of supported AWS regions
func (c *AWSCollector) SupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"ca-central-1", "sa-east-1",
	}
}

// GetDefaultConfig returns the default configuration for the collector
func (c *AWSCollector) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"region":  "us-east-1",
		"profile": "",
	}
}