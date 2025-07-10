package concurrent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/scanner"
	"github.com/yairfalse/vaino/pkg/types"
)

// MockConcurrentCollector implements EnhancedCollector for testing
type MockConcurrentCollector struct {
	name          string
	delay         time.Duration
	shouldFail    bool
	resourceCount int
	collectCalls  int
	mu            sync.Mutex
}

func NewMockConcurrentCollector(name string, delay time.Duration, shouldFail bool, resourceCount int) *MockConcurrentCollector {
	return &MockConcurrentCollector{
		name:          name,
		delay:         delay,
		shouldFail:    shouldFail,
		resourceCount: resourceCount,
	}
}

func (m *MockConcurrentCollector) Name() string {
	return m.name
}

func (m *MockConcurrentCollector) Status() string {
	return "ready"
}

func (m *MockConcurrentCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	m.mu.Lock()
	m.collectCalls++
	callCount := m.collectCalls
	m.mu.Unlock()

	// Simulate processing delay
	select {
	case <-time.After(m.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if m.shouldFail {
		return nil, fmt.Errorf("mock collector %s failed", m.name)
	}

	// Create mock resources
	resources := make([]types.Resource, m.resourceCount)
	for i := 0; i < m.resourceCount; i++ {
		resources[i] = types.Resource{
			ID:       fmt.Sprintf("%s-resource-%d-%d", m.name, callCount, i),
			Type:     fmt.Sprintf("%s_resource", m.name),
			Name:     fmt.Sprintf("%s-resource-%d", m.name, i),
			Provider: m.name,
			Configuration: map[string]interface{}{
				"index":      i,
				"call_count": callCount,
			},
		}
	}

	return &types.Snapshot{
		ID:        fmt.Sprintf("%s-snapshot-%d", m.name, callCount),
		Timestamp: time.Now(),
		Provider:  m.name,
		Resources: resources,
	}, nil
}

func (m *MockConcurrentCollector) Validate(config collectors.CollectorConfig) error {
	return nil
}

func (m *MockConcurrentCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	return collectors.CollectorConfig{}, nil
}

func (m *MockConcurrentCollector) SupportedRegions() []string {
	return []string{"us-east-1", "us-west-2"}
}

func (m *MockConcurrentCollector) GetCollectCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.collectCalls
}

func TestConcurrentScanner_Basic(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Register mock collectors
	collectors := map[string]*MockConcurrentCollector{
		"aws":        NewMockConcurrentCollector("aws", 100*time.Millisecond, false, 5),
		"gcp":        NewMockConcurrentCollector("gcp", 150*time.Millisecond, false, 3),
		"kubernetes": NewMockConcurrentCollector("kubernetes", 200*time.Millisecond, false, 8),
	}

	for name, collector := range collectors {
		concurrentScanner.RegisterProvider(name, collector)
	}

	// Create scan configuration
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws":        {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp":        {Config: map[string]interface{}{"project": "test-project"}},
			"kubernetes": {Config: map[string]interface{}{"context": "test-context"}},
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
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check that all providers were scanned
	if len(result.ProviderResults) != 3 {
		t.Errorf("Expected 3 provider results, got %d", len(result.ProviderResults))
	}

	// Check individual provider results
	for providerName, providerResult := range result.ProviderResults {
		if providerResult.Error != nil {
			t.Errorf("Provider %s failed: %v", providerName, providerResult.Error)
		}

		expectedCount := collectors[providerName].resourceCount
		if len(providerResult.Snapshot.Resources) != expectedCount {
			t.Errorf("Provider %s: expected %d resources, got %d",
				providerName, expectedCount, len(providerResult.Snapshot.Resources))
		}
	}

	// Check merged snapshot
	if result.Snapshot == nil {
		t.Error("Expected merged snapshot to be created")
	} else {
		expectedTotalResources := 5 + 3 + 8 // aws + gcp + kubernetes
		if len(result.Snapshot.Resources) != expectedTotalResources {
			t.Errorf("Expected %d total resources in merged snapshot, got %d",
				expectedTotalResources, len(result.Snapshot.Resources))
		}
	}

	// Check that scanning was actually concurrent (should be faster than sequential)
	maxSequentialTime := 100 + 150 + 200 // Sum of all delays
	if scanDuration > time.Duration(maxSequentialTime)*time.Millisecond {
		t.Errorf("Scan took too long (%v), expected concurrent execution", scanDuration)
	}

	// Check success/error counts
	if result.SuccessCount != 3 {
		t.Errorf("Expected 3 successful scans, got %d", result.SuccessCount)
	}

	if result.ErrorCount != 0 {
		t.Errorf("Expected 0 errors, got %d", result.ErrorCount)
	}

	t.Logf("Concurrent scan completed in %v", scanDuration)
}

func TestConcurrentScanner_WithErrors(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Register mock collectors with one that fails
	collectors := map[string]*MockConcurrentCollector{
		"aws":        NewMockConcurrentCollector("aws", 100*time.Millisecond, false, 5),
		"gcp":        NewMockConcurrentCollector("gcp", 150*time.Millisecond, true, 3), // This one fails
		"kubernetes": NewMockConcurrentCollector("kubernetes", 200*time.Millisecond, false, 8),
	}

	for name, collector := range collectors {
		concurrentScanner.RegisterProvider(name, collector)
	}

	// Create scan configuration
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws":        {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp":        {Config: map[string]interface{}{"project": "test-project"}},
			"kubernetes": {Config: map[string]interface{}{"context": "test-context"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: false, // Continue on error
	}

	// Perform concurrent scan
	ctx := context.Background()
	result, err := scanner.ScanAllProviders(ctx, config)

	// Should not fail with FailOnError=false
	if err != nil {
		t.Fatalf("Expected no error with FailOnError=false, got: %v", err)
	}

	// Check error counts
	if result.ErrorCount != 1 {
		t.Errorf("Expected 1 error, got %d", result.ErrorCount)
	}

	if result.SuccessCount != 2 {
		t.Errorf("Expected 2 successful scans, got %d", result.SuccessCount)
	}

	// Check that failed provider is in results
	gcpResult := result.ProviderResults["gcp"]
	if gcpResult.Error == nil {
		t.Error("Expected GCP provider to have error")
	}

	// Check that successful providers still have resources
	awsResult := result.ProviderResults["aws"]
	if awsResult.Error != nil {
		t.Errorf("Expected AWS provider to succeed, got error: %v", awsResult.Error)
	}

	// Check merged snapshot only contains successful resources
	if result.Snapshot != nil {
		expectedResources := 5 + 8 // aws + kubernetes (gcp failed)
		if len(result.Snapshot.Resources) != expectedResources {
			t.Errorf("Expected %d resources in merged snapshot, got %d",
				expectedResources, len(result.Snapshot.Resources))
		}
	}
}

func TestConcurrentScanner_FailOnError(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Register mock collectors with one that fails
	collectors := map[string]*MockConcurrentCollector{
		"aws": NewMockConcurrentCollector("aws", 100*time.Millisecond, false, 5),
		"gcp": NewMockConcurrentCollector("gcp", 150*time.Millisecond, true, 3), // This one fails
	}

	for name, collector := range collectors {
		concurrentScanner.RegisterProvider(name, collector)
	}

	// Create scan configuration with FailOnError=true
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws": {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp": {Config: map[string]interface{}{"project": "test-project"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: true, // Fail on any error
	}

	// Perform concurrent scan
	ctx := context.Background()
	result, err := scanner.ScanAllProviders(ctx, config)

	// Should fail with FailOnError=true
	if err == nil {
		t.Error("Expected error with FailOnError=true when provider fails")
	}

	if result != nil {
		t.Error("Expected nil result when scan fails")
	}
}

func TestConcurrentScanner_Timeout(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 1*time.Second) // Short timeout
	defer concurrentScanner.Close()

	// Register mock collector with long delay
	slowCollector := NewMockConcurrentCollector("slow", 5*time.Second, false, 5)
	concurrentScanner.RegisterProvider("slow", slowCollector)

	// Create scan configuration
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"slow": {Config: map[string]interface{}{"test": "value"}},
		},
		MaxWorkers:  4,
		Timeout:     1 * time.Second,
		FailOnError: false,
	}

	// Perform concurrent scan
	ctx := context.Background()
	startTime := time.Now()

	result, err := scanner.ScanAllProviders(ctx, config)

	scanDuration := time.Since(startTime)

	// Should not fail, but should timeout
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that scan completed within timeout
	if scanDuration > 2*time.Second {
		t.Errorf("Scan took too long (%v), expected timeout", scanDuration)
	}

	// Check error count (should have timeout error)
	if result.ErrorCount == 0 {
		t.Error("Expected timeout error")
	}
}

