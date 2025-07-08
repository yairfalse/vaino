package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeEC2Instance(t *testing.T) {
	tests := []struct {
		name     string
		instance ec2Types.Instance
		expected map[string]interface{}
	}{
		{
			name: "basic instance normalization",
			instance: ec2Types.Instance{
				InstanceId:       aws.String("i-1234567890abcdef0"),
				InstanceType:     ec2Types.InstanceTypeT2Micro,
				State:            &ec2Types.InstanceState{Name: ec2Types.InstanceStateNameRunning},
				VpcId:            aws.String("vpc-12345"),
				SubnetId:         aws.String("subnet-12345"),
				PrivateIpAddress: aws.String("10.0.1.10"),
				PublicIpAddress:  aws.String("54.123.45.67"),
				ImageId:          aws.String("ami-12345"),
				KeyName:          aws.String("my-key"),
				LaunchTime:       aws.Time(time.Now()),
				Placement: &ec2Types.Placement{
					AvailabilityZone: aws.String("us-east-1a"),
				},
				Monitoring: &ec2Types.Monitoring{
					State: ec2Types.MonitoringStateEnabled,
				},
				Tags: []ec2Types.Tag{
					{Key: aws.String("Name"), Value: aws.String("test-instance")},
					{Key: aws.String("Environment"), Value: aws.String("production")},
				},
			},
			expected: map[string]interface{}{
				"instance_type":       "t2.micro",
				"state":              "running",
				"vpc_id":             "vpc-12345",
				"subnet_id":          "subnet-12345",
				"availability_zone":  "us-east-1a",
				"private_ip_address": "10.0.1.10",
				"public_ip_address":  "54.123.45.67",
				"image_id":           "ami-12345",
				"key_name":           "my-key",
				"monitoring":         true,
			},
		},
		{
			name: "instance without public IP",
			instance: ec2Types.Instance{
				InstanceId:       aws.String("i-0987654321fedcba0"),
				InstanceType:     ec2Types.InstanceTypeT3Large,
				State:            &ec2Types.InstanceState{Name: ec2Types.InstanceStateNameRunning},
				VpcId:            aws.String("vpc-67890"),
				SubnetId:         aws.String("subnet-67890"),
				PrivateIpAddress: aws.String("10.0.2.20"),
				LaunchTime:       aws.Time(time.Now()),
				Placement: &ec2Types.Placement{
					AvailabilityZone: aws.String("us-west-2b"),
				},
				Monitoring: &ec2Types.Monitoring{
					State: ec2Types.MonitoringStateDisabled,
				},
			},
			expected: map[string]interface{}{
				"instance_type":       "t3.large",
				"state":              "running",
				"vpc_id":             "vpc-67890",
				"subnet_id":          "subnet-67890",
				"availability_zone":  "us-west-2b",
				"private_ip_address": "10.0.2.20",
				"public_ip_address":  "",
				"monitoring":         false,
			},
		},
	}

	normalizer := NewNormalizer("us-east-1")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizeEC2Instance(tt.instance)

			// Check basic fields
			assert.Equal(t, aws.ToString(tt.instance.InstanceId), result.ID)
			assert.Equal(t, "aws_instance", result.Type)
			assert.Equal(t, "aws", result.Provider)
			assert.Equal(t, "us-east-1", result.Region)

			// Check configuration
			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, result.Configuration[key], "Configuration field %s mismatch", key)
			}

			// Check tags
			if len(tt.instance.Tags) > 0 {
				assert.NotEmpty(t, result.Tags)
				for _, tag := range tt.instance.Tags {
					assert.Equal(t, aws.ToString(tag.Value), result.Tags[aws.ToString(tag.Key)])
				}
			}
		})
	}
}

