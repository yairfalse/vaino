package aws

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	logsTypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	elbv2Types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/yairfalse/vaino/pkg/types"
)

// Normalizer converts AWS resources to VAINO format
type Normalizer struct {
	region string
}

// NewNormalizer creates a new AWS resource normalizer
func NewNormalizer(region string) *Normalizer {
	return &Normalizer{region: region}
}

// NormalizeEC2Instance converts an EC2 instance to VAINO format
func (n *Normalizer) NormalizeEC2Instance(instance ec2Types.Instance) types.Resource {
	return types.Resource{
		ID:       aws.ToString(instance.InstanceId),
		Type:     "aws_instance",
		Provider: "aws",
		Name:     getInstanceName(instance.Tags),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"instance_type":      string(instance.InstanceType),
			"state":              string(instance.State.Name),
			"vpc_id":             aws.ToString(instance.VpcId),
			"subnet_id":          aws.ToString(instance.SubnetId),
			"availability_zone":  aws.ToString(instance.Placement.AvailabilityZone),
			"private_ip_address": aws.ToString(instance.PrivateIpAddress),
			"public_ip_address":  aws.ToString(instance.PublicIpAddress),
			"image_id":           aws.ToString(instance.ImageId),
			"key_name":           aws.ToString(instance.KeyName),
			"security_groups":    normalizeSecurityGroupRefs(instance.SecurityGroups),
			"monitoring":         instance.Monitoring.State == ec2Types.MonitoringStateEnabled,
		},
		Tags: convertAWSTags(instance.Tags),
		Metadata: types.ResourceMetadata{
			UpdatedAt: aws.ToTime(instance.LaunchTime),
		},
	}
}

// NormalizeSecurityGroup converts a security group to VAINO format
func (n *Normalizer) NormalizeSecurityGroup(sg ec2Types.SecurityGroup) types.Resource {
	return types.Resource{
		ID:       aws.ToString(sg.GroupId),
		Type:     "aws_security_group",
		Provider: "aws",
		Name:     aws.ToString(sg.GroupName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"name":        aws.ToString(sg.GroupName),
			"description": aws.ToString(sg.Description),
			"vpc_id":      aws.ToString(sg.VpcId),
			"ingress":     normalizeSecurityGroupRules(sg.IpPermissions),
			"egress":      normalizeSecurityGroupRules(sg.IpPermissionsEgress),
		},
		Tags: convertAWSTags(sg.Tags),
		Metadata: types.ResourceMetadata{
			UpdatedAt: time.Now(), // Security groups don't have a last modified time
		},
	}
}

// NormalizeS3Bucket converts an S3 bucket to VAINO format
func (n *Normalizer) NormalizeS3Bucket(bucket s3Types.Bucket) types.Resource {
	return types.Resource{
		ID:       aws.ToString(bucket.Name),
		Type:     "aws_s3_bucket",
		Provider: "aws",
		Name:     aws.ToString(bucket.Name),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"bucket": aws.ToString(bucket.Name),
		},
		Tags: make(map[string]string), // Tags will be fetched separately
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(bucket.CreationDate),
		},
	}
}

// NormalizeVPC converts a VPC to VAINO format
func (n *Normalizer) NormalizeVPC(vpc ec2Types.Vpc) types.Resource {
	return types.Resource{
		ID:       aws.ToString(vpc.VpcId),
		Type:     "aws_vpc",
		Provider: "aws",
		Name:     getVPCName(vpc.Tags),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"cidr_block":                           aws.ToString(vpc.CidrBlock),
			"state":                                string(vpc.State),
			"dhcp_options_id":                      aws.ToString(vpc.DhcpOptionsId),
			"instance_tenancy":                     string(vpc.InstanceTenancy),
			"enable_dns_hostnames":                 false, // Will be set separately
			"enable_dns_support":                   false, // Will be set separately
			"enable_network_address_usage_metrics": false, // Will be set separately
		},
		Tags: convertAWSTags(vpc.Tags),
		Metadata: types.ResourceMetadata{
			UpdatedAt: time.Now(), // VPCs don't have a last modified time
		},
	}
}

