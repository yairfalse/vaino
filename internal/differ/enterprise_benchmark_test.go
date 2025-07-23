//go:build enterprise
// +build enterprise

package differ

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// BenchmarkEnterpriseDiffEngine_10K tests performance with 10,000 resources
func BenchmarkEnterpriseDiffEngine_10K(b *testing.B) {
	baseline := generateTestSnapshot("baseline", 10000)
	current := generateTestSnapshot("current", 10000)

	// Modify 10% of resources to create realistic drift
	modifyResources(current.Resources, 0.1)

	options := EnterpriseDiffOptions{
		MaxWorkers:        8,
		ParallelThreshold: 100,
		EnableCaching:     true,
		EnableIndexing:    true,
		EnableCorrelation: true,
		EnableRiskScoring: true,
	}

	engine := NewEnterpriseDifferEngine(options)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report, err := engine.Compare(baseline, current)
		if err != nil {
			b.Fatalf("Diff failed: %v", err)
		}

		// Verify we get expected results
		if report.Summary.TotalChanges == 0 {
			b.Fatalf("Expected changes but got none")
		}
	}
}

// BenchmarkEnterpriseDiffEngine_50K tests performance with 50,000 resources
func BenchmarkEnterpriseDiffEngine_50K(b *testing.B) {
	baseline := generateTestSnapshot("baseline", 50000)
	current := generateTestSnapshot("current", 50000)

	// Modify 5% of resources
	modifyResources(current.Resources, 0.05)

	options := EnterpriseDiffOptions{
		MaxWorkers:        16,
		ParallelThreshold: 100,
		EnableCaching:     true,
		EnableIndexing:    true,
		EnableCorrelation: false, // Disable for very large datasets
		EnableRiskScoring: false, // Disable for very large datasets
	}

	engine := NewEnterpriseDifferEngine(options)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		report, err := engine.Compare(baseline, current)
		duration := time.Since(start)

		if err != nil {
			b.Fatalf("Diff failed: %v", err)
		}

		// Performance requirement: <10s for 50K resources
		if duration > 10*time.Second {
			b.Fatalf("Performance requirement failed: %v > 10s", duration)
		}

		if report.Summary.TotalChanges == 0 {
			b.Fatalf("Expected changes but got none")
		}
	}
}

// BenchmarkParallelVsSequential compares parallel vs sequential processing
func BenchmarkParallelVsSequential(b *testing.B) {
	baseline := generateTestSnapshot("baseline", 5000)
	current := generateTestSnapshot("current", 5000)
	modifyResources(current.Resources, 0.1)

	b.Run("Sequential", func(b *testing.B) {
		options := EnterpriseDiffOptions{
			MaxWorkers:        1,
			ParallelThreshold: 999999, // Disable parallel
			EnableCaching:     false,
			EnableIndexing:    false,
		}
		engine := NewEnterpriseDifferEngine(options)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Compare(baseline, current)
			if err != nil {
				b.Fatalf("Diff failed: %v", err)
			}
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		options := EnterpriseDiffOptions{
			MaxWorkers:        8,
			ParallelThreshold: 100,
			EnableCaching:     false,
			EnableIndexing:    false,
		}
		engine := NewEnterpriseDifferEngine(options)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Compare(baseline, current)
			if err != nil {
				b.Fatalf("Diff failed: %v", err)
			}
		}
	})
}

