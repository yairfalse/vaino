package aws

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/yairfalse/vaino/internal/collectors"
	vainoerrors "github.com/yairfalse/vaino/internal/errors"
	"github.com/yairfalse/vaino/pkg/types"
)

// AWSCollector implements the enhanced collector interface for AWS
type AWSCollector struct {
	clients    *AWSClients
	normalizer *Normalizer
	region     string
	profile    string
}

// NewAWSCollector creates a new AWS collector
func NewAWSCollector() *AWSCollector {
	return &AWSCollector{}
}

// Status returns the current status of the AWS collector
func (c *AWSCollector) Status() string {
	// Check for AWS credentials in environment
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return "ready (environment credentials)"
	}

	// Check for AWS profile
	if os.Getenv("AWS_PROFILE") != "" {
		return "ready (profile: " + os.Getenv("AWS_PROFILE") + ")"
	}

	// Check for default profile
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		credentialsFile := homeDir + "/.aws/credentials"
		if _, err := os.Stat(credentialsFile); err == nil {
			return "ready (default profile)"
		}
	}

	// Check for IAM instance profile (when running on EC2)
	if os.Getenv("AWS_REGION") != "" || os.Getenv("AWS_DEFAULT_REGION") != "" {
		return "ready (instance profile)"
	}

	return "warning: no AWS credentials configured"
}

// AutoDiscover automatically discovers AWS configuration
func (c *AWSCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	// For AWS, auto-discovery means using default credentials and region
	return collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region":  "", // Will use default region from AWS config
			"profile": "", // Will use default profile
		},
	}, nil
}

// Validate validates the collector configuration
func (c *AWSCollector) Validate(config collectors.CollectorConfig) error {
	// Extract configuration
	region := ""
	profile := ""

	if config.Config != nil {
		if r, ok := config.Config["region"].(string); ok {
			region = r
		}
		if p, ok := config.Config["profile"].(string); ok {
			profile = p
		}
	}

	// Test AWS client creation
	ctx := context.Background()
	clientConfig := ClientConfig{
		Region:  region,
		Profile: profile,
	}

	clients, err := NewAWSClients(ctx, clientConfig)
	if err != nil {
		return c.wrapAWSError(err, "Failed to create AWS clients")
	}

	// Test credentials
	if err := clients.ValidateCredentials(ctx); err != nil {
		return c.wrapAWSError(err, "AWS credentials validation failed")
	}

	return nil
}

// Collect collects AWS resources
func (c *AWSCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	// Extract configuration
	region := ""
	profile := ""

	if config.Config != nil {
		if r, ok := config.Config["region"].(string); ok {
			region = r
		}
		if p, ok := config.Config["profile"].(string); ok {
			profile = p
		}
	}

	// Create AWS clients
	clientConfig := ClientConfig{
		Region:  region,
		Profile: profile,
	}

	clients, err := NewAWSClients(ctx, clientConfig)
	if err != nil {
		return nil, c.wrapAWSError(err, "Failed to create AWS clients")
	}

	c.clients = clients
	c.region = clients.GetRegion()
	c.profile = profile
	c.normalizer = NewNormalizer(c.region)

	// Generate snapshot ID
	snapshotID := fmt.Sprintf("aws-%d", time.Now().Unix())

	var allResources []types.Resource

	// Collect resources from different services
	services := []struct {
		name      string
		collector func(ctx context.Context) ([]types.Resource, error)
	}{
		{"EC2", c.CollectEC2Resources},
		{"S3", c.CollectS3Resources},
		{"VPC", c.CollectVPCResources},
		{"RDS", c.CollectRDSResources},
		{"Lambda", c.CollectLambdaResources},
		{"IAM", c.CollectIAMResources},
		{"DynamoDB", c.CollectDynamoDBResources},
		{"ECS", c.CollectECSResources},
		{"EKS", c.CollectEKSResources},
		{"CloudWatch", c.CollectCloudWatchResources},
		{"CloudFormation", c.CollectCloudFormationResources},
		{"ELB", c.CollectELBResources},
	}

	for _, service := range services {
		resources, err := service.collector(ctx)
		if err != nil {
			// Return authentication errors immediately, don't continue
			if isAuthenticationError(err) {
				return nil, c.wrapAWSError(err, fmt.Sprintf("Authentication failed for %s service", service.name))
			}

			// Check for permission errors - continue with warning
			if isPermissionError(err) {
				fmt.Printf("Warning: Insufficient permissions for %s service: %v\n", service.name, err)
				continue
			}

			// Check for rate limiting - continue with warning
			if isRateLimitError(err) {
				fmt.Printf("Warning: Rate limit exceeded for %s service: %v\n", service.name, err)
				continue
			}

			// For other errors, log and continue
			fmt.Printf("Warning: Failed to collect %s resources: %v\n", service.name, err)
			continue
		}
		allResources = append(allResources, resources...)
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        snapshotID,
		Timestamp: time.Now(),
		Provider:  "aws",
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    len(allResources),
			AdditionalData: map[string]interface{}{
				"profile": c.profile,
				"region":  c.region,
			},
		},
	}

	return snapshot, nil
}

