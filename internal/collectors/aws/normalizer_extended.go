package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	cloudformationTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	cloudwatchTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	elbv2Types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/yairfalse/vaino/pkg/types"
)

// NormalizeEBSVolume converts an EBS volume to VAINO format
func (n *Normalizer) NormalizeEBSVolume(volume ec2Types.Volume) types.Resource {
	return types.Resource{
		ID:       aws.ToString(volume.VolumeId),
		Type:     "aws_ebs_volume",
		Provider: "aws",
		Name:     getVolumeNameFromTags(volume.Tags),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"volume_type":       string(volume.VolumeType),
			"size":              aws.ToInt32(volume.Size),
			"iops":              aws.ToInt32(volume.Iops),
			"encrypted":         aws.ToBool(volume.Encrypted),
			"kms_key_id":        aws.ToString(volume.KmsKeyId),
			"availability_zone": aws.ToString(volume.AvailabilityZone),
			"state":             string(volume.State),
			"snapshot_id":       aws.ToString(volume.SnapshotId),
			"multi_attach":      aws.ToBool(volume.MultiAttachEnabled),
		},
		Tags: extractEC2Tags(volume.Tags),
	}
}

// NormalizeEBSSnapshot converts an EBS snapshot to VAINO format
func (n *Normalizer) NormalizeEBSSnapshot(snapshot ec2Types.Snapshot) types.Resource {
	return types.Resource{
		ID:       aws.ToString(snapshot.SnapshotId),
		Type:     "aws_ebs_snapshot",
		Provider: "aws",
		Name:     getSnapshotNameFromTags(snapshot.Tags),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"volume_id":    aws.ToString(snapshot.VolumeId),
			"volume_size":  aws.ToInt32(snapshot.VolumeSize),
			"description":  aws.ToString(snapshot.Description),
			"encrypted":    aws.ToBool(snapshot.Encrypted),
			"kms_key_id":   aws.ToString(snapshot.KmsKeyId),
			"owner_id":     aws.ToString(snapshot.OwnerId),
			"state":        string(snapshot.State),
			"progress":     aws.ToString(snapshot.Progress),
			"storage_tier": string(snapshot.StorageTier),
		},
		Tags: extractEC2Tags(snapshot.Tags),
	}
}

// NormalizeKeyPair converts a key pair to VAINO format
func (n *Normalizer) NormalizeKeyPair(keyPair ec2Types.KeyPairInfo) types.Resource {
	return types.Resource{
		ID:       aws.ToString(keyPair.KeyPairId),
		Type:     "aws_key_pair",
		Provider: "aws",
		Name:     aws.ToString(keyPair.KeyName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"key_name":         aws.ToString(keyPair.KeyName),
			"key_fingerprint":  aws.ToString(keyPair.KeyFingerprint),
			"key_type":         string(keyPair.KeyType),
			"public_key":       aws.ToString(keyPair.PublicKey),
			"create_time":      keyPair.CreateTime,
		},
		Tags: extractEC2Tags(keyPair.Tags),
	}
}

// NormalizeCloudFormationStack converts a CloudFormation stack to VAINO format
func (n *Normalizer) NormalizeCloudFormationStack(stack cloudformationTypes.Stack) types.Resource {
	return types.Resource{
		ID:       aws.ToString(stack.StackId),
		Type:     "aws_cloudformation_stack",
		Provider: "aws",
		Name:     aws.ToString(stack.StackName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"stack_name":        aws.ToString(stack.StackName),
			"description":       aws.ToString(stack.Description),
			"status":            string(stack.StackStatus),
			"status_reason":     aws.ToString(stack.StackStatusReason),
			"creation_time":     stack.CreationTime,
			"last_updated_time": stack.LastUpdatedTime,
			"role_arn":          aws.ToString(stack.RoleARN),
			"enable_termination_protection": aws.ToBool(stack.EnableTerminationProtection),
			"drift_status":      string(stack.DriftInformation.StackDriftStatus),
		},
		Tags: extractStackTags(stack.Tags),
	}
}

// NormalizeCloudFormationStackSet converts a CloudFormation stack set to VAINO format
func (n *Normalizer) NormalizeCloudFormationStackSet(stackSet cloudformationTypes.StackSet) types.Resource {
	return types.Resource{
		ID:       aws.ToString(stackSet.StackSetId),
		Type:     "aws_cloudformation_stack_set",
		Provider: "aws",
		Name:     aws.ToString(stackSet.StackSetName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"stack_set_name":  aws.ToString(stackSet.StackSetName),
			"description":     aws.ToString(stackSet.Description),
			"status":          string(stackSet.Status),
			"administration_role_arn": aws.ToString(stackSet.AdministrationRoleARN),
			"execution_role_name":     aws.ToString(stackSet.ExecutionRoleName),
			"auto_deployment": map[string]interface{}{
				"enabled": stackSet.AutoDeployment != nil && aws.ToBool(stackSet.AutoDeployment.Enabled),
			},
		},
		Tags: extractStackSetTags(stackSet.Tags),
	}
}

