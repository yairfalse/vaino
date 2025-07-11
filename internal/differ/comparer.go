package differ

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/yairfalse/vaino/pkg/types"
)

// DefaultComparer implements deep comparison of resources
type DefaultComparer struct {
	options DiffOptions
}

// Compare compares two resources and returns changes with error handling
func (c *DefaultComparer) Compare(baseline, current types.Resource) ([]Change, error) {
	changes := c.CompareResources(baseline, current)
	return changes, nil
}

// CompareResources compares two resources and returns a list of changes
func (c *DefaultComparer) CompareResources(baseline, current types.Resource) []Change {
	var changes []Change

	// Compare basic fields
	if baseline.Name != current.Name {
		changes = append(changes, Change{
			Type:        ChangeTypeModified,
			ResourceID:  current.ID,
			Path:        "name",
			Field:       "name",
			OldValue:    baseline.Name,
			NewValue:    current.Name,
			Description: fmt.Sprintf("Name changed from '%s' to '%s'", baseline.Name, current.Name),
		})
	}

	if baseline.Region != current.Region {
		changes = append(changes, Change{
			Type:        ChangeTypeModified,
			ResourceID:  current.ID,
			Path:        "region",
			Field:       "region",
			OldValue:    baseline.Region,
			NewValue:    current.Region,
			Description: fmt.Sprintf("Region changed from '%s' to '%s'", baseline.Region, current.Region),
		})
	}

	if baseline.Namespace != current.Namespace {
		changes = append(changes, Change{
			Type:        ChangeTypeModified,
			ResourceID:  current.ID,
			Path:        "namespace",
			Field:       "namespace",
			OldValue:    baseline.Namespace,
			NewValue:    current.Namespace,
			Description: fmt.Sprintf("Namespace changed from '%s' to '%s'", baseline.Namespace, current.Namespace),
		})
	}

	// Compare tags
	tagChanges := c.compareTags(current.ID, baseline.Tags, current.Tags)
	changes = append(changes, tagChanges...)

	// Compare configuration
	configChanges := c.CompareConfiguration("configuration", baseline.Configuration, current.Configuration)
	for i := range configChanges {
		configChanges[i].ResourceID = current.ID
	}
	changes = append(changes, configChanges...)

	// Filter out ignored fields
	changes = c.filterIgnoredFields(changes)

	return changes
}

// CompareConfiguration performs deep comparison of configuration maps
func (c *DefaultComparer) CompareConfiguration(basePath string, baseline, current map[string]interface{}) []Change {
	var changes []Change

	// Check for removed keys
	for key, baselineValue := range baseline {
		path := c.buildPath(basePath, key)

		if c.shouldIgnoreField(path) {
			continue
		}

		if currentValue, exists := current[key]; !exists {
			changes = append(changes, Change{
				Type:        ChangeTypeRemoved,
				Path:        path,
				Field:       key,
				OldValue:    baselineValue,
				NewValue:    nil,
				Description: fmt.Sprintf("Configuration key '%s' was removed", path),
			})
		} else {
			// Compare values
			valueChanges := c.compareValues(path, key, baselineValue, currentValue)
			changes = append(changes, valueChanges...)
		}
	}

	// Check for added keys
	for key, currentValue := range current {
		path := c.buildPath(basePath, key)

		if c.shouldIgnoreField(path) {
			continue
		}

		if _, exists := baseline[key]; !exists {
			changes = append(changes, Change{
				Type:        ChangeTypeAdded,
				Path:        path,
				Field:       key,
				OldValue:    nil,
				NewValue:    currentValue,
				Description: fmt.Sprintf("Configuration key '%s' was added", path),
			})
		}
	}

	return changes
}