func TestNormalizeSecurityGroup(t *testing.T) {
	sg := ec2Types.SecurityGroup{
		GroupId:     aws.String("sg-0123456789abcdef0"),
		GroupName:   aws.String("web-server-sg"),
		Description: aws.String("Security group for web servers"),
		VpcId:       aws.String("vpc-12345"),
		IpPermissions: []ec2Types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges: []ec2Types.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				},
			},
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(443),
				ToPort:     aws.Int32(443),
				IpRanges: []ec2Types.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				},
			},
		},
		IpPermissionsEgress: []ec2Types.IpPermission{
			{
				IpProtocol: aws.String("-1"),
				IpRanges: []ec2Types.IpRange{
					{CidrIp: aws.String("0.0.0.0/0")},
				},
			},
		},
		Tags: []ec2Types.Tag{
			{Key: aws.String("Name"), Value: aws.String("web-sg")},
		},
	}

	normalizer := NewNormalizer("us-east-1")
	result := normalizer.NormalizeSecurityGroup(sg)

	assert.Equal(t, "sg-0123456789abcdef0", result.ID)
	assert.Equal(t, "aws_security_group", result.Type)
	assert.Equal(t, "web-server-sg", result.Name)
	assert.Equal(t, "web-server-sg", result.Configuration["name"])
	assert.Equal(t, "Security group for web servers", result.Configuration["description"])
	assert.Equal(t, "vpc-12345", result.Configuration["vpc_id"])

	// Check ingress rules
	ingress := result.Configuration["ingress"].([]map[string]interface{})
	assert.Len(t, ingress, 2)
	assert.Equal(t, "tcp", ingress[0]["ip_protocol"])
	assert.Equal(t, int32(80), ingress[0]["from_port"])
	assert.Equal(t, int32(80), ingress[0]["to_port"])
	assert.Contains(t, ingress[0]["cidr_blocks"], "0.0.0.0/0")

	// Check egress rules
	egress := result.Configuration["egress"].([]map[string]interface{})
	assert.Len(t, egress, 1)
	assert.Equal(t, "-1", egress[0]["ip_protocol"])
}

func TestNormalizeS3Bucket(t *testing.T) {
	creationDate := time.Now().Add(-24 * time.Hour)
	bucket := s3Types.Bucket{
		Name:         aws.String("my-test-bucket"),
		CreationDate: aws.Time(creationDate),
	}

	normalizer := NewNormalizer("us-east-1")
	result := normalizer.NormalizeS3Bucket(bucket)

	assert.Equal(t, "my-test-bucket", result.ID)
	assert.Equal(t, "aws_s3_bucket", result.Type)
	assert.Equal(t, "my-test-bucket", result.Name)
	assert.Equal(t, "my-test-bucket", result.Configuration["bucket"])
	assert.Equal(t, creationDate.Unix(), result.Metadata.CreatedAt.Unix())
}

func TestNormalizeRDSInstance(t *testing.T) {
	createTime := time.Now().Add(-48 * time.Hour)
	dbInstance := rdsTypes.DBInstance{
		DBInstanceIdentifier: aws.String("mydb-instance"),
		DBInstanceClass:     aws.String("db.t3.micro"),
		Engine:              aws.String("mysql"),
		EngineVersion:       aws.String("8.0.28"),
		DBName:              aws.String("mydatabase"),
		MasterUsername:      aws.String("admin"),
		AllocatedStorage:    aws.Int32(20),
		StorageType:         aws.String("gp2"),
		StorageEncrypted:    aws.Bool(true),
		MultiAZ:             aws.Bool(false),
		PubliclyAccessible:  aws.Bool(false),
		BackupRetentionPeriod: aws.Int32(7),
		InstanceCreateTime:  aws.Time(createTime),
		DBSubnetGroup: &rdsTypes.DBSubnetGroup{
			DBSubnetGroupName: aws.String("default-vpc-12345"),
		},
		VpcSecurityGroups: []rdsTypes.VpcSecurityGroupMembership{
			{
				VpcSecurityGroupId: aws.String("sg-12345"),
			},
		},
	}

	normalizer := NewNormalizer("us-east-1")
	result := normalizer.NormalizeRDSInstance(dbInstance)

	assert.Equal(t, "mydb-instance", result.ID)
	assert.Equal(t, "aws_db_instance", result.Type)
	assert.Equal(t, "mydb-instance", result.Name)
	
	// Check configuration
	config := result.Configuration
	assert.Equal(t, "mydb-instance", config["db_instance_identifier"])
	assert.Equal(t, "db.t3.micro", config["db_instance_class"])
	assert.Equal(t, "mysql", config["engine"])
	assert.Equal(t, "8.0.28", config["engine_version"])
	assert.Equal(t, "mydatabase", config["db_name"])
	assert.Equal(t, "admin", config["username"])
	assert.Equal(t, int32(20), config["allocated_storage"])
	assert.Equal(t, "gp2", config["storage_type"])
	assert.Equal(t, true, config["storage_encrypted"])
	assert.Equal(t, false, config["multi_az"])
	assert.Equal(t, false, config["publicly_accessible"])
	assert.Equal(t, int32(7), config["backup_retention_period"])
	assert.Equal(t, "default-vpc-12345", config["db_subnet_group_name"])
	
	securityGroups := config["vpc_security_group_ids"].([]string)
	assert.Contains(t, securityGroups, "sg-12345")
}