// Name returns the collector name
func (c *AWSCollector) Name() string {
	return "aws"
}

// GetDescription returns the collector description
func (c *AWSCollector) GetDescription() string {
	return "AWS resource collector for EC2, S3, VPC, RDS, Lambda, IAM, DynamoDB, ECS, EKS, CloudWatch, CloudFormation, and ELB"
}

// GetVersion returns the collector version
func (c *AWSCollector) GetVersion() string {
	return "1.0.0"
}

// SupportsRegion returns whether the collector supports multiple regions
func (c *AWSCollector) SupportsRegion() bool {
	return true
}

// SupportedRegions returns the list of supported AWS regions
func (c *AWSCollector) SupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"ca-central-1", "sa-east-1",
	}
}

// GetDefaultConfig returns the default configuration for the collector
func (c *AWSCollector) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"region":  "us-east-1",
		"profile": "",
	}
}

// wrapAWSError wraps AWS errors with detailed context and solutions
func (c *AWSCollector) wrapAWSError(err error, message string) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for authentication errors
	if isAuthenticationError(err) {
		return vainoerrors.New(vainoerrors.ErrorTypeAuthentication, vainoerrors.ProviderAWS, message).
			WithCause(errStr).
			WithSolutions(c.getAuthenticationSolutions(errStr)...).
			WithVerify("aws sts get-caller-identity").
			WithHelp("vaino validate aws")
	}

	// Check for permission errors
	if isPermissionError(err) {
		return vainoerrors.New(vainoerrors.ErrorTypePermission, vainoerrors.ProviderAWS, message).
			WithCause(errStr).
			WithSolutions(c.getPermissionSolutions(errStr)...).
			WithVerify("aws iam get-user").
			WithHelp("vaino validate aws")
	}

	// Check for configuration errors
	if isConfigurationError(err) {
		return vainoerrors.New(vainoerrors.ErrorTypeConfiguration, vainoerrors.ProviderAWS, message).
			WithCause(errStr).
			WithSolutions(c.getConfigurationSolutions(errStr)...).
			WithVerify("aws configure list").
			WithHelp("vaino validate aws")
	}

	// Check for rate limiting
	if isRateLimitError(err) {
		return vainoerrors.New(vainoerrors.ErrorTypeRateLimit, vainoerrors.ProviderAWS, message).
			WithCause(errStr).
			WithSolutions(
				"Wait and retry the operation",
				"Consider using exponential backoff",
				"Reduce the number of concurrent requests",
			)
	}

	// Check for network errors
	if isNetworkError(err) {
		return vainoerrors.New(vainoerrors.ErrorTypeNetwork, vainoerrors.ProviderAWS, message).
			WithCause(errStr).
			WithSolutions(
				"Check internet connectivity",
				"Verify AWS service endpoints are accessible",
				"Check VPC/security group settings if running on EC2",
				"Try a different AWS region",
			)
	}

	// Default to configuration error
	return vainoerrors.New(vainoerrors.ErrorTypeConfiguration, vainoerrors.ProviderAWS, message).
		WithCause(errStr).
		WithSolutions(
			"Check AWS configuration",
			"Verify all required parameters are provided",
			"Check AWS service availability",
		)
}

