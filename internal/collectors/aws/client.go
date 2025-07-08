package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// AWSClients holds all AWS service clients
type AWSClients struct {
	EC2    *ec2.Client
	S3     *s3.Client
	RDS    *rds.Client
	Lambda *lambda.Client
	IAM    *iam.Client
	Config aws.Config
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
		EC2:    ec2.NewFromConfig(cfg),
		S3:     s3.NewFromConfig(cfg),
		RDS:    rds.NewFromConfig(cfg),
		Lambda: lambda.NewFromConfig(cfg),
		IAM:    iam.NewFromConfig(cfg),
		Config: cfg,
	}
	
	return clients, nil
}

// GetRegion returns the configured region
func (c *AWSClients) GetRegion() string {
	return c.Config.Region
}

// ValidateCredentials tests AWS credentials by making a simple API call
func (c *AWSClients) ValidateCredentials(ctx context.Context) error {
	// Use STS GetCallerIdentity to validate credentials
	_, err := c.IAM.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		// If GetUser fails, try listing IAM users (works with most policies)
		_, err = c.IAM.ListUsers(ctx, &iam.ListUsersInput{MaxItems: aws.Int32(1)})
		if err != nil {
			return fmt.Errorf("failed to validate AWS credentials: %w", err)
		}
	}
	return nil
}