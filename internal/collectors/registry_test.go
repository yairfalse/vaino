package collectors

import (
	"context"
	"testing"

	"github.com/yairfalse/vaino/pkg/types"
)

// Use the MockCollector from registry.go

// MockEnhancedCollector for testing
type MockEnhancedCollector struct {
	name         string
	status       string
	collectFunc  func(ctx context.Context, config CollectorConfig) (*types.Snapshot, error)
	validateFunc func(config CollectorConfig) error
}

func (m *MockEnhancedCollector) Name() string {
	return m.name
}

func (m *MockEnhancedCollector) Status() string {
	return m.status
}

func (m *MockEnhancedCollector) Collect(ctx context.Context, config CollectorConfig) (*types.Snapshot, error) {
	if m.collectFunc != nil {
		return m.collectFunc(ctx, config)
	}
	return &types.Snapshot{
		ID:       "mock-snapshot",
		Provider: m.name,
	}, nil
}

func (m *MockEnhancedCollector) Validate(config CollectorConfig) error {
	if m.validateFunc != nil {
		return m.validateFunc(config)
	}
	return nil
}

func (m *MockEnhancedCollector) AutoDiscover() (CollectorConfig, error) {
	return CollectorConfig{}, nil
}

func (m *MockEnhancedCollector) SupportedRegions() []string {
	return []string{"us-east-1", "us-west-2"}
}

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

func TestEnhancedRegistry_RegisterEnhanced(t *testing.T) {
	registry := NewEnhancedRegistry()

	collector := &MockEnhancedCollector{
		name:   "enhanced-test",
		status: "ready",
	}

	registry.RegisterEnhanced(collector)

	// Test enhanced retrieval
	retrieved, err := registry.GetEnhanced("enhanced-test")
	if err != nil {
		t.Fatalf("failed to get enhanced collector: %v", err)
	}

	if retrieved.Name() != "enhanced-test" {
		t.Errorf("expected name 'enhanced-test', got '%s'", retrieved.Name())
	}

	// Enhanced collectors are not automatically available as legacy collectors
	_, err = registry.GetLegacy("enhanced-test")
	if err == nil {
		t.Error("expected error getting enhanced collector as legacy")
	}
}

func TestEnhancedRegistry_List(t *testing.T) {
	registry := NewEnhancedRegistry()

	// Add enhanced collector
	enhanced := &MockEnhancedCollector{
		name:   "enhanced1",
		status: "ready",
	}
	registry.RegisterEnhanced(enhanced)

	// Add legacy collector
	legacy := NewMockCollector("legacy1", "ready")
	registry.RegisterLegacy(legacy)

	// Test enhanced list
	enhancedNames := registry.ListEnhanced()
	if len(enhancedNames) != 1 || enhancedNames[0] != "enhanced1" {
		t.Errorf("expected [enhanced1], got %v", enhancedNames)
	}

	// Test legacy list
	legacyNames := registry.ListLegacy()
	if len(legacyNames) != 1 || legacyNames[0] != "legacy1" {
		t.Errorf("expected [legacy1], got %v", legacyNames)
	}

	// Test all list
	allNames := registry.ListAll()
	if len(allNames) != 2 {
		t.Errorf("expected 2 total collectors, got %d", len(allNames))
	}

	nameMap := make(map[string]bool)
	for _, name := range allNames {
		nameMap[name] = true
	}

	if !nameMap["enhanced1"] || !nameMap["legacy1"] {
		t.Error("expected both enhanced1 and legacy1 in all list")
	}
}

func TestEnhancedRegistry_GetNonExistent(t *testing.T) {
	registry := NewEnhancedRegistry()

	_, err := registry.GetEnhanced("non-existent")
	if err == nil {
		t.Error("expected error getting non-existent enhanced collector")
	}

	_, err = registry.GetLegacy("non-existent")
	if err == nil {
		t.Error("expected error getting non-existent legacy collector")
	}
}

func TestEnhancedRegistry_GetSupportedProviders(t *testing.T) {
	registry := NewEnhancedRegistry()

	// Initially empty
	providers := registry.GetSupportedProviders()
	if len(providers) != 0 {
		t.Errorf("expected empty providers list, got %d", len(providers))
	}

	// Add collectors
	registry.RegisterEnhanced(&MockEnhancedCollector{
		name:   "terraform",
		status: "ready",
	})
	registry.RegisterLegacy(NewMockCollector("aws", "ready"))

	providers = registry.GetSupportedProviders()
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}

	providerMap := make(map[string]bool)
	for _, provider := range providers {
		providerMap[provider] = true
	}

	if !providerMap["terraform"] || !providerMap["aws"] {
		t.Error("expected both terraform and aws in supported providers")
	}
}

func TestEnhancedRegistry_GetEnhancedProviders(t *testing.T) {
	registry := NewEnhancedRegistry()

	// Add enhanced and legacy collectors
	registry.RegisterEnhanced(&MockEnhancedCollector{
		name:   "terraform",
		status: "ready",
	})
	registry.RegisterLegacy(NewMockCollector("aws", "ready"))

	enhancedProviders := registry.GetEnhancedProviders()
	if len(enhancedProviders) != 1 || enhancedProviders[0] != "terraform" {
		t.Errorf("expected [terraform], got %v", enhancedProviders)
	}
}

func TestEnhancedRegistry_GetStatus(t *testing.T) {
	registry := NewEnhancedRegistry()

	// Add collectors with different statuses
	registry.RegisterEnhanced(&MockEnhancedCollector{
		name:   "terraform",
		status: "ready",
	})
	registry.RegisterLegacy(NewMockCollector("aws", "error"))

	statuses := registry.GetStatus()
	if len(statuses) != 2 {
		t.Errorf("expected 2 status entries, got %d", len(statuses))
	}

	if statuses["terraform"] != "ready" {
		t.Errorf("expected terraform status 'ready', got '%s'", statuses["terraform"])
	}

	if statuses["aws"] != "error" {
		t.Errorf("expected aws status 'error', got '%s'", statuses["aws"])
	}
}

func TestDefaultEnhancedRegistry(t *testing.T) {
	// Reset default enhanced registry for testing
	defaultEnhancedRegistry = nil

	registry1 := DefaultEnhancedRegistry()
	registry2 := DefaultEnhancedRegistry()

	// Should return the same instance
	if registry1 != registry2 {
		t.Error("DefaultEnhancedRegistry should return the same instance")
	}

	// Test that it works
	collector := &MockEnhancedCollector{
		name:   "default-enhanced-test",
		status: "ready",
	}
	registry1.RegisterEnhanced(collector)

	retrieved, err := registry2.GetEnhanced("default-enhanced-test")
	if err != nil {
		t.Fatalf("failed to get collector: %v", err)
	}

	if retrieved.Name() != "default-enhanced-test" {
		t.Error("expected same collector through both references")
	}
}
