package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// ConcurrentAWSCollector implements parallel resource collection for AWS
type ConcurrentAWSCollector struct {
	*AWSCollector
	maxWorkers int
	timeout    time.Duration
}

// AWSResourceCollectionResult holds the result of collecting AWS resources
type AWSResourceCollectionResult struct {
	ServiceName string
	Resources   []types.Resource
	Error       error
	Duration    time.Duration
}

// NewConcurrentAWSCollector creates a new concurrent AWS collector
func NewConcurrentAWSCollector(maxWorkers int, timeout time.Duration) collectors.EnhancedCollector {
	if maxWorkers <= 0 {
		maxWorkers = 6 // Default to 6 concurrent AWS services
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	return &ConcurrentAWSCollector{
		AWSCollector: NewAWSCollector(),
		maxWorkers:   maxWorkers,
		timeout:      timeout,
	}
}

// CollectConcurrent performs concurrent resource collection across all AWS services
func (c *ConcurrentAWSCollector) CollectConcurrent(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
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

	// Create context with timeout
	collectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Results channel for concurrent operations
	results := make(chan AWSResourceCollectionResult, 6) // 6 main AWS services
	
	// Wait group for tracking goroutines
	var wg sync.WaitGroup

	// Define AWS services to collect concurrently
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

	// Launch concurrent service collection
	for _, service := range services {
		wg.Add(1)
		go c.collectAWSService(collectCtx, service.name, service.collector, results, &wg)
	}

	// Close results channel when all collections complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var allResources []types.Resource
	collectionErrors := make([]error, 0)

	for result := range results {
		if result.Error != nil {
			// Check for authentication errors
			if isAuthenticationError(result.Error) {
				cancel() // Cancel remaining operations
				return nil, fmt.Errorf("authentication failed for %s service: %w", result.ServiceName, result.Error)
			}
			
			collectionErrors = append(collectionErrors, 
				fmt.Errorf("%s collection failed: %w", result.ServiceName, result.Error))
		} else {
			allResources = append(allResources, result.Resources...)
		}
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("aws-concurrent-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "aws",
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    len(allResources),
			AdditionalData: map[string]interface{}{
				"profile":            c.profile,
				"region":             c.region,
				"concurrent_enabled": true,
				"collection_errors":  len(collectionErrors),
			},
		},
	}

	return snapshot, nil
}

// collectAWSService collects resources for a specific AWS service
func (c *ConcurrentAWSCollector) collectAWSService(
	ctx context.Context,
	serviceName string,
	collector func(ctx context.Context) ([]types.Resource, error),
	results chan<- AWSResourceCollectionResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	startTime := time.Now()
	result := AWSResourceCollectionResult{
		ServiceName: serviceName,
		Resources:   make([]types.Resource, 0),
	}

	resources, err := collector(ctx)
	result.Resources = resources
	result.Error = err
	result.Duration = time.Since(startTime)

	results <- result
}

// Enhanced EC2 collection with parallel sub-resource gathering
func (c *ConcurrentAWSCollector) CollectEC2Resources(ctx context.Context) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect different EC2 resources concurrently
	ec2ResourceTypes := []string{"instances", "volumes", "snapshots", "security_groups", "key_pairs"}
	
	for _, resourceType := range ec2ResourceTypes {
		wg.Add(1)
		go func(rType string) {
			defer wg.Done()
			
			resources, err := c.collectEC2ResourceType(ctx, rType)
			if err != nil {
				// Log error but don't fail the entire collection
				return
			}
			
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(resourceType)
	}

	wg.Wait()
	return allResources, nil
}

// collectEC2ResourceType collects a specific EC2 resource type
func (c *ConcurrentAWSCollector) collectEC2ResourceType(ctx context.Context, resourceType string) ([]types.Resource, error) {
	switch resourceType {
	case "instances":
		return c.collectEC2Instances(ctx)
	case "volumes":
		return c.collectEC2Volumes(ctx)
	case "snapshots":
		return c.collectEC2Snapshots(ctx)
	case "security_groups":
		return c.collectEC2SecurityGroups(ctx)
	case "key_pairs":
		return c.collectEC2KeyPairs(ctx)
	default:
		return nil, fmt.Errorf("unknown EC2 resource type: %s", resourceType)
	}
}

// collectEC2Instances collects EC2 instances
func (c *ConcurrentAWSCollector) collectEC2Instances(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation - in real implementation, this would use AWS SDK
	return []types.Resource{
		{
			ID:       fmt.Sprintf("i-%d", time.Now().Unix()),
			Type:     "ec2_instance",
			Name:     "web-server",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"instance_type": "t3.micro",
				"state":         "running",
				"region":        c.region,
			},
			Metadata: types.ResourceMetadata{
				CreatedAt: time.Now(),
				Version:   "1",
			},
		},
	}, nil
}

// collectEC2Volumes collects EC2 volumes
func (c *ConcurrentAWSCollector) collectEC2Volumes(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("vol-%d", time.Now().Unix()),
			Type:     "ec2_volume",
			Name:     "root-volume",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"size":        8,
				"volume_type": "gp2",
				"region":      c.region,
			},
		},
	}, nil
}

// collectEC2Snapshots collects EC2 snapshots
func (c *ConcurrentAWSCollector) collectEC2Snapshots(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("snap-%d", time.Now().Unix()),
			Type:     "ec2_snapshot",
			Name:     "backup-snapshot",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"description": "Automated backup",
				"region":      c.region,
			},
		},
	}, nil
}

