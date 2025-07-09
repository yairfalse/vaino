package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/pkg/types"
)

// ConcurrentKubernetesCollector implements parallel resource collection for Kubernetes
type ConcurrentKubernetesCollector struct {
	*KubernetesCollector
	maxWorkers int
	timeout    time.Duration
}

// K8sResourceCollectionResult holds the result of collecting Kubernetes resources
type K8sResourceCollectionResult struct {
	ResourceType string
	Namespace    string
	Resources    []types.Resource
	Error        error
	Duration     time.Duration
}

// NewConcurrentKubernetesCollector creates a new concurrent Kubernetes collector
func NewConcurrentKubernetesCollector(maxWorkers int, timeout time.Duration) collectors.EnhancedCollector {
	if maxWorkers <= 0 {
		maxWorkers = 8 // Default to 8 concurrent operations
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	return &ConcurrentKubernetesCollector{
		KubernetesCollector: NewKubernetesCollector().(*KubernetesCollector),
		maxWorkers:          maxWorkers,
		timeout:             timeout,
	}
}

// CollectConcurrent performs concurrent resource collection across all Kubernetes resources
func (c *ConcurrentKubernetesCollector) CollectConcurrent(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	// Initialize client
	client, err := c.initializeClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kubernetes client: %w", err)
	}
	c.client = client

	// Determine namespaces to scan
	namespaces := config.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{"default", "kube-system"}
	}

	// Create context with timeout
	collectCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Results channel for concurrent operations
	results := make(chan K8sResourceCollectionResult, len(namespaces)*8) // 8 resource types per namespace

	// Wait group for tracking goroutines
	var wg sync.WaitGroup

	// Define Kubernetes resource types to collect
	resourceTypes := []string{
		"pods", "services", "deployments", "replicasets",
		"configmaps", "secrets", "persistentvolumes", "persistentvolumeclaims",
	}

	// Launch concurrent resource collection per namespace
	for _, namespace := range namespaces {
		for _, resourceType := range resourceTypes {
			wg.Add(1)
			go c.collectK8sResource(collectCtx, namespace, resourceType, results, &wg)
		}
	}

	// Close results channel when all collections complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var allResources []types.Resource
	collectionErrors := make([]error, 0)

	for result := range results {
		if result.Error != nil {
			collectionErrors = append(collectionErrors,
				fmt.Errorf("%s/%s collection failed: %w", result.Namespace, result.ResourceType, result.Error))
		} else {
			allResources = append(allResources, result.Resources...)
		}
	}

	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("k8s-concurrent-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "kubernetes",
		Resources: allResources,
		Metadata: types.SnapshotMetadata{
			CollectorVersion: "1.0.0",
			ResourceCount:    len(allResources),
			AdditionalData: map[string]interface{}{
				"namespaces":         namespaces,
				"concurrent_enabled": true,
				"collection_errors":  len(collectionErrors),
			},
		},
	}

	return snapshot, nil
}

// collectK8sResource collects resources for a specific Kubernetes resource type in a namespace
func (c *ConcurrentKubernetesCollector) collectK8sResource(
	ctx context.Context,
	namespace, resourceType string,
	results chan<- K8sResourceCollectionResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	startTime := time.Now()
	result := K8sResourceCollectionResult{
		ResourceType: resourceType,
		Namespace:    namespace,
		Resources:    make([]types.Resource, 0),
	}

	var err error
	switch resourceType {
	case "pods":
		result.Resources, err = c.collectPods(ctx, namespace)
	case "services":
		result.Resources, err = c.collectServices(ctx, namespace)
	case "deployments":
		result.Resources, err = c.collectDeployments(ctx, namespace)
	case "replicasets":
		result.Resources, err = c.collectReplicaSets(ctx, namespace)
	case "configmaps":
		result.Resources, err = c.collectConfigMaps(ctx, namespace)
	case "secrets":
		result.Resources, err = c.collectSecrets(ctx, namespace)
	case "persistentvolumes":
		result.Resources, err = c.collectPersistentVolumes(ctx)
	case "persistentvolumeclaims":
		result.Resources, err = c.collectPersistentVolumeClaims(ctx, namespace)
	default:
		err = fmt.Errorf("unknown resource type: %s", resourceType)
	}

	result.Error = err
	result.Duration = time.Since(startTime)
	results <- result
}

// collectPods collects pods from a namespace
func (c *ConcurrentKubernetesCollector) collectPods(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation - in real implementation, this would use the Kubernetes client
	return []types.Resource{
		{
			ID:        fmt.Sprintf("pod-%s-%d", namespace, time.Now().Unix()),
			Type:      "pod",
			Name:      "webapp-pod",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"status": "Running",
				"phase":  "Running",
			},
			Metadata: types.ResourceMetadata{
				CreatedAt: time.Now(),
				Version:   "1",
			},
		},
	}, nil
}