// compareValues compares two values recursively
func (c *DefaultComparer) compareValues(path, key string, baseline, current interface{}) []Change {
	var changes []Change

	// Handle nil values
	if baseline == nil && current == nil {
		return changes
	}
	if baseline == nil && current != nil {
		changes = append(changes, Change{
			Type:        ChangeTypeAdded,
			Path:        path,
			Field:       key,
			OldValue:    nil,
			NewValue:    current,
			Description: fmt.Sprintf("Value at '%s' was set to %v", path, current),
		})
		return changes
	}
	if baseline != nil && current == nil {
		changes = append(changes, Change{
			Type:        ChangeTypeRemoved,
			Path:        path,
			Field:       key,
			OldValue:    baseline,
			NewValue:    nil,
			Description: fmt.Sprintf("Value at '%s' was removed", path),
		})
		return changes
	}

	// Get reflection types
	baselineType := reflect.TypeOf(baseline)
	currentType := reflect.TypeOf(current)

	// Type change
	if baselineType != currentType {
		changes = append(changes, Change{
			Type:        ChangeTypeModified,
			Path:        path,
			Field:       key,
			OldValue:    baseline,
			NewValue:    current,
			Description: fmt.Sprintf("Type changed from %T to %T at '%s'", baseline, current, path),
		})
		return changes
	}

	// Handle different types
	switch baselineValue := baseline.(type) {
	case map[string]interface{}:
		if currentMap, ok := current.(map[string]interface{}); ok {
			nestedChanges := c.CompareConfiguration(path, baselineValue, currentMap)
			changes = append(changes, nestedChanges...)
		}
	case []interface{}:
		if currentSlice, ok := current.([]interface{}); ok {
			sliceChanges := c.compareSlices(path, key, baselineValue, currentSlice)
			changes = append(changes, sliceChanges...)
		}
	default:
		// Simple value comparison
		if !reflect.DeepEqual(baseline, current) {
			changes = append(changes, Change{
				Type:        ChangeTypeModified,
				Path:        path,
				Field:       key,
				OldValue:    baseline,
				NewValue:    current,
				Description: fmt.Sprintf("Value changed from %v to %v at '%s'", baseline, current, path),
			})
		}
	}

	return changes
}

// compareSlices compares two slices and detects changes
func (c *DefaultComparer) compareSlices(path, key string, baseline, current []interface{}) []Change {
	var changes []Change

	// Simple approach: detect length changes and element changes
	if len(baseline) != len(current) {
		changes = append(changes, Change{
			Type:        ChangeTypeModified,
			Path:        path,
			Field:       key,
			OldValue:    baseline,
			NewValue:    current,
			Description: fmt.Sprintf("Array length changed from %d to %d at '%s'", len(baseline), len(current), path),
		})
		return changes
	}

	// Compare elements
	for i := 0; i < len(baseline); i++ {
		elementPath := fmt.Sprintf("%s[%d]", path, i)
		elementChanges := c.compareValues(elementPath, fmt.Sprintf("%s[%d]", key, i), baseline[i], current[i])
		changes = append(changes, elementChanges...)
	}

	return changes
}

// compareTags compares tag maps
func (c *DefaultComparer) compareTags(resourceID string, baseline, current map[string]string) []Change {
	var changes []Change

	// Check for removed tags
	for key, baselineValue := range baseline {
		if currentValue, exists := current[key]; !exists {
			changes = append(changes, Change{
				Type:        ChangeTypeRemoved,
				ResourceID:  resourceID,
				Path:        fmt.Sprintf("tags.%s", key),
				Field:       fmt.Sprintf("tags.%s", key),
				OldValue:    baselineValue,
				NewValue:    nil,
				Description: fmt.Sprintf("Tag '%s' was removed", key),
			})
		} else if baselineValue != currentValue {
			changes = append(changes, Change{
				Type:        ChangeTypeModified,
				ResourceID:  resourceID,
				Path:        fmt.Sprintf("tags.%s", key),
				Field:       fmt.Sprintf("tags.%s", key),
				OldValue:    baselineValue,
				NewValue:    currentValue,
				Description: fmt.Sprintf("Tag '%s' changed from '%s' to '%s'", key, baselineValue, currentValue),
			})
		}
	}

	// Check for added tags
	for key, currentValue := range current {
		if _, exists := baseline[key]; !exists {
			changes = append(changes, Change{
				Type:        ChangeTypeAdded,
				ResourceID:  resourceID,
				Path:        fmt.Sprintf("tags.%s", key),
				Field:       fmt.Sprintf("tags.%s", key),
				OldValue:    nil,
				NewValue:    currentValue,
				Description: fmt.Sprintf("Tag '%s' was added with value '%s'", key, currentValue),
			})
		}
	}

	return changes
}

