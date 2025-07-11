package aws

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodbstreamsTypes "github.com/aws/aws-sdk-go-v2/service/dynamodbstreams/types"
	"github.com/yairfalse/vaino/pkg/types"
)

// NormalizeDynamoDBTable converts a DynamoDB table to VAINO format
func (n *Normalizer) NormalizeDynamoDBTable(table dynamodbTypes.TableDescription) types.Resource {
	return types.Resource{
		ID:       aws.ToString(table.TableArn),
		Type:     "aws_dynamodb_table",
		Provider: "aws",
		Name:     aws.ToString(table.TableName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"table_name":               aws.ToString(table.TableName),
			"table_status":             string(table.TableStatus),
			"billing_mode":             string(table.BillingModeSummary.BillingMode),
			"item_count":               aws.ToInt64(table.ItemCount),
			"table_size_bytes":         aws.ToInt64(table.TableSizeBytes),
			"key_schema":               normalizeKeySchema(table.KeySchema),
			"attribute_definitions":    normalizeAttributeDefinitions(table.AttributeDefinitions),
			"provisioned_throughput":   normalizeProvisionedThroughput(table.ProvisionedThroughput),
			"global_secondary_indexes": normalizeGlobalSecondaryIndexes(table.GlobalSecondaryIndexes),
			"local_secondary_indexes":  normalizeLocalSecondaryIndexes(table.LocalSecondaryIndexes),
			"stream_specification":     normalizeStreamSpecification(table.StreamSpecification),
			"sse_description":          normalizeSSEDescription(table.SSEDescription),
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(table.CreationDateTime),
		},
	}
}

// NormalizeDynamoDBStream converts a DynamoDB stream to VAINO format
func (n *Normalizer) NormalizeDynamoDBStream(stream dynamodbstreamsTypes.StreamDescription) types.Resource {
	return types.Resource{
		ID:       aws.ToString(stream.StreamArn),
		Type:     "aws_dynamodb_stream",
		Provider: "aws",
		Name:     aws.ToString(stream.StreamLabel),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"stream_arn":       aws.ToString(stream.StreamArn),
			"stream_label":     aws.ToString(stream.StreamLabel),
			"stream_status":    string(stream.StreamStatus),
			"stream_view_type": string(stream.StreamViewType),
			"table_name":       aws.ToString(stream.TableName),
			"shard_count":      len(stream.Shards),
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(stream.CreationRequestDateTime),
		},
	}
}

// normalizeKeySchema converts DynamoDB key schema to a normalized format
func normalizeKeySchema(keySchema []dynamodbTypes.KeySchemaElement) []map[string]interface{} {
	var normalized []map[string]interface{}
	for _, element := range keySchema {
		normalized = append(normalized, map[string]interface{}{
			"attribute_name": aws.ToString(element.AttributeName),
			"key_type":       string(element.KeyType),
		})
	}
	return normalized
}

// normalizeAttributeDefinitions converts DynamoDB attribute definitions
func normalizeAttributeDefinitions(attributes []dynamodbTypes.AttributeDefinition) []map[string]interface{} {
	var normalized []map[string]interface{}
	for _, attr := range attributes {
		normalized = append(normalized, map[string]interface{}{
			"attribute_name": aws.ToString(attr.AttributeName),
			"attribute_type": string(attr.AttributeType),
		})
	}
	return normalized
}

// normalizeProvisionedThroughput converts DynamoDB provisioned throughput
func normalizeProvisionedThroughput(throughput *dynamodbTypes.ProvisionedThroughputDescription) map[string]interface{} {
	if throughput == nil {
		return nil
	}

	return map[string]interface{}{
		"read_capacity_units":  aws.ToInt64(throughput.ReadCapacityUnits),
		"write_capacity_units": aws.ToInt64(throughput.WriteCapacityUnits),
		"last_increase_date":   formatDynamoDBTime(throughput.LastIncreaseDateTime),
		"last_decrease_date":   formatDynamoDBTime(throughput.LastDecreaseDateTime),
		"number_of_decreases":  aws.ToInt64(throughput.NumberOfDecreasesToday),
	}
}

// normalizeGlobalSecondaryIndexes converts DynamoDB GSIs
func normalizeGlobalSecondaryIndexes(indexes []dynamodbTypes.GlobalSecondaryIndexDescription) []map[string]interface{} {
	var normalized []map[string]interface{}
	for _, index := range indexes {
		gsi := map[string]interface{}{
			"index_name":             aws.ToString(index.IndexName),
			"index_status":           string(index.IndexStatus),
			"key_schema":             normalizeKeySchema(index.KeySchema),
			"provisioned_throughput": normalizeProvisionedThroughput(index.ProvisionedThroughput),
			"index_size_bytes":       aws.ToInt64(index.IndexSizeBytes),
			"item_count":             aws.ToInt64(index.ItemCount),
		}

		if index.Projection != nil {
			gsi["projection"] = map[string]interface{}{
				"projection_type":    string(index.Projection.ProjectionType),
				"non_key_attributes": index.Projection.NonKeyAttributes,
			}
		}

		normalized = append(normalized, gsi)
	}
	return normalized
}

// normalizeLocalSecondaryIndexes converts DynamoDB LSIs
func normalizeLocalSecondaryIndexes(indexes []dynamodbTypes.LocalSecondaryIndexDescription) []map[string]interface{} {
	var normalized []map[string]interface{}
	for _, index := range indexes {
		lsi := map[string]interface{}{
			"index_name":       aws.ToString(index.IndexName),
			"key_schema":       normalizeKeySchema(index.KeySchema),
			"index_size_bytes": aws.ToInt64(index.IndexSizeBytes),
			"item_count":       aws.ToInt64(index.ItemCount),
		}

		if index.Projection != nil {
			lsi["projection"] = map[string]interface{}{
				"projection_type":    string(index.Projection.ProjectionType),
				"non_key_attributes": index.Projection.NonKeyAttributes,
			}
		}

		normalized = append(normalized, lsi)
	}
	return normalized
}

// normalizeStreamSpecification converts DynamoDB stream specification
func normalizeStreamSpecification(spec *dynamodbTypes.StreamSpecification) map[string]interface{} {
	if spec == nil {
		return nil
	}

	return map[string]interface{}{
		"stream_enabled":   aws.ToBool(spec.StreamEnabled),
		"stream_view_type": string(spec.StreamViewType),
	}
}

// normalizeSSEDescription converts DynamoDB SSE description
func normalizeSSEDescription(sse *dynamodbTypes.SSEDescription) map[string]interface{} {
	if sse == nil {
		return nil
	}

	return map[string]interface{}{
		"status":                       string(sse.Status),
		"sse_type":                     string(sse.SSEType),
		"kms_master_key_arn":           aws.ToString(sse.KMSMasterKeyArn),
		"inaccessible_encryption_date": formatDynamoDBTime(sse.InaccessibleEncryptionDateTime),
	}
}

// formatDynamoDBTime formats DynamoDB time to string
func formatDynamoDBTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}