// NormalizeSubnet converts a subnet to VAINO format
func (n *Normalizer) NormalizeSubnet(subnet ec2Types.Subnet) types.Resource {
	return types.Resource{
		ID:       aws.ToString(subnet.SubnetId),
		Type:     "aws_subnet",
		Provider: "aws",
		Name:     getSubnetName(subnet.Tags),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"vpc_id":                          aws.ToString(subnet.VpcId),
			"cidr_block":                      aws.ToString(subnet.CidrBlock),
			"availability_zone":               aws.ToString(subnet.AvailabilityZone),
			"availability_zone_id":            aws.ToString(subnet.AvailabilityZoneId),
			"state":                           string(subnet.State),
			"map_public_ip_on_launch":         aws.ToBool(subnet.MapPublicIpOnLaunch),
			"assign_ipv6_address_on_creation": aws.ToBool(subnet.AssignIpv6AddressOnCreation),
		},
		Tags: convertAWSTags(subnet.Tags),
		Metadata: types.ResourceMetadata{
			UpdatedAt: time.Now(), // Subnets don't have a last modified time
		},
	}
}

// NormalizeRDSInstance converts an RDS instance to VAINO format
func (n *Normalizer) NormalizeRDSInstance(instance rdsTypes.DBInstance) types.Resource {
	return types.Resource{
		ID:       aws.ToString(instance.DBInstanceIdentifier),
		Type:     "aws_db_instance",
		Provider: "aws",
		Name:     aws.ToString(instance.DBInstanceIdentifier),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"db_instance_identifier":  aws.ToString(instance.DBInstanceIdentifier),
			"db_instance_class":       aws.ToString(instance.DBInstanceClass),
			"engine":                  aws.ToString(instance.Engine),
			"engine_version":          aws.ToString(instance.EngineVersion),
			"db_name":                 aws.ToString(instance.DBName),
			"username":                aws.ToString(instance.MasterUsername),
			"allocated_storage":       aws.ToInt32(instance.AllocatedStorage),
			"storage_type":            aws.ToString(instance.StorageType),
			"storage_encrypted":       aws.ToBool(instance.StorageEncrypted),
			"multi_az":                aws.ToBool(instance.MultiAZ),
			"publicly_accessible":     aws.ToBool(instance.PubliclyAccessible),
			"backup_retention_period": aws.ToInt32(instance.BackupRetentionPeriod),
			"db_subnet_group_name":    aws.ToString(instance.DBSubnetGroup.DBSubnetGroupName),
			"vpc_security_group_ids":  normalizeDBSecurityGroups(instance.VpcSecurityGroups),
		},
		Tags: make(map[string]string), // Tags will be fetched separately
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(instance.InstanceCreateTime),
		},
	}
}

// NormalizeLambdaFunction converts a Lambda function to VAINO format
func (n *Normalizer) NormalizeLambdaFunction(function lambdaTypes.FunctionConfiguration) types.Resource {
	return types.Resource{
		ID:       aws.ToString(function.FunctionArn),
		Type:     "aws_lambda_function",
		Provider: "aws",
		Name:     aws.ToString(function.FunctionName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"function_name": aws.ToString(function.FunctionName),
			"role":          aws.ToString(function.Role),
			"handler":       aws.ToString(function.Handler),
			"runtime":       string(function.Runtime),
			"timeout":       aws.ToInt32(function.Timeout),
			"memory_size":   aws.ToInt32(function.MemorySize),
			"description":   aws.ToString(function.Description),
			"kms_key_arn":   aws.ToString(function.KMSKeyArn),
		},
		Tags: make(map[string]string), // Tags will be fetched separately
		Metadata: types.ResourceMetadata{
			UpdatedAt: parseTimeString(function.LastModified),
		},
	}
}

// Helper functions

