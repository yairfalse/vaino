package types

import (
	"fmt"
	"time"
)

// Resource represents a single infrastructure resource
type Resource struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`         // e.g., "aws_instance", "kubernetes_deployment"
	Provider     string                 `json:"provider"`     // e.g., "aws", "kubernetes", "terraform"
	Name         string                 `json:"name"`
	Region       string                 `json:"region"`
	Config       map[string]interface{} `json:"config"`       // resource configuration
	Tags         map[string]string      `json:"tags"`
	State        string                 `json:"state"`        // e.g., "running", "stopped"
	LastModified time.Time              `json:"last_modified"`
	Source       string                 `json:"source"`       // where this resource was discovered
}

// Validate checks if the Resource has all required fields
func (r *Resource) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	if r.Type == "" {
		return fmt.Errorf("resource type cannot be empty")
	}
	if r.Provider == "" {
		return fmt.Errorf("resource provider cannot be empty")
	}
	if r.Name == "" {
		return fmt.Errorf("resource name cannot be empty")
	}
	if r.Source == "" {
		return fmt.Errorf("resource source cannot be empty")
	}
	return nil
}

// GetConfigValue retrieves a configuration value by key
func (r *Resource) GetConfigValue(key string) (interface{}, bool) {
	if r.Config == nil {
		return nil, false
	}
	val, exists := r.Config[key]
	return val, exists
}

// GetTag retrieves a tag value by key
func (r *Resource) GetTag(key string) (string, bool) {
	if r.Tags == nil {
		return "", false
	}
	val, exists := r.Tags[key]
	return val, exists
}

// Key returns a unique key for the resource
func (r *Resource) Key() string {
	return fmt.Sprintf("%s/%s/%s", r.Provider, r.Type, r.ID)
}

// IsActive checks if the resource is in an active state
func (r *Resource) IsActive() bool {
	activeStates := map[string]bool{
		"running":   true,
		"active":    true,
		"available": true,
		"healthy":   true,
	}
	return activeStates[r.State]
}