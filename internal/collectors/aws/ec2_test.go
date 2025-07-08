package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestCollectEC2Instances(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockEC2Client)
	
	// Setup mock response
	expectedOutput := &ec2.DescribeInstancesOutput{
		Reservations: []ec2Types.Reservation{
			{
				Instances: []ec2Types.Instance{
					{
						InstanceId:       aws.String("i-1234567890abcdef0"),
						InstanceType:     ec2Types.InstanceTypeT2Micro,
						State:            &ec2Types.InstanceState{Name: ec2Types.InstanceStateNameRunning},
						VpcId:            aws.String("vpc-12345"),
						SubnetId:         aws.String("subnet-12345"),
						PrivateIpAddress: aws.String("10.0.1.10"),
						PublicIpAddress:  aws.String("54.123.45.67"),
						ImageId:          aws.String("ami-12345"),
						LaunchTime:       aws.Time(time.Now()),
						Placement: &ec2Types.Placement{
							AvailabilityZone: aws.String("us-east-1a"),
						},
						Monitoring: &ec2Types.Monitoring{
							State: ec2Types.MonitoringStateEnabled,
						},
						Tags: []ec2Types.Tag{
							{Key: aws.String("Name"), Value: aws.String("test-instance")},
						},
					},
					{
						InstanceId:   aws.String("i-terminated"),
						InstanceType: ec2Types.InstanceTypeT2Micro,
						State:        &ec2Types.InstanceState{Name: ec2Types.InstanceStateNameTerminated},
						LaunchTime:   aws.Time(time.Now()),
						Placement: &ec2Types.Placement{
							AvailabilityZone: aws.String("us-east-1a"),
						},
						Monitoring: &ec2Types.Monitoring{
							State: ec2Types.MonitoringStateDisabled,
						},
					},
				},
			},
		},
		NextToken: nil,
	}
	
	mockClient.On("DescribeInstances", ctx, mock.AnythingOfType("*ec2.DescribeInstancesInput")).Return(expectedOutput, nil)
	
	// Create collector with mocked client
	collector := &AWSCollector{
		clients: &AWSClients{
			EC2: mockClient,
		},
		normalizer: NewNormalizer("us-east-1"),
	}
	
	// Test collection
	resources, err := collector.collectEC2Instances(ctx)
	
	assert.NoError(t, err)
	assert.Len(t, resources, 1) // Only running instance, terminated should be filtered
	assert.Equal(t, "i-1234567890abcdef0", resources[0].ID)
	assert.Equal(t, "aws_instance", resources[0].Type)
	assert.Equal(t, "test-instance", resources[0].Name)
	
	mockClient.AssertExpectations(t)
}

func TestCollectSecurityGroups(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockEC2Client)
	
	// Setup mock response
	expectedOutput := &ec2.DescribeSecurityGroupsOutput{
		SecurityGroups: []ec2Types.SecurityGroup{
			{
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
				},
				Tags: []ec2Types.Tag{
					{Key: aws.String("Name"), Value: aws.String("web-sg")},
				},
			},
			{
				GroupId:     aws.String("sg-default"),
				GroupName:   aws.String("default"),
				Description: aws.String("default VPC security group"),
				VpcId:       aws.String("vpc-12345"),
			},
		},
		NextToken: nil,
	}
	
	mockClient.On("DescribeSecurityGroups", ctx, mock.AnythingOfType("*ec2.DescribeSecurityGroupsInput")).Return(expectedOutput, nil)
	
	// Create collector with mocked client
	collector := &AWSCollector{
		clients: &AWSClients{
			EC2: mockClient,
		},
		normalizer: NewNormalizer("us-east-1"),
	}
	
	// Test collection
	resources, err := collector.collectSecurityGroups(ctx)
	
	assert.NoError(t, err)
	assert.Len(t, resources, 2)
	
	// Check first security group
	assert.Equal(t, "sg-0123456789abcdef0", resources[0].ID)
	assert.Equal(t, "aws_security_group", resources[0].Type)
	assert.Equal(t, "web-server-sg", resources[0].Name)
	
	// Check second security group
	assert.Equal(t, "sg-default", resources[1].ID)
	assert.Equal(t, "aws_security_group", resources[1].Type)
	assert.Equal(t, "default", resources[1].Name)
	
	mockClient.AssertExpectations(t)
}

func TestCollectEC2ResourcesWithPagination(t *testing.T) {
	ctx := context.Background()
	mockClient := new(MockEC2Client)
	
	// First page of instances
	firstPage := &ec2.DescribeInstancesOutput{
		Reservations: []ec2Types.Reservation{
			{
				Instances: []ec2Types.Instance{
					{
						InstanceId:   aws.String("i-page1"),
						InstanceType: ec2Types.InstanceTypeT2Micro,
						State:        &ec2Types.InstanceState{Name: ec2Types.InstanceStateNameRunning},
						LaunchTime:   aws.Time(time.Now()),
						Placement: &ec2Types.Placement{
							AvailabilityZone: aws.String("us-east-1a"),
						},
						Monitoring: &ec2Types.Monitoring{
							State: ec2Types.MonitoringStateDisabled,
						},
					},
				},
			},
		},
		NextToken: aws.String("token1"),
	}
	
	// Second page of instances
	secondPage := &ec2.DescribeInstancesOutput{
		Reservations: []ec2Types.Reservation{
			{
				Instances: []ec2Types.Instance{
					{
						InstanceId:   aws.String("i-page2"),
						InstanceType: ec2Types.InstanceTypeT2Micro,
						State:        &ec2Types.InstanceState{Name: ec2Types.InstanceStateNameRunning},
						LaunchTime:   aws.Time(time.Now()),
						Placement: &ec2Types.Placement{
							AvailabilityZone: aws.String("us-east-1b"),
						},
						Monitoring: &ec2Types.Monitoring{
							State: ec2Types.MonitoringStateDisabled,
						},
					},
				},
			},
		},
		NextToken: nil, // No more pages
	}
	
	// Setup mock calls
	mockClient.On("DescribeInstances", ctx, &ec2.DescribeInstancesInput{}).Return(firstPage, nil).Once()
	mockClient.On("DescribeInstances", ctx, &ec2.DescribeInstancesInput{NextToken: aws.String("token1")}).Return(secondPage, nil).Once()
	
	// Empty security groups for simplicity
	mockClient.On("DescribeSecurityGroups", ctx, mock.AnythingOfType("*ec2.DescribeSecurityGroupsInput")).Return(&ec2.DescribeSecurityGroupsOutput{}, nil)
	
	// Create collector with mocked client
	collector := &AWSCollector{
		clients: &AWSClients{
			EC2: mockClient,
		},
		normalizer: NewNormalizer("us-east-1"),
	}
	
	// Test collection
	resources, err := collector.CollectEC2Resources(ctx)
	
	assert.NoError(t, err)
	// Should have 2 instances from pagination
	instanceCount := 0
	for _, r := range resources {
		if r.Type == "aws_instance" {
			instanceCount++
		}
	}
	assert.Equal(t, 2, instanceCount)
	
	mockClient.AssertExpectations(t)
}