func getInstanceName(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func getVPCName(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func getSubnetName(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func convertAWSTags(tags []ec2Types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	return result
}

func normalizeSecurityGroupRefs(groups []ec2Types.GroupIdentifier) []string {
	var result []string
	for _, group := range groups {
		result = append(result, aws.ToString(group.GroupId))
	}
	return result
}

func normalizeSecurityGroupRules(rules []ec2Types.IpPermission) []map[string]interface{} {
	var result []map[string]interface{}
	for _, rule := range rules {
		ruleMap := map[string]interface{}{
			"ip_protocol": aws.ToString(rule.IpProtocol),
			"from_port":   aws.ToInt32(rule.FromPort),
			"to_port":     aws.ToInt32(rule.ToPort),
		}

		// Add CIDR blocks
		var cidrBlocks []string
		for _, ipRange := range rule.IpRanges {
			cidrBlocks = append(cidrBlocks, aws.ToString(ipRange.CidrIp))
		}
		ruleMap["cidr_blocks"] = cidrBlocks

		// Add security group references
		var securityGroups []string
		for _, userIdGroupPair := range rule.UserIdGroupPairs {
			securityGroups = append(securityGroups, aws.ToString(userIdGroupPair.GroupId))
		}
		ruleMap["security_groups"] = securityGroups

		result = append(result, ruleMap)
	}
	return result
}

func normalizeDBSecurityGroups(groups []rdsTypes.VpcSecurityGroupMembership) []string {
	var result []string
	for _, group := range groups {
		result = append(result, aws.ToString(group.VpcSecurityGroupId))
	}
	return result
}

// parseTimeString parses a time string from AWS Lambda
func parseTimeString(timeStr *string) time.Time {
	if timeStr == nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, *timeStr)
	if err != nil {
		return time.Time{}
	}
	return t
}

// NormalizeEBSVolume converts an EBS volume to VAINO format
func (n *Normalizer) NormalizeEBSVolume(volume ec2Types.Volume) types.Resource {
	return types.Resource{
		ID:       aws.ToString(volume.VolumeId),
		Type:     "aws_ebs_volume",
		Provider: "aws",
		Name:     getVolumeNameFromTags(volume.Tags),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"size":              aws.ToInt32(volume.Size),
			"volume_type":       string(volume.VolumeType),
			"state":             string(volume.State),
			"availability_zone": aws.ToString(volume.AvailabilityZone),
			"encrypted":         aws.ToBool(volume.Encrypted),
			"snapshot_id":       aws.ToString(volume.SnapshotId),
			"iops":              aws.ToInt32(volume.Iops),
			"throughput":        aws.ToInt32(volume.Throughput),
		},
		Tags: convertAWSTags(volume.Tags),
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(volume.CreateTime),
		},
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
			"volume_id":   aws.ToString(snapshot.VolumeId),
			"description": aws.ToString(snapshot.Description),
			"state":       string(snapshot.State),
			"progress":    aws.ToString(snapshot.Progress),
			"volume_size": aws.ToInt32(snapshot.VolumeSize),
			"encrypted":   aws.ToBool(snapshot.Encrypted),
			"owner_id":    aws.ToString(snapshot.OwnerId),
		},
		Tags: convertAWSTags(snapshot.Tags),
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(snapshot.StartTime),
		},
	}
}

// NormalizeKeyPair converts an EC2 key pair to VAINO format
func (n *Normalizer) NormalizeKeyPair(keyPair ec2Types.KeyPairInfo) types.Resource {
	return types.Resource{
		ID:       aws.ToString(keyPair.KeyPairId),
		Type:     "aws_key_pair",
		Provider: "aws",
		Name:     aws.ToString(keyPair.KeyName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"key_name":        aws.ToString(keyPair.KeyName),
			"key_fingerprint": aws.ToString(keyPair.KeyFingerprint),
			"key_type":        string(keyPair.KeyType),
		},
		Tags: convertAWSTags(keyPair.Tags),
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(keyPair.CreateTime),
		},
	}
}

// getVolumeNameFromTags extracts the Name tag from volume tags
func getVolumeNameFromTags(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

// getSnapshotNameFromTags extracts the Name tag from snapshot tags
func getSnapshotNameFromTags(tags []ec2Types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

// CloudWatch Normalizers

// NormalizeCloudWatchAlarm converts a CloudWatch alarm to VAINO format
func (n *Normalizer) NormalizeCloudWatchAlarm(alarm cloudwatchTypes.MetricAlarm) types.Resource {
	var createdAt time.Time
	if alarm.AlarmConfigurationUpdatedTimestamp != nil {
		createdAt = *alarm.AlarmConfigurationUpdatedTimestamp
	}

	return types.Resource{
		ID:       aws.ToString(alarm.AlarmArn),
		Type:     "aws_cloudwatch_alarm",
		Provider: "aws",
		Name:     aws.ToString(alarm.AlarmName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"alarm_name":                aws.ToString(alarm.AlarmName),
			"alarm_description":         aws.ToString(alarm.AlarmDescription),
			"metric_name":               aws.ToString(alarm.MetricName),
			"namespace":                 aws.ToString(alarm.Namespace),
			"statistic":                 string(alarm.Statistic),
			"comparison_operator":       string(alarm.ComparisonOperator),
			"threshold":                 aws.ToFloat64(alarm.Threshold),
			"evaluation_periods":        aws.ToInt32(alarm.EvaluationPeriods),
			"period":                    aws.ToInt32(alarm.Period),
			"state_value":               string(alarm.StateValue),
			"state_reason":              aws.ToString(alarm.StateReason),
			"actions_enabled":           aws.ToBool(alarm.ActionsEnabled),
			"alarm_actions":             alarm.AlarmActions,
			"ok_actions":                alarm.OKActions,
			"insufficient_data_actions": alarm.InsufficientDataActions,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
		},
	}
}

// NormalizeCloudWatchCompositeAlarm converts a CloudWatch composite alarm to VAINO format
func (n *Normalizer) NormalizeCloudWatchCompositeAlarm(alarm cloudwatchTypes.CompositeAlarm) types.Resource {
	var createdAt time.Time
	if alarm.AlarmConfigurationUpdatedTimestamp != nil {
		createdAt = *alarm.AlarmConfigurationUpdatedTimestamp
	}

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
			"state_reason":      aws.ToString(alarm.StateReason),
			"actions_enabled":   aws.ToBool(alarm.ActionsEnabled),
			"alarm_actions":     alarm.AlarmActions,
			"ok_actions":        alarm.OKActions,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
		},
	}
}