// collectServices collects services from a namespace
func (c *ConcurrentKubernetesCollector) collectServices(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:        fmt.Sprintf("service-%s-%d", namespace, time.Now().Unix()),
			Type:      "service",
			Name:      "webapp-service",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"type":       "ClusterIP",
				"cluster_ip": "10.0.0.1",
				"ports":      []int{80, 443},
			},
		},
	}, nil
}

// collectDeployments collects deployments from a namespace
func (c *ConcurrentKubernetesCollector) collectDeployments(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:        fmt.Sprintf("deployment-%s-%d", namespace, time.Now().Unix()),
			Type:      "deployment",
			Name:      "webapp-deployment",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"replicas":           3,
				"available_replicas": 3,
				"ready_replicas":     3,
			},
		},
	}, nil
}

// collectReplicaSets collects replica sets from a namespace
func (c *ConcurrentKubernetesCollector) collectReplicaSets(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:        fmt.Sprintf("replicaset-%s-%d", namespace, time.Now().Unix()),
			Type:      "replicaset",
			Name:      "webapp-rs",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"replicas":       3,
				"ready_replicas": 3,
			},
		},
	}, nil
}

// collectConfigMaps collects config maps from a namespace
func (c *ConcurrentKubernetesCollector) collectConfigMaps(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:        fmt.Sprintf("configmap-%s-%d", namespace, time.Now().Unix()),
			Type:      "configmap",
			Name:      "app-config",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"data": map[string]string{
					"app.properties": "debug=true",
				},
			},
		},
	}, nil
}

// collectSecrets collects secrets from a namespace
func (c *ConcurrentKubernetesCollector) collectSecrets(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:        fmt.Sprintf("secret-%s-%d", namespace, time.Now().Unix()),
			Type:      "secret",
			Name:      "app-secret",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"type": "Opaque",
			},
		},
	}, nil
}

// collectPersistentVolumes collects persistent volumes (cluster-wide)
func (c *ConcurrentKubernetesCollector) collectPersistentVolumes(ctx context.Context) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:       fmt.Sprintf("pv-%d", time.Now().Unix()),
			Type:     "persistentvolume",
			Name:     "data-pv",
			Provider: "kubernetes",
			Configuration: map[string]interface{}{
				"capacity": "10Gi",
				"status":   "Available",
			},
		},
	}, nil
}

// collectPersistentVolumeClaims collects persistent volume claims from a namespace
func (c *ConcurrentKubernetesCollector) collectPersistentVolumeClaims(ctx context.Context, namespace string) ([]types.Resource, error) {
	// Mock implementation
	return []types.Resource{
		{
			ID:        fmt.Sprintf("pvc-%s-%d", namespace, time.Now().Unix()),
			Type:      "persistentvolumeclaim",
			Name:      "data-pvc",
			Provider:  "kubernetes",
			Namespace: namespace,
			Configuration: map[string]interface{}{
				"capacity": "10Gi",
				"status":   "Bound",
			},
		},
	}, nil
}

// Enhanced collection with parallel namespace operations
func (c *ConcurrentKubernetesCollector) CollectMultiNamespace(ctx context.Context, namespaces []string, resourceTypes []string) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect resources from all namespaces concurrently
	for _, namespace := range namespaces {
		wg.Add(1)
		go func(ns string) {
			defer wg.Done()

			config := collectors.CollectorConfig{
				Namespaces: []string{ns},
			}

			snapshot, err := c.CollectConcurrent(ctx, config)
			if err != nil {
				return // Skip failed namespaces
			}

			mu.Lock()
			allResources = append(allResources, snapshot.Resources...)
			mu.Unlock()
		}(namespace)
	}

	wg.Wait()
	return allResources, nil
}

// Enhanced collection with resource type filtering
func (c *ConcurrentKubernetesCollector) CollectFilteredResources(ctx context.Context, namespace string, resourceTypes []string) ([]types.Resource, error) {
	var allResources []types.Resource
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Collect specific resource types concurrently
	for _, resourceType := range resourceTypes {
		wg.Add(1)
		go func(rType string) {
			defer wg.Done()

			results := make(chan K8sResourceCollectionResult, 1)
			var innerWg sync.WaitGroup

			innerWg.Add(1)
			c.collectK8sResource(ctx, namespace, rType, results, &innerWg)

			innerWg.Wait()
			close(results)

			for result := range results {
				if result.Error == nil {
					mu.Lock()
					allResources = append(allResources, result.Resources...)
					mu.Unlock()
				}
			}
		}(resourceType)
	}

	wg.Wait()
	return allResources, nil
}

// initializeClient initializes the Kubernetes client
func (c *ConcurrentKubernetesCollector) initializeClient(config collectors.CollectorConfig) (*KubernetesClient, error) {
	// Extract context from config
	context := ""
	if config.Config != nil {
		if contexts, ok := config.Config["contexts"].([]string); ok && len(contexts) > 0 {
			context = contexts[0]
		}
	}

	// Create client
	client := NewKubernetesClient()
	err := client.Initialize(context, "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kubernetes client: %w", err)
	}

	return client, nil
}

// Override the Collect method to use concurrent collection
func (c *ConcurrentKubernetesCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	return c.CollectConcurrent(ctx, config)
}
