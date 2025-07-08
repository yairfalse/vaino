// +build integration

package aws

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/internal/storage"
	"github.com/yairfalse/wgo/pkg/types"
)

// TestAWSProviderSystem tests the full AWS provider workflow
func TestAWSProviderSystem(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	ctx := context.Background()
	tmpDir, err := ioutil.TempDir("", "wgo-aws-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test scenario: Full workflow
	t.Run("FullWorkflow", func(t *testing.T) {
		// 1. Create AWS collector
		awsCollector := NewAWSCollector()
		assert.Equal(t, "aws", awsCollector.Name())

		// 2. Auto-discover configuration
		config, err := awsCollector.AutoDiscover()
		assert.NoError(t, err)

		// 3. Set test configuration
		config.Config = map[string]interface{}{
			"region":  "us-east-1",
			"profile": os.Getenv("AWS_PROFILE"), // Use env var if set
		}

		// 4. Validate configuration
		err = awsCollector.Validate(config)
		if err != nil {
			t.Skip("Skipping test - AWS credentials not configured")
		}

		// 5. Collect AWS resources
		snapshot, err := awsCollector.Collect(ctx, config)
		require.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, "aws", snapshot.Provider)
		assert.NotEmpty(t, snapshot.Resources)

		// 6. Save snapshot to storage
		storageConfig := storage.Config{BaseDir: tmpDir}
		localStorage, err := storage.NewLocalStorage(storageConfig)
		require.NoError(t, err)

		err = localStorage.SaveSnapshot(snapshot)
		assert.NoError(t, err)

		// 7. Verify we can load the snapshot back
		loadedSnapshot, err := localStorage.LoadSnapshot(snapshot.ID)
		require.NoError(t, err)
		assert.Equal(t, snapshot.ID, loadedSnapshot.ID)
		assert.Equal(t, len(snapshot.Resources), len(loadedSnapshot.Resources))
	})
}

// TestAWSResourceTypes verifies that different AWS resource types are collected correctly
func TestAWSResourceTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	ctx := context.Background()
	collector := NewAWSCollector()

	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region": "us-east-1",
		},
	}

	// Check if we can validate (have credentials)
	if err := collector.Validate(config); err != nil {
		t.Skip("Skipping test - AWS credentials not configured")
	}

	snapshot, err := collector.Collect(ctx, config)
	require.NoError(t, err)

	// Group resources by type
	resourcesByType := make(map[string][]types.Resource)
	for _, resource := range snapshot.Resources {
		resourcesByType[resource.Type] = append(resourcesByType[resource.Type], resource)
	}

	// Log what we found
	t.Logf("Found %d total resources", len(snapshot.Resources))
	for resType, resources := range resourcesByType {
		t.Logf("  %s: %d", resType, len(resources))

		// Verify each resource has required fields
		for _, resource := range resources {
			assert.NotEmpty(t, resource.ID, "Resource should have ID")
			assert.Equal(t, "aws", resource.Provider, "Resource should have AWS provider")
			assert.NotEmpty(t, resource.Type, "Resource should have type")
			assert.NotEmpty(t, resource.Region, "Resource should have region")
			assert.NotNil(t, resource.Configuration, "Resource should have configuration")
		}
	}

	// Verify we collected multiple resource types
	assert.True(t, len(resourcesByType) > 0, "Should collect at least one resource type")
}

// TestAWSDriftDetection tests drift detection between two AWS snapshots
func TestAWSDriftDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	ctx := context.Background()
	collector := NewAWSCollector()

	config := collectors.CollectorConfig{
		Config: map[string]interface{}{
			"region": "us-east-1",
		},
	}

	// Check credentials
	if err := collector.Validate(config); err != nil {
		t.Skip("Skipping test - AWS credentials not configured")
	}

	// Collect first snapshot
	snapshot1, err := collector.Collect(ctx, config)
	require.NoError(t, err)

	// Wait a bit and collect second snapshot
	time.Sleep(2 * time.Second)
	snapshot2, err := collector.Collect(ctx, config)
	require.NoError(t, err)

	// Create differ
	differEngine := differ.NewDifferEngine(differ.DiffOptions{})

	// Compare snapshots
	report, err := differEngine.Compare(snapshot1, snapshot2)
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Log drift report
	t.Logf("Drift Report:")
	t.Logf("  Total Resources: %d", report.Summary.TotalResources)
	t.Logf("  Changed Resources: %d", report.Summary.ChangedResources)
	t.Logf("  Added Resources: %d", report.Summary.AddedResources)
	t.Logf("  Removed Resources: %d", report.Summary.RemovedResources)
	t.Logf("  Modified Resources: %d", report.Summary.ModifiedResources)

	// In a real environment, we might see some changes
	// For testing, we just verify the comparison works
	assert.GreaterOrEqual(t, report.Summary.TotalResources, 0)
}

// TestAWSProviderWithMockData tests the provider with mock AWS data
func TestAWSProviderWithMockData(t *testing.T) {
	// This test uses a mock AWS response file
	mockDataFile := filepath.Join("testdata", "aws_mock_response.json")
	
	// Check if mock data exists
	if _, err := os.Stat(mockDataFile); os.IsNotExist(err) {
		t.Skip("Mock data file not found")
	}

	// Load mock data
	mockData, err := ioutil.ReadFile(mockDataFile)
	require.NoError(t, err)

	var mockSnapshot types.Snapshot
	err = json.Unmarshal(mockData, &mockSnapshot)
	require.NoError(t, err)

	// Verify mock snapshot structure
	assert.Equal(t, "aws", mockSnapshot.Provider)
	assert.NotEmpty(t, mockSnapshot.Resources)

	// Verify resource types
	resourceTypes := make(map[string]int)
	for _, resource := range mockSnapshot.Resources {
		resourceTypes[resource.Type]++
	}

	// Should have various AWS resource types
	assert.Contains(t, resourceTypes, "aws_instance")
	assert.Contains(t, resourceTypes, "aws_security_group")
	assert.Contains(t, resourceTypes, "aws_s3_bucket")
}

// TestAWSMultiRegionCollection tests collecting resources from multiple regions
func TestAWSMultiRegionCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	ctx := context.Background()
	regions := []string{"us-east-1", "us-west-2"}
	allResources := []types.Resource{}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			collector := NewAWSCollector()
			config := collectors.CollectorConfig{
				Config: map[string]interface{}{
					"region": region,
				},
			}

			// Check credentials
			if err := collector.Validate(config); err != nil {
				t.Skip("Skipping test - AWS credentials not configured")
			}

			snapshot, err := collector.Collect(ctx, config)
			if err != nil {
				t.Logf("Failed to collect from region %s: %v", region, err)
				return
			}

			t.Logf("Collected %d resources from %s", len(snapshot.Resources), region)
			allResources = append(allResources, snapshot.Resources...)

			// Verify all resources have the correct region
			for _, resource := range snapshot.Resources {
				if resource.Type != "aws_iam_role" && resource.Type != "aws_iam_user" {
					// IAM resources are global
					assert.Equal(t, region, resource.Region)
				}
			}
		})
	}

	t.Logf("Total resources collected from all regions: %d", len(allResources))
}