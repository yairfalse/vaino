package collectors

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/yairfalse/wgo/pkg/types"
)

// mockCollector implements the Collector interface for testing
type mockCollector struct {
	name string
}

func (m *mockCollector) Name() string {
	return m.name
}

func (m *mockCollector) Collect(ctx context.Context, config Config) (*types.Snapshot, error) {
	return &types.Snapshot{}, nil
}

func (m *mockCollector) Validate(config Config) error {
	return nil
}

func TestNewCollectorRegistry(t *testing.T) {
	registry := NewCollectorRegistry()
	if registry == nil {
		t.Fatal("NewCollectorRegistry() returned nil")
	}
	if registry.collectors == nil {
		t.Error("Expected collectors map to be initialized")
	}
}

func TestCollectorRegistry_Register(t *testing.T) {
	registry := NewCollectorRegistry()

	tests := []struct {
		collector Collector
		name      string
		wantErr   bool
	}{
		{
			name:      "valid collector",
			collector: &mockCollector{name: "test-collector"},
			wantErr:   false,
		},
		{
			name:      "nil collector",
			collector: nil,
			wantErr:   true,
		},
		{
			name:      "collector with empty name",
			collector: &mockCollector{name: ""},
			wantErr:   true,
		},
		{
			name:      "collector with whitespace name",
			collector: &mockCollector{name: "   "},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.collector)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Test duplicate registration
	collector := &mockCollector{name: "duplicate-test"}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("First registration should succeed: %v", err)
	}

	err = registry.Register(collector)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestCollectorRegistry_Get(t *testing.T) {
	registry := NewCollectorRegistry()
	testCollector := &mockCollector{name: "test-collector"}
	_ = registry.Register(testCollector) // Error intentionally ignored for test setup

	tests := []struct {
		name          string
		collectorName string
		wantErr       bool
		wantNil       bool
	}{
		{
			name:          "existing collector",
			collectorName: "test-collector",
			wantErr:       false,
			wantNil:       false,
		},
		{
			name:          "non-existing collector",
			collectorName: "non-existing",
			wantErr:       true,
			wantNil:       true,
		},
		{
			name:          "empty name",
			collectorName: "",
			wantErr:       true,
			wantNil:       true,
		},
		{
			name:          "whitespace name",
			collectorName: "   ",
			wantErr:       true,
			wantNil:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := registry.Get(tt.collectorName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (collector == nil) != tt.wantNil {
				t.Errorf("Get() collector = %v, wantNil %v", collector, tt.wantNil)
			}
			if !tt.wantErr && collector.Name() != tt.collectorName {
				t.Errorf("Get() returned collector with name %v, want %v", collector.Name(), tt.collectorName)
			}
		})
	}
}

func TestCollectorRegistry_List(t *testing.T) {
	registry := NewCollectorRegistry()

	// Initially empty
	list := registry.List()
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %v", list)
	}

	// Add some collectors
	collectors := []*mockCollector{
		{name: "collector-c"},
		{name: "collector-a"},
		{name: "collector-b"},
	}

	for _, collector := range collectors {
		_ = registry.Register(collector) // Error intentionally ignored for test setup
	}

	list = registry.List()
	expected := []string{"collector-a", "collector-b", "collector-c"}

	if !reflect.DeepEqual(list, expected) {
		t.Errorf("List() = %v, want %v", list, expected)
	}

	// Verify the list is sorted
	if !sort.StringsAreSorted(list) {
		t.Error("Expected list to be sorted")
	}
}

func TestCollectorRegistry_Count(t *testing.T) {
	registry := NewCollectorRegistry()

	// Initially empty
	if registry.Count() != 0 {
		t.Errorf("Expected count 0, got %d", registry.Count())
	}

	// Add collectors
	for i := 0; i < 3; i++ {
		collector := &mockCollector{name: fmt.Sprintf("collector-%d", i)}
		_ = registry.Register(collector) // Error intentionally ignored for test setup
		if registry.Count() != i+1 {
			t.Errorf("Expected count %d, got %d", i+1, registry.Count())
		}
	}
}

func TestCollectorRegistry_Exists(t *testing.T) {
	registry := NewCollectorRegistry()
	testCollector := &mockCollector{name: "test-collector"}
	_ = registry.Register(testCollector) // Error intentionally ignored for test setup

	tests := []struct {
		name          string
		collectorName string
		expected      bool
	}{
		{
			name:          "existing collector",
			collectorName: "test-collector",
			expected:      true,
		},
		{
			name:          "non-existing collector",
			collectorName: "non-existing",
			expected:      false,
		},
		{
			name:          "empty name",
			collectorName: "",
			expected:      false,
		},
		{
			name:          "whitespace name",
			collectorName: "   ",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.Exists(tt.collectorName)
			if result != tt.expected {
				t.Errorf("Exists() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCollectorRegistry_Unregister(t *testing.T) {
	registry := NewCollectorRegistry()
	testCollector := &mockCollector{name: "test-collector"}
	_ = registry.Register(testCollector) // Error intentionally ignored for test setup

	tests := []struct {
		name          string
		collectorName string
		wantErr       bool
	}{
		{
			name:          "existing collector",
			collectorName: "test-collector",
			wantErr:       false,
		},
		{
			name:          "non-existing collector",
			collectorName: "non-existing",
			wantErr:       true,
		},
		{
			name:          "empty name",
			collectorName: "",
			wantErr:       true,
		},
		{
			name:          "whitespace name",
			collectorName: "   ",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Re-register the collector for each test
			if tt.collectorName == "test-collector" {
				_ = registry.Register(testCollector) // Error intentionally ignored for test setup
			}

			err := registry.Unregister(tt.collectorName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unregister() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify the collector is actually removed
			if !tt.wantErr && registry.Exists(tt.collectorName) {
				t.Error("Expected collector to be removed after unregistration")
			}
		})
	}
}

func TestCollectorRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewCollectorRegistry()

	// Test concurrent registration and access
	done := make(chan bool, 10)

	// Start multiple goroutines registering collectors
	for i := 0; i < 5; i++ {
		go func(id int) {
			collector := &mockCollector{name: fmt.Sprintf("concurrent-collector-%d", id)}
			_ = registry.Register(collector) // Error intentionally ignored for test setup
			done <- true
		}(i)
	}

	// Start multiple goroutines accessing collectors
	for i := 0; i < 5; i++ {
		go func(id int) {
			registry.List()
			registry.Count()
			registry.Exists(fmt.Sprintf("concurrent-collector-%d", id))
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	if registry.Count() != 5 {
		t.Errorf("Expected 5 registered collectors, got %d", registry.Count())
	}
}

func TestDefaultRegistry(t *testing.T) {
	if DefaultRegistry == nil {
		t.Error("Expected DefaultRegistry to be initialized")
	}

	// The default registry should have some collectors registered via init
	// We can't test the exact count since it depends on which packages are imported
	// but we can test that it's a valid registry
	list := DefaultRegistry.List()
	if list == nil {
		t.Error("Expected DefaultRegistry.List() to return a non-nil slice")
	}
}