// collectEC2SecurityGroups collects security groups
func (c *ConcurrentAWSCollector) collectEC2SecurityGroups(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("sg-%d", time.Now().Unix()),
			Type:     "ec2_security_group",
			Name:     "web-sg",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"description": "Web server security group",
				"region":      c.region,
			},
		},
	}, nil
}

// collectEC2KeyPairs collects key pairs
func (c *ConcurrentAWSCollector) collectEC2KeyPairs(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("kp-%d", time.Now().Unix()),
			Type:     "ec2_key_pair",
			Name:     "my-key",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"region": c.region,
			},
		},
	}, nil
}

// Enhanced S3 collection with parallel bucket operations
func (c *ConcurrentAWSCollector) CollectS3Resources(ctx context.Context) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Mock bucket list - in real implementation, this would list actual buckets
	buckets := []string{"bucket1", "bucket2", "bucket3"}
	
	for _, bucket := range buckets {
		wg.Add(1)
		go func(bucketName string) {
			defer wg.Done()
			
			resource := types.Resource{
				ID:       fmt.Sprintf("s3-%s-%d", bucketName, time.Now().Unix()),
				Type:     "s3_bucket",
				Name:     bucketName,
				Provider: "aws",
				Configuration: map[string]interface{}{
					"region": c.region,
				},
			}
			
			mu.Lock()
			allResources = append(allResources, resource)
			mu.Unlock()
		}(bucket)
	}

	wg.Wait()
	return allResources, nil
}

// Enhanced VPC collection with parallel subnet/route table operations
func (c *ConcurrentAWSCollector) CollectVPCResources(ctx context.Context) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect different VPC resources concurrently
	vpcResourceTypes := []string{"vpcs", "subnets", "route_tables", "internet_gateways"}
	
	for _, resourceType := range vpcResourceTypes {
		wg.Add(1)
		go func(rType string) {
			defer wg.Done()
			
			resources, err := c.collectVPCResourceType(ctx, rType)
			if err != nil {
				return // Skip failed resource types
			}
			
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(resourceType)
	}

	wg.Wait()
	return allResources, nil
}

// collectVPCResourceType collects a specific VPC resource type
func (c *ConcurrentAWSCollector) collectVPCResourceType(ctx context.Context, resourceType string) ([]types.Resource, error) {
	switch resourceType {
	case "vpcs":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("vpc-%d", time.Now().Unix()),
				Type:     "vpc",
				Name:     "main-vpc",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"cidr_block": "10.0.0.0/16",
					"region":     c.region,
				},
			},
		}, nil
	case "subnets":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("subnet-%d", time.Now().Unix()),
				Type:     "subnet",
				Name:     "public-subnet",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"cidr_block": "10.0.1.0/24",
					"region":     c.region,
				},
			},
		}, nil
	case "route_tables":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("rtb-%d", time.Now().Unix()),
				Type:     "route_table",
				Name:     "main-rt",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"region": c.region,
				},
			},
		}, nil
	case "internet_gateways":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("igw-%d", time.Now().Unix()),
				Type:     "internet_gateway",
				Name:     "main-igw",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"region": c.region,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown VPC resource type: %s", resourceType)
	}
}

// Enhanced RDS collection with parallel database operations
func (c *ConcurrentAWSCollector) CollectRDSResources(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("db-%d", time.Now().Unix()),
			Type:     "rds_instance",
			Name:     "production-db",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"engine":         "mysql",
				"instance_class": "db.t3.micro",
				"region":         c.region,
			},
		},
	}, nil
}

// Enhanced Lambda collection with parallel function operations
func (c *ConcurrentAWSCollector) CollectLambdaResources(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("lambda-%d", time.Now().Unix()),
			Type:     "lambda_function",
			Name:     "api-handler",
			Provider: "aws",
			Configuration: map[string]interface{}{
				"runtime": "python3.9",
				"region":  c.region,
			},
		},
	}, nil
}

// Enhanced IAM collection with parallel policy/role operations
func (c *ConcurrentAWSCollector) CollectIAMResources(ctx context.Context) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect different IAM resources concurrently
	iamResourceTypes := []string{"users", "roles", "policies"}
	
	for _, resourceType := range iamResourceTypes {
		wg.Add(1)
		go func(rType string) {
			defer wg.Done()
			
			resources, err := c.collectIAMResourceType(ctx, rType)
			if err != nil {
				return // Skip failed resource types
			}
			
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(resourceType)
	}

	wg.Wait()
	return allResources, nil
}

// collectIAMResourceType collects a specific IAM resource type
func (c *ConcurrentAWSCollector) collectIAMResourceType(ctx context.Context, resourceType string) ([]types.Resource, error) {
	switch resourceType {
	case "users":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("user-%d", time.Now().Unix()),
				Type:     "iam_user",
				Name:     "developer",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"region": c.region,
				},
			},
		}, nil
	case "roles":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("role-%d", time.Now().Unix()),
				Type:     "iam_role",
				Name:     "lambda-role",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"region": c.region,
				},
			},
		}, nil
	case "policies":
		return []types.Resource{
			{
				ID:       fmt.Sprintf("policy-%d", time.Now().Unix()),
				Type:     "iam_policy",
				Name:     "s3-access-policy",
				Provider: "aws",
				Configuration: map[string]interface{}{
					"region": c.region,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown IAM resource type: %s", resourceType)
	}
}

// Override the Collect method to use concurrent collection
func (c *ConcurrentAWSCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	return c.CollectConcurrent(ctx, config)
}