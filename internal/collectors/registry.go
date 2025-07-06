package collectors

import (
	"fmt"
	"sync"

	"github.com/yairfalse/wgo/pkg/types"
)

type Collector interface {
	Name() string
	Collect() ([]types.Resource, error)
	Health() error
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

func (r *CollectorRegistry) Register(collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[collector.Name()] = collector
}

func (r *CollectorRegistry) Get(name string) (Collector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	collector, exists := r.collectors[name]
	if !exists {
		return nil, fmt.Errorf("collector %s not found", name)
	}
	
	return collector, nil
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

func (r *CollectorRegistry) CollectAll() ([]types.Resource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var allResources []types.Resource
	
	for name, collector := range r.collectors {
		resources, err := collector.Collect()
		if err != nil {
			return nil, fmt.Errorf("collector %s failed: %w", name, err)
		}
		allResources = append(allResources, resources...)
	}
	
	return allResources, nil
}

// Default registry instance
var defaultRegistry = NewRegistry()

func DefaultRegistry() *CollectorRegistry {
	return defaultRegistry
}