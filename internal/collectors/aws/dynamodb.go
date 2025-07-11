package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectDynamoDBResources collects DynamoDB tables, global tables, and streams
func (c *AWSCollector) CollectDynamoDBResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect DynamoDB tables
	tables, err := c.collectDynamoDBTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect DynamoDB tables: %w", err)
	}
	resources = append(resources, tables...)

	// Collect DynamoDB streams
	streams, err := c.collectDynamoDBStreams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect DynamoDB streams: %w", err)
	}
	resources = append(resources, streams...)

	return resources, nil
}

// collectDynamoDBTables fetches all DynamoDB tables in the region
func (c *AWSCollector) collectDynamoDBTables(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var exclusiveStartTableName *string

	for {
		input := &dynamodb.ListTablesInput{}
		if exclusiveStartTableName != nil {
			input.ExclusiveStartTableName = exclusiveStartTableName
		}

		result, err := c.clients.DynamoDB.ListTables(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list DynamoDB tables: %w", err)
		}

		// Process each table
		for _, tableName := range result.TableNames {
			// Get detailed table information
			describeInput := &dynamodb.DescribeTableInput{
				TableName: &tableName,
			}

			tableDesc, err := c.clients.DynamoDB.DescribeTable(ctx, describeInput)
			if err != nil {
				// Skip tables we can't access
				continue
			}

			resource := c.normalizer.NormalizeDynamoDBTable(*tableDesc.Table)

			// Try to get table tags
			if tags, err := c.getDynamoDBTableTags(ctx, tableName); err == nil {
				resource.Tags = tags
			}

			resources = append(resources, resource)
		}

		// Check if there are more results
		exclusiveStartTableName = result.LastEvaluatedTableName
		if exclusiveStartTableName == nil {
			break
		}
	}

	return resources, nil
}

// collectDynamoDBStreams fetches all DynamoDB streams
func (c *AWSCollector) collectDynamoDBStreams(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	input := &dynamodb.ListStreamsInput{}
	result, err := c.clients.DynamoDBStreams.ListStreams(ctx, input)
	if err != nil {
		// Streams API might not be available in all regions, skip gracefully
		return resources, nil
	}

	// Process each stream
	for _, stream := range result.Streams {
		// Get detailed stream information
		describeInput := &dynamodb.DescribeStreamInput{
			StreamArn: stream.StreamArn,
		}

		streamDesc, err := c.clients.DynamoDBStreams.DescribeStream(ctx, describeInput)
		if err != nil {
			// Skip streams we can't access
			continue
		}

		resource := c.normalizer.NormalizeDynamoDBStream(*streamDesc.StreamDescription)
		resources = append(resources, resource)
	}

	return resources, nil
}

// getDynamoDBTableTags fetches tags for a DynamoDB table
func (c *AWSCollector) getDynamoDBTableTags(ctx context.Context, tableName string) (map[string]string, error) {
	// First, get the table ARN
	describeInput := &dynamodb.DescribeTableInput{
		TableName: &tableName,
	}

	tableDesc, err := c.clients.DynamoDB.DescribeTable(ctx, describeInput)
	if err != nil {
		return make(map[string]string), nil
	}

	if tableDesc.Table.TableArn == nil {
		return make(map[string]string), nil
	}

	// Get tags using the table ARN
	input := &dynamodb.ListTagsOfResourceInput{
		ResourceArn: tableDesc.Table.TableArn,
	}

	result, err := c.clients.DynamoDB.ListTagsOfResource(ctx, input)
	if err != nil {
		// Return empty tags if we can't access them
		return make(map[string]string), nil
	}

	tags := make(map[string]string)
	for _, tag := range result.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}
