package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectEKSResources collects EKS clusters, node groups, and Fargate profiles
func (c *AWSCollector) CollectEKSResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect EKS clusters
	clusters, err := c.collectEKSClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect EKS clusters: %w", err)
	}
	resources = append(resources, clusters...)

	// For each cluster, collect node groups and Fargate profiles
	for _, cluster := range clusters {
		clusterName := cluster.Name

		// Collect node groups for this cluster
		nodeGroups, err := c.collectEKSNodeGroups(ctx, clusterName)
		if err != nil {
			// Continue with other clusters if one fails
			continue
		}
		resources = append(resources, nodeGroups...)

		// Collect Fargate profiles for this cluster
		fargateProfiles, err := c.collectEKSFargateProfiles(ctx, clusterName)
		if err != nil {
			// Continue with other clusters if one fails
			continue
		}
		resources = append(resources, fargateProfiles...)
	}

	return resources, nil
}

// collectEKSClusters fetches all EKS clusters
func (c *AWSCollector) collectEKSClusters(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &eks.ListClustersInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EKS.ListClusters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list EKS clusters: %w", err)
		}

		// Process each cluster
		for _, clusterName := range result.Clusters {
			// Get detailed information about the cluster
			describeInput := &eks.DescribeClusterInput{
				Name: &clusterName,
			}

			describeResult, err := c.clients.EKS.DescribeCluster(ctx, describeInput)
			if err != nil {
				// Skip clusters we can't access
				continue
			}

			resource := c.normalizer.NormalizeEKSCluster(*describeResult.Cluster)
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

// collectEKSNodeGroups fetches all EKS node groups for a cluster
func (c *AWSCollector) collectEKSNodeGroups(ctx context.Context, clusterName string) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &eks.ListNodegroupsInput{
			ClusterName: &clusterName,
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EKS.ListNodegroups(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list EKS node groups: %w", err)
		}

		// Process each node group
		for _, nodeGroupName := range result.Nodegroups {
			// Get detailed information about the node group
			describeInput := &eks.DescribeNodegroupInput{
				ClusterName:   &clusterName,
				NodegroupName: &nodeGroupName,
			}

			describeResult, err := c.clients.EKS.DescribeNodegroup(ctx, describeInput)
			if err != nil {
				// Skip node groups we can't access
				continue
			}

			resource := c.normalizer.NormalizeEKSNodeGroup(*describeResult.Nodegroup, clusterName)
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

// collectEKSFargateProfiles fetches all EKS Fargate profiles for a cluster
func (c *AWSCollector) collectEKSFargateProfiles(ctx context.Context, clusterName string) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &eks.ListFargateProfilesInput{
			ClusterName: &clusterName,
		}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EKS.ListFargateProfiles(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list EKS Fargate profiles: %w", err)
		}

		// Process each Fargate profile
		for _, profileName := range result.FargateProfileNames {
			// Get detailed information about the Fargate profile
			describeInput := &eks.DescribeFargateProfileInput{
				ClusterName:        &clusterName,
				FargateProfileName: &profileName,
			}

			describeResult, err := c.clients.EKS.DescribeFargateProfile(ctx, describeInput)
			if err != nil {
				// Skip profiles we can't access
				continue
			}

			resource := c.normalizer.NormalizeEKSFargateProfile(*describeResult.FargateProfile, clusterName)
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
