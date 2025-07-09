package collectors

import (
	"context"
	"fmt"
	"sync"

	"github.com/yairfalse/wgo/pkg/types"
)

// EnhancedRegistry manages enhanced collectors that support full collection
type EnhancedRegistry struct {
	mu               sync.RWMutex
	collectors       map[string]EnhancedCollector
	legacyCollectors map[string]Collector
}

// NewEnhancedRegistry creates a new enhanced collector registry
func NewEnhancedRegistry() *EnhancedRegistry {
	return &EnhancedRegistry{
		collectors:       make(map[string]EnhancedCollector),
		legacyCollectors: make(map[string]Collector),
	}
}

// RegisterEnhanced registers an enhanced collector
func (r *EnhancedRegistry) RegisterEnhanced(collector EnhancedCollector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[collector.Name()] = collector
}

// RegisterLegacy registers a legacy collector (basic interface)
func (r *EnhancedRegistry) RegisterLegacy(collector Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.legacyCollectors[collector.Name()] = collector
}

// GetEnhanced returns an enhanced collector by name
func (r *EnhancedRegistry) GetEnhanced(name string) (EnhancedCollector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.collectors[name]
	if !exists {
		return nil, fmt.Errorf("enhanced collector %s not found", name)
	}

	return collector, nil
}

// GetLegacy returns a legacy collector by name
func (r *EnhancedRegistry) GetLegacy(name string) (Collector, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collector, exists := r.legacyCollectors[name]
	if !exists {
		return nil, fmt.Errorf("legacy collector %s not found", name)
	}

	return collector, nil
}

// ListEnhanced returns all enhanced collector names
func (r *EnhancedRegistry) ListEnhanced() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}

	return names
}

// ListLegacy returns all legacy collector names
func (r *EnhancedRegistry) ListLegacy() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.legacyCollectors))
	for name := range r.legacyCollectors {
		names = append(names, name)
	}

	return names
}

// ListAll returns all collector names (both enhanced and legacy)
func (r *EnhancedRegistry) ListAll() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.collectors)+len(r.legacyCollectors))

	for name := range r.collectors {
		names = append(names, name)
	}

	for name := range r.legacyCollectors {
		names = append(names, name)
	}

	return names
}

// CollectFromProvider performs collection using the specified provider
func (r *EnhancedRegistry) CollectFromProvider(ctx context.Context, providerName string, config CollectorConfig) (*types.Snapshot, error) {
	// Try enhanced collectors first
	if collector, err := r.GetEnhanced(providerName); err == nil {
		return collector.Collect(ctx, config)
	}

	// Enhanced collector not found
	return nil, fmt.Errorf("enhanced collector %s not found - only enhanced collectors support collection", providerName)
}

// ValidateConfig validates configuration for a specific provider
func (r *EnhancedRegistry) ValidateConfig(providerName string, config CollectorConfig) error {
	collector, err := r.GetEnhanced(providerName)
	if err != nil {
		return fmt.Errorf("enhanced collector %s not found", providerName)
	}

	return collector.Validate(config)
}

// AutoDiscover performs auto-discovery for a specific provider
func (r *EnhancedRegistry) AutoDiscover(providerName string) (CollectorConfig, error) {
	collector, err := r.GetEnhanced(providerName)
	if err != nil {
		return CollectorConfig{}, fmt.Errorf("enhanced collector %s not found", providerName)
	}

	return collector.AutoDiscover()
}

// GetCollectorInfo returns information about a collector
func (r *EnhancedRegistry) GetCollectorInfo(providerName string) (CollectorInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check enhanced collectors
	if collector, exists := r.collectors[providerName]; exists {
		return CollectorInfo{
			Name:        collector.Name(),
			Provider:    collector.Name(),
			Version:     "1.0.0", // Would be better to get from collector
			Description: fmt.Sprintf("Enhanced %s collector", collector.Name()),
			Features:    []string{"collection", "validation", "auto-discovery"},
			Status:      collector.Status(),
		}, nil
	}

	// Check legacy collectors
	if collector, exists := r.legacyCollectors[providerName]; exists {
		return CollectorInfo{
			Name:        collector.Name(),
			Provider:    collector.Name(),
			Version:     "1.0.0",
			Description: fmt.Sprintf("Legacy %s collector", collector.Name()),
			Features:    []string{"status"},
			Status:      collector.Status(),
		}, nil
	}

	return CollectorInfo{}, fmt.Errorf("collector %s not found", providerName)
}

// GetSupportedProviders returns all supported provider names
func (r *EnhancedRegistry) GetSupportedProviders() []string {
	return r.ListAll()
}

// GetEnhancedProviders returns only enhanced provider names
func (r *EnhancedRegistry) GetEnhancedProviders() []string {
	return r.ListEnhanced()
}

// IsEnhanced checks if a provider has enhanced collection capabilities
func (r *EnhancedRegistry) IsEnhanced(providerName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.collectors[providerName]
	return exists
}

// GetStatus returns the status of all collectors
func (r *EnhancedRegistry) GetStatus() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]string)

	for name, collector := range r.collectors {
		status[name] = collector.Status()
	}

	for name, collector := range r.legacyCollectors {
		status[name] = collector.Status()
	}

	return status
}

// DefaultEnhancedRegistry is the global enhanced registry instance
var defaultEnhancedRegistry *EnhancedRegistry

// DefaultEnhancedRegistry returns the default enhanced registry
func DefaultEnhancedRegistry() *EnhancedRegistry {
	if defaultEnhancedRegistry == nil {
		defaultEnhancedRegistry = NewEnhancedRegistry()
	}
	return defaultEnhancedRegistry
}
