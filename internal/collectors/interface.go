package collectors

import (
	"context"

	"github.com/yairfalse/wgo/pkg/types"
)

// CollectorConfig holds configuration for a collector
type CollectorConfig struct {
	// Provider-specific configuration
	Config map[string]interface{} `json:"config"`
	
	// Common options
	Regions    []string `json:"regions,omitempty"`
	Namespaces []string `json:"namespaces,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	
	// File paths for file-based collectors
	StatePaths []string `json:"state_paths,omitempty"`
	
	// Timeout settings
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// EnhancedCollector defines the interface for infrastructure collectors
// This extends the basic Collector interface with collection capabilities
type EnhancedCollector interface {
	// Basic interface methods
	Name() string
	Status() string
	
	// Enhanced collection methods
	Collect(ctx context.Context, config CollectorConfig) (*types.Snapshot, error)
	Validate(config CollectorConfig) error
	
	// Discovery methods
	AutoDiscover() (CollectorConfig, error)
	SupportedRegions() []string
}

// CollectorInfo provides metadata about a collector
type CollectorInfo struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Status      string   `json:"status"`
}