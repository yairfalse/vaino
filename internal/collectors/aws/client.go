package aws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
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
	CloudWatch      *cloudwatch.Client
	CloudWatchLogs  *cloudwatchlogs.Client
	CloudFormation  *cloudformation.Client
	ELB             *elasticloadbalancing.Client
	ELBv2           *elasticloadbalancingv2.Client
	STS             *sts.Client
	Config          aws.Config
}

// ClientConfig holds configuration for AWS client creation
type ClientConfig struct {
	Region     string
	Profile    string
	MaxRetries int
	Timeout    time.Duration
}

// NewAWSClients creates and configures AWS service clients
func NewAWSClients(ctx context.Context, clientConfig ClientConfig) (*AWSClients, error) {
	// Set default values
	if clientConfig.MaxRetries == 0 {
		clientConfig.MaxRetries = 3
	}
	if clientConfig.Timeout == 0 {
		clientConfig.Timeout = 30 * time.Second
	}

	// Build AWS config with optional profile and region
	var opts []func(*config.LoadOptions) error

	if clientConfig.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(clientConfig.Profile))
	}

	if clientConfig.Region != "" {
		opts = append(opts, config.WithRegion(clientConfig.Region))
	}

	// Add retry configuration
	opts = append(opts, config.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), clientConfig.MaxRetries)
	}))

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Enhance error handling for credential retrieval
	if err := validateAWSCredentials(ctx, cfg); err != nil {
		return nil, err
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
		CloudWatch:      cloudwatch.NewFromConfig(cfg),
		CloudWatchLogs:  cloudwatchlogs.NewFromConfig(cfg),
		CloudFormation:  cloudformation.NewFromConfig(cfg),
		ELB:             elasticloadbalancing.NewFromConfig(cfg),
		ELBv2:           elasticloadbalancingv2.NewFromConfig(cfg),
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
	result, err := c.STS.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}

	// Verify we got valid identity information
	if result.Account == nil || result.Arn == nil {
		return fmt.Errorf("received invalid identity information from AWS")
	}

	return nil
}

// validateAWSCredentials provides enhanced credential validation with detailed error reporting
func validateAWSCredentials(ctx context.Context, cfg aws.Config) error {
	// First check if credentials are available
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return enhanceCredentialError(err)
	}

	// Validate credential fields
	if creds.AccessKeyID == "" {
		return fmt.Errorf("AWS Access Key ID is empty")
	}

	if creds.SecretAccessKey == "" {
		return fmt.Errorf("AWS Secret Access Key is empty")
	}

	// Check for expired credentials
	if !creds.Expires.IsZero() && time.Now().After(creds.Expires) {
		return fmt.Errorf("AWS credentials have expired (expired at: %v)", creds.Expires)
	}

	return nil
}

// enhanceCredentialError provides more helpful error messages for credential issues
func enhanceCredentialError(err error) error {
	_ = err.Error()

	// Check for common credential issues
	if fmt.Sprintf("%v", err) == "no EC2 IMDS role found" {
		return fmt.Errorf("no AWS credentials found: %w\n\nSuggestions:\n"+
			"1. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables\n"+
			"2. Run 'aws configure' to set up credentials\n"+
			"3. Use AWS_PROFILE environment variable\n"+
			"4. If running on EC2, ensure instance has IAM role attached", err)
	}

	if fmt.Sprintf("%v", err) == "failed to refresh cached credentials" {
		return fmt.Errorf("failed to refresh AWS credentials: %w\n\nSuggestions:\n"+
			"1. Check if credentials have expired\n"+
			"2. Verify network connectivity\n"+
			"3. Re-run 'aws configure' or refresh tokens", err)
	}

	// Check environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" && os.Getenv("AWS_PROFILE") == "" {
		return fmt.Errorf("no AWS credentials configured: %w\n\nSuggestions:\n"+
			"1. Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables\n"+
			"2. Set AWS_PROFILE environment variable\n"+
			"3. Run 'aws configure' to set up default profile\n"+
			"4. If running on EC2, attach IAM role to instance", err)
	}

	return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
}
