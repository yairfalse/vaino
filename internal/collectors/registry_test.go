package collectors

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/yairfalse/wgo/pkg/types"
)

// Mock collector for testing
type mockCollector struct {
	name      string
	resources []types.Resource
	err       error
	healthErr error
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) Collect() ([]types.Resource, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resources, nil
}

func (m *mockCollector) Health() error {
	return m.healthErr
}

func TestCollectorRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	collector := &mockCollector{name: "test-collector"}

	registry.Register(collector)

	// Verify collector was registered
	retrieved, err := registry.Get("test-collector")
	if err != nil {
		t.Fatalf("Failed to get registered collector: %v", err)
	}

	if retrieved.Name() != "test-collector" {
		t.Errorf("Expected collector name 'test-collector', got %s", retrieved.Name())
	}
}

func TestCollectorRegistry_GetNonExistent(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent collector")
	}

	if err.Error() != "collector nonexistent not found" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestCollectorRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("Expected empty list, got %d items", len(names))
	}

	// Add collectors
	collectors := []*mockCollector{
		{name: "aws"},
		{name: "kubernetes"},
		{name: "terraform"},
	}

	for _, collector := range collectors {
		registry.Register(collector)
	}

	names = registry.List()
	if len(names) != 3 {
		t.Errorf("Expected 3 collectors, got %d", len(names))
	}

	// Check all names are present
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	expectedNames := []string{"aws", "kubernetes", "terraform"}
	for _, expected := range expectedNames {
		if !nameSet[expected] {
			t.Errorf("Expected collector %s not found in list", expected)
		}
	}
}

func TestCollectorRegistry_CollectAll(t *testing.T) {
	registry := NewRegistry()

	// Create collectors with different resources
	awsCollector := &mockCollector{
		name: "aws",
		resources: []types.Resource{
			{ID: "i-123", Type: "ec2:instance", Provider: "aws"},
			{ID: "vol-456", Type: "ec2:volume", Provider: "aws"},
		},
	}

	k8sCollector := &mockCollector{
		name: "kubernetes",
		resources: []types.Resource{
			{ID: "pod-789", Type: "pod", Provider: "kubernetes"},
		},
	}

	registry.Register(awsCollector)
	registry.Register(k8sCollector)

	// Collect all resources
	resources, err := registry.CollectAll()
	if err != nil {
		t.Fatalf("CollectAll failed: %v", err)
	}

	if len(resources) != 3 {
		t.Errorf("Expected 3 total resources, got %d", len(resources))
	}

	// Verify we got resources from both collectors
	awsCount := 0
	k8sCount := 0
	for _, resource := range resources {
		switch resource.Provider {
		case "aws":
			awsCount++
		case "kubernetes":
			k8sCount++
		}
	}

	if awsCount != 2 {
		t.Errorf("Expected 2 AWS resources, got %d", awsCount)
	}
	if k8sCount != 1 {
		t.Errorf("Expected 1 Kubernetes resource, got %d", k8sCount)
	}
}

func TestCollectorRegistry_CollectAllWithError(t *testing.T) {
	registry := NewRegistry()

	// Create a failing collector
	failingCollector := &mockCollector{
		name: "failing",
		err:  errors.New("collection failed"),
	}

	workingCollector := &mockCollector{
		name:      "working",
		resources: []types.Resource{{ID: "test", Provider: "working"}},
	}

	registry.Register(failingCollector)
	registry.Register(workingCollector)

	// CollectAll should fail if any collector fails
	_, err := registry.CollectAll()
	if err == nil {
		t.Error("Expected CollectAll to fail when a collector fails")
	}

	if err.Error() != "collector failing failed: collection failed" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestCollectorRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	
	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent registration
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			collector := &mockCollector{
				name:      fmt.Sprintf("collector-%d", id),
				resources: []types.Resource{},
			}
			registry.Register(collector)
		}(i)
	}
	wg.Wait()

	// Verify all collectors were registered
	names := registry.List()
	if len(names) != numGoroutines {
		t.Errorf("Expected %d collectors, got %d", numGoroutines, len(names))
	}

	// Concurrent access
	wg.Add(numGoroutines * 2)
	
	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			_, err := registry.Get(fmt.Sprintf("collector-%d", id))
			if err != nil {
				t.Errorf("Failed to get collector-%d: %v", id, err)
			}
		}(i)
	}

	// List operations
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			registry.List()
		}()
	}

	wg.Wait()
}

func TestDefaultRegistry(t *testing.T) {
	// Test that default registry is accessible
	defaultReg := DefaultRegistry()
	if defaultReg == nil {
		t.Error("Default registry should not be nil")
	}

	// Test that it works like a normal registry
	collector := &mockCollector{name: "default-test"}
	defaultReg.Register(collector)

	retrieved, err := defaultReg.Get("default-test")
	if err != nil {
		t.Errorf("Failed to get collector from default registry: %v", err)
	}

	if retrieved.Name() != "default-test" {
		t.Errorf("Expected 'default-test', got %s", retrieved.Name())
	}
}