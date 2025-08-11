package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectEC2Resources collects EC2 instances, security groups, volumes, snapshots, and key pairs
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

	// Collect EBS volumes
	volumes, err := c.collectEBSVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect EBS volumes: %w", err)
	}
	resources = append(resources, volumes...)

	// Collect EBS snapshots
	snapshots, err := c.collectEBSSnapshots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect EBS snapshots: %w", err)
	}
	resources = append(resources, snapshots...)

	// Collect key pairs
	keyPairs, err := c.collectKeyPairs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect key pairs: %w", err)
	}
	resources = append(resources, keyPairs...)

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

// collectEBSVolumes fetches all EBS volumes in the region
func (c *AWSCollector) collectEBSVolumes(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &ec2.DescribeVolumesInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EC2.DescribeVolumes(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe volumes: %w", err)
		}

		// Process volumes
		for _, volume := range result.Volumes {
			resource := c.normalizer.NormalizeEBSVolume(volume)
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

// collectEBSSnapshots fetches all EBS snapshots owned by the account
func (c *AWSCollector) collectEBSSnapshots(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	// Only get snapshots owned by the current account
	ownerAlias := "self"

	for {
		input := &ec2.DescribeSnapshotsInput{
			OwnerAliases: []string{ownerAlias},
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EC2.DescribeSnapshots(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe snapshots: %w", err)
		}

		// Process snapshots
		for _, snapshot := range result.Snapshots {
			resource := c.normalizer.NormalizeEBSSnapshot(snapshot)
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

// collectKeyPairs fetches all EC2 key pairs in the region
func (c *AWSCollector) collectKeyPairs(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	input := &ec2.DescribeKeyPairsInput{}
	result, err := c.clients.EC2.DescribeKeyPairs(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe key pairs: %w", err)
	}

	// Process key pairs
	for _, keyPair := range result.KeyPairs {
		resource := c.normalizer.NormalizeKeyPair(keyPair)
		resources = append(resources, resource)
	}

	return resources, nil
}