// getAuthenticationSolutions returns context-specific authentication solutions
func (c *AWSCollector) getAuthenticationSolutions(errStr string) []string {
	solutions := []string{
		"Verify AWS credentials are configured correctly",
		"Check AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables",
		"Ensure AWS profile is valid if using profiles",
	}

	if strings.Contains(errStr, "InvalidAccessKeyId") {
		solutions = append(solutions, "Verify the AWS Access Key ID is correct")
	}

	if strings.Contains(errStr, "SignatureDoesNotMatch") {
		solutions = append(solutions, "Verify the AWS Secret Access Key is correct")
	}

	if strings.Contains(errStr, "TokenRefreshRequired") {
		solutions = append(solutions, "Refresh your AWS session token")
	}

	if strings.Contains(errStr, "RequestExpired") {
		solutions = append(solutions, "Check system clock synchronization")
	}

	if strings.Contains(errStr, "no valid credentials") {
		solutions = append(solutions,
			"Run 'aws configure' to set up credentials",
			"Set AWS_PROFILE environment variable",
			"Use IAM instance profile if running on EC2")
	}

	return solutions
}

// getPermissionSolutions returns context-specific permission solutions
func (c *AWSCollector) getPermissionSolutions(errStr string) []string {
	solutions := []string{
		"Verify IAM user has necessary permissions",
		"Check IAM policies attached to user/role",
		"Ensure resource-level permissions are granted",
	}

	if strings.Contains(errStr, "Access Denied") || strings.Contains(errStr, "UnauthorizedOperation") {
		solutions = append(solutions,
			"Add required IAM permissions for the specific service",
			"Check if MFA is required for this operation",
			"Verify the resource exists and you have access")
	}

	return solutions
}

// getConfigurationSolutions returns context-specific configuration solutions
func (c *AWSCollector) getConfigurationSolutions(errStr string) []string {
	solutions := []string{
		"Check AWS configuration files (~/.aws/config, ~/.aws/credentials)",
		"Verify region is correctly specified",
		"Ensure profile configuration is valid",
	}

	if strings.Contains(errStr, "region") {
		solutions = append(solutions,
			"Set AWS_REGION or AWS_DEFAULT_REGION environment variable",
			"Specify region in AWS config file",
			"Use --region flag if available")
	}

	if strings.Contains(errStr, "profile") {
		solutions = append(solutions,
			"Check if the specified profile exists",
			"Verify profile configuration in ~/.aws/config",
			"Use 'aws configure list-profiles' to see available profiles")
	}

	return solutions
}

// isAuthenticationError checks if an error is related to authentication
func isAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Common AWS authentication error patterns
	authErrorPatterns := []string{
		"unauthorizedoperation",
		"invaliduserid.notfound",
		"authfailure",
		"signaturedoesnotmatch",
		"tokenrefreshrequired",
		"accessdenied",
		"invalidaccesskeyid",
		"requestexpired",
		"no valid credentials",
		"credential provider",
		"unable to load aws config",
		"nocredentialproviders",
		"credentials not found",
		"unable to retrieve credentials",
		"authentication",
		"unauthenticated",
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// isPermissionError checks if an error is related to permissions
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	permissionErrorPatterns := []string{
		"access denied",
		"unauthorized",
		"forbidden",
		"permission denied",
		"insufficient privileges",
		"not authorized",
		"operation denied",
	}

	for _, pattern := range permissionErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// isConfigurationError checks if an error is related to configuration
func isConfigurationError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	configErrorPatterns := []string{
		"invalidparameter",
		"validation failed",
		"invalid configuration",
		"configuration error",
		"invalid region",
		"unknown region",
		"profile not found",
		"invalid profile",
		"missing required parameter",
	}

	for _, pattern := range configErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// isRateLimitError checks if an error is related to rate limiting
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	rateLimitPatterns := []string{
		"throttling",
		"requestlimitexceeded",
		"too many requests",
		"rate exceeded",
		"throttled",
		"slowdown",
	}

	for _, pattern := range rateLimitPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for HTTP 429 status
	var re *awshttp.ResponseError
	if aws.ErrorAs(err, &re) {
		if re.ResponseError.HTTPStatusCode() == 429 {
			return true
		}
	}

	return false
}

// isNetworkError checks if an error is related to network connectivity
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	networkErrorPatterns := []string{
		"connection refused",
		"connection timeout",
		"network unreachable",
		"no such host",
		"connection reset",
		"timeout",
		"network error",
		"dns resolution",
		"connection failed",
	}

	for _, pattern := range networkErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for HTTP connection errors
	var re *awshttp.ResponseError
	if aws.ErrorAs(err, &re) {
		// Network-related HTTP status codes
		status := re.ResponseError.HTTPStatusCode()
		if status == 502 || status == 503 || status == 504 {
			return true
		}
	}

	return false
}
