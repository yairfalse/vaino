package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/yairfalse/wgo/pkg/types"
)

// CollectLambdaResources collects Lambda functions
func (c *AWSCollector) CollectLambdaResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		input := &lambda.ListFunctionsInput{}
		if marker != nil {
			input.Marker = marker
		}

		result, err := c.clients.Lambda.ListFunctions(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list Lambda functions: %w", err)
		}

		// Process Lambda functions
		for _, function := range result.Functions {
			resource := c.normalizer.NormalizeLambdaFunction(function)

			// Try to get function tags
			if tags, err := c.getLambdaFunctionTags(ctx, *function.FunctionArn); err == nil {
				resource.Tags = tags
			}

			// Try to get function environment variables
			if envVars, err := c.getLambdaFunctionConfig(ctx, *function.FunctionName); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["environment"] = envVars
			}

			resources = append(resources, resource)
		}

		// Check if there are more results
		marker = result.NextMarker
		if marker == nil {
			break
		}
	}

	return resources, nil
}

// getLambdaFunctionTags fetches tags for a Lambda function
func (c *AWSCollector) getLambdaFunctionTags(ctx context.Context, functionArn string) (map[string]string, error) {
	result, err := c.clients.Lambda.ListTags(ctx, &lambda.ListTagsInput{
		Resource: &functionArn,
	})
	if err != nil {
		return make(map[string]string), nil
	}

	return result.Tags, nil
}

// getLambdaFunctionConfig fetches additional configuration for a Lambda function
func (c *AWSCollector) getLambdaFunctionConfig(ctx context.Context, functionName string) (map[string]interface{}, error) {
	result, err := c.clients.Lambda.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: &functionName,
	})
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{})

	// Add environment variables
	if result.Configuration.Environment != nil && result.Configuration.Environment.Variables != nil {
		config["variables"] = result.Configuration.Environment.Variables
	}

	// Add VPC configuration
	if result.Configuration.VpcConfig != nil {
		vpcConfig := map[string]interface{}{
			"subnet_ids":         result.Configuration.VpcConfig.SubnetIds,
			"security_group_ids": result.Configuration.VpcConfig.SecurityGroupIds,
		}
		if result.Configuration.VpcConfig.VpcId != nil {
			vpcConfig["vpc_id"] = *result.Configuration.VpcConfig.VpcId
		}
		config["vpc_config"] = vpcConfig
	}

	// Add dead letter config
	if result.Configuration.DeadLetterConfig != nil && result.Configuration.DeadLetterConfig.TargetArn != nil {
		config["dead_letter_config"] = map[string]interface{}{
			"target_arn": *result.Configuration.DeadLetterConfig.TargetArn,
		}
	}

	return config, nil
}
