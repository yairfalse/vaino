package types

import (
	"fmt"
	"time"
)

// Snapshot represents a point-in-time capture of infrastructure state
type Snapshot struct {
	ID        string             `json:"id"`
	Timestamp time.Time          `json:"timestamp"`
	Provider  string             `json:"provider"`
	Resources []Resource         `json:"resources"`
	Metadata  SnapshotMetadata   `json:"metadata"`
}

// SnapshotMetadata contains metadata about the snapshot collection process
type SnapshotMetadata struct {
	CollectorVersion string        `json:"collector_version"`
	CollectionTime   time.Duration `json:"collection_time"`
	ResourceCount    int           `json:"resource_count"`
	Regions          []string      `json:"regions,omitempty"`
	Namespaces       []string      `json:"namespaces,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}

// Validate checks if the snapshot has all required fields
func (s Snapshot) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("snapshot ID is required")
	}
	if s.Provider == "" {
		return fmt.Errorf("snapshot provider is required")
	}
	if s.Timestamp.IsZero() {
		return fmt.Errorf("snapshot timestamp is required")
	}
	
	for i, resource := range s.Resources {
		if err := resource.Validate(); err != nil {
			return fmt.Errorf("resource %d invalid: %w", i, err)
		}
	}
	
	return s.Metadata.Validate()
}

// ResourceCount returns the number of resources in the snapshot
func (s Snapshot) ResourceCount() int {
	return len(s.Resources)
}

// ResourcesByProvider groups resources by provider
func (s Snapshot) ResourcesByProvider() map[string][]Resource {
	byProvider := make(map[string][]Resource)
	for _, resource := range s.Resources {
		byProvider[resource.Provider] = append(byProvider[resource.Provider], resource)
	}
	return byProvider
}

// ResourcesByType groups resources by type
func (s Snapshot) ResourcesByType() map[string][]Resource {
	byType := make(map[string][]Resource)
	for _, resource := range s.Resources {
		byType[resource.Type] = append(byType[resource.Type], resource)
	}
	return byType
}

// FindResource finds a resource by ID
func (s Snapshot) FindResource(id string) (Resource, bool) {
	for _, resource := range s.Resources {
		if resource.ID == id {
			return resource, true
		}
	}
	return Resource{}, false
}

// Baseline represents a known good state for comparison
type Baseline struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	SnapshotID  string            `json:"snapshot_id"`
	CreatedAt   time.Time         `json:"created_at"`
	Tags        map[string]string `json:"tags,omitempty"`
	Version     string            `json:"version"`
}

// Validate checks if the baseline has all required fields
func (b Baseline) Validate() error {
	if b.ID == "" {
		return fmt.Errorf("baseline ID is required")
	}
	if b.Name == "" {
		return fmt.Errorf("baseline name is required")
	}
	if b.SnapshotID == "" {
		return fmt.Errorf("baseline snapshot ID is required")
	}
	if b.CreatedAt.IsZero() {
		return fmt.Errorf("baseline created time is required")
	}
	return nil
}

// Validate checks if the snapshot metadata is valid
func (sm SnapshotMetadata) Validate() error {
	if sm.CollectorVersion == "" {
		return fmt.Errorf("collector version is required")
	}
	if sm.ResourceCount < 0 {
		return fmt.Errorf("resource count cannot be negative")
	}
	if sm.CollectionTime < 0 {
		return fmt.Errorf("collection time cannot be negative")
	}
	return nil
}