package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// EC2ClientInterface defines the EC2 client methods we use
type EC2ClientInterface interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeVpcAttribute(ctx context.Context, params *ec2.DescribeVpcAttributeInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcAttributeOutput, error)
}

// AWSClients holds all AWS service clients
type AWSClients struct {
	EC2             EC2ClientInterface
	S3              *s3.Client
	RDS             *rds.Client
	Lambda          *lambda.Client
	IAM             *iam.Client
	DynamoDB        *dynamodb.Client
	DynamoDBStreams *dynamodbstreams.Client
	ECS             *ecs.Client
	EKS             *eks.Client
	STS             *sts.Client
	Config          aws.Config
}

// ClientConfig holds configuration for AWS client creation
type ClientConfig struct {
	Region  string
	Profile string
}

// NewAWSClients creates and configures AWS service clients
func NewAWSClients(ctx context.Context, clientConfig ClientConfig) (*AWSClients, error) {
	// Build AWS config with optional profile and region
	var opts []func(*config.LoadOptions) error

	if clientConfig.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(clientConfig.Profile))
	}

	if clientConfig.Region != "" {
		opts = append(opts, config.WithRegion(clientConfig.Region))
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Verify credentials are available
	if _, err := cfg.Credentials.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Create service clients
	clients := &AWSClients{
		EC2:             ec2.NewFromConfig(cfg),
		S3:              s3.NewFromConfig(cfg),
		RDS:             rds.NewFromConfig(cfg),
		Lambda:          lambda.NewFromConfig(cfg),
		IAM:             iam.NewFromConfig(cfg),
		DynamoDB:        dynamodb.NewFromConfig(cfg),
		DynamoDBStreams: dynamodbstreams.NewFromConfig(cfg),
		ECS:             ecs.NewFromConfig(cfg),
		EKS:             eks.NewFromConfig(cfg),
		STS:             sts.NewFromConfig(cfg),
		Config:          cfg,
	}

	return clients, nil
}

// GetRegion returns the configured region
func (c *AWSClients) GetRegion() string {
	return c.Config.Region
}

// ValidateCredentials tests AWS credentials by making a simple API call
func (c *AWSClients) ValidateCredentials(ctx context.Context) error {
	// Use STS GetCallerIdentity to validate credentials - works with any valid AWS credentials
	_, err := c.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}
	return nil
}
