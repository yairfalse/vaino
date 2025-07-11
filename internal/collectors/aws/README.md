# AWS Provider for VAINO

This package implements AWS resource collection for VAINO drift detection.

## Supported Services

- **EC2**: Instances, Security Groups, VPCs, Subnets
- **S3**: Buckets with versioning and encryption info
- **RDS**: Database instances
- **Lambda**: Functions with environment variables
- **IAM**: Roles and Users (basic)

## Authentication

The AWS provider supports multiple authentication methods:

1. **Environment Variables**:
   ```bash
   export AWS_ACCESS_KEY_ID=your-key
   export AWS_SECRET_ACCESS_KEY=your-secret
   export AWS_REGION=us-east-1
   ```

2. **AWS CLI Configuration**:
   ```bash
   aws configure
   ```

3. **AWS Profiles**:
   ```bash
   vaino scan --provider aws --profile production
   ```

4. **IAM Roles** (when running on EC2)

## Usage

```bash
# Scan AWS resources in default region
vaino scan --provider aws

# Scan specific region
vaino scan --provider aws --region us-west-2

# Scan with specific profile
vaino scan --provider aws --profile production

# Save output to file
vaino scan --provider aws --output-file aws-snapshot.json
```

## Testing

### Unit Tests

Run unit tests for the AWS provider:

```bash
# Run all AWS tests
go test -v ./internal/collectors/aws/...

# Run specific test files
go test -v ./internal/collectors/aws/normalizer_test.go ./internal/collectors/aws/normalizer.go

# Run with coverage
go test -v -cover ./internal/collectors/aws/...
```

### Integration Tests

Integration tests require AWS credentials:

```bash
# Run integration tests (requires AWS credentials)
go test -v -tags=integration ./internal/collectors/aws/...

# Run with specific AWS profile
AWS_PROFILE=test go test -v -tags=integration ./internal/collectors/aws/...
```

### System Tests

System tests verify the full workflow:

```bash
# Run system tests
go test -v -tags=integration ./internal/collectors/aws/aws_system_test.go
```

### E2E Tests

End-to-end tests verify the complete drift detection workflow:

```bash
# Build vaino first
go build -o vaino ./cmd/vaino

# Run E2E tests
go test -v -tags=e2e ./test/e2e/aws_drift_detection_test.go
```

## Test Structure

```
internal/collectors/aws/
├── normalizer_test.go      # Unit tests for resource normalization
├── client_test.go          # Unit tests for AWS client setup
├── collector_test.go       # Integration tests for collector
├── aws_test.go            # Package-level tests
└── aws_system_test.go     # System integration tests

test/e2e/
└── aws_drift_detection_test.go  # E2E workflow tests
```

## Mocking AWS Services

For unit tests without AWS credentials, use the mock data:

```go
// Load mock snapshot
mockData, _ := ioutil.ReadFile("testdata/aws_mock_response.json")
var snapshot types.Snapshot
json.Unmarshal(mockData, &snapshot)
```

## Common Issues

1. **Credentials Error**: Ensure AWS credentials are configured
2. **Region Error**: Some resources are region-specific
3. **Rate Limiting**: AWS API rate limits may affect large scans
4. **IAM Permissions**: Ensure IAM user/role has read permissions for all services

## Required IAM Permissions

Minimum IAM permissions needed:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:Describe*",
        "s3:ListBucket",
        "s3:GetBucket*",
        "rds:Describe*",
        "lambda:List*",
        "lambda:Get*",
        "iam:List*",
        "iam:Get*"
      ],
      "Resource": "*"
    }
  ]
}
```