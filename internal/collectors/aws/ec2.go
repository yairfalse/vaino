package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/yairfalse/wgo/pkg/types"
)

// CollectEC2Resources collects EC2 instances and security groups
func (c *AWSCollector) CollectEC2Resources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	
	// Collect EC2 instances
	instances, err := c.collectEC2Instances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect EC2 instances: %w", err)
	}
	resources = append(resources, instances...)
	
	// Collect security groups
	securityGroups, err := c.collectSecurityGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect security groups: %w", err)
	}
	resources = append(resources, securityGroups...)
	
	return resources, nil
}

// collectEC2Instances fetches all EC2 instances in the region
func (c *AWSCollector) collectEC2Instances(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string
	
	for {
		input := &ec2.DescribeInstancesInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}
		
		result, err := c.clients.EC2.DescribeInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}
		
		// Process reservations and instances
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				// Skip terminated instances
				if instance.State.Name == "terminated" {
					continue
				}
				
				resource := c.normalizer.NormalizeEC2Instance(instance)
				resources = append(resources, resource)
			}
		}
		
		// Check if there are more results
		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}
	
	return resources, nil
}

// collectSecurityGroups fetches all security groups in the region
func (c *AWSCollector) collectSecurityGroups(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string
	
	for {
		input := &ec2.DescribeSecurityGroupsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}
		
		result, err := c.clients.EC2.DescribeSecurityGroups(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe security groups: %w", err)
		}
		
		// Process security groups
		for _, sg := range result.SecurityGroups {
			resource := c.normalizer.NormalizeSecurityGroup(sg)
			resources = append(resources, resource)
		}
		
		// Check if there are more results
		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}
	
	return resources, nil
}