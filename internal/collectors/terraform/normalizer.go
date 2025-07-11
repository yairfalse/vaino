package terraform

import (
	"fmt"
	"strings"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// ResourceNormalizer converts Terraform resources to VAINO Resource format
type ResourceNormalizer struct {
	typeMapping map[string]ResourceTypeInfo
}

// ResourceTypeInfo contains information about how to normalize a specific resource type
type ResourceTypeInfo struct {
	Category    string
	IDField     string
	NameField   string
	RegionField string
	TagsField   string
}

// NewResourceNormalizer creates a new resource normalizer
func NewResourceNormalizer() *ResourceNormalizer {
	return &ResourceNormalizer{
		typeMapping: initializeTypeMappings(),
	}
}

// NormalizeResources converts a Terraform state to VAINO resources
func (n *ResourceNormalizer) NormalizeResources(tfState *TerraformState) ([]types.Resource, error) {
	var resources []types.Resource

	for _, tfResource := range tfState.Resources {
		// Skip data sources, only process managed resources
		if tfResource.Mode != "managed" {
			continue
		}

		// Process each instance of the resource
		for i, instance := range tfResource.Instances {
			resource, err := n.normalizeResource(tfResource, instance, i)
			if err != nil {
				// Log warning but continue processing other resources
				continue
			}
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// normalizeResource converts a single Terraform resource instance to VAINO Resource
func (n *ResourceNormalizer) normalizeResource(tfResource TerraformResource, instance TerraformInstance, instanceIndex int) (types.Resource, error) {
	// Get type info for normalization
	typeInfo := n.getTypeInfo(tfResource.Type)

	// Generate resource ID
	resourceID := n.generateResourceID(tfResource, instance, instanceIndex, typeInfo)

	// Extract resource name
	resourceName := n.extractResourceName(tfResource, instance, typeInfo)

	// Extract region
	region := n.extractRegion(instance, typeInfo)

	// Extract tags
	tags := n.extractTags(instance, typeInfo)

	// Create metadata
	metadata := types.ResourceMetadata{
		CreatedAt:    n.extractCreatedAt(instance),
		UpdatedAt:    n.extractUpdatedAt(instance),
		Version:      n.extractVersion(instance),
		Dependencies: instance.Dependencies,
		Checksum:     n.generateChecksum(instance.Attributes),
	}

	resource := types.Resource{
		ID:            resourceID,
		Type:          tfResource.Type,
		Name:          resourceName,
		Provider:      "terraform",
		Region:        region,
		Configuration: n.sanitizeConfiguration(instance.Attributes),
		Metadata:      metadata,
		Tags:          tags,
	}

	return resource, nil
}

// generateResourceID creates a unique ID for the resource
func (n *ResourceNormalizer) generateResourceID(tfResource TerraformResource, instance TerraformInstance, instanceIndex int, typeInfo ResourceTypeInfo) string {
	// Try to use the actual resource ID from Terraform
	if typeInfo.IDField != "" {
		if id, exists := instance.Attributes[typeInfo.IDField]; exists {
			if idStr, ok := id.(string); ok && idStr != "" {
				return idStr
			}
		}
	}

	// Try common ID fields
	idFields := []string{"id", "arn", "name", "identifier"}
	for _, field := range idFields {
		if id, exists := instance.Attributes[field]; exists {
			if idStr, ok := id.(string); ok && idStr != "" {
				return idStr
			}
		}
	}

	// Fallback: use Terraform address
	address := tfResource.Type + "." + tfResource.Name
	if tfResource.Module != "" {
		address = "module." + tfResource.Module + "." + address
	}

	// Add instance index for count/for_each resources
	if instanceIndex > 0 || len(tfResource.Instances) > 1 {
		address += fmt.Sprintf("[%d]", instanceIndex)
	}

	return address
}

// extractResourceName gets the human-readable name for the resource
func (n *ResourceNormalizer) extractResourceName(tfResource TerraformResource, instance TerraformInstance, typeInfo ResourceTypeInfo) string {
	// Try configured name field
	if typeInfo.NameField != "" {
		if name, exists := instance.Attributes[typeInfo.NameField]; exists {
			if nameStr, ok := name.(string); ok && nameStr != "" {
				return nameStr
			}
		}
	}

	// Try common name fields
	nameFields := []string{"name", "display_name", "title", "identifier"}
	for _, field := range nameFields {
		if name, exists := instance.Attributes[field]; exists {
			if nameStr, ok := name.(string); ok && nameStr != "" {
				return nameStr
			}
		}
	}

	// Fallback to Terraform resource name
	return tfResource.Name
}

// extractRegion gets the region/location for the resource
func (n *ResourceNormalizer) extractRegion(instance TerraformInstance, typeInfo ResourceTypeInfo) string {
	// Try configured region field
	if typeInfo.RegionField != "" {
		if region, exists := instance.Attributes[typeInfo.RegionField]; exists {
			if regionStr, ok := region.(string); ok && regionStr != "" {
				return regionStr
			}
		}
	}

	// Try common region/location fields
	regionFields := []string{"region", "location", "zone", "availability_zone", "placement"}
	for _, field := range regionFields {
		if region, exists := instance.Attributes[field]; exists {
			if regionStr, ok := region.(string); ok && regionStr != "" {
				return regionStr
			}
		}
	}

	return ""
}

// extractTags gets tags/labels from the resource
func (n *ResourceNormalizer) extractTags(instance TerraformInstance, typeInfo ResourceTypeInfo) map[string]string {
	tags := make(map[string]string)

	// Try configured tags field
	tagsField := typeInfo.TagsField
	if tagsField == "" {
		tagsField = "tags"
	}

	tagFields := []string{tagsField, "tags", "labels", "metadata"}

	for _, field := range tagFields {
		if tagData, exists := instance.Attributes[field]; exists {
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

// extractCreatedAt attempts to find creation timestamp
func (n *ResourceNormalizer) extractCreatedAt(instance TerraformInstance) time.Time {
	timeFields := []string{"created_time", "creation_date", "create_time", "created_at", "time_created"}

	for _, field := range timeFields {
		if timeVal, exists := instance.Attributes[field]; exists {
			if timeStr, ok := timeVal.(string); ok {
				if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
					return t
				}
				// Try other common formats
				formats := []string{
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

	return time.Time{}
}

// extractUpdatedAt attempts to find last modification timestamp
func (n *ResourceNormalizer) extractUpdatedAt(instance TerraformInstance) time.Time {
	timeFields := []string{"last_modified", "updated_time", "modification_date", "updated_at", "time_updated"}

	for _, field := range timeFields {
		if timeVal, exists := instance.Attributes[field]; exists {
			if timeStr, ok := timeVal.(string); ok {
				if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
					return t
				}
			}
		}
	}

	return time.Time{}
}

// extractVersion attempts to find resource version
func (n *ResourceNormalizer) extractVersion(instance TerraformInstance) string {
	versionFields := []string{"version", "resource_version", "etag", "revision"}

	for _, field := range versionFields {
		if version, exists := instance.Attributes[field]; exists {
			if versionStr, ok := version.(string); ok && versionStr != "" {
				return versionStr
			}
		}
	}

	return ""
}

// generateChecksum creates a checksum for the resource configuration
func (n *ResourceNormalizer) generateChecksum(attributes map[string]interface{}) string {
	// This is a simplified checksum - in production you'd want a proper hash
	return fmt.Sprintf("tf-%d", len(attributes))
}

// sanitizeConfiguration removes sensitive fields from configuration
func (n *ResourceNormalizer) sanitizeConfiguration(attributes map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	// List of sensitive fields to exclude
	sensitiveFields := map[string]bool{
		"password":          true,
		"secret":            true,
		"private_key":       true,
		"access_key":        true,
		"secret_key":        true,
		"token":             true,
		"api_key":           true,
		"connection_string": true,
	}

	for k, v := range attributes {
		// Skip sensitive fields
		if sensitiveFields[strings.ToLower(k)] {
			sanitized[k] = "[REDACTED]"
			continue
		}

		// Skip fields that contain sensitive keywords
		lowerKey := strings.ToLower(k)
		if strings.Contains(lowerKey, "password") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "key") && strings.Contains(lowerKey, "private") {
			sanitized[k] = "[REDACTED]"
			continue
		}

		sanitized[k] = v
	}

	return sanitized
}

// getTypeInfo returns normalization info for a resource type
func (n *ResourceNormalizer) getTypeInfo(resourceType string) ResourceTypeInfo {
	if info, exists := n.typeMapping[resourceType]; exists {
		return info
	}

	// Return default mapping
	return ResourceTypeInfo{
		Category:    "unknown",
		IDField:     "id",
		NameField:   "name",
		RegionField: "region",
		TagsField:   "tags",
	}
}

// initializeTypeMappings creates the mapping for common Terraform resource types
func initializeTypeMappings() map[string]ResourceTypeInfo {
	return map[string]ResourceTypeInfo{
		// AWS Resources
		"aws_instance": {
			Category:    "compute",
			IDField:     "id",
			NameField:   "tags.Name",
			RegionField: "availability_zone",
			TagsField:   "tags",
		},
		"aws_s3_bucket": {
			Category:    "storage",
			IDField:     "id",
			NameField:   "bucket",
			RegionField: "region",
			TagsField:   "tags",
		},
		"aws_vpc": {
			Category:    "network",
			IDField:     "id",
			NameField:   "tags.Name",
			RegionField: "region",
			TagsField:   "tags",
		},
		"aws_security_group": {
			Category:    "security",
			IDField:     "id",
			NameField:   "name",
			RegionField: "region",
			TagsField:   "tags",
		},
		"aws_rds_instance": {
			Category:    "database",
			IDField:     "id",
			NameField:   "identifier",
			RegionField: "availability_zone",
			TagsField:   "tags",
		},
		"aws_lambda_function": {
			Category:    "compute",
			IDField:     "arn",
			NameField:   "function_name",
			RegionField: "region",
			TagsField:   "tags",
		},

		// Azure Resources
		"azurerm_virtual_machine": {
			Category:    "compute",
			IDField:     "id",
			NameField:   "name",
			RegionField: "location",
			TagsField:   "tags",
		},
		"azurerm_storage_account": {
			Category:    "storage",
			IDField:     "id",
			NameField:   "name",
			RegionField: "location",
			TagsField:   "tags",
		},

		// Google Cloud Resources
		"google_compute_instance": {
			Category:    "compute",
			IDField:     "id",
			NameField:   "name",
			RegionField: "zone",
			TagsField:   "labels",
		},
		"google_storage_bucket": {
			Category:    "storage",
			IDField:     "id",
			NameField:   "name",
			RegionField: "location",
			TagsField:   "labels",
		},

		// Kubernetes Resources
		"kubernetes_deployment": {
			Category:    "workload",
			IDField:     "id",
			NameField:   "metadata.0.name",
			RegionField: "",
			TagsField:   "metadata.0.labels",
		},
		"kubernetes_service": {
			Category:    "network",
			IDField:     "id",
			NameField:   "metadata.0.name",
			RegionField: "",
			TagsField:   "metadata.0.labels",
		},
	}
}
