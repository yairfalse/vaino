package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/yairfalse/vaino/pkg/types"
)

// CollectCloudWatchResources collects CloudWatch metrics, alarms, and log groups
func (c *AWSCollector) CollectCloudWatchResources(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource

	// Collect CloudWatch alarms
	alarms, err := c.collectCloudWatchAlarms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CloudWatch alarms: %w", err)
	}
	resources = append(resources, alarms...)

	// Collect CloudWatch log groups
	logGroups, err := c.collectCloudWatchLogGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CloudWatch log groups: %w", err)
	}
	resources = append(resources, logGroups...)

	// Collect CloudWatch dashboards
	dashboards, err := c.collectCloudWatchDashboards(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect CloudWatch dashboards: %w", err)
	}
	resources = append(resources, dashboards...)

	return resources, nil
}

// collectCloudWatchAlarms fetches all CloudWatch alarms
func (c *AWSCollector) collectCloudWatchAlarms(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &cloudwatch.DescribeAlarmsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.CloudWatch.DescribeAlarms(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe CloudWatch alarms: %w", err)
		}

		// Process metric alarms
		for _, alarm := range result.MetricAlarms {
			resource := c.normalizer.NormalizeCloudWatchAlarm(alarm)

			// Get alarm tags
			if tags, err := c.getCloudWatchAlarmTags(ctx, *alarm.AlarmArn); err == nil {
				resource.Tags = tags
			}

			resources = append(resources, resource)
		}

		// Process composite alarms
		for _, alarm := range result.CompositeAlarms {
			resource := c.normalizer.NormalizeCloudWatchCompositeAlarm(alarm)

			// Get alarm tags
			if tags, err := c.getCloudWatchAlarmTags(ctx, *alarm.AlarmArn); err == nil {
				resource.Tags = tags
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

// collectCloudWatchLogGroups fetches all CloudWatch log groups
func (c *AWSCollector) collectCloudWatchLogGroups(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &cloudwatchlogs.DescribeLogGroupsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.CloudWatchLogs.DescribeLogGroups(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log groups: %w", err)
		}

		// Process log groups
		for _, logGroup := range result.LogGroups {
			resource := c.normalizer.NormalizeCloudWatchLogGroup(logGroup)

			// Get log group tags
			if tags, err := c.getCloudWatchLogGroupTags(ctx, *logGroup.LogGroupName); err == nil {
				resource.Tags = tags
			}

			// Get retention settings and other details
			if details, err := c.getCloudWatchLogGroupDetails(ctx, *logGroup.LogGroupName); err == nil {
				if resource.Configuration == nil {
					resource.Configuration = make(map[string]interface{})
				}
				for k, v := range details {
					resource.Configuration[k] = v
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

// collectCloudWatchDashboards fetches all CloudWatch dashboards
func (c *AWSCollector) collectCloudWatchDashboards(ctx context.Context) ([]types.Resource, error) {
	var resources []types.Resource
	var nextToken *string

	for {
		input := &cloudwatch.ListDashboardsInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.clients.CloudWatch.ListDashboards(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list CloudWatch dashboards: %w", err)
		}

		// Process dashboards
		for _, dashboard := range result.DashboardEntries {
			// Get dashboard details
			detailInput := &cloudwatch.GetDashboardInput{
				DashboardName: dashboard.DashboardName,
			}

			detailResult, err := c.clients.CloudWatch.GetDashboard(ctx, detailInput)
			if err != nil {
				// Skip dashboards we can't access
				continue
			}

			resource := c.normalizer.NormalizeCloudWatchDashboard(*dashboard.DashboardName, detailResult)
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

// getCloudWatchAlarmTags fetches tags for a CloudWatch alarm
func (c *AWSCollector) getCloudWatchAlarmTags(ctx context.Context, alarmArn string) (map[string]string, error) {
	result, err := c.clients.CloudWatch.ListTagsForResource(ctx, &cloudwatch.ListTagsForResourceInput{
		ResourceARN: &alarmArn,
	})
	if err != nil {
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

// getCloudWatchLogGroupTags fetches tags for a CloudWatch log group
func (c *AWSCollector) getCloudWatchLogGroupTags(ctx context.Context, logGroupName string) (map[string]string, error) {
	result, err := c.clients.CloudWatchLogs.ListTagsLogGroup(ctx, &cloudwatchlogs.ListTagsLogGroupInput{
		LogGroupName: &logGroupName,
	})
	if err != nil {
		return make(map[string]string), nil
	}

	return result.Tags, nil
}

// getCloudWatchLogGroupDetails fetches additional details for a log group
func (c *AWSCollector) getCloudWatchLogGroupDetails(ctx context.Context, logGroupName string) (map[string]interface{}, error) {
	details := make(map[string]interface{})

	// Get metric filters
	filterResult, err := c.clients.CloudWatchLogs.DescribeMetricFilters(ctx, &cloudwatchlogs.DescribeMetricFiltersInput{
		LogGroupName: &logGroupName,
	})
	if err == nil && len(filterResult.MetricFilters) > 0 {
		var filters []map[string]interface{}
		for _, filter := range filterResult.MetricFilters {
			filterInfo := map[string]interface{}{
				"filter_name":    *filter.FilterName,
				"filter_pattern": *filter.FilterPattern,
			}
			if filter.MetricTransformations != nil && len(filter.MetricTransformations) > 0 {
				filterInfo["metric_name"] = *filter.MetricTransformations[0].MetricName
				filterInfo["metric_namespace"] = *filter.MetricTransformations[0].MetricNamespace
			}
			filters = append(filters, filterInfo)
		}
		details["metric_filters"] = filters
	}

	// Get subscription filters
	subscriptionResult, err := c.clients.CloudWatchLogs.DescribeSubscriptionFilters(ctx, &cloudwatchlogs.DescribeSubscriptionFiltersInput{
		LogGroupName: &logGroupName,
	})
	if err == nil && len(subscriptionResult.SubscriptionFilters) > 0 {
		var subscriptions []map[string]interface{}
		for _, sub := range subscriptionResult.SubscriptionFilters {
			subInfo := map[string]interface{}{
				"filter_name":     *sub.FilterName,
				"filter_pattern":  *sub.FilterPattern,
				"destination_arn": *sub.DestinationArn,
			}
			subscriptions = append(subscriptions, subInfo)
		}
		details["subscription_filters"] = subscriptions
	}

	return details, nil
}