func TestNormalizeLambdaFunction(t *testing.T) {
	lastModified := time.Now().Format(time.RFC3339)
	function := lambdaTypes.FunctionConfiguration{
		FunctionArn:  aws.String("arn:aws:lambda:us-east-1:123456789012:function:my-function"),
		FunctionName: aws.String("my-function"),
		Role:         aws.String("arn:aws:iam::123456789012:role/lambda-role"),
		Handler:      aws.String("index.handler"),
		Runtime:      lambdaTypes.RuntimeNodejs18x,
		Timeout:      aws.Int32(30),
		MemorySize:   aws.Int32(256),
		Description:  aws.String("My Lambda function"),
		LastModified: aws.String(lastModified),
		KMSKeyArn:    aws.String("arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"),
	}

	normalizer := NewNormalizer("us-east-1")
	result := normalizer.NormalizeLambdaFunction(function)

	assert.Equal(t, "arn:aws:lambda:us-east-1:123456789012:function:my-function", result.ID)
	assert.Equal(t, "aws_lambda_function", result.Type)
	assert.Equal(t, "my-function", result.Name)
	
	// Check configuration
	config := result.Configuration
	assert.Equal(t, "my-function", config["function_name"])
	assert.Equal(t, "arn:aws:iam::123456789012:role/lambda-role", config["role"])
	assert.Equal(t, "index.handler", config["handler"])
	assert.Equal(t, "nodejs18.x", config["runtime"])
	assert.Equal(t, int32(30), config["timeout"])
	assert.Equal(t, int32(256), config["memory_size"])
	assert.Equal(t, "My Lambda function", config["description"])
	assert.Equal(t, "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012", config["kms_key_arn"])
}

func TestConvertAWSTags(t *testing.T) {
	tags := []ec2Types.Tag{
		{Key: aws.String("Name"), Value: aws.String("test-resource")},
		{Key: aws.String("Environment"), Value: aws.String("production")},
		{Key: aws.String("Team"), Value: aws.String("platform")},
	}

	result := convertAWSTags(tags)

	assert.Len(t, result, 3)
	assert.Equal(t, "test-resource", result["Name"])
	assert.Equal(t, "production", result["Environment"])
	assert.Equal(t, "platform", result["Team"])
}

func TestNormalizeSecurityGroupRules(t *testing.T) {
	rules := []ec2Types.IpPermission{
		{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int32(22),
			ToPort:     aws.Int32(22),
			IpRanges: []ec2Types.IpRange{
				{CidrIp: aws.String("10.0.0.0/8")},
				{CidrIp: aws.String("172.16.0.0/12")},
			},
		},
		{
			IpProtocol: aws.String("tcp"),
			FromPort:   aws.Int32(443),
			ToPort:     aws.Int32(443),
			UserIdGroupPairs: []ec2Types.UserIdGroupPair{
				{GroupId: aws.String("sg-12345")},
			},
		},
	}

	result := normalizeSecurityGroupRules(rules)

	require.Len(t, result, 2)
	
	// Check first rule
	assert.Equal(t, "tcp", result[0]["ip_protocol"])
	assert.Equal(t, int32(22), result[0]["from_port"])
	assert.Equal(t, int32(22), result[0]["to_port"])
	cidrBlocks := result[0]["cidr_blocks"].([]string)
	assert.Contains(t, cidrBlocks, "10.0.0.0/8")
	assert.Contains(t, cidrBlocks, "172.16.0.0/12")
	
	// Check second rule
	assert.Equal(t, "tcp", result[1]["ip_protocol"])
	assert.Equal(t, int32(443), result[1]["from_port"])
	assert.Equal(t, int32(443), result[1]["to_port"])
	securityGroups := result[1]["security_groups"].([]string)
	assert.Contains(t, securityGroups, "sg-12345")
}