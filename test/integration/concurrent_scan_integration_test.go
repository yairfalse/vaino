package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/internal/scanner"
	"github.com/yairfalse/wgo/pkg/types"
)

// TestConcurrentScanIntegration tests end-to-end concurrent scanning
func TestConcurrentScanIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Create test Terraform state file
	terraformState := `{
		"version": 4,
		"terraform_version": "1.5.0",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"schema_version": 1,
						"attributes": {
							"id": "i-1234567890abcdef0",
							"instance_type": "t3.micro",
							"ami": "ami-12345678"
						}
					}
				]
			}
		]
	}`

	stateFile := filepath.Join(tmpDir, "terraform.tfstate")
	err := os.WriteFile(stateFile, []byte(terraformState), 0644)
	if err != nil {
		t.Fatalf("Failed to create test state file: %v", err)
	}

	// Test concurrent scanner with real components
	t.Run("ConcurrentScanner_Integration", func(t *testing.T) {
		scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
		defer scanner.Close()

		// Register actual collectors (they will use mock data since we don't have real credentials)
		scanner.RegisterProvider("terraform", terraform.NewTerraformCollector())

		// Create scan configuration
		config := scanner.ScanConfig{
			Providers: map[string]collectors.CollectorConfig{
				"terraform": {
					StatePaths: []string{stateFile},
					Config: map[string]interface{}{
						"path": tmpDir,
					},
				},
			},
			MaxWorkers:  4,
			Timeout:     30 * time.Second,
			FailOnError: false,
		}

		// Perform concurrent scan
		ctx := context.Background()
		startTime := time.Now()

		result, err := scanner.ScanAllProviders(ctx, config)

		scanDuration := time.Since(startTime)

		// Validate results
		if err != nil {
			t.Fatalf("Concurrent scan failed: %v", err)
		}

		if result == nil {
			t.Fatal("Expected non-nil result")
		}

		// Check terraform provider result
		terraformResult := result.ProviderResults["terraform"]
		if terraformResult == nil {
			t.Fatal("Expected terraform result")
		}

		if terraformResult.Error != nil {
			t.Fatalf("Terraform scan failed: %v", terraformResult.Error)
		}

		if len(terraformResult.Snapshot.Resources) == 0 {
			t.Error("Expected terraform resources")
		}

		// Check merged snapshot
		if result.Snapshot == nil {
			t.Error("Expected merged snapshot")
		}

		t.Logf("Concurrent scan completed in %v", scanDuration)
		t.Logf("Found %d resources", len(result.Snapshot.Resources))
	})

	// Test CLI integration
	// t.Run("CLI_ConcurrentScan", func(t *testing.T) {
	// 	// This test would normally use exec.Command to test the CLI
	// 	// but we'll test the command functions directly
	//
	// 	// Skip command test - function is not exported
	// 	// cmd := commands.NewScanCommand()
	//
	// 	// Set up command flags
	// 	// cmd.Flags().Set("concurrent", "true")
	// 	// cmd.Flags().Set("max-workers", "2")
	// 	// cmd.Flags().Set("quiet", "true")
	// 	// cmd.Flags().Set("state-file", stateFile)
	// 	// cmd.Flags().Set("provider", "terraform")
	//
	// 	// Note: This would normally execute the command
	// 	// In a real integration test, you'd use exec.Command or similar
	// 	// t.Log("CLI command would be executed here")
	// })
}

func TestConcurrentScanPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Performance comparison test
	t.Run("Sequential_vs_Concurrent", func(t *testing.T) {
		// Create mock collectors with simulated delays
		mockCollectors := map[string]*MockDelayedCollector{
			"aws":        NewMockDelayedCollector("aws", 200*time.Millisecond, 50),
			"gcp":        NewMockDelayedCollector("gcp", 300*time.Millisecond, 30),
			"kubernetes": NewMockDelayedCollector("kubernetes", 250*time.Millisecond, 100),
		}

		// Test sequential execution
		t.Run("Sequential", func(t *testing.T) {
			ctx := context.Background()
			config := collectors.CollectorConfig{
				Config: map[string]interface{}{"test": "value"},
			}

			startTime := time.Now()

			for name, collector := range mockCollectors {
				snapshot, err := collector.Collect(ctx, config)
				if err != nil {
					t.Errorf("Sequential scan failed for %s: %v", name, err)
				}
				if snapshot == nil {
					t.Errorf("Expected snapshot for %s", name)
				}
			}

			sequentialDuration := time.Since(startTime)
			t.Logf("Sequential scan took: %v", sequentialDuration)
		})

		// Test concurrent execution
		t.Run("Concurrent", func(t *testing.T) {
			scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
			defer scanner.Close()

			// Register providers
			for name, collector := range mockCollectors {
				scanner.RegisterProvider(name, collector)
			}

			// Create scan configuration
			config := scanner.ScanConfig{
				Providers: map[string]collectors.CollectorConfig{
					"aws":        {Config: map[string]interface{}{"test": "value"}},
					"gcp":        {Config: map[string]interface{}{"test": "value"}},
					"kubernetes": {Config: map[string]interface{}{"test": "value"}},
				},
				MaxWorkers:  4,
				Timeout:     30 * time.Second,
				FailOnError: false,
			}

			ctx := context.Background()
			startTime := time.Now()

			result, err := scanner.ScanAllProviders(ctx, config)

			concurrentDuration := time.Since(startTime)

			if err != nil {
				t.Fatalf("Concurrent scan failed: %v", err)
			}

			if result.SuccessCount != 3 {
				t.Errorf("Expected 3 successful scans, got %d", result.SuccessCount)
			}

			t.Logf("Concurrent scan took: %v", concurrentDuration)

			// Concurrent should be significantly faster
			if concurrentDuration >= 500*time.Millisecond { // Should be much less than sequential
				t.Errorf("Concurrent scan not fast enough: %v", concurrentDuration)
			}
		})
	})
}

func TestConcurrentScanErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling test in short mode")
	}

	scanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer scanner.Close()

	// Register providers with mixed success/failure
	scanner.RegisterProvider("success", NewMockDelayedCollector("success", 100*time.Millisecond, 10))
	scanner.RegisterProvider("failure", NewMockFailingCollector("failure", "simulated failure"))

	// Test with FailOnError=false
	t.Run("ContinueOnError", func(t *testing.T) {
		config := scanner.ScanConfig{
			Providers: map[string]collectors.CollectorConfig{
				"success": {Config: map[string]interface{}{"test": "value"}},
				"failure": {Config: map[string]interface{}{"test": "value"}},
			},
			MaxWorkers:  4,
			Timeout:     30 * time.Second,
			FailOnError: false,
		}

		ctx := context.Background()
		result, err := scanner.ScanAllProviders(ctx, config)

		// Should not fail
		if err != nil {
			t.Fatalf("Expected no error with FailOnError=false, got: %v", err)
		}

		// Should have 1 success and 1 failure
		if result.SuccessCount != 1 {
			t.Errorf("Expected 1 success, got %d", result.SuccessCount)
		}

		if result.ErrorCount != 1 {
			t.Errorf("Expected 1 error, got %d", result.ErrorCount)
		}
	})

	// Test with FailOnError=true
	t.Run("FailOnError", func(t *testing.T) {
		config := scanner.ScanConfig{
			Providers: map[string]collectors.CollectorConfig{
				"success": {Config: map[string]interface{}{"test": "value"}},
				"failure": {Config: map[string]interface{}{"test": "value"}},
			},
			MaxWorkers:  4,
			Timeout:     30 * time.Second,
			FailOnError: true,
		}

		ctx := context.Background()
		result, err := scanner.ScanAllProviders(ctx, config)

		// Should fail
		if err == nil {
			t.Error("Expected error with FailOnError=true")
		}

		if result != nil {
			t.Error("Expected nil result when scan fails")
		}
	})
}

// MockDelayedCollector simulates a collector with processing delay
type MockDelayedCollector struct {
	name          string
	delay         time.Duration
	resourceCount int
}

func NewMockDelayedCollector(name string, delay time.Duration, resourceCount int) *MockDelayedCollector {
	return &MockDelayedCollector{
		name:          name,
		delay:         delay,
		resourceCount: resourceCount,
	}
}

func (m *MockDelayedCollector) Name() string {
	return m.name
}

func (m *MockDelayedCollector) Status() string {
	return "ready"
}

func (m *MockDelayedCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	// Simulate processing delay
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Create mock resources
	resources := make([]types.Resource, m.resourceCount)
	for i := 0; i < m.resourceCount; i++ {
		resources[i] = types.Resource{
			ID:       fmt.Sprintf("%s-resource-%d", m.name, i),
			Type:     fmt.Sprintf("%s_resource", m.name),
			Name:     fmt.Sprintf("resource-%d", i),
			Provider: m.name,
			Configuration: map[string]interface{}{
				"index": i,
			},
		}
	}

	return &types.Snapshot{
		ID:        fmt.Sprintf("%s-snapshot", m.name),
		Timestamp: time.Now(),
		Provider:  m.name,
		Resources: resources,
	}, nil
}

func (m *MockDelayedCollector) Validate(config collectors.CollectorConfig) error {
	return nil
}

func (m *MockDelayedCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	return collectors.CollectorConfig{}, nil
}

func (m *MockDelayedCollector) SupportedRegions() []string {
	return []string{"us-east-1", "us-west-2"}
}

// MockFailingCollector simulates a collector that always fails
type MockFailingCollector struct {
	name  string
	error string
}

func NewMockFailingCollector(name, errorMsg string) *MockFailingCollector {
	return &MockFailingCollector{
		name:  name,
		error: errorMsg,
	}
}

func (m *MockFailingCollector) Name() string {
	return m.name
}

func (m *MockFailingCollector) Status() string {
	return "error"
}

func (m *MockFailingCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	return nil, fmt.Errorf(m.error)
}

func (m *MockFailingCollector) Validate(config collectors.CollectorConfig) error {
	return nil
}

func (m *MockFailingCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	return collectors.CollectorConfig{}, nil
}

func (m *MockFailingCollector) SupportedRegions() []string {
	return []string{}
}
