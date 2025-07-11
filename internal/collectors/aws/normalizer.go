package aws

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
