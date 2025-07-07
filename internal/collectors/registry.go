package collectors

import (
	"sync"
)

type Collector interface {
	Name() string
	Status() string
}

type CollectorRegistry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

func NewRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make(map[string]Collector),
	}
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