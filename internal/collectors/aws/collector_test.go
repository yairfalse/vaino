package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yairfalse/vaino/internal/collectors"
)

func TestAWSCollectorInterface(t *testing.T) {
	// Verify AWSCollector implements EnhancedCollector interface
	var _ collectors.EnhancedCollector = (*AWSCollector)(nil)

	collector := NewAWSCollector()
	assert.NotNil(t, collector)
}

func TestAWSCollectorMethods(t *testing.T) {
	collector := NewAWSCollector()

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "aws", collector.Name())
	})

	t.Run("Status", func(t *testing.T) {
		// Without clients configured
		assert.Equal(t, "not_configured", collector.Status())
	})

	t.Run("GetDescription", func(t *testing.T) {
		assert.Contains(t, collector.GetDescription(), "AWS resource collector")
		assert.Contains(t, collector.GetDescription(), "EC2")
		assert.Contains(t, collector.GetDescription(), "S3")
		assert.Contains(t, collector.GetDescription(), "Lambda")
	})

	t.Run("GetVersion", func(t *testing.T) {
		assert.Equal(t, "1.0.0", collector.GetVersion())
	})

	t.Run("SupportsRegion", func(t *testing.T) {
		assert.True(t, collector.SupportsRegion())
	})

	t.Run("SupportedRegions", func(t *testing.T) {
		regions := collector.SupportedRegions()
		assert.Contains(t, regions, "us-east-1")
		assert.Contains(t, regions, "us-west-2")
		assert.Contains(t, regions, "eu-west-1")
		assert.True(t, len(regions) > 10) // Should have many regions
	})

	t.Run("GetDefaultConfig", func(t *testing.T) {
		config := collector.GetDefaultConfig()
		assert.Equal(t, "us-east-1", config["region"])
		assert.Equal(t, "", config["profile"])
	})
}

func TestAutoDiscover(t *testing.T) {
	collector := NewAWSCollector()

	config, err := collector.AutoDiscover()
	assert.NoError(t, err)
	assert.NotNil(t, config.Config)

	// Auto-discover should return empty config (uses defaults)
	assert.Equal(t, "", config.Config["region"])
	assert.Equal(t, "", config.Config["profile"])
}

func TestValidate(t *testing.T) {
	collector := NewAWSCollector()

	tests := []struct {
		name        string
		config      collectors.CollectorConfig
		expectError bool
	}{
		{
			name: "empty config",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{},
			},
			expectError: true, // Will fail without credentials
		},
		{
			name: "with region",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"region": "us-west-2",
				},
			},
			expectError: true, // Will fail without credentials
		},
		{
			name: "with profile",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"profile": "production",
					"region":  "eu-west-1",
				},
			},
			expectError: true, // Will fail without actual profile
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.Validate(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectWithoutCredentials(t *testing.T) {
	// This test verifies proper error handling when credentials are missing
	collector := NewAWSCollector()
	ctx := context.Background()

	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region": "us-east-1",
		},
	}

	snapshot, err := collector.Collect(ctx, config)
	assert.Error(t, err)
	assert.Nil(t, snapshot)
	assert.Contains(t, err.Error(), "failed to create AWS clients")
}

// Integration test that would run with real AWS credentials
func TestCollectIntegration(t *testing.T) {
	// Skip this test if AWS credentials are not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if AWS credentials are available
	ctx := context.Background()
	testConfig := ClientConfig{
		Region: "us-east-1",
	}

	_, err := NewAWSClients(ctx, testConfig)
	if err != nil {
		t.Skip("Skipping integration test - AWS credentials not available")
	}

	// Run actual collection
	collector := NewAWSCollector()
	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region": "us-east-1",
		},
	}

	// Validate configuration
	err = collector.Validate(config)
	require.NoError(t, err)

	// Collect resources
	snapshot, err := collector.Collect(ctx, config)
	require.NoError(t, err)
	assert.NotNil(t, snapshot)

	// Verify snapshot metadata
	assert.Equal(t, "aws", snapshot.Provider)
	assert.NotEmpty(t, snapshot.ID)
	assert.NotZero(t, snapshot.Timestamp)

	// Log resource counts for debugging
	resourceTypes := make(map[string]int)
	for _, resource := range snapshot.Resources {
		resourceTypes[resource.Type]++
	}

	t.Logf("Collected resources:")
	for resType, count := range resourceTypes {
		t.Logf("  %s: %d", resType, count)
	}
}
