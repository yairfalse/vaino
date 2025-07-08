package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/yairfalse/wgo/pkg/types"
)

// CollectS3Resources collects S3 buckets
func (c *AWSCollector) CollectS3Resources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	
	// List all buckets
	result, err := c.clients.S3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 buckets: %w", err)
	}
	
	// Process each bucket
	for _, bucket := range result.Buckets {
		// Check if bucket is in the current region
		bucketRegion, err := c.getBucketRegion(ctx, *bucket.Name)
		if err != nil {
			// Skip buckets we can't access
			continue
		}
		
		// Only include buckets in the current region
		if bucketRegion != c.normalizer.region {
			continue
		}
		
		resource := c.normalizer.NormalizeS3Bucket(bucket)
		
		// Try to get bucket tags
		if tags, err := c.getBucketTags(ctx, *bucket.Name); err == nil {
			resource.Tags = tags
		}
		
		// Try to get bucket versioning info
		if versioning, err := c.getBucketVersioning(ctx, *bucket.Name); err == nil {
			if resource.Configuration == nil {
				resource.Configuration = make(map[string]interface{})
			}
			resource.Configuration["versioning"] = versioning
		}
		
		// Try to get bucket encryption info
		if encryption, err := c.getBucketEncryption(ctx, *bucket.Name); err == nil {
			if resource.Configuration == nil {
				resource.Configuration = make(map[string]interface{})
			}
			resource.Configuration["server_side_encryption"] = encryption
		}
		
		resources = append(resources, resource)
	}
	
	return resources, nil
}

// getBucketRegion returns the region of a bucket
func (c *AWSCollector) getBucketRegion(ctx context.Context, bucketName string) (string, error) {
	result, err := c.clients.S3.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return "", err
	}
	
	// AWS returns empty string for us-east-1
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}
	
	return string(result.LocationConstraint), nil
}

// getBucketTags fetches tags for a bucket
func (c *AWSCollector) getBucketTags(ctx context.Context, bucketName string) (map[string]string, error) {
	result, err := c.clients.S3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: &bucketName,
	})
	if err != nil {
		// Return empty tags if bucket has no tags or we can't access them
		return make(map[string]string), nil
	}
	
	tags := make(map[string]string)
	for _, tag := range result.TagSet {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}
	
	return tags, nil
}

// getBucketVersioning fetches versioning configuration for a bucket
func (c *AWSCollector) getBucketVersioning(ctx context.Context, bucketName string) (map[string]interface{}, error) {
	result, err := c.clients.S3.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return nil, err
	}
	
	versioning := map[string]interface{}{
		"enabled": string(result.Status) == "Enabled",
		"mfa_delete": string(result.MFADelete) == "Enabled",
	}
	
	return versioning, nil
}

// getBucketEncryption fetches encryption configuration for a bucket
func (c *AWSCollector) getBucketEncryption(ctx context.Context, bucketName string) (map[string]interface{}, error) {
	result, err := c.clients.S3.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: &bucketName,
	})
	if err != nil {
		// Return no encryption if bucket is not encrypted
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}
	
	encryption := map[string]interface{}{
		"enabled": true,
		"rules":   make([]map[string]interface{}, 0),
	}
	
	rules := make([]map[string]interface{}, 0)
	for _, rule := range result.ServerSideEncryptionConfiguration.Rules {
		if rule.ApplyServerSideEncryptionByDefault != nil {
			ruleMap := map[string]interface{}{
				"sse_algorithm": string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm),
			}
			
			if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
				ruleMap["kms_master_key_id"] = *rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID
			}
			
			rules = append(rules, ruleMap)
		}
	}
	encryption["rules"] = rules
	
	return encryption, nil
}