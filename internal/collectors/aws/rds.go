package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/yairfalse/wgo/pkg/types"
)

// CollectRDSResources collects RDS database instances
func (c *AWSCollector) CollectRDSResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		input := &rds.DescribeDBInstancesInput{}
		if marker != nil {
			input.Marker = marker
		}

		result, err := c.clients.RDS.DescribeDBInstances(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe RDS instances: %w", err)
		}

		// Process DB instances
		for _, instance := range result.DBInstances {
			resource := c.normalizer.NormalizeRDSInstance(instance)

			// Try to get instance tags
			if tags, err := c.getRDSInstanceTags(ctx, *instance.DBInstanceArn); err == nil {
				resource.Tags = tags
			}

			resources = append(resources, resource)
		}

		// Check if there are more results
		marker = result.Marker
		if marker == nil {
			break
		}
	}

	return resources, nil
}

// getRDSInstanceTags fetches tags for an RDS instance
func (c *AWSCollector) getRDSInstanceTags(ctx context.Context, instanceArn string) (map[string]string, error) {
	result, err := c.clients.RDS.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
		ResourceName: &instanceArn,
	})
	if err != nil {
		return make(map[string]string), nil
	}

	tags := make(map[string]string)
	for _, tag := range result.TagList {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}
