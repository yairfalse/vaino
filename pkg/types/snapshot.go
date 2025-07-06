package types

import (
	"fmt"
	"time"
)

// Snapshot represents a point-in-time capture of infrastructure resources
type Snapshot struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Provider  string            `json:"provider"`
	Region    string            `json:"region"`
	Resources []Resource        `json:"resources"`
	Metadata  map[string]string `json:"metadata"`
}

// Validate checks if the Snapshot has all required fields
func (s *Snapshot) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("snapshot ID cannot be empty")
	}
	if s.Timestamp.IsZero() {
		return fmt.Errorf("snapshot timestamp cannot be zero")
	}
	if s.Provider == "" {
		return fmt.Errorf("snapshot provider cannot be empty")
	}
	
	// Validate all resources
	for i, resource := range s.Resources {
		if err := resource.Validate(); err != nil {
			return fmt.Errorf("invalid resource at index %d: %w", i, err)
		}
	}
	
	return nil
}

// GetMetadata retrieves a metadata value by key
func (s *Snapshot) GetMetadata(key string) (string, bool) {
	if s.Metadata == nil {
		return "", false
	}
	val, exists := s.Metadata[key]
	return val, exists
}

// ResourceCount returns the number of resources in the snapshot
func (s *Snapshot) ResourceCount() int {
	return len(s.Resources)
}

// ResourcesByType returns all resources of a specific type
func (s *Snapshot) ResourcesByType(resourceType string) []Resource {
	var filtered []Resource
	for _, r := range s.Resources {
		if r.Type == resourceType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ResourcesByProvider returns all resources from a specific provider
func (s *Snapshot) ResourcesByProvider(provider string) []Resource {
	var filtered []Resource
	for _, r := range s.Resources {
		if r.Provider == provider {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// FindResourceByID finds a resource by its ID
func (s *Snapshot) FindResourceByID(id string) (*Resource, bool) {
	for _, r := range s.Resources {
		if r.ID == id {
			return &r, true
		}
	}
	return nil, false
}