// NormalizeCloudWatchLogGroup converts a CloudWatch log group to VAINO format
func (n *Normalizer) NormalizeCloudWatchLogGroup(logGroup logsTypes.LogGroup) types.Resource {
	var createdAt time.Time
	if logGroup.CreationTime != nil {
		createdAt = time.Unix(*logGroup.CreationTime/1000, 0)
	}

	return types.Resource{
		ID:       aws.ToString(logGroup.LogGroupName),
		Type:     "aws_cloudwatch_log_group",
		Provider: "aws",
		Name:     aws.ToString(logGroup.LogGroupName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"log_group_name":      aws.ToString(logGroup.LogGroupName),
			"retention_in_days":   aws.ToInt32(logGroup.RetentionInDays),
			"stored_bytes":        aws.ToInt64(logGroup.StoredBytes),
			"metric_filter_count": aws.ToInt32(logGroup.MetricFilterCount),
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: createdAt,
		},
	}
}

// NormalizeCloudWatchDashboard converts a CloudWatch dashboard to VAINO format
func (n *Normalizer) NormalizeCloudWatchDashboard(dashboardName string, dashboard *cloudwatch.GetDashboardOutput) types.Resource {
	var lastModified time.Time
	if dashboard.DashboardArn != nil {
		// Extract last modified time from ARN or use current time as fallback
		lastModified = time.Now()
	}

	return types.Resource{
		ID:       dashboardName,
		Type:     "aws_cloudwatch_dashboard",
		Provider: "aws",
		Name:     dashboardName,
		Region:   n.region,
		Configuration: map[string]interface{}{
			"dashboard_name": dashboardName,
			"dashboard_arn":  aws.ToString(dashboard.DashboardArn),
			"dashboard_body": aws.ToString(dashboard.DashboardBody),
		},
		Metadata: types.ResourceMetadata{
			UpdatedAt: lastModified,
		},
	}
}

// CloudFormation Normalizers

// NormalizeCloudFormationStack converts a CloudFormation stack to VAINO format
func (n *Normalizer) NormalizeCloudFormationStack(stack cloudformation.Stack) types.Resource {
	return types.Resource{
		ID:       aws.ToString(stack.StackId),
		Type:     "aws_cloudformation_stack",
		Provider: "aws",
		Name:     aws.ToString(stack.StackName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"stack_name":                    aws.ToString(stack.StackName),
			"stack_status":                  string(stack.StackStatus),
			"stack_status_reason":           aws.ToString(stack.StackStatusReason),
			"description":                   aws.ToString(stack.Description),
			"disable_rollback":              aws.ToBool(stack.DisableRollback),
			"enable_termination_protection": aws.ToBool(stack.EnableTerminationProtection),
			"drift_information":             stack.DriftInformation,
			"capabilities":                  stack.Capabilities,
			"notification_arns":             stack.NotificationARNs,
			"role_arn":                      aws.ToString(stack.RoleARN),
			"timeout_in_minutes":            aws.ToInt32(stack.TimeoutInMinutes),
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(stack.CreationTime),
			UpdatedAt: aws.ToTime(stack.LastUpdatedTime),
		},
	}
}

