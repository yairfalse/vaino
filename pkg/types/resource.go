package types

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Resource represents a single infrastructure resource
type Resource struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Name          string                 `json:"name"`
	Provider      string                 `json:"provider"`
	Region        string                 `json:"region,omitempty"`
	Namespace     string                 `json:"namespace,omitempty"`
	Configuration map[string]interface{} `json:"configuration"`
	Metadata      ResourceMetadata       `json:"metadata"`
	Tags          map[string]string      `json:"tags,omitempty"`
}

// ResourceMetadata contains metadata about the resource
type ResourceMetadata struct {
	CreatedAt      time.Time              `json:"created_at,omitempty"`
	UpdatedAt      time.Time              `json:"updated_at,omitempty"`
	Version        string                 `json:"version,omitempty"`
	Checksum       string                 `json:"checksum,omitempty"`
	Dependencies   []string               `json:"dependencies,omitempty"`
	StateFile      string                 `json:"state_file,omitempty"`
	StateVersion   string                 `json:"state_version,omitempty"`
	AdditionalData map[string]interface{} `json:"additional_data,omitempty"`
}

// Validate checks if the resource has all required fields
func (r Resource) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("resource ID is required")
	}
	if r.Type == "" {
		return fmt.Errorf("resource type is required")
	}
	if r.Provider == "" {
		return fmt.Errorf("resource provider is required")
	}
	return nil
}

// String returns a string representation of the resource
func (r Resource) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%s:%s:%s", r.Provider, r.Type, r.ID))

	if r.Name != "" {
		parts = append(parts, fmt.Sprintf("(%s)", r.Name))
	}

	if r.Region != "" {
		parts = append(parts, fmt.Sprintf("in %s", r.Region))
	} else if r.Namespace != "" {
		parts = append(parts, fmt.Sprintf("in %s", r.Namespace))
	}

	return strings.Join(parts, " ")
}

// Equals compares two resources for equality
func (r Resource) Equals(other Resource) bool {
	if r.ID != other.ID || r.Type != other.Type || r.Provider != other.Provider {
		return false
	}

	if r.Name != other.Name || r.Region != other.Region || r.Namespace != other.Namespace {
		return false
	}

	// Compare configuration maps
	if len(r.Configuration) != len(other.Configuration) {
		return false
	}
	for k, v := range r.Configuration {
		if otherV, exists := other.Configuration[k]; !exists || v != otherV {
			return false
		}
	}

	// Compare tags
	if len(r.Tags) != len(other.Tags) {
		return false
	}
	for k, v := range r.Tags {
		if otherV, exists := other.Tags[k]; !exists || v != otherV {
			return false
		}
	}

	return true
}

// Hash returns a hash of the resource for quick comparison
func (r Resource) Hash() string {
	data, _ := json.Marshal(struct {
		ID            string                 `json:"id"`
		Type          string                 `json:"type"`
		Provider      string                 `json:"provider"`
		Name          string                 `json:"name"`
		Region        string                 `json:"region"`
		Namespace     string                 `json:"namespace"`
		Configuration map[string]interface{} `json:"configuration"`
		Tags          map[string]string      `json:"tags"`
	}{
		ID:            r.ID,
		Type:          r.Type,
		Provider:      r.Provider,
		Name:          r.Name,
		Region:        r.Region,
		Namespace:     r.Namespace,
		Configuration: r.Configuration,
		Tags:          r.Tags,
	})

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// DriftResult represents the result of comparing two resources
type DriftResult struct {
	ResourceID  string        `json:"resource_id"`
	DriftType   DriftType     `json:"drift_type"`
	Severity    DriftSeverity `json:"severity"`
	Changes     []Change      `json:"changes,omitempty"`
	RiskScore   float64       `json:"risk_score"`
	Description string        `json:"description"`
	Timestamp   time.Time     `json:"timestamp"`
}

// DriftType represents the type of drift detected
type DriftType string

const (
	DriftTypeCreated  DriftType = "created"
	DriftTypeDeleted  DriftType = "deleted"
	DriftTypeModified DriftType = "modified"
	DriftTypeMigrated DriftType = "migrated"
)

// DriftSeverity represents the severity of drift
type DriftSeverity string

const (
	DriftSeverityLow      DriftSeverity = "low"
	DriftSeverityMedium   DriftSeverity = "medium"
	DriftSeverityHigh     DriftSeverity = "high"
	DriftSeverityCritical DriftSeverity = "critical"
)
