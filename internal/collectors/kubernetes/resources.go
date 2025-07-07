package kubernetes

import (
	"context"
	"fmt"
	"sync"

	"github.com/yairfalse/wgo/pkg/types"
)

// ResourceType represents different Kubernetes resource types
type ResourceType string

const (
	// Workload resources
	ResourceTypeDeployment  ResourceType = "deployment"
	ResourceTypeStatefulSet ResourceType = "statefulset"
	ResourceTypeDaemonSet   ResourceType = "daemonset"
	
	// Service resources
	ResourceTypeService ResourceType = "service"
	ResourceTypeIngress ResourceType = "ingress"
	
	// Config resources
	ResourceTypeConfigMap ResourceType = "configmap"
	ResourceTypeSecret    ResourceType = "secret"
	
	// Storage resources
	ResourceTypePersistentVolume      ResourceType = "persistentvolume"
	ResourceTypePersistentVolumeClaim ResourceType = "persistentvolumeclaim"
	
	// Security resources
	ResourceTypeServiceAccount ResourceType = "serviceaccount"
	ResourceTypeRole           ResourceType = "role"
	ResourceTypeRoleBinding    ResourceType = "rolebinding"
)

// ResourceGroup represents a logical grouping of related resources
type ResourceGroup struct {
	Name      string         `json:"name"`
	Resources []ResourceType `json:"resources"`
}

// GetSupportedResourceGroups returns all supported resource groups
func GetSupportedResourceGroups() []ResourceGroup {
	return []ResourceGroup{
		{
			Name:      "workloads",
			Resources: []ResourceType{ResourceTypeDeployment, ResourceTypeStatefulSet, ResourceTypeDaemonSet},
		},
		{
			Name:      "services",
			Resources: []ResourceType{ResourceTypeService, ResourceTypeIngress},
		},
		{
			Name:      "config",
			Resources: []ResourceType{ResourceTypeConfigMap, ResourceTypeSecret},
		},
		{
			Name:      "storage",
			Resources: []ResourceType{ResourceTypePersistentVolume, ResourceTypePersistentVolumeClaim},
		},
		{
			Name:      "security",
			Resources: []ResourceType{ResourceTypeServiceAccount, ResourceTypeRole, ResourceTypeRoleBinding},
		},
	}
}

// GetAllSupportedResourceTypes returns all supported resource types
func GetAllSupportedResourceTypes() []ResourceType {
	var allTypes []ResourceType
	for _, group := range GetSupportedResourceGroups() {
		allTypes = append(allTypes, group.Resources...)
	}
	return allTypes
}

// ResourceCollectionStats holds statistics about resource collection
type ResourceCollectionStats struct {
	TotalResources      int                        `json:"total_resources"`
	ResourcesByType     map[ResourceType]int       `json:"resources_by_type"`
	ResourcesByGroup    map[string]int             `json:"resources_by_group"`
	ResourcesByNamespace map[string]int            `json:"resources_by_namespace"`
	CollectionErrors    []ResourceCollectionError  `json:"collection_errors,omitempty"`
}

// ResourceCollectionError represents an error that occurred during resource collection
type ResourceCollectionError struct {
	ResourceType ResourceType `json:"resource_type"`
	Namespace    string       `json:"namespace,omitempty"`
	Error        string       `json:"error"`
}

// CollectResourcesInParallel collects resources from multiple namespaces in parallel
func (k *KubernetesCollector) CollectResourcesInParallel(ctx context.Context, namespaces []string, resourceTypes []ResourceType) ([]types.Resource, *ResourceCollectionStats, error) {
	if len(namespaces) == 0 {
		namespaces = []string{""} // Default to all namespaces
	}
	
	if len(resourceTypes) == 0 {
		resourceTypes = GetAllSupportedResourceTypes()
	}
	
	stats := &ResourceCollectionStats{
		ResourcesByType:      make(map[ResourceType]int),
		ResourcesByGroup:     make(map[string]int),
		ResourcesByNamespace: make(map[string]int),
		CollectionErrors:     []ResourceCollectionError{},
	}
	
	var (
		allResources []types.Resource
		mu           sync.Mutex
		wg           sync.WaitGroup
	)
	
	// Collect resources for each namespace in parallel
	for _, namespace := range namespaces {
		wg.Add(1)
		go func(ns string) {
			defer wg.Done()
			
			nsResources, nsErrors := k.collectResourcesForNamespace(ctx, ns, resourceTypes)
			
			mu.Lock()
			allResources = append(allResources, nsResources...)
			stats.CollectionErrors = append(stats.CollectionErrors, nsErrors...)
			stats.ResourcesByNamespace[ns] = len(nsResources)
			mu.Unlock()
		}(namespace)
	}
	
	wg.Wait()
	
	// Calculate statistics
	stats.TotalResources = len(allResources)
	for _, resource := range allResources {
		resourceType := ResourceType(resource.Type)
		stats.ResourcesByType[resourceType]++
		
		// Find which group this resource belongs to
		for _, group := range GetSupportedResourceGroups() {
			for _, groupResourceType := range group.Resources {
				if groupResourceType == resourceType {
					stats.ResourcesByGroup[group.Name]++
					break
				}
			}
		}
	}
	
	return allResources, stats, nil
}