// BenchmarkCachePerformance tests caching impact
func BenchmarkCachePerformance(b *testing.B) {
	baseline := generateTestSnapshot("baseline", 1000)
	current := generateTestSnapshot("current", 1000)
	modifyResources(current.Resources, 0.1)

	b.Run("WithoutCache", func(b *testing.B) {
		options := EnterpriseDiffOptions{
			MaxWorkers:        4,
			ParallelThreshold: 100,
			EnableCaching:     false,
		}
		engine := NewEnterpriseDifferEngine(options)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Compare(baseline, current)
			if err != nil {
				b.Fatalf("Diff failed: %v", err)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		options := EnterpriseDiffOptions{
			MaxWorkers:        4,
			ParallelThreshold: 100,
			EnableCaching:     true,
		}
		engine := NewEnterpriseDifferEngine(options)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Compare(baseline, current)
			if err != nil {
				b.Fatalf("Diff failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryUsage tests memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
	baseline := generateTestSnapshot("baseline", 10000)
	current := generateTestSnapshot("current", 10000)
	modifyResources(current.Resources, 0.1)

	options := EnterpriseDiffOptions{
		MaxWorkers:        8,
		ParallelThreshold: 100,
		EnableCaching:     true,
		EnableIndexing:    true,
	}
	engine := NewEnterpriseDifferEngine(options)

	// Track memory usage
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Compare(baseline, current)
		if err != nil {
			b.Fatalf("Diff failed: %v", err)
		}
	}
}

// TestEnterpriseDiffEngine_PerformanceRequirements validates performance requirements
func TestEnterpriseDiffEngine_PerformanceRequirements(t *testing.T) {
	tests := []struct {
		name          string
		resourceCount int
		maxDuration   time.Duration
		modifyPercent float64
	}{
		{
			name:          "1K Resources",
			resourceCount: 1000,
			maxDuration:   500 * time.Millisecond,
			modifyPercent: 0.1,
		},
		{
			name:          "5K Resources",
			resourceCount: 5000,
			maxDuration:   1500 * time.Millisecond,
			modifyPercent: 0.1,
		},
		{
			name:          "10K Resources",
			resourceCount: 10000,
			maxDuration:   3 * time.Second,
			modifyPercent: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseline := generateTestSnapshot("baseline", tt.resourceCount)
			current := generateTestSnapshot("current", tt.resourceCount)
			modifyResources(current.Resources, tt.modifyPercent)

			options := EnterpriseDiffOptions{
				MaxWorkers:        8,
				ParallelThreshold: 100,
				EnableCaching:     true,
				EnableIndexing:    true,
				EnableCorrelation: true,
				EnableRiskScoring: true,
			}
			engine := NewEnterpriseDifferEngine(options)

			start := time.Now()
			report, err := engine.Compare(baseline, current)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Diff failed: %v", err)
			}

			if duration > tt.maxDuration {
				t.Errorf("Performance requirement failed: %v > %v for %d resources",
					duration, tt.maxDuration, tt.resourceCount)
			}

			expectedChanges := int(float64(tt.resourceCount) * tt.modifyPercent)
			if report.Summary.TotalChanges < expectedChanges/2 {
				t.Errorf("Expected at least %d changes, got %d",
					expectedChanges/2, report.Summary.TotalChanges)
			}

			t.Logf("Performance: %v for %d resources (%d changes detected)",
				duration, tt.resourceCount, report.Summary.TotalChanges)
		})
	}
}

// TestEnterpriseDiffEngine_CorrelationPerformance tests correlation analysis performance
func TestEnterpriseDiffEngine_CorrelationPerformance(t *testing.T) {
	baseline := generateTestSnapshot("baseline", 2000)
	current := generateTestSnapshot("current", 2000)

	// Create correlated changes (scaling scenario)
	createScalingScenario(baseline.Resources, current.Resources)

	options := EnterpriseDiffOptions{
		MaxWorkers:        8,
		ParallelThreshold: 100,
		EnableCorrelation: true,
	}
	engine := NewEnterpriseDifferEngine(options)

	start := time.Now()
	report, err := engine.Compare(baseline, current)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Correlation analysis should not significantly impact performance
	maxDuration := 2 * time.Second
	if duration > maxDuration {
		t.Errorf("Correlation analysis too slow: %v > %v", duration, maxDuration)
	}

	// Should detect changes
	if report.Summary.TotalChanges == 0 {
		t.Errorf("Expected changes but got none")
	}

	t.Logf("Correlation analysis: %v for %d resources (%d changes)",
		duration, len(baseline.Resources), report.Summary.TotalChanges)
}

// TestEnterpriseDiffEngine_CompliancePerformance tests compliance analysis performance
func TestEnterpriseDiffEngine_CompliancePerformance(t *testing.T) {
	baseline := generateTestSnapshot("baseline", 1000)
	current := generateTestSnapshot("current", 1000)

	// Create security-related changes
	createSecurityChanges(baseline.Resources, current.Resources)

	options := EnterpriseDiffOptions{
		MaxWorkers:       8,
		EnableCompliance: true,
		ComplianceRules:  GetBuiltInComplianceRules(),
	}
	engine := NewEnterpriseDifferEngine(options)

	start := time.Now()
	report, err := engine.Compare(baseline, current)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Compliance analysis should complete in reasonable time
	maxDuration := 1 * time.Second
	if duration > maxDuration {
		t.Errorf("Compliance analysis too slow: %v > %v", duration, maxDuration)
	}

	// Should have compliance report
	if report.ComplianceReport == nil {
		t.Errorf("Expected compliance report but got none")
	}

	t.Logf("Compliance analysis: %v for %d resources", duration, len(baseline.Resources))
}

// TestEnterpriseDiffEngine_StreamingPerformance tests streaming performance
func TestEnterpriseDiffEngine_StreamingPerformance(t *testing.T) {
	baseline := generateTestSnapshot("baseline", 1000)
	current := generateTestSnapshot("current", 1000)
	modifyResources(current.Resources, 0.2) // 20% changes

	options := EnterpriseDiffOptions{
		MaxWorkers:       8,
		StreamingEnabled: true,
		StreamBufferSize: 100,
	}
	engine := NewEnterpriseDifferEngine(options)

	// Monitor streaming
	streamedChanges := 0
	go func() {
		for range engine.GetChangeStream() {
			streamedChanges++
		}
	}()

	start := time.Now()
	_, err := engine.Compare(baseline, current)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Wait a bit for streaming to complete
	time.Sleep(100 * time.Millisecond)

	// Should not significantly impact performance
	maxDuration := 1 * time.Second
	if duration > maxDuration {
		t.Errorf("Streaming analysis too slow: %v > %v", duration, maxDuration)
	}

	if streamedChanges == 0 {
		t.Errorf("Expected streamed changes but got none")
	}

	t.Logf("Streaming analysis: %v, streamed %d changes", duration, streamedChanges)
}

// Helper functions for benchmarks and tests

func generateTestSnapshot(id string, resourceCount int) *types.Snapshot {
	resources := make([]types.Resource, resourceCount)

	resourceTypes := []string{
		"aws_instance", "aws_security_group", "aws_s3_bucket", "aws_iam_role",
		"kubernetes_deployment", "kubernetes_service", "kubernetes_configmap",
		"gcp_compute_instance", "gcp_storage_bucket", "gcp_sql_instance",
	}

	for i := 0; i < resourceCount; i++ {
		resourceType := resourceTypes[i%len(resourceTypes)]
		resources[i] = types.Resource{
			ID:       fmt.Sprintf("%s-%d", resourceType, i),
			Type:     resourceType,
			Name:     fmt.Sprintf("resource-%d", i),
			Provider: getProviderFromType(resourceType),
			Region:   getRegionForIndex(i),
			Configuration: map[string]interface{}{
				"instance_type": "t3.micro",
				"ami":           "ami-12345678",
				"tags": map[string]string{
					"Name":        fmt.Sprintf("resource-%d", i),
					"Environment": getEnvironmentForIndex(i),
				},
				"security_groups": []string{"sg-12345"},
				"subnet_id":       "subnet-12345",
			},
			Tags: map[string]string{
				"Name":        fmt.Sprintf("resource-%d", i),
				"Environment": getEnvironmentForIndex(i),
				"Team":        "platform",
			},
		}
	}

	return &types.Snapshot{
		ID:        id,
		Timestamp: time.Now(),
		Provider:  "multi",
		Resources: resources,
	}
}

func modifyResources(resources []types.Resource, percent float64) {
	modifyCount := int(float64(len(resources)) * percent)

	for i := 0; i < modifyCount; i++ {
		idx := i % len(resources)
		resource := &resources[idx]

		// Modify configuration
		if config, ok := resource.Configuration["tags"].(map[string]string); ok {
			config["Modified"] = "true"
			config["ModifiedAt"] = time.Now().Format(time.RFC3339)
		}

		// Change instance type occasionally
		if i%5 == 0 {
			resource.Configuration["instance_type"] = "t3.small"
		}

		// Add new configuration occasionally
		if i%7 == 0 {
			resource.Configuration["monitoring"] = true
		}
	}
}

func createScalingScenario(baseline, current []types.Resource) {
	// Simulate auto-scaling by modifying instances and ASGs
	for i := range current {
		if current[i].Type == "aws_instance" || current[i].Type == "aws_autoscaling_group" {
			if i%3 == 0 {
				current[i].Configuration["desired_capacity"] = 5 // Scale up
			}
			current[i].Configuration["last_scaling_activity"] = time.Now().Format(time.RFC3339)
		}
	}
}

func createSecurityChanges(baseline, current []types.Resource) {
	// Simulate security-related changes
	for i := range current {
		if current[i].Type == "aws_security_group" || current[i].Type == "aws_iam_role" {
			// Add overly permissive rule
			if i%4 == 0 {
				current[i].Configuration["ingress_rules"] = []map[string]interface{}{
					{
						"from_port":   80,
						"to_port":     80,
						"protocol":    "tcp",
						"cidr_blocks": []string{"0.0.0.0/0"},
					},
				}
			}
		}
	}
}

func getProviderFromType(resourceType string) string {
	if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	}
	if strings.HasPrefix(resourceType, "kubernetes_") {
		return "kubernetes"
	}
	if strings.HasPrefix(resourceType, "gcp_") {
		return "gcp"
	}
	return "unknown"
}

func getRegionForIndex(index int) string {
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	return regions[index%len(regions)]
}

func getEnvironmentForIndex(index int) string {
	envs := []string{"production", "staging", "development", "test"}
	return envs[index%len(envs)]
}
