package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectECSResources collects ECS clusters, services, and tasks
func (c *AWSCollector) CollectECSResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect ECS clusters
	clusters, err := c.collectECSClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect ECS clusters: %w", err)
	}
	resources = append(resources, clusters...)

	// For each cluster, collect services and tasks
	for _, cluster := range clusters {
		clusterArn := cluster.ID

		// Collect services for this cluster
		services, err := c.collectECSServices(ctx, clusterArn)
		if err != nil {
			// Continue with other clusters if one fails
			continue
		}
		resources = append(resources, services...)

		// Collect tasks for this cluster
		tasks, err := c.collectECSTasks(ctx, clusterArn)
		if err != nil {
			// Continue with other clusters if one fails
			continue
		}
		resources = append(resources, tasks...)
	}

	return resources, nil
}

// collectECSClusters fetches all ECS clusters
func (c *AWSCollector) collectECSClusters(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &ecs.ListClustersInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.ECS.ListClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list ECS clusters: %w", err)
		}

		if len(result.ClusterArns) > 0 {
			// Get detailed information about clusters
			describeInput := &ecs.DescribeClustersInput{
				Clusters: result.ClusterArns,
				Include:  []string{"ATTACHMENTS", "CONFIGURATIONS", "STATISTICS", "TAGS"},
			}

			describeResult, err := c.clients.ECS.DescribeClusters(ctx, describeInput)
			if err != nil {
				return nil, fmt.Errorf("failed to describe ECS clusters: %w", err)
			}

			// Process each cluster
			for _, cluster := range describeResult.Clusters {
				resource := c.normalizer.NormalizeECSCluster(cluster)
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

// collectECSServices fetches all ECS services for a cluster
func (c *AWSCollector) collectECSServices(ctx context.Context, clusterArn string) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &ecs.ListServicesInput{
			Cluster: &clusterArn,
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.ECS.ListServices(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list ECS services: %w", err)
		}

		if len(result.ServiceArns) > 0 {
			// Get detailed information about services
			describeInput := &ecs.DescribeServicesInput{
				Cluster:  &clusterArn,
				Services: result.ServiceArns,
				Include:  []string{"TAGS"},
			}

			describeResult, err := c.clients.ECS.DescribeServices(ctx, describeInput)
			if err != nil {
				return nil, fmt.Errorf("failed to describe ECS services: %w", err)
			}

			// Process each service
			for _, service := range describeResult.Services {
				resource := c.normalizer.NormalizeECSService(service, clusterArn)
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

// collectECSTasks fetches all ECS tasks for a cluster
func (c *AWSCollector) collectECSTasks(ctx context.Context, clusterArn string) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &ecs.ListTasksInput{
			Cluster: &clusterArn,
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.ECS.ListTasks(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list ECS tasks: %w", err)
		}

		if len(result.TaskArns) > 0 {
			// Get detailed information about tasks
			describeInput := &ecs.DescribeTasksInput{
				Cluster: &clusterArn,
				Tasks:   result.TaskArns,
				Include: []string{"TAGS"},
			}

			describeResult, err := c.clients.ECS.DescribeTasks(ctx, describeInput)
			if err != nil {
				return nil, fmt.Errorf("failed to describe ECS tasks: %w", err)
			}

			// Process each task
			for _, task := range describeResult.Tasks {
				resource := c.normalizer.NormalizeECSTask(task, clusterArn)
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
