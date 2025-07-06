package types

import (
	"errors"
	"strings"
	"time"
)

// Snapshot represents a point-in-time capture of infrastructure resources
type Snapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
	ID        string            `json:"id"`
	Provider  string            `json:"provider"`
	Region    string            `json:"region"`
	Resources []Resource        `json:"resources"`
}

// Validate checks if the Snapshot has all required fields and valid values
func (s *Snapshot) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return errors.New("snapshot ID is required")
	}
	if s.Timestamp.IsZero() {
		return errors.New("snapshot timestamp is required")
	}
	if strings.TrimSpace(s.Provider) == "" {
		return errors.New("snapshot provider is required")
	}
	if strings.TrimSpace(s.Region) == "" {
		return errors.New("snapshot region is required")
	}
	if s.Resources == nil {
		return errors.New("snapshot resources cannot be nil")
	}

	// Validate each resource in the snapshot
	for i := range s.Resources {
		if err := s.Resources[i].Validate(); err != nil {
			return errors.New("resource at index " + string(rune(i)) + " is invalid: " + err.Error())
		}
	}

	return nil
}

// ResourceCount returns the number of resources in the snapshot
func (s *Snapshot) ResourceCount() int {
	return len(s.Resources)
}

// GetResourceByID returns a resource by its ID, or nil if not found
func (s *Snapshot) GetResourceByID(id string) *Resource {
	for i := range s.Resources {
		if s.Resources[i].ID == id {
			return &s.Resources[i]
		}
	}
	return nil
}

// GetResourcesByType returns all resources of a specific type
func (s *Snapshot) GetResourcesByType(resourceType string) []Resource {
	var resources []Resource
	for i := range s.Resources {
		if strings.EqualFold(s.Resources[i].Type, resourceType) {
			resources = append(resources, s.Resources[i])
		}
	}
	return resources
}

// GetResourcesByProvider returns all resources from a specific provider
func (s *Snapshot) GetResourcesByProvider(provider string) []Resource {
	var resources []Resource
	for i := range s.Resources {
		if strings.EqualFold(s.Resources[i].Provider, provider) {
			resources = append(resources, s.Resources[i])
		}
	}
	return resources
}

// GetResourcesByTag returns all resources that have a specific tag key-value pair
func (s *Snapshot) GetResourcesByTag(key, value string) []Resource {
	var resources []Resource
	for i := range s.Resources {
		resource := &s.Resources[i]
		if tagValue := resource.GetTag(key); tagValue == value {
			resources = append(resources, *resource)
		}
	}
	return resources
}

// GetResourcesByState returns all resources in a specific state
func (s *Snapshot) GetResourcesByState(state string) []Resource {
	var resources []Resource
	for i := range s.Resources {
		resource := &s.Resources[i]
		if strings.EqualFold(resource.State, state) {
			resources = append(resources, *resource)
		}
	}
	return resources
}

// GetActiveResources returns all resources that are in an active state
func (s *Snapshot) GetActiveResources() []Resource {
	var resources []Resource
	for i := range s.Resources {
		resource := &s.Resources[i]
		if resource.IsActive() {
			resources = append(resources, *resource)
		}
	}
	return resources
}

// AddResource adds a resource to the snapshot
func (s *Snapshot) AddResource(resource *Resource) error {
	if err := resource.Validate(); err != nil {
		return err
	}

	// Check for duplicate resource IDs
	for i := range s.Resources {
		existing := &s.Resources[i]
		if existing.ID == resource.ID {
			return errors.New("resource with ID " + resource.ID + " already exists in snapshot")
		}
	}

	s.Resources = append(s.Resources, *resource)
	return nil
}

// RemoveResource removes a resource from the snapshot by ID
func (s *Snapshot) RemoveResource(id string) bool {
	for i := range s.Resources {
		if s.Resources[i].ID == id {
			s.Resources = append(s.Resources[:i], s.Resources[i+1:]...)
			return true
		}
	}
	return false
}

// GetMetadata returns the value of a specific metadata key
func (s *Snapshot) GetMetadata(key string) string {
	if s.Metadata == nil {
		return ""
	}
	return s.Metadata[key]
}

// SetMetadata sets a metadata key-value pair
func (s *Snapshot) SetMetadata(key, value string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[key] = value
}

// Clone creates a deep copy of the snapshot
func (s *Snapshot) Clone() *Snapshot {
	clone := &Snapshot{
		ID:        s.ID,
		Timestamp: s.Timestamp,
		Provider:  s.Provider,
		Region:    s.Region,
	}

	// Deep copy Resources
	if s.Resources != nil {
		clone.Resources = make([]Resource, len(s.Resources))
		for i := range s.Resources {
			clone.Resources[i] = *s.Resources[i].Clone()
		}
	}

	// Deep copy Metadata
	if s.Metadata != nil {
		clone.Metadata = make(map[string]string, len(s.Metadata))
		for k, v := range s.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

// String returns a string representation of the snapshot
func (s *Snapshot) String() string {
	return s.Provider + ":" + s.Region + " snapshot " + s.ID + " (" + s.Timestamp.Format(time.RFC3339) + ")"
}
