package collectors

import (
	"testing"
)

// Use the MockCollector from registry.go

func TestCollectorRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	collector := NewMockCollector("test-collector", "ready")

	registry.Register(collector)

	// Test retrieval
	retrieved, exists := registry.Get("test-collector")
	if !exists {
		t.Error("expected collector to exist after registration")
	}

	if retrieved.Name() != "test-collector" {
		t.Errorf("expected name 'test-collector', got '%s'", retrieved.Name())
	}

	if retrieved.Status() != "ready" {
		t.Errorf("expected status 'ready', got '%s'", retrieved.Status())
	}
}

func TestCollectorRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Initially empty
	names := registry.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %d items", len(names))
	}

	// Add collectors
	registry.Register(NewMockCollector("collector1", "ready"))
	registry.Register(NewMockCollector("collector2", "error"))

	names = registry.List()
	if len(names) != 2 {
		t.Errorf("expected 2 collectors, got %d", len(names))
	}

	// Check that both names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["collector1"] || !nameMap["collector2"] {
		t.Error("expected both collector1 and collector2 to be listed")
	}
}

func TestCollectorRegistry_GetNonExistent(t *testing.T) {
	registry := NewRegistry()

	_, exists := registry.Get("non-existent")
	if exists {
		t.Error("expected non-existent collector to not exist")
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Reset default registry for testing
	defaultRegistry = nil

	registry1 := DefaultRegistry()
	registry2 := DefaultRegistry()

	// Should return the same instance
	if registry1 != registry2 {
		t.Error("DefaultRegistry should return the same instance")
	}

	// Test that it works
	collector := NewMockCollector("default-test", "ready")
	registry1.Register(collector)

	retrieved, exists := registry2.Get("default-test")
	if !exists {
		t.Error("expected collector to be accessible through both references")
	}

	if retrieved.Name() != "default-test" {
		t.Error("expected same collector through both references")
	}
}