// NormalizeCloudWatchAlarm converts a CloudWatch alarm to VAINO format
func (n *Normalizer) NormalizeCloudWatchAlarm(alarm cloudwatchTypes.MetricAlarm) types.Resource {
	return types.Resource{
		ID:       aws.ToString(alarm.AlarmArn),
		Type:     "aws_cloudwatch_alarm",
		Provider: "aws",
		Name:     aws.ToString(alarm.AlarmName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"alarm_name":        aws.ToString(alarm.AlarmName),
			"alarm_description": aws.ToString(alarm.AlarmDescription),
			"metric_name":       aws.ToString(alarm.MetricName),
			"namespace":         aws.ToString(alarm.Namespace),
			"statistic":         string(alarm.Statistic),
			"comparison_operator": string(alarm.ComparisonOperator),
			"threshold":         aws.ToFloat64(alarm.Threshold),
			"evaluation_periods": aws.ToInt32(alarm.EvaluationPeriods),
			"period":            aws.ToInt32(alarm.Period),
			"treat_missing_data": aws.ToString(alarm.TreatMissingData),
			"state_value":       string(alarm.StateValue),
			"actions_enabled":   aws.ToBool(alarm.ActionsEnabled),
		},
		Tags: map[string]string{},
	}
}

// NormalizeCloudWatchCompositeAlarm converts a CloudWatch composite alarm to VAINO format
func (n *Normalizer) NormalizeCloudWatchCompositeAlarm(alarm cloudwatchTypes.CompositeAlarm) types.Resource {
	return types.Resource{
		ID:       aws.ToString(alarm.AlarmArn),
		Type:     "aws_cloudwatch_composite_alarm",
		Provider: "aws",
		Name:     aws.ToString(alarm.AlarmName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"alarm_name":        aws.ToString(alarm.AlarmName),
			"alarm_description": aws.ToString(alarm.AlarmDescription),
			"alarm_rule":        aws.ToString(alarm.AlarmRule),
			"state_value":       string(alarm.StateValue),
			"actions_enabled":   aws.ToBool(alarm.ActionsEnabled),
		},
		Tags: map[string]string{},
	}
}

// NormalizeCloudWatchLogGroup converts a CloudWatch log group to VAINO format
func (n *Normalizer) NormalizeCloudWatchLogGroup(logGroup logsTypes.LogGroup) types.Resource {
	return types.Resource{
		ID:       aws.ToString(logGroup.Arn),
		Type:     "aws_cloudwatch_log_group",
		Provider: "aws",
		Name:     aws.ToString(logGroup.LogGroupName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"log_group_name":     aws.ToString(logGroup.LogGroupName),
			"retention_in_days":  aws.ToInt32(logGroup.RetentionInDays),
			"kms_key_id":         aws.ToString(logGroup.KmsKeyId),
			"stored_bytes":       aws.ToInt64(logGroup.StoredBytes),
			"metric_filter_count": aws.ToInt32(logGroup.MetricFilterCount),
		},
		Tags: map[string]string{},
	}
}

// NormalizeCloudWatchDashboard converts a CloudWatch dashboard to VAINO format
func (n *Normalizer) NormalizeCloudWatchDashboard(dashboardName string, dashboardOutput interface{}) types.Resource {
	return types.Resource{
		ID:       dashboardName,
		Type:     "aws_cloudwatch_dashboard",
		Provider: "aws",
		Name:     dashboardName,
		Region:   n.region,
		Configuration: map[string]interface{}{
			"dashboard_name": dashboardName,
		},
		Tags: map[string]string{},
	}
}

// NormalizeClassicLoadBalancer converts a classic load balancer to VAINO format
func (n *Normalizer) NormalizeClassicLoadBalancer(lb elbTypes.LoadBalancerDescription) types.Resource {
	return types.Resource{
		ID:       aws.ToString(lb.LoadBalancerName),
		Type:     "aws_elb",
		Provider: "aws",
		Name:     aws.ToString(lb.LoadBalancerName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"load_balancer_name": aws.ToString(lb.LoadBalancerName),
			"dns_name":           aws.ToString(lb.DNSName),
			"scheme":             aws.ToString(lb.Scheme),
			"vpc_id":             aws.ToString(lb.VPCId),
			"availability_zones": lb.AvailabilityZones,
			"subnets":            lb.Subnets,
			"instances":          extractInstanceIds(lb.Instances),
		},
		Tags: map[string]string{},
	}
}

// NormalizeModernLoadBalancer converts an ALB/NLB to VAINO format
func (n *Normalizer) NormalizeModernLoadBalancer(lb elbv2Types.LoadBalancer) types.Resource {
	return types.Resource{
		ID:       aws.ToString(lb.LoadBalancerArn),
		Type:     "aws_" + string(lb.Type),
		Provider: "aws",
		Name:     aws.ToString(lb.LoadBalancerName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"load_balancer_name": aws.ToString(lb.LoadBalancerName),
			"load_balancer_arn":  aws.ToString(lb.LoadBalancerArn),
			"dns_name":           aws.ToString(lb.DNSName),
			"type":               string(lb.Type),
			"scheme":             string(lb.Scheme),
			"vpc_id":             aws.ToString(lb.VpcId),
			"state":              string(lb.State.Code),
			"ip_address_type":    string(lb.IpAddressType),
		},
		Tags: map[string]string{},
	}
}

// Helper functions for extracting names from tags
func getVolumeNameFromTags(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func getSnapshotNameFromTags(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func extractEC2Tags(tags []ec2Types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func extractStackTags(tags []cloudformationTypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func extractStackSetTags(tags []cloudformationTypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func extractInstanceIds(instances []elbTypes.Instance) []string {
	var ids []string
	for _, instance := range instances {
		ids = append(ids, aws.ToString(instance.InstanceId))
	}
	return ids
}