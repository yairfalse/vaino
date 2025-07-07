package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// TerraformState represents the structure of a Terraform state file
type TerraformState struct {
	Version          int                      `json:"version"`
	TerraformVersion string                   `json:"terraform_version"`
	Serial           int                      `json:"serial"`
	Lineage          string                   `json:"lineage"`
	Outputs          map[string]interface{}   `json:"outputs"`
	Resources        []TerraformResource      `json:"resources"`
	Modules          []TerraformModule        `json:"modules,omitempty"` // Legacy format
}

// TerraformResource represents a resource in Terraform state
type TerraformResource struct {
	Mode         string                   `json:"mode"`
	Type         string                   `json:"type"`
	Name         string                   `json:"name"`
	Provider     string                   `json:"provider"`
	Instances    []TerraformInstance      `json:"instances"`
	EachMode     string                   `json:"each,omitempty"`
	Module       string                   `json:"module,omitempty"`
}

// TerraformInstance represents an instance of a resource
type TerraformInstance struct {
	SchemaVersion  int                    `json:"schema_version"`
	Attributes     map[string]interface{} `json:"attributes"`
	Dependencies   []string               `json:"dependencies,omitempty"`
	CreateBeforeDestroy bool              `json:"create_before_destroy,omitempty"`
	Tainted        bool                   `json:"tainted,omitempty"`
	Deposed        []interface{}          `json:"deposed,omitempty"`
}

// TerraformModule represents a module in legacy state format (< 0.12)
type TerraformModule struct {
	Path         []string                           `json:"path"`
	Outputs      map[string]interface{}             `json:"outputs"`
	Resources    map[string]LegacyTerraformResource `json:"resources"`
	Dependencies []string                           `json:"dependencies"`
}

