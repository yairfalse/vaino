package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/yairfalse/wgo/pkg/types"
)

// CollectVPCResources collects VPC, subnets, and related networking resources
func (c *AWSCollector) CollectVPCResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect VPCs
	vpcs, err := c.collectVPCs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect VPCs: %w", err)
	}
	resources = append(resources, vpcs...)

	// Collect subnets
	subnets, err := c.collectSubnets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect subnets: %w", err)
	}
	resources = append(resources, subnets...)

	return resources, nil
}

// collectVPCs fetches all VPCs in the region
func (c *AWSCollector) collectVPCs(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &ec2.DescribeVpcsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EC2.DescribeVpcs(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe VPCs: %w", err)
		}

		// Process VPCs
		for _, vpc := range result.Vpcs {
			resource := c.normalizer.NormalizeVPC(vpc)

			// Get VPC attributes
			if attributes, err := c.getVPCAttributes(ctx, *vpc.VpcId); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				for key, value := range attributes {
					resource.Configuration[key] = value
				}
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

// collectSubnets fetches all subnets in the region
func (c *AWSCollector) collectSubnets(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &ec2.DescribeSubnetsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.EC2.DescribeSubnets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe subnets: %w", err)
		}

		// Process subnets
		for _, subnet := range result.Subnets {
			resource := c.normalizer.NormalizeSubnet(subnet)
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

// getVPCAttributes fetches additional VPC attributes
func (c *AWSCollector) getVPCAttributes(ctx context.Context, vpcId string) (map[string]interface{}, error) {
	attributes := make(map[string]interface{})

	// Get DNS hostnames attribute
	dnsHostnamesResult, err := c.clients.EC2.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		VpcId:     &vpcId,
		Attribute: "enableDnsHostnames",
	})
	if err == nil && dnsHostnamesResult.EnableDnsHostnames != nil {
		attributes["enable_dns_hostnames"] = *dnsHostnamesResult.EnableDnsHostnames.Value
	}

	// Get DNS support attribute
	dnsSupportResult, err := c.clients.EC2.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		VpcId:     &vpcId,
		Attribute: "enableDnsSupport",
	})
	if err == nil && dnsSupportResult.EnableDnsSupport != nil {
		attributes["enable_dns_support"] = *dnsSupportResult.EnableDnsSupport.Value
	}

	// Get network address usage metrics attribute
	networkAddressUsageResult, err := c.clients.EC2.DescribeVpcAttribute(ctx, &ec2.DescribeVpcAttributeInput{
		VpcId:     &vpcId,
		Attribute: "enableNetworkAddressUsageMetrics",
	})
	if err == nil && networkAddressUsageResult.EnableNetworkAddressUsageMetrics != nil {
		attributes["enable_network_address_usage_metrics"] = *networkAddressUsageResult.EnableNetworkAddressUsageMetrics.Value
	}

	return attributes, nil
}
