package collectors

import (
	"fmt"
	"sync"
	"testing"
)

// Mock collector for testing
type mockCollector struct {
	name   string
	status string
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) Status() string {
	if m.status == "" {
		return "ready"
	}
	return m.status
}

func TestCollectorRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	collector := &mockCollector{name: "test-collector"}

	registry.Register(collector)

	// Verify collector was registered
	retrieved, exists := registry.Get("test-collector")
	if !exists {
		t.Fatalf("Failed to get registered collector")
	}

	if retrieved.Name() != "test-collector" {
		t.Errorf("Expected collector name 'test-collector', got %s", retrieved.Name())
	}
}

func TestCollectorRegistry_GetNonExistent(t *testing.T) {
	registry := NewRegistry()

	_, exists := registry.Get("nonexistent")
	if exists {
		t.Error("Expected false for nonexistent collector")
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
				name: fmt.Sprintf("collector-%d", id),
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
			_, exists := registry.Get(fmt.Sprintf("collector-%d", id))
			if !exists {
				t.Errorf("Failed to get collector-%d", id)
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
	// Just test that we can create a registry
	registry := NewRegistry()
	if registry == nil {
		t.Error("Registry should not be nil")
	}

	// Test that it works like a normal registry
	collector := &mockCollector{name: "default-test"}
	registry.Register(collector)

	retrieved, exists := registry.Get("default-test")
	if !exists {
		t.Error("Failed to get collector from registry")
	}

	if retrieved.Name() != "default-test" {
		t.Errorf("Expected 'default-test', got %s", retrieved.Name())
	}
}