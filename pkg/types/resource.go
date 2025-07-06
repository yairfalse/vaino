package types

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Resource represents a cloud infrastructure resource
type Resource struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Provider     string            `json:"provider"`
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	Config       map[string]any    `json:"config"`
	Tags         map[string]string `json:"tags"`
	State        string            `json:"state"`
	LastModified time.Time         `json:"last_modified"`
	Source       string            `json:"source"`
}

// Validate checks if the Resource has all required fields and valid values
func (r *Resource) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("resource ID is required")
	}
	if strings.TrimSpace(r.Type) == "" {
		return errors.New("resource type is required")
	}
	if strings.TrimSpace(r.Provider) == "" {
		return errors.New("resource provider is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("resource name is required")
	}
	if strings.TrimSpace(r.Region) == "" {
		return errors.New("resource region is required")
	}
	if r.LastModified.IsZero() {
		return errors.New("last modified time is required")
	}
	if strings.TrimSpace(r.Source) == "" {
		return errors.New("resource source is required")
	}
	return nil
}

// IsActive returns true if the resource is in an active state
func (r *Resource) IsActive() bool {
	activeStates := []string{"running", "active", "available", "in-use", "attached"}
	for _, state := range activeStates {
		if strings.EqualFold(r.State, state) {
			return true
		}
	}
	return false
}

// GetTag returns the value of a specific tag, or empty string if not found
func (r *Resource) GetTag(key string) string {
	if r.Tags == nil {
		return ""
	}
	return r.Tags[key]
}

// SetTag sets a tag key-value pair
func (r *Resource) SetTag(key, value string) {
	if r.Tags == nil {
		r.Tags = make(map[string]string)
	}
	r.Tags[key] = value
}

// GetConfigValue returns a configuration value by key
func (r *Resource) GetConfigValue(key string) (any, bool) {
	if r.Config == nil {
		return nil, false
	}
	value, exists := r.Config[key]
	return value, exists
}

// SetConfigValue sets a configuration key-value pair
func (r *Resource) SetConfigValue(key string, value any) {
	if r.Config == nil {
		r.Config = make(map[string]any)
	}
	r.Config[key] = value
}

// Clone creates a deep copy of the resource
func (r *Resource) Clone() *Resource {
	clone := &Resource{
		ID:           r.ID,
		Type:         r.Type,
		Provider:     r.Provider,
		Name:         r.Name,
		Region:       r.Region,
		State:        r.State,
		LastModified: r.LastModified,
		Source:       r.Source,
	}

	// Deep copy Tags
	if r.Tags != nil {
		clone.Tags = make(map[string]string, len(r.Tags))
		for k, v := range r.Tags {
			clone.Tags[k] = v
		}
	}

	// Deep copy Config
	if r.Config != nil {
		clone.Config = make(map[string]any, len(r.Config))
		for k, v := range r.Config {
			// For simplicity, we'll use JSON marshaling/unmarshaling for deep copy
			// This works for JSON-serializable types
			data, err := json.Marshal(v)
			if err != nil {
				clone.Config[k] = v // fallback to shallow copy
			} else {
				var clonedValue any
				if err := json.Unmarshal(data, &clonedValue); err != nil {
					clone.Config[k] = v // fallback to shallow copy
				} else {
					clone.Config[k] = clonedValue
				}
			}
		}
	}

	return clone
}

// String returns a string representation of the resource
func (r *Resource) String() string {
	return r.Provider + ":" + r.Type + ":" + r.Name + " (" + r.ID + ")"
}
