package collectors

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// CollectorRegistry manages the registration and retrieval of collectors
type CollectorRegistry struct {
	collectors map[string]Collector
	mu         sync.RWMutex
}

// NewCollectorRegistry creates a new collector registry
func NewCollectorRegistry() *CollectorRegistry {
	registry := &CollectorRegistry{
		collectors: make(map[string]Collector),
	}

	return registry
}

// Register adds a collector to the registry
func (r *CollectorRegistry) Register(collector Collector) error {
	if collector == nil {
		return errors.New("collector cannot be nil")
	}

	name := strings.TrimSpace(collector.Name())
	if name == "" {
		return errors.New("collector name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collectors[name]; exists {
		return fmt.Errorf("collector with name '%s' already registered", name)
	}

	r.collectors[name] = collector
	return nil
}

// Get retrieves a collector by name
func (r *CollectorRegistry) Get(name string) (Collector, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("collector name cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.collectors[name]
	if !exists {
		return nil, fmt.Errorf("collector with name '%s' not found", name)
	}

	return collector, nil
}

// List returns a sorted list of all registered collector names
func (r *CollectorRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Count returns the number of registered collectors
func (r *CollectorRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.collectors)
}

// Exists checks if a collector with the given name is registered
func (r *CollectorRegistry) Exists(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.collectors[name]
	return exists
}

// Unregister removes a collector from the registry
func (r *CollectorRegistry) Unregister(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("collector name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.collectors[name]; !exists {
		return fmt.Errorf("collector with name '%s' not found", name)
	}

	delete(r.collectors, name)
	return nil
}

// DefaultRegistry is the default global collector registry
var DefaultRegistry = NewCollectorRegistry()