// collectResourcesForNamespace collects all specified resource types for a single namespace
func (k *KubernetesCollector) collectResourcesForNamespace(ctx context.Context, namespace string, resourceTypes []ResourceType) ([]types.Resource, []ResourceCollectionError) {
	var (
		resources []types.Resource
		errors    []ResourceCollectionError
	)
	
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case ResourceTypeDeployment:
			if deployments, err := k.client.GetDeployments(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, dep := range deployments {
					resource := k.normalizer.NormalizeDeployment(&dep)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeStatefulSet:
			if statefulSets, err := k.client.GetStatefulSets(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, ss := range statefulSets {
					resource := k.normalizer.NormalizeStatefulSet(&ss)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeDaemonSet:
			if daemonSets, err := k.client.GetDaemonSets(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, ds := range daemonSets {
					resource := k.normalizer.NormalizeDaemonSet(&ds)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeService:
			if services, err := k.client.GetServices(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, svc := range services {
					resource := k.normalizer.NormalizeService(&svc)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeIngress:
			if ingresses, err := k.client.GetIngresses(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, ing := range ingresses {
					resource := k.normalizer.NormalizeIngress(&ing)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeConfigMap:
			if configMaps, err := k.client.GetConfigMaps(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, cm := range configMaps {
					resource := k.normalizer.NormalizeConfigMap(&cm)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeSecret:
			if secrets, err := k.client.GetSecrets(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, secret := range secrets {
					resource := k.normalizer.NormalizeSecret(&secret)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypePersistentVolume:
			// PVs are cluster-wide, only collect once
			if namespace == "" || namespace == "default" {
				if pvs, err := k.client.GetPersistentVolumes(ctx); err != nil {
					errors = append(errors, ResourceCollectionError{
						ResourceType: resourceType,
						Error:        err.Error(),
					})
				} else {
					for _, pv := range pvs {
						resource := k.normalizer.NormalizePersistentVolume(&pv)
						resources = append(resources, resource)
					}
				}
			}
			
		case ResourceTypePersistentVolumeClaim:
			if pvcs, err := k.client.GetPersistentVolumeClaims(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, pvc := range pvcs {
					resource := k.normalizer.NormalizePersistentVolumeClaim(&pvc)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeServiceAccount:
			if serviceAccounts, err := k.client.GetServiceAccounts(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, sa := range serviceAccounts {
					resource := k.normalizer.NormalizeServiceAccount(&sa)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeRole:
			if roles, err := k.client.GetRoles(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, role := range roles {
					resource := k.normalizer.NormalizeRole(&role)
					resources = append(resources, resource)
				}
			}
			
		case ResourceTypeRoleBinding:
			if roleBindings, err := k.client.GetRoleBindings(ctx, namespace); err != nil {
				errors = append(errors, ResourceCollectionError{
					ResourceType: resourceType,
					Namespace:    namespace,
					Error:        err.Error(),
				})
			} else {
				for _, rb := range roleBindings {
					resource := k.normalizer.NormalizeRoleBinding(&rb)
					resources = append(resources, resource)
				}
			}
		}
	}
	
	return resources, errors
}

// FilterResourcesByLabels filters resources based on label selectors
func FilterResourcesByLabels(resources []types.Resource, labelSelector map[string]string) []types.Resource {
	if len(labelSelector) == 0 {
		return resources
	}
	
	var filtered []types.Resource
	for _, resource := range resources {
		matches := true
		for key, value := range labelSelector {
			if resourceValue, exists := resource.Tags[key]; !exists || resourceValue != value {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, resource)
		}
	}
	
	return filtered
}

// GroupResourcesByNamespace groups resources by their namespace
func GroupResourcesByNamespace(resources []types.Resource) map[string][]types.Resource {
	grouped := make(map[string][]types.Resource)
	
	for _, resource := range resources {
		namespace := resource.Namespace
		if namespace == "" {
			namespace = "cluster-wide"
		}
		grouped[namespace] = append(grouped[namespace], resource)
	}
	
	return grouped
}

// GroupResourcesByType groups resources by their type
func GroupResourcesByType(resources []types.Resource) map[string][]types.Resource {
	grouped := make(map[string][]types.Resource)
	
	for _, resource := range resources {
		grouped[resource.Type] = append(grouped[resource.Type], resource)
	}
	
	return grouped
}

// GetResourceTypeFromString converts a string to ResourceType
func GetResourceTypeFromString(s string) (ResourceType, error) {
	for _, resourceType := range GetAllSupportedResourceTypes() {
		if string(resourceType) == s {
			return resourceType, nil
		}
	}
	return "", fmt.Errorf("unsupported resource type: %s", s)
}

// ValidateResourceTypes validates that all provided resource types are supported
func ValidateResourceTypes(resourceTypes []string) error {
	supportedTypes := GetAllSupportedResourceTypes()
	supportedMap := make(map[string]bool)
	for _, t := range supportedTypes {
		supportedMap[string(t)] = true
	}
	
	for _, resourceType := range resourceTypes {
		if !supportedMap[resourceType] {
			return fmt.Errorf("unsupported resource type: %s", resourceType)
		}
	}
	
	return nil
}