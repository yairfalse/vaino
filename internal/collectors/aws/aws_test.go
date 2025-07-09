package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAWSPackage verifies the AWS package structure
func TestAWSPackage(t *testing.T) {
	// Test that all components are available
	t.Run("Components", func(t *testing.T) {
		// Verify normalizer
		normalizer := NewNormalizer("us-east-1")
		assert.NotNil(t, normalizer)
		assert.Equal(t, "us-east-1", normalizer.region)

		// Verify collector
		collector := NewAWSCollector()
		assert.NotNil(t, collector)
		assert.Equal(t, "aws", collector.Name())
		assert.Equal(t, "not_configured", collector.Status())
	})

	// Test supported regions
	t.Run("Regions", func(t *testing.T) {
		collector := NewAWSCollector()
		regions := collector.SupportedRegions()

		// Should support major regions
		assert.Contains(t, regions, "us-east-1")
		assert.Contains(t, regions, "us-west-2")
		assert.Contains(t, regions, "eu-west-1")
		assert.Contains(t, regions, "ap-southeast-1")
	})

	// Test default configuration
	t.Run("DefaultConfig", func(t *testing.T) {
		collector := NewAWSCollector()
		config := collector.GetDefaultConfig()

		assert.Equal(t, "us-east-1", config["region"])
		assert.Equal(t, "", config["profile"])
	})
}
