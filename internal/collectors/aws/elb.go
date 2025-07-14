package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectELBResources collects Elastic Load Balancer resources (Classic, ALB, NLB)
func (c *AWSCollector) CollectELBResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect Classic Load Balancers (ELB v1)
	classicLBs, err := c.collectClassicLoadBalancers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Classic Load Balancers: %w", err)
	}
	resources = append(resources, classicLBs...)

	// Collect Application and Network Load Balancers (ELB v2)
	modernLBs, err := c.collectModernLoadBalancers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect ALB/NLB Load Balancers: %w", err)
	}
	resources = append(resources, modernLBs...)

	return resources, nil
}

// collectClassicLoadBalancers fetches all Classic Load Balancers (ELB v1)
func (c *AWSCollector) collectClassicLoadBalancers(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		input := &elasticloadbalancing.DescribeLoadBalancersInput{}
		if marker != nil {
			input.Marker = marker
		}

		result, err := c.clients.ELB.DescribeLoadBalancers(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe Classic Load Balancers: %w", err)
		}

		// Process Classic Load Balancers
		for _, lb := range result.LoadBalancerDescriptions {
			resource := c.normalizer.NormalizeClassicLoadBalancer(lb)

			// Get load balancer tags
			if tags, err := c.getClassicLoadBalancerTags(ctx, *lb.LoadBalancerName); err == nil {
				resource.Tags = tags
			}

			// Get load balancer attributes
			if attributes, err := c.getClassicLoadBalancerAttributes(ctx, *lb.LoadBalancerName); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["attributes"] = attributes
			}

			// Get health check configuration
			if lb.HealthCheck != nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["health_check"] = map[string]interface{}{
					"target":              *lb.HealthCheck.Target,
					"interval":            *lb.HealthCheck.Interval,
					"timeout":             *lb.HealthCheck.Timeout,
					"healthy_threshold":   *lb.HealthCheck.HealthyThreshold,
					"unhealthy_threshold": *lb.HealthCheck.UnhealthyThreshold,
				}
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

// collectModernLoadBalancers fetches all ALB and NLB Load Balancers (ELB v2)
func (c *AWSCollector) collectModernLoadBalancers(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var marker *string

	for {
		input := &elasticloadbalancingv2.DescribeLoadBalancersInput{}
		if marker != nil {
			input.Marker = marker
		}

		result, err := c.clients.ELBv2.DescribeLoadBalancers(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe ALB/NLB Load Balancers: %w", err)
		}

		// Process ALB/NLB Load Balancers
		for _, lb := range result.LoadBalancers {
			resource := c.normalizer.NormalizeModernLoadBalancer(lb)

			// Get load balancer tags
			if tags, err := c.getModernLoadBalancerTags(ctx, *lb.LoadBalancerArn); err == nil {
				resource.Tags = tags
			}

			// Get load balancer attributes
			if attributes, err := c.getModernLoadBalancerAttributes(ctx, *lb.LoadBalancerArn); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["attributes"] = attributes
			}

			// Get listeners for this load balancer
			if listeners, err := c.getLoadBalancerListeners(ctx, *lb.LoadBalancerArn); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["listeners"] = listeners
			}

			// Get target groups for this load balancer
			if targetGroups, err := c.getLoadBalancerTargetGroups(ctx, *lb.LoadBalancerArn); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				resource.Configuration["target_groups"] = targetGroups
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

// getClassicLoadBalancerTags fetches tags for a Classic Load Balancer
func (c *AWSCollector) getClassicLoadBalancerTags(ctx context.Context, lbName string) (map[string]string, error) {
	result, err := c.clients.ELB.DescribeTags(ctx, &elasticloadbalancing.DescribeTagsInput{
		LoadBalancerNames: []string{lbName},
	})
	if err != nil {
		return make(map[string]string), nil
	}

	tags := make(map[string]string)
	for _, tagDesc := range result.TagDescriptions {
		for _, tag := range tagDesc.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	return tags, nil
}

// getModernLoadBalancerTags fetches tags for an ALB/NLB Load Balancer
func (c *AWSCollector) getModernLoadBalancerTags(ctx context.Context, lbArn string) (map[string]string, error) {
	result, err := c.clients.ELBv2.DescribeTags(ctx, &elasticloadbalancingv2.DescribeTagsInput{
		ResourceArns: []string{lbArn},
	})
	if err != nil {
		return make(map[string]string), nil
	}

	tags := make(map[string]string)
	for _, tagDesc := range result.TagDescriptions {
		for _, tag := range tagDesc.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}

	return tags, nil
}

// getClassicLoadBalancerAttributes fetches attributes for a Classic Load Balancer
func (c *AWSCollector) getClassicLoadBalancerAttributes(ctx context.Context, lbName string) (map[string]interface{}, error) {
	result, err := c.clients.ELB.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancing.DescribeLoadBalancerAttributesInput{
		LoadBalancerName: &lbName,
	})
	if err != nil {
		return nil, err
	}

	attributes := make(map[string]interface{})
	if result.LoadBalancerAttributes != nil {
		if result.LoadBalancerAttributes.AccessLog != nil {
			attributes["access_log_enabled"] = result.LoadBalancerAttributes.AccessLog.Enabled
			if result.LoadBalancerAttributes.AccessLog.S3BucketName != nil {
				attributes["access_log_s3_bucket"] = *result.LoadBalancerAttributes.AccessLog.S3BucketName
			}
		}
		if result.LoadBalancerAttributes.ConnectionDraining != nil {
			attributes["connection_draining_enabled"] = result.LoadBalancerAttributes.ConnectionDraining.Enabled
			if result.LoadBalancerAttributes.ConnectionDraining.Timeout != nil {
				attributes["connection_draining_timeout"] = *result.LoadBalancerAttributes.ConnectionDraining.Timeout
			}
		}
		if result.LoadBalancerAttributes.ConnectionSettings != nil {
			attributes["idle_timeout"] = *result.LoadBalancerAttributes.ConnectionSettings.IdleTimeout
		}
		if result.LoadBalancerAttributes.CrossZoneLoadBalancing != nil {
			attributes["cross_zone_load_balancing"] = result.LoadBalancerAttributes.CrossZoneLoadBalancing.Enabled
		}
	}

	return attributes, nil
}

// getModernLoadBalancerAttributes fetches attributes for an ALB/NLB Load Balancer
func (c *AWSCollector) getModernLoadBalancerAttributes(ctx context.Context, lbArn string) (map[string]interface{}, error) {
	result, err := c.clients.ELBv2.DescribeLoadBalancerAttributes(ctx, &elasticloadbalancingv2.DescribeLoadBalancerAttributesInput{
		LoadBalancerArn: &lbArn,
	})
	if err != nil {
		return nil, err
	}

	attributes := make(map[string]interface{})
	for _, attr := range result.Attributes {
		if attr.Key != nil && attr.Value != nil {
			attributes[*attr.Key] = *attr.Value
		}
	}

	return attributes, nil
}

// getLoadBalancerListeners fetches listeners for an ALB/NLB Load Balancer
func (c *AWSCollector) getLoadBalancerListeners(ctx context.Context, lbArn string) ([]map[string]interface{}, error) {
	var listeners []map[string]interface{}
	var marker *string

	for {
		input := &elasticloadbalancingv2.DescribeListenersInput{
			LoadBalancerArn: &lbArn,
		}
		if marker != nil {
			input.Marker = marker
		}

		result, err := c.clients.ELBv2.DescribeListeners(ctx, input)
		if err != nil {
			return listeners, err
		}

		for _, listener := range result.Listeners {
			listenerInfo := map[string]interface{}{
				"port":     *listener.Port,
				"protocol": string(listener.Protocol),
			}

			if listener.SslPolicy != nil {
				listenerInfo["ssl_policy"] = *listener.SslPolicy
			}

			if len(listener.Certificates) > 0 {
				var certs []string
				for _, cert := range listener.Certificates {
					if cert.CertificateArn != nil {
						certs = append(certs, *cert.CertificateArn)
					}
				}
				listenerInfo["certificates"] = certs
			}

			// Get default actions
			if len(listener.DefaultActions) > 0 {
				var actions []map[string]interface{}
				for _, action := range listener.DefaultActions {
					actionInfo := map[string]interface{}{
						"type": string(action.Type),
					}
					if action.TargetGroupArn != nil {
						actionInfo["target_group_arn"] = *action.TargetGroupArn
					}
					actions = append(actions, actionInfo)
				}
				listenerInfo["default_actions"] = actions
			}

			listeners = append(listeners, listenerInfo)
		}

		marker = result.NextMarker
		if marker == nil {
			break
		}
	}

	return listeners, nil
}

// getLoadBalancerTargetGroups fetches target groups for an ALB/NLB Load Balancer
func (c *AWSCollector) getLoadBalancerTargetGroups(ctx context.Context, lbArn string) ([]map[string]interface{}, error) {
	result, err := c.clients.ELBv2.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
		LoadBalancerArn: &lbArn,
	})
	if err != nil {
		return nil, err
	}

	var targetGroups []map[string]interface{}
	for _, tg := range result.TargetGroups {
		tgInfo := map[string]interface{}{
			"name":     *tg.TargetGroupName,
			"port":     *tg.Port,
			"protocol": string(tg.Protocol),
		}

		if tg.HealthCheckPath != nil {
			tgInfo["health_check_path"] = *tg.HealthCheckPath
		}
		tgInfo["health_check_protocol"] = string(tg.HealthCheckProtocol)
		if tg.HealthCheckIntervalSeconds != nil {
			tgInfo["health_check_interval"] = *tg.HealthCheckIntervalSeconds
		}

		// Get target health for this target group
		if targets, err := c.getTargetGroupTargets(ctx, *tg.TargetGroupArn); err == nil {
			tgInfo["targets"] = targets
		}

		targetGroups = append(targetGroups, tgInfo)
	}

	return targetGroups, nil
}

// getTargetGroupTargets fetches target health for a target group
func (c *AWSCollector) getTargetGroupTargets(ctx context.Context, tgArn string) ([]map[string]interface{}, error) {
	result, err := c.clients.ELBv2.DescribeTargetHealth(ctx, &elasticloadbalancingv2.DescribeTargetHealthInput{
		TargetGroupArn: &tgArn,
	})
	if err != nil {
		return nil, err
	}

	var targets []map[string]interface{}
	for _, target := range result.TargetHealthDescriptions {
		targetInfo := map[string]interface{}{
			"id":   *target.Target.Id,
			"port": *target.Target.Port,
		}

		if target.TargetHealth != nil {
			targetInfo["health_state"] = string(target.TargetHealth.State)
			targetInfo["health_reason"] = string(target.TargetHealth.Reason)
			if target.TargetHealth.Description != nil {
				targetInfo["health_description"] = *target.TargetHealth.Description
			}
		}

		targets = append(targets, targetInfo)
	}

	return targets, nil
}
