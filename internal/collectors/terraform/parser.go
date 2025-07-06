package terraform

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// State represents the structure of a Terraform state file
type State struct {
	Outputs          map[string]interface{} `json:"outputs,omitempty"`
	TerraformVersion string                 `json:"terraform_version"`
	Lineage          string                 `json:"lineage"`
	Resources        []Resource    `json:"resources,omitempty"`
	Version          int                    `json:"version"`
	Serial           int                    `json:"serial"`
}

// Resource represents a resource in the Terraform state
type Resource struct {
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Mode       string                 `json:"mode"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Provider   string                 `json:"provider"`
	Each       string                 `json:"each,omitempty"`
	Module     string                 `json:"module,omitempty"`
	Instances  []Instance    `json:"instances"`
	DependsOn  []string               `json:"depends_on,omitempty"`
}

// Instance represents an instance of a resource
type Instance struct {
	Attributes          map[string]interface{} `json:"attributes"`
	Private             string                 `json:"private,omitempty"`
	Dependencies        []string               `json:"dependencies,omitempty"`
	SchemaVersion       int                    `json:"schema_version"`
	CreateBeforeDestroy bool                   `json:"create_before_destroy,omitempty"`
}

// ParseStateFile parses a Terraform state file and returns a list of resources
func ParseStateFile(filePath string) ([]types.Resource, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, errors.New("file path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("state file does not exist: %s", filePath)
	}

	// Open and read the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file JSON: %w", err)
	}

	// Convert to our Resource types
	resources := make([]types.Resource, 0)
	for i := range state.Resources {
		parsedResources, err := parseResource(&state.Resources[i], filePath)
		if err != nil {
			resType := state.Resources[i].Type
			resName := state.Resources[i].Name
			return nil, fmt.Errorf("failed to parse resource %s.%s: %w", resType, resName, err)
		}
		resources = append(resources, parsedResources...)
	}

	return resources, nil
}

// parseResource converts a Terraform resource to our Resource type
func parseResource(tfResource *Resource, sourceFile string) ([]types.Resource, error) {
	if tfResource.Mode != "managed" {
		// Skip data sources and other non-managed resources
		return nil, nil
	}

	resources := make([]types.Resource, 0, len(tfResource.Instances))

	for i, instance := range tfResource.Instances {
		// Extract resource ID
		resourceID := extractResourceID(instance.Attributes)
		if resourceID == "" {
			resourceID = fmt.Sprintf("%s.%s[%d]", tfResource.Type, tfResource.Name, i)
		}

		// Extract region
		region := extractRegion(instance.Attributes)
		if region == "" {
			region = "unknown"
		}

		// Extract provider
		provider := extractProvider(tfResource.Provider)

		// Extract tags
		tags := extractTags(instance.Attributes)

		// Extract state
		state := extractState(instance.Attributes)

		// Create resource
		resource := types.Resource{
			ID:           resourceID,
			Type:         tfResource.Type,
			Provider:     provider,
			Name:         tfResource.Name,
			Region:       region,
			Config:       instance.Attributes,
			Tags:         tags,
			State:        state,
			LastModified: time.Now(), // Use current time as we don't have last modified in state
			Source:       sourceFile,
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// extractResourceID extracts the resource ID from attributes
func extractResourceID(attributes map[string]interface{}) string {
	// Common ID fields in Terraform resources
	idFields := []string{"id", "arn", "instance_id", "resource_id"}

	for _, field := range idFields {
		if id, ok := attributes[field]; ok {
			if idStr, ok := id.(string); ok && idStr != "" {
				return idStr
			}
		}
	}

	return ""
}

// extractRegion extracts the region from attributes
func extractRegion(attributes map[string]interface{}) string {
	regionFields := []string{"region", "availability_zone", "zone", "location"}

	for _, field := range regionFields {
		if region, ok := attributes[field]; ok {
			if regionStr, ok := region.(string); ok && regionStr != "" {
				if strings.Contains(field, "zone") {
					return extractRegionFromZone(regionStr)
				}
				return regionStr
			}
		}
	}

	return ""
}

// extractRegionFromZone extracts region from zone/availability zone strings
func extractRegionFromZone(zoneStr string) string {
	if len(zoneStr) <= 1 {
		return zoneStr
	}

	// Check for GCP-style zones like "europe-west1-b" first
	if region := extractGCPRegion(zoneStr); region != "" {
		return region
	}

	// Check for AWS-style zones like "us-east-1a"
	return extractAWSRegion(zoneStr)
}

// extractGCPRegion extracts region from GCP zone format
func extractGCPRegion(zoneStr string) string {
	lastDash := strings.LastIndex(zoneStr, "-")
	if lastDash <= 0 || lastDash >= len(zoneStr)-1 {
		return ""
	}

	suffixAfterDash := zoneStr[lastDash+1:]
	if len(suffixAfterDash) == 1 && isLetter(suffixAfterDash[0]) {
		return zoneStr[:lastDash]
	}
	return ""
}

// extractAWSRegion extracts region from AWS zone format
func extractAWSRegion(zoneStr string) string {
	if zoneStr == "" {
		return zoneStr
	}

	lastChar := zoneStr[len(zoneStr)-1]
	if isLetter(lastChar) {
		return zoneStr[:len(zoneStr)-1]
	}
	return zoneStr
}

// isLetter checks if a byte is a letter
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// extractProvider extracts the provider name from the provider string
func extractProvider(provider string) string {
	if provider == "" {
		return ProviderName
	}

	// Provider strings are usually in format: "provider[\"registry.terraform.io/hashicorp/aws\"]"
	// or "provider.aws"
	if strings.Contains(provider, "/") {
		parts := strings.Split(provider, "/")
		if len(parts) > 0 {
			return strings.Trim(parts[len(parts)-1], "\"]")
		}
	}

	// Handle "provider.aws" format
	if strings.HasPrefix(provider, "provider.") {
		return strings.TrimPrefix(provider, "provider.")
	}

	return strings.Trim(provider, "\"[]")
}

// extractTags extracts tags from attributes
func extractTags(attributes map[string]interface{}) map[string]string {
	tags := make(map[string]string)

	// Common tag fields
	tagFields := []string{"tags", "labels", "metadata"}

	for _, field := range tagFields {
		if tagData, ok := attributes[field]; ok {
			if tagMap, ok := tagData.(map[string]interface{}); ok {
				for k, v := range tagMap {
					if vStr, ok := v.(string); ok {
						tags[k] = vStr
					}
				}
			}
		}
	}

	return tags
}

// extractState extracts the state from attributes
func extractState(attributes map[string]interface{}) string {
	// Common state fields
	stateFields := []string{"state", "status", "lifecycle_state", "instance_state"}

	for _, field := range stateFields {
		if state, ok := attributes[field]; ok {
			if stateStr, ok := state.(string); ok && stateStr != "" {
				return stateStr
			}
		}
	}

	// Default to "running" if no state found
	return "running"
}

// FindStateFiles finds all Terraform state files in the given paths
func FindStateFiles(paths []string) ([]string, error) {
	var stateFiles []string

	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}

		// Check if path exists
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to stat path %s: %w", path, err)
		}

		if info.IsDir() {
			// Find state files in directory
			files, err := findStateFilesInDir(path)
			if err != nil {
				return nil, err
			}
			stateFiles = append(stateFiles, files...)
		} else {
			// Check if it's a state file
			if isStateFile(path) {
				stateFiles = append(stateFiles, path)
			}
		}
	}

	return stateFiles, nil
}

// findStateFilesInDir recursively finds state files in a directory
func findStateFilesInDir(dir string) ([]string, error) {
	var stateFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isStateFile(path) {
			stateFiles = append(stateFiles, path)
		}

		return nil
	})

	return stateFiles, err
}

// isStateFile checks if a file is a Terraform state file
func isStateFile(filePath string) bool {
	name := filepath.Base(filePath)
	return strings.HasSuffix(name, ".tfstate") ||
		strings.HasSuffix(name, ".tfstate.backup") ||
		name == "terraform.tfstate"
}
