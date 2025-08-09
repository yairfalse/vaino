package collectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/yairfalse/vaino/pkg/types"
)

type CollectorRegistry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

func NewRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]Collector),
	}
}

func NewCollectorRegistry() *CollectorRegistry {
	return NewRegistry()
}

// defaultRegistry is the global registry instance
var defaultRegistry *CollectorRegistry

// DefaultRegistry returns the default collector registry
func DefaultRegistry() *CollectorRegistry {
	if defaultRegistry == nil {
		defaultRegistry = NewRegistry()
	}
	return defaultRegistry
}

func (r *CollectorRegistry) Register(collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[collector.Name()] = collector
}

func (r *CollectorRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}

	return names
}

func (r *CollectorRegistry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.collectors[name]
	return collector, exists
}

type MockCollector struct {
	name   string
	status string
}

func NewMockCollector(name, status string) Collector {
	return &MockCollector{
		name:   name,
		status: status,
	}
}

func (c *MockCollector) Name() string {
	return c.name
}

func (c *MockCollector) Status() string {
	return c.status
}

func (c *MockCollector) Collect(ctx context.Context, config CollectorConfig) (*types.Snapshot, error) {
	return &types.Snapshot{}, nil
}

func (c *MockCollector) Validate(config CollectorConfig) error {
	return nil
}

func (c *MockCollector) AutoDiscover() (CollectorConfig, error) {
	return CollectorConfig{}, nil
}

func (c *MockCollector) SupportedRegions() []string {
	return []string{}
}

func (c *MockCollector) CollectSeparate(ctx context.Context, config CollectorConfig) ([]*types.Snapshot, error) {
	return nil, fmt.Errorf("separate collection not supported by mock collector")
}

// Enhanced registry compatibility methods
func (r *CollectorRegistry) ListEnhanced() []string {
	return r.List()
}

func (r *CollectorRegistry) ListLegacy() []string {
	return []string{} // All collectors are enhanced now
}

func (r *CollectorRegistry) GetEnhanced(name string) (Collector, error) {
	collector, exists := r.Get(name)
	if !exists {
		return nil, fmt.Errorf("collector %s not found", name)
	}
	return collector, nil
}

func (r *CollectorRegistry) IsEnhanced(name string) bool {
	_, exists := r.Get(name)
	return exists
}