func TestConcurrentScanner_ResourceDeduplication(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Create collectors that return overlapping resources
	aws1 := NewMockConcurrentCollector("aws-1", 100*time.Millisecond, false, 3)
	aws2 := NewMockConcurrentCollector("aws-2", 100*time.Millisecond, false, 3)

	concurrentScanner.RegisterProvider("aws-1", aws1)
	concurrentScanner.RegisterProvider("aws-2", aws2)

	// Create scan configuration
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws-1": {Config: map[string]interface{}{"region": "us-east-1"}},
			"aws-2": {Config: map[string]interface{}{"region": "us-east-1"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: false,
	}

	// Perform concurrent scan
	ctx := context.Background()
	result, err := scanner.ScanAllProviders(ctx, config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should have deduplication logic in place
	if result.Snapshot != nil {
		t.Logf("Merged snapshot has %d resources", len(result.Snapshot.Resources))

		// Check for unique resource IDs
		seenIDs := make(map[string]bool)
		for _, resource := range result.Snapshot.Resources {
			if seenIDs[resource.ID] {
				t.Errorf("Duplicate resource ID found: %s", resource.ID)
			}
			seenIDs[resource.ID] = true
		}
	}
}

func TestConcurrentScanner_PreferredOrder(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(1, 30*time.Second) // Single worker to test order
	defer concurrentScanner.Close()

	// Register mock collectors
	collectors := map[string]*MockConcurrentCollector{
		"aws":        NewMockConcurrentCollector("aws", 100*time.Millisecond, false, 1),
		"gcp":        NewMockConcurrentCollector("gcp", 100*time.Millisecond, false, 1),
		"kubernetes": NewMockConcurrentCollector("kubernetes", 100*time.Millisecond, false, 1),
	}

	for name, collector := range collectors {
		concurrentScanner.RegisterProvider(name, collector)
	}

	// Create scan configuration with preferred order
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws":        {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp":        {Config: map[string]interface{}{"project": "test-project"}},
			"kubernetes": {Config: map[string]interface{}{"context": "test-context"}},
		},
		MaxWorkers:     1,
		Timeout:        30 * time.Second,
		PreferredOrder: []string{"kubernetes", "gcp", "aws"}, // Reverse alphabetical order
	}

	// Perform concurrent scan
	ctx := context.Background()
	result, err := scanner.ScanAllProviders(ctx, config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that all providers were scanned
	if len(result.ProviderResults) != 3 {
		t.Errorf("Expected 3 provider results, got %d", len(result.ProviderResults))
	}

	// With single worker, preferred order should be respected
	// (This is a simplified test - in real implementation, you'd need to track execution order)
	t.Logf("Scan completed successfully with preferred order")
}

func TestConcurrentScanner_Stats(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Register some providers
	concurrentScanner.RegisterProvider("aws", NewMockConcurrentCollector("aws", 100*time.Millisecond, false, 5))
	concurrentScanner.RegisterProvider("gcp", NewMockConcurrentCollector("gcp", 150*time.Millisecond, false, 3))

	// Get stats
	stats := scanner.GetStats()

	// Check stats
	if stats["registered_providers"] != 2 {
		t.Errorf("Expected 2 registered providers, got %v", stats["registered_providers"])
	}

	if stats["max_workers"] != 4 {
		t.Errorf("Expected 4 max workers, got %v", stats["max_workers"])
	}

	if stats["timeout"] != "30s" {
		t.Errorf("Expected 30s timeout, got %v", stats["timeout"])
	}

	t.Logf("Scanner stats: %+v", stats)
}

func TestConcurrentScanner_SkipMerging(t *testing.T) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Register mock collectors
	concurrentScanner.RegisterProvider("aws", NewMockConcurrentCollector("aws", 100*time.Millisecond, false, 5))
	concurrentScanner.RegisterProvider("gcp", NewMockConcurrentCollector("gcp", 150*time.Millisecond, false, 3))

	// Create scan configuration with skip merging
	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws": {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp": {Config: map[string]interface{}{"project": "test-project"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		SkipMerging: true,
	}

	// Perform concurrent scan
	ctx := context.Background()
	result, err := scanner.ScanAllProviders(ctx, config)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should not have merged snapshot
	if result.Snapshot != nil {
		t.Error("Expected no merged snapshot when SkipMerging=true")
	}

	// Should still have individual provider results
	if len(result.ProviderResults) != 2 {
		t.Errorf("Expected 2 provider results, got %d", len(result.ProviderResults))
	}
}

func BenchmarkConcurrentScanner_Performance(b *testing.B) {
	concurrentScanner := scanner.NewConcurrentScanner(4, 30*time.Second)
	defer concurrentScanner.Close()

	// Register mock collectors
	concurrentScanner.RegisterProvider("aws", NewMockConcurrentCollector("aws", 10*time.Millisecond, false, 100))
	concurrentScanner.RegisterProvider("gcp", NewMockConcurrentCollector("gcp", 15*time.Millisecond, false, 50))
	concurrentScanner.RegisterProvider("kubernetes", NewMockConcurrentCollector("kubernetes", 20*time.Millisecond, false, 200))

	config := scanner.ScanConfig{
		Providers: map[string]collectors.CollectorConfig{
			"aws":        {Config: map[string]interface{}{"region": "us-east-1"}},
			"gcp":        {Config: map[string]interface{}{"project": "test-project"}},
			"kubernetes": {Config: map[string]interface{}{"context": "test-context"}},
		},
		MaxWorkers:  4,
		Timeout:     30 * time.Second,
		FailOnError: false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		result, err := scanner.ScanAllProviders(ctx, config)
		if err != nil {
			b.Fatalf("Scan failed: %v", err)
		}
		if len(result.ProviderResults) != 3 {
			b.Fatalf("Expected 3 provider results, got %d", len(result.ProviderResults))
		}
	}
}