// LegacyTerraformResource represents resource format in Terraform < 0.12
type LegacyTerraformResource struct {
	Type         string                 `json:"type"`
	Primary      LegacyPrimaryResource  `json:"primary"`
	Dependencies []string               `json:"depends_on,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
}

// LegacyPrimaryResource represents the primary instance in legacy format
type LegacyPrimaryResource struct {
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
	Tainted    bool                   `json:"tainted,omitempty"`
}

// StateParser handles parsing of Terraform state files
type StateParser struct{}

// NewStateParser creates a new state parser
func NewStateParser() *StateParser {
	return &StateParser{}
}

// ParseStateFile reads and parses a Terraform state file
func (p *StateParser) ParseStateFile(filePath string) (*TerraformState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	if len(data) == 0 {
		return &TerraformState{
			Version:   4,
			Resources: []TerraformResource{},
		}, nil
	}
	
	var state TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file JSON: %w", err)
	}
	
	// Handle legacy state format (version < 4)
	if state.Version < 4 && len(state.Modules) > 0 {
		return p.convertLegacyState(&state)
	}
	
	return &state, nil
}

// convertLegacyState converts legacy state format to modern format
func (p *StateParser) convertLegacyState(legacyState *TerraformState) (*TerraformState, error) {
	var modernResources []TerraformResource
	
	for _, module := range legacyState.Modules {
		for resourceKey, resource := range module.Resources {
			// Parse resource key (e.g., "aws_instance.web" -> type: "aws_instance", name: "web")
			resourceType, resourceName := p.parseResourceKey(resourceKey)
			
			modernResource := TerraformResource{
				Mode:     "managed",
				Type:     resourceType,
				Name:     resourceName,
				Provider: resource.Provider,
				Instances: []TerraformInstance{
					{
						SchemaVersion: 0,
						Attributes:    resource.Primary.Attributes,
						Dependencies:  resource.Dependencies,
						Tainted:       resource.Primary.Tainted,
					},
				},
				Module: p.formatModulePath(module.Path),
			}
			
			modernResources = append(modernResources, modernResource)
		}
	}
	
	return &TerraformState{
		Version:          legacyState.Version,
		TerraformVersion: legacyState.TerraformVersion,
		Serial:           legacyState.Serial,
		Lineage:          legacyState.Lineage,
		Outputs:          legacyState.Outputs,
		Resources:        modernResources,
	}, nil
}

// parseResourceKey splits a resource key into type and name
func (p *StateParser) parseResourceKey(key string) (string, string) {
	// Handle different formats:
	// - "aws_instance.web"
	// - "module.vpc.aws_instance.web"
	// - "aws_instance.web.0" (count)
	
	parts := []string{}
	current := ""
	inBrackets := false
	
	for _, char := range key {
		if char == '[' {
			inBrackets = true
			current += string(char)
		} else if char == ']' {
			inBrackets = false
			current += string(char)
		} else if char == '.' && !inBrackets {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	
	if len(parts) >= 2 {
		// Skip module prefixes
		for i, part := range parts {
			if part != "module" && i+1 < len(parts) {
				// Check if next part is numeric (count index)
				if i+2 < len(parts) {
					if _, err := strconv.Atoi(parts[i+2]); err == nil {
						return part, parts[i+1] // type, name
					}
				}
				return part, parts[i+1] // type, name
			}
		}
	}
	
	// Fallback
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	
	return "unknown", key
}

// formatModulePath converts module path array to string
func (p *StateParser) formatModulePath(path []string) string {
	if len(path) <= 1 {
		return ""
	}
	// Skip the "root" path element
	if path[0] == "root" {
		path = path[1:]
	}
	if len(path) == 0 {
		return ""
	}
	
	result := ""
	for i, segment := range path {
		if i > 0 {
			result += "."
		}
		result += segment
	}
	return result
}

// GetResourcesByType returns all resources of a specific type
func (s *TerraformState) GetResourcesByType(resourceType string) []TerraformResource {
	var results []TerraformResource
	for _, resource := range s.Resources {
		if resource.Type == resourceType {
			results = append(results, resource)
		}
	}
	return results
}

// GetResourceByAddress returns a resource by its Terraform address
func (s *TerraformState) GetResourceByAddress(address string) (TerraformResource, bool) {
	for _, resource := range s.Resources {
		if p := (&StateParser{}); p.getResourceAddress(resource) == address {
			return resource, true
		}
	}
	return TerraformResource{}, false
}

// getResourceAddress constructs the Terraform address for a resource
func (p *StateParser) getResourceAddress(resource TerraformResource) string {
	address := resource.Type + "." + resource.Name
	if resource.Module != "" {
		address = "module." + resource.Module + "." + address
	}
	return address
}

// GetCreatedAt attempts to extract creation time from resource attributes
func (r *TerraformResource) GetCreatedAt() time.Time {
	for _, instance := range r.Instances {
		// Common timestamp attributes
		timeAttrs := []string{"created_time", "creation_date", "create_time", "created_at"}
		
		for _, attr := range timeAttrs {
			if val, exists := instance.Attributes[attr]; exists {
				if timeStr, ok := val.(string); ok {
					// Try different time formats
					formats := []string{
						time.RFC3339,
						"2006-01-02T15:04:05Z",
						"2006-01-02T15:04:05.000Z",
						"2006-01-02 15:04:05",
					}
					
					for _, format := range formats {
						if t, err := time.Parse(format, timeStr); err == nil {
							return t
						}
					}
				}
			}
		}
	}
	
	return time.Time{}
}

// GetTags extracts tags from a Terraform resource
func (r *TerraformResource) GetTags() map[string]string {
	tags := make(map[string]string)
	
	for _, instance := range r.Instances {
		// Check different tag attribute names
		tagAttrs := []string{"tags", "labels", "tag"}
		
		for _, attr := range tagAttrs {
			if val, exists := instance.Attributes[attr]; exists {
				if tagMap, ok := val.(map[string]interface{}); ok {
					for k, v := range tagMap {
						if strVal, ok := v.(string); ok {
							tags[k] = strVal
						}
					}
				}
			}
		}
	}
	
	return tags
}