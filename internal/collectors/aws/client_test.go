package aws

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientConfig(t *testing.T) {
	tests := []struct {
		name         string
		config       ClientConfig
		envVars      map[string]string
		expectError  bool
		expectRegion string
	}{
		{
			name: "default configuration",
			config: ClientConfig{
				Region:  "",
				Profile: "",
			},
			envVars:      map[string]string{},
			expectError:  true, // Will fail without credentials
			expectRegion: "",
		},
		{
			name: "with specific region",
			config: ClientConfig{
				Region:  "us-west-2",
				Profile: "",
			},
			envVars:      map[string]string{},
			expectError:  true, // Will fail without credentials
			expectRegion: "us-west-2",
		},
		{
			name: "with profile",
			config: ClientConfig{
				Region:  "eu-west-1",
				Profile: "production",
			},
			envVars:      map[string]string{},
			expectError:  true, // Will fail without actual profile
			expectRegion: "eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				oldVal := os.Getenv(k)
				os.Setenv(k, v)
				defer os.Setenv(k, oldVal)
			}

			ctx := context.Background()
			_, err := NewAWSClients(ctx, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetRegion(t *testing.T) {
	// This test verifies the GetRegion method works correctly
	// In a real environment with credentials, this would be more comprehensive
	
	clients := &AWSClients{
		Config: mockAWSConfig("us-east-1"),
	}
	
	assert.Equal(t, "us-east-1", clients.GetRegion())
}

// mockAWSConfig creates a mock AWS config for testing
func mockAWSConfig(region string) aws.Config {
	return aws.Config{
		Region: region,
	}
}

// TestValidateCredentials would require mocking AWS services
// In a real test environment, we would use aws-sdk-go-v2's testing utilities
func TestValidateCredentials(t *testing.T) {
	t.Skip("Skipping credential validation test - requires AWS service mocking")
}