// buildPath constructs a dot-notation path
func (c *DefaultComparer) buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", basePath, key)
}

// shouldIgnoreField checks if a field should be ignored based on options
func (c *DefaultComparer) shouldIgnoreField(path string) bool {
	for _, ignoredField := range c.options.IgnoreFields {
		if strings.Contains(path, ignoredField) {
			return true
		}
	}
	return false
}

// filterIgnoredFields removes changes for fields that should be ignored
func (c *DefaultComparer) filterIgnoredFields(changes []Change) []Change {
	var filtered []Change

	for _, change := range changes {
		if !c.shouldIgnoreField(change.Path) {
			filtered = append(filtered, change)
		}
	}

	return filtered
}

// SmartComparer is an enhanced comparer with context-aware comparison
type SmartComparer struct {
	options        DiffOptions
	typeComparers  map[string]TypeComparer
	fieldComparers map[string]FieldComparer
}

// TypeComparer defines resource-type-specific comparison logic
type TypeComparer interface {
	CompareResources(baseline, current types.Resource) []Change
	GetCriticalFields() []string
}

// FieldComparer defines field-specific comparison logic
type FieldComparer interface {
	CompareField(path, field string, baseline, current interface{}) []Change
}

// NewSmartComparer creates a comparer with type-specific logic
func NewSmartComparer(options DiffOptions) *SmartComparer {
	comparer := &SmartComparer{
		options:        options,
		typeComparers:  make(map[string]TypeComparer),
		fieldComparers: make(map[string]FieldComparer),
	}

	// Register default type comparers
	comparer.typeComparers["aws_instance"] = &EC2InstanceComparer{}
	comparer.typeComparers["aws_security_group"] = &SecurityGroupComparer{}
	comparer.typeComparers["kubernetes_deployment"] = &KubernetesDeploymentComparer{}

	// Register field comparers
	comparer.fieldComparers["security_groups"] = &SecurityGroupFieldComparer{}
	comparer.fieldComparers["tags"] = &TagFieldComparer{}

	return comparer
}

// CompareResources uses type-specific comparison logic
func (c *SmartComparer) CompareResources(baseline, current types.Resource) []Change {
	// Try type-specific comparer first
	resourceType := fmt.Sprintf("%s_%s", baseline.Provider, baseline.Type)
	if typeComparer, exists := c.typeComparers[resourceType]; exists {
		return typeComparer.CompareResources(baseline, current)
	}

	// Fall back to default comparison
	defaultComparer := &DefaultComparer{options: c.options}
	return defaultComparer.CompareResources(baseline, current)
}

