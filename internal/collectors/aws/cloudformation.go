package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectCloudFormationResources collects CloudFormation stacks and stack sets
func (c *AWSCollector) CollectCloudFormationResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect CloudFormation stacks
	stacks, err := c.collectCloudFormationStacks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CloudFormation stacks: %w", err)
	}
	resources = append(resources, stacks...)

	// Collect CloudFormation stack sets
	stackSets, err := c.collectCloudFormationStackSets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CloudFormation stack sets: %w", err)
	}
	resources = append(resources, stackSets...)

	return resources, nil
}

// collectCloudFormationStacks fetches all CloudFormation stacks
func (c *AWSCollector) collectCloudFormationStacks(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &cloudformation.DescribeStacksInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.CloudFormation.DescribeStacks(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe CloudFormation stacks: %w", err)
		}

		// Process stacks
		for _, stack := range result.Stacks {
			resource := c.normalizer.NormalizeCloudFormationStack(stack)

			// Get stack tags
			resource.Tags = make(map[string]string)
			for _, tag := range stack.Tags {
				if tag.Key != nil && tag.Value != nil {
					resource.Tags[*tag.Key] = *tag.Value
				}
			}

			// Get stack resources and events for additional context
			if stackDetails, err := c.getCloudFormationStackDetails(ctx, *stack.StackName); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				for k, v := range stackDetails {
					resource.Configuration[k] = v
				}
			}

			// Get stack drift detection status
			if driftInfo, err := c.getCloudFormationStackDrift(ctx, *stack.StackName); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["drift_detection"] = driftInfo
			}

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

// collectCloudFormationStackSets fetches all CloudFormation stack sets
func (c *AWSCollector) collectCloudFormationStackSets(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &cloudformation.ListStackSetsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.CloudFormation.ListStackSets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list CloudFormation stack sets: %w", err)
		}

		// Process stack sets
		for _, stackSetSummary := range result.Summaries {
			// Get detailed stack set information
			describeInput := &cloudformation.DescribeStackSetInput{
				StackSetName: stackSetSummary.StackSetName,
			}

			describeResult, err := c.clients.CloudFormation.DescribeStackSet(ctx, describeInput)
			if err != nil {
				// Skip stack sets we can't access
				continue
			}

			resource := c.normalizer.NormalizeCloudFormationStackSet(*describeResult.StackSet)

			// Get stack set tags
			resource.Tags = make(map[string]string)
			for _, tag := range describeResult.StackSet.Tags {
				if tag.Key != nil && tag.Value != nil {
					resource.Tags[*tag.Key] = *tag.Value
				}
			}

			// Get stack instances for this stack set
			if instances, err := c.getCloudFormationStackSetInstances(ctx, *stackSetSummary.StackSetName); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["stack_instances"] = instances
			}

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

// getCloudFormationStackDetails fetches additional details for a CloudFormation stack
func (c *AWSCollector) getCloudFormationStackDetails(ctx context.Context, stackName string) (map[string]interface{}, error) {
	details := make(map[string]interface{})

	// Get stack resources
	resourcesResult, err := c.clients.CloudFormation.ListStackResources(ctx, &cloudformation.ListStackResourcesInput{
		StackName: &stackName,
	})
	if err == nil {
		var stackResources []map[string]interface{}
		for _, resource := range resourcesResult.StackResourceSummaries {
			resourceInfo := map[string]interface{}{
				"logical_resource_id": *resource.LogicalResourceId,
				"resource_type":       *resource.ResourceType,
				"resource_status":     string(resource.ResourceStatus),
			}
			if resource.PhysicalResourceId != nil {
				resourceInfo["physical_resource_id"] = *resource.PhysicalResourceId
			}
			if resource.ResourceStatusReason != nil {
				resourceInfo["status_reason"] = *resource.ResourceStatusReason
			}
			stackResources = append(stackResources, resourceInfo)
		}
		details["resources"] = stackResources
		details["resource_count"] = len(stackResources)
	}

	// Get stack outputs
	if len(stackResources) > 0 {
		// Re-describe the stack to get outputs
		describeResult, err := c.clients.CloudFormation.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: &stackName,
		})
		if err == nil && len(describeResult.Stacks) > 0 {
			stack := describeResult.Stacks[0]
			if len(stack.Outputs) > 0 {
				var outputs []map[string]interface{}
				for _, output := range stack.Outputs {
					outputInfo := map[string]interface{}{
						"output_key": *output.OutputKey,
					}
					if output.OutputValue != nil {
						outputInfo["output_value"] = *output.OutputValue
					}
					if output.Description != nil {
						outputInfo["description"] = *output.Description
					}
					outputs = append(outputs, outputInfo)
				}
				details["outputs"] = outputs
			}

			// Add parameters
			if len(stack.Parameters) > 0 {
				var parameters []map[string]interface{}
				for _, param := range stack.Parameters {
					paramInfo := map[string]interface{}{
						"parameter_key": *param.ParameterKey,
					}
					if param.ParameterValue != nil {
						paramInfo["parameter_value"] = *param.ParameterValue
					}
					parameters = append(parameters, paramInfo)
				}
				details["parameters"] = parameters
			}
		}
	}

	return details, nil
}

// getCloudFormationStackDrift fetches drift detection information for a stack
func (c *AWSCollector) getCloudFormationStackDrift(ctx context.Context, stackName string) (map[string]interface{}, error) {
	// Detect stack drift
	detectInput := &cloudformation.DetectStackDriftInput{
		StackName: &stackName,
	}

	detectResult, err := c.clients.CloudFormation.DetectStackDrift(ctx, detectInput)
	if err != nil {
		return nil, err
	}

	driftInfo := map[string]interface{}{
		"stack_drift_detection_id": *detectResult.StackDriftDetectionId,
	}

	// Get drift detection status
	statusInput := &cloudformation.DescribeStackDriftDetectionStatusInput{
		StackDriftDetectionId: detectResult.StackDriftDetectionId,
	}

	statusResult, err := c.clients.CloudFormation.DescribeStackDriftDetectionStatus(ctx, statusInput)
	if err == nil {
		driftInfo["detection_status"] = string(statusResult.DetectionStatus)
		if statusResult.StackDriftStatus != nil {
			driftInfo["stack_drift_status"] = string(*statusResult.StackDriftStatus)
		}
		if statusResult.DriftedStackResourceCount != nil {
			driftInfo["drifted_resource_count"] = *statusResult.DriftedStackResourceCount
		}
	}

	return driftInfo, nil
}

// getCloudFormationStackSetInstances fetches stack instances for a stack set
func (c *AWSCollector) getCloudFormationStackSetInstances(ctx context.Context, stackSetName string) ([]map[string]interface{}, error) {
	var instances []map[string]interface{}
	var nextToken *string

	for {
		input := &cloudformation.ListStackInstancesInput{
			StackSetName: &stackSetName,
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.CloudFormation.ListStackInstances(ctx, input)
		if err != nil {
			return instances, err
		}

		for _, instance := range result.Summaries {
			instanceInfo := map[string]interface{}{
				"account": *instance.Account,
				"region":  *instance.Region,
				"status":  string(instance.Status),
			}
			if instance.StatusReason != nil {
				instanceInfo["status_reason"] = *instance.StatusReason
			}
			if instance.StackId != nil {
				instanceInfo["stack_id"] = *instance.StackId
			}
			instances = append(instances, instanceInfo)
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return instances, nil
}
