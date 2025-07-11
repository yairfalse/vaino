//go:build enhanced
// +build enhanced

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/yairfalse/vaino/pkg/types"
)

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

	for {
		input := &ec2.DescribeSnapshotsInput{
			OwnerIds: []string{"self"}, // Only get snapshots owned by this account
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

// collectKeyPairs fetches all key pairs in the region
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