// CompareConfiguration uses field-specific comparison logic where available
func (c *SmartComparer) CompareConfiguration(basePath string, baseline, current map[string]interface{}) []Change {
	var changes []Change

	defaultComparer := &DefaultComparer{options: c.options}

	for key, baselineValue := range baseline {
		path := c.buildPath(basePath, key)

		if fieldComparer, exists := c.fieldComparers[key]; exists {
			if currentValue, exists := current[key]; exists {
				fieldChanges := fieldComparer.CompareField(path, key, baselineValue, currentValue)
				changes = append(changes, fieldChanges...)
			}
		} else {
			// Use default comparison
			if currentValue, exists := current[key]; exists {
				valueChanges := defaultComparer.compareValues(path, key, baselineValue, currentValue)
				changes = append(changes, valueChanges...)
			} else {
				changes = append(changes, Change{
					Type:        ChangeTypeRemoved,
					Path:        path,
					Field:       key,
					OldValue:    baselineValue,
					NewValue:    nil,
					Description: fmt.Sprintf("Configuration key '%s' was removed", path),
				})
			}
		}
	}

	// Check for added keys
	for key, currentValue := range current {
		if _, exists := baseline[key]; !exists {
			path := c.buildPath(basePath, key)
			changes = append(changes, Change{
				Type:        ChangeTypeAdded,
				Path:        path,
				Field:       key,
				OldValue:    nil,
				NewValue:    currentValue,
				Description: fmt.Sprintf("Configuration key '%s' was added", path),
			})
		}
	}

	return changes
}

func (c *SmartComparer) buildPath(basePath, key string) string {
	if basePath == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", basePath, key)
}

// Example type comparers

// EC2InstanceComparer specialized for AWS EC2 instances
type EC2InstanceComparer struct{}

func (c *EC2InstanceComparer) CompareResources(baseline, current types.Resource) []Change {
	// Implement EC2-specific comparison logic
	defaultComparer := &DefaultComparer{}
	changes := defaultComparer.CompareResources(baseline, current)

	// Add EC2-specific logic, like marking instance type changes as high severity
	for i := range changes {
		if changes[i].Path == "configuration.instance_type" {
			changes[i].Impact = "Instance type change may affect performance and cost"
		}
	}

	return changes
}

func (c *EC2InstanceComparer) GetCriticalFields() []string {
	return []string{"instance_type", "security_groups", "subnet_id", "iam_instance_profile"}
}

// SecurityGroupComparer specialized for security groups
type SecurityGroupComparer struct{}

func (c *SecurityGroupComparer) CompareResources(baseline, current types.Resource) []Change {
	defaultComparer := &DefaultComparer{}
	changes := defaultComparer.CompareResources(baseline, current)

	// Mark all security group changes as potentially high risk
	for i := range changes {
		if strings.Contains(changes[i].Path, "ingress") || strings.Contains(changes[i].Path, "egress") {
			changes[i].Impact = "Security rule change may affect network access"
		}
	}

	return changes
}

func (c *SecurityGroupComparer) GetCriticalFields() []string {
	return []string{"ingress", "egress", "rules"}
}

// KubernetesDeploymentComparer specialized for Kubernetes deployments
type KubernetesDeploymentComparer struct{}

func (c *KubernetesDeploymentComparer) CompareResources(baseline, current types.Resource) []Change {
	defaultComparer := &DefaultComparer{}
	changes := defaultComparer.CompareResources(baseline, current)

	// Add Kubernetes-specific logic
	for i := range changes {
		if changes[i].Path == "configuration.spec.replicas" {
			changes[i].Impact = "Replica count change affects availability and resource usage"
		}
	}

	return changes
}

func (c *KubernetesDeploymentComparer) GetCriticalFields() []string {
	return []string{"spec.replicas", "spec.template.spec.containers", "spec.strategy"}
}

// Example field comparers

// SecurityGroupFieldComparer for security group fields
type SecurityGroupFieldComparer struct{}

func (c *SecurityGroupFieldComparer) CompareField(path, field string, baseline, current interface{}) []Change {
	// Implement security-group-specific comparison
	defaultComparer := &DefaultComparer{}
	return defaultComparer.compareValues(path, field, baseline, current)
}

// TagFieldComparer for tag fields
type TagFieldComparer struct{}

func (c *TagFieldComparer) CompareField(path, field string, baseline, current interface{}) []Change {
	// Implement tag-specific comparison with special handling for certain tags
	defaultComparer := &DefaultComparer{}
	return defaultComparer.compareValues(path, field, baseline, current)
}