// NormalizeCloudFormationStackSet converts a CloudFormation stack set to VAINO format
func (n *Normalizer) NormalizeCloudFormationStackSet(stackSet cloudformation.StackSet) types.Resource {
	return types.Resource{
		ID:       aws.ToString(stackSet.StackSetId),
		Type:     "aws_cloudformation_stack_set",
		Provider: "aws",
		Name:     aws.ToString(stackSet.StackSetName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"stack_set_name":          aws.ToString(stackSet.StackSetName),
			"description":             aws.ToString(stackSet.Description),
			"status":                  string(stackSet.Status),
			"auto_deployment":         stackSet.AutoDeployment,
			"capabilities":            stackSet.Capabilities,
			"permission_model":        string(stackSet.PermissionModel),
			"organizational_unit_ids": stackSet.OrganizationalUnitIds,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(stackSet.CreationTimestamp),
		},
	}
}

// ELB Normalizers

// NormalizeClassicLoadBalancer converts a Classic Load Balancer to VAINO format
func (n *Normalizer) NormalizeClassicLoadBalancer(lb elbTypes.LoadBalancerDescription) types.Resource {
	return types.Resource{
		ID:       aws.ToString(lb.LoadBalancerName),
		Type:     "aws_classic_load_balancer",
		Provider: "aws",
		Name:     aws.ToString(lb.LoadBalancerName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"load_balancer_name": aws.ToString(lb.LoadBalancerName),
			"dns_name":           aws.ToString(lb.DNSName),
			"scheme":             aws.ToString(lb.Scheme),
			"vpc_id":             aws.ToString(lb.VPCId),
			"subnets":            lb.Subnets,
			"security_groups":    lb.SecurityGroups,
			"instances":          normalizeELBInstances(lb.Instances),
			"listeners":          normalizeELBListeners(lb.ListenerDescriptions),
			"availability_zones": lb.AvailabilityZones,
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(lb.CreatedTime),
		},
	}
}

// NormalizeModernLoadBalancer converts an ALB/NLB Load Balancer to VAINO format
func (n *Normalizer) NormalizeModernLoadBalancer(lb elbv2Types.LoadBalancer) types.Resource {
	return types.Resource{
		ID:       aws.ToString(lb.LoadBalancerArn),
		Type:     "aws_load_balancer_v2",
		Provider: "aws",
		Name:     aws.ToString(lb.LoadBalancerName),
		Region:   n.region,
		Configuration: map[string]interface{}{
			"load_balancer_name": aws.ToString(lb.LoadBalancerName),
			"dns_name":           aws.ToString(lb.DNSName),
			"scheme":             string(lb.Scheme),
			"type":               string(lb.Type),
			"state":              lb.State,
			"vpc_id":             aws.ToString(lb.VpcId),
			"availability_zones": normalizeELBv2AvailabilityZones(lb.AvailabilityZones),
			"security_groups":    lb.SecurityGroups,
			"ip_address_type":    string(lb.IpAddressType),
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: aws.ToTime(lb.CreatedTime),
		},
	}
}

// Helper functions for ELB normalization

func normalizeELBInstances(instances []elbTypes.Instance) []map[string]interface{} {
	var result []map[string]interface{}
	for _, instance := range instances {
		result = append(result, map[string]interface{}{
			"instance_id": aws.ToString(instance.InstanceId),
		})
	}
	return result
}

func normalizeELBListeners(listeners []elbTypes.ListenerDescription) []map[string]interface{} {
	var result []map[string]interface{}
	for _, listener := range listeners {
		if listener.Listener != nil {
			result = append(result, map[string]interface{}{
				"protocol":           aws.ToString(listener.Listener.Protocol),
				"load_balancer_port": aws.ToInt32(listener.Listener.LoadBalancerPort),
				"instance_port":      aws.ToInt32(listener.Listener.InstancePort),
				"instance_protocol":  aws.ToString(listener.Listener.InstanceProtocol),
				"ssl_certificate_id": aws.ToString(listener.Listener.SSLCertificateId),
			})
		}
	}
	return result
}

func normalizeELBv2AvailabilityZones(zones []elbv2Types.AvailabilityZone) []map[string]interface{} {
	var result []map[string]interface{}
	for _, zone := range zones {
		result = append(result, map[string]interface{}{
			"zone_name":               aws.ToString(zone.ZoneName),
			"subnet_id":               aws.ToString(zone.SubnetId),
			"load_balancer_addresses": zone.LoadBalancerAddresses,
		})
	}
	return result
}
