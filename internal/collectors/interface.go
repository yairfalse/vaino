package collectors

import (
	"context"

	"github.com/yairfalse/wgo/pkg/types"
)

// Config represents the configuration for a collector
type Config struct {
	Options  map[string]interface{} `json:"options,omitempty"`
	Provider string                 `json:"provider"`
	Region   string                 `json:"region"`
	Paths    []string               `json:"paths,omitempty"`
}

// Collector defines the interface that all resource collectors must implement
type Collector interface {
	// Name returns the name of the collector
	Name() string

	// Collect gathers resources from the specified configuration and returns a snapshot
	Collect(ctx context.Context, config Config) (*types.Snapshot, error)

	// Validate checks if the provided configuration is valid for this collector
	Validate(config Config) error
}
