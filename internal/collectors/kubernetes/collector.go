package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	wgoerrors "github.com/yairfalse/wgo/internal/errors"
	"github.com/yairfalse/wgo/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubernetesCollector implements the EnhancedCollector interface for Kubernetes
type KubernetesCollector struct {
	clientset  kubernetes.Interface
	config     *rest.Config
	client     *KubernetesClient
	normalizer *ResourceNormalizer
	version    string
}

// NewKubernetesCollector creates a new Kubernetes collector
func NewKubernetesCollector() collectors.EnhancedCollector {
	return &KubernetesCollector{
		client:     NewKubernetesClient(),
		normalizer: NewResourceNormalizer(),
		version:    "1.0.0",
	}
}

// Name returns the collector name
func (k *KubernetesCollector) Name() string {
	return "kubernetes"
}

// Status returns the current status of the collector
func (k *KubernetesCollector) Status() string {
	// Try to initialize client to check connectivity
	_, err := k.client.GetConfig("", "")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return "ready"
}

// Collect performs collection of Kubernetes resources
func (k *KubernetesCollector) Collect(ctx context.Context, config collectors.CollectorConfig) (*types.Snapshot, error) {
	startTime := time.Now()
	
	// Initialize Kubernetes client
	kubeConfig := k.extractKubernetesConfig(config)
	err := k.client.Initialize(kubeConfig.Context, kubeConfig.Kubeconfig)
	if err != nil {
		// Check if it's an authentication error
		if isKubernetesAuthError(err) {
			return nil, wgoerrors.New(wgoerrors.ErrorTypeAuthentication, wgoerrors.ProviderKubernetes,
				"Kubernetes authentication failed").
				WithCause(err.Error()).
				WithSolutions(
					"Check your kubeconfig file exists and is valid",
					"Verify kubectl can connect to the cluster",
					"Ensure the current context is correct",
					"Check if your cluster credentials have expired",
				).
				WithVerify("kubectl cluster-info").
				WithHelp("wgo validate kubernetes")
		}
		
		return nil, fmt.Errorf("failed to initialize Kubernetes client: %w", err)
	}
	
	// Create snapshot
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("k8s-snapshot-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "kubernetes",
		Resources: []types.Resource{},
		Metadata: types.SnapshotMetadata{
			CollectorVersion: k.version,
			CollectionTime:   0, // Will be set at the end
			ResourceCount:    0, // Will be set at the end
			Tags:             config.Tags,
		},
	}
	
	// Collect resources by type
	resources := []types.Resource{}
	
	// Collect workloads
	workloadResources, err := k.collectWorkloads(ctx, kubeConfig.Namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to collect workloads: %w", err)
	}
	resources = append(resources, workloadResources...)
	
	// Collect services
	serviceResources, err := k.collectServices(ctx, kubeConfig.Namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to collect services: %w", err)
	}
	resources = append(resources, serviceResources...)
	
	// Collect config resources
	configResources, err := k.collectConfig(ctx, kubeConfig.Namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to collect config: %w", err)
	}
	resources = append(resources, configResources...)
	
	// Collect storage resources
	storageResources, err := k.collectStorage(ctx, kubeConfig.Namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to collect storage: %w", err)
	}
	resources = append(resources, storageResources...)
	
	// Collect security resources
	securityResources, err := k.collectSecurity(ctx, kubeConfig.Namespaces)
	if err != nil {
		return nil, fmt.Errorf("failed to collect security: %w", err)
	}
	resources = append(resources, securityResources...)
	
	// Update snapshot
	snapshot.Resources = resources
	snapshot.Metadata.CollectionTime = time.Since(startTime)
	snapshot.Metadata.ResourceCount = len(resources)
	
	// Add regions and namespaces to metadata
	if len(kubeConfig.Namespaces) > 0 {
		snapshot.Metadata.Namespaces = kubeConfig.Namespaces
	}
	
	return snapshot, nil
}

// Validate validates the collector configuration
func (k *KubernetesCollector) Validate(config collectors.CollectorConfig) error {
	kubeConfig := k.extractKubernetesConfig(config)
	
	// Try to initialize client
	_, err := k.client.GetConfig(kubeConfig.Context, kubeConfig.Kubeconfig)
	if err != nil {
		return fmt.Errorf("invalid Kubernetes configuration: %w", err)
	}
	
	return nil
}

// AutoDiscover performs auto-discovery of Kubernetes configuration
func (k *KubernetesCollector) AutoDiscover() (collectors.CollectorConfig, error) {
	// Try to get default kubeconfig
	_, err := k.client.GetConfig("", "")
	if err != nil {
		return collectors.CollectorConfig{}, fmt.Errorf("failed to auto-discover Kubernetes config: %w", err)
	}
	
	// Get available contexts
	contexts, err := k.client.GetAvailableContexts("")
	if err != nil {
		contexts = []string{} // Not critical if we can't get contexts
	}
	
	// Get available namespaces
	namespaces, err := k.client.GetAvailableNamespaces()
	if err != nil {
		namespaces = []string{"default"} // Fallback to default
	}
	
	collectorConfig := collectors.CollectorConfig{
		Namespaces: namespaces,
		Config: map[string]interface{}{
			"contexts":   contexts,
			"kubeconfig": "", // Use default
		},
	}
	
	return collectorConfig, nil
}

// SupportedRegions returns supported regions (not applicable for Kubernetes)
func (k *KubernetesCollector) SupportedRegions() []string {
	return []string{} // Kubernetes doesn't have regions
}

// KubernetesConfig holds Kubernetes-specific configuration
type KubernetesConfig struct {
	Context    string   `json:"context"`
	Kubeconfig string   `json:"kubeconfig"`
	Namespaces []string `json:"namespaces"`
}

// extractKubernetesConfig extracts Kubernetes-specific config from CollectorConfig
func (k *KubernetesCollector) extractKubernetesConfig(config collectors.CollectorConfig) KubernetesConfig {
	kubeConfig := KubernetesConfig{
		Namespaces: config.Namespaces,
	}
	
	if config.Config != nil {
		if context, ok := config.Config["context"].(string); ok {
			kubeConfig.Context = context
		}
		if kubeconfig, ok := config.Config["kubeconfig"].(string); ok {
			kubeConfig.Kubeconfig = kubeconfig
		}
	}
	
	// Default to all namespaces if none specified
	if len(kubeConfig.Namespaces) == 0 {
		kubeConfig.Namespaces = []string{""}
	}
	
	return kubeConfig
}

// collectWorkloads collects Kubernetes workload resources
func (k *KubernetesCollector) collectWorkloads(ctx context.Context, namespaces []string) ([]types.Resource, error) {
	var resources []types.Resource
	
	for _, namespace := range namespaces {
		// Collect Deployments
		deployments, err := k.client.GetDeployments(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get deployments in namespace %s: %w", namespace, err)
		}
		for _, deployment := range deployments {
			resource := k.normalizer.NormalizeDeployment(&deployment)
			resources = append(resources, resource)
		}
		
		// Collect StatefulSets
		statefulSets, err := k.client.GetStatefulSets(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get statefulsets in namespace %s: %w", namespace, err)
		}
		for _, ss := range statefulSets {
			resource := k.normalizer.NormalizeStatefulSet(&ss)
			resources = append(resources, resource)
		}
		
		// Collect DaemonSets
		daemonSets, err := k.client.GetDaemonSets(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get daemonsets in namespace %s: %w", namespace, err)
		}
		for _, ds := range daemonSets {
			resource := k.normalizer.NormalizeDaemonSet(&ds)
			resources = append(resources, resource)
		}
	}
	
	return resources, nil
}

// collectServices collects Kubernetes service resources
func (k *KubernetesCollector) collectServices(ctx context.Context, namespaces []string) ([]types.Resource, error) {
	var resources []types.Resource
	
	for _, namespace := range namespaces {
		// Collect Services
		services, err := k.client.GetServices(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get services in namespace %s: %w", namespace, err)
		}
		for _, service := range services {
			resource := k.normalizer.NormalizeService(&service)
			resources = append(resources, resource)
		}
		
		// Collect Ingresses
		ingresses, err := k.client.GetIngresses(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get ingresses in namespace %s: %w", namespace, err)
		}
		for _, ingress := range ingresses {
			resource := k.normalizer.NormalizeIngress(&ingress)
			resources = append(resources, resource)
		}
	}
	
	return resources, nil
}

// collectConfig collects Kubernetes configuration resources
func (k *KubernetesCollector) collectConfig(ctx context.Context, namespaces []string) ([]types.Resource, error) {
	var resources []types.Resource
	
	for _, namespace := range namespaces {
		// Collect ConfigMaps
		configMaps, err := k.client.GetConfigMaps(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get configmaps in namespace %s: %w", namespace, err)
		}
		for _, cm := range configMaps {
			resource := k.normalizer.NormalizeConfigMap(&cm)
			resources = append(resources, resource)
		}
		
		// Collect Secrets
		secrets, err := k.client.GetSecrets(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets in namespace %s: %w", namespace, err)
		}
		for _, secret := range secrets {
			resource := k.normalizer.NormalizeSecret(&secret)
			resources = append(resources, resource)
		}
	}
	
	return resources, nil
}

// collectStorage collects Kubernetes storage resources
func (k *KubernetesCollector) collectStorage(ctx context.Context, namespaces []string) ([]types.Resource, error) {
	var resources []types.Resource
	
	// Collect PersistentVolumes (cluster-wide)
	pvs, err := k.client.GetPersistentVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get persistent volumes: %w", err)
	}
	for _, pv := range pvs {
		resource := k.normalizer.NormalizePersistentVolume(&pv)
		resources = append(resources, resource)
	}
	
	for _, namespace := range namespaces {
		// Collect PersistentVolumeClaims
		pvcs, err := k.client.GetPersistentVolumeClaims(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get persistent volume claims in namespace %s: %w", namespace, err)
		}
		for _, pvc := range pvcs {
			resource := k.normalizer.NormalizePersistentVolumeClaim(&pvc)
			resources = append(resources, resource)
		}
	}
	
	return resources, nil
}

// collectSecurity collects Kubernetes security resources
func (k *KubernetesCollector) collectSecurity(ctx context.Context, namespaces []string) ([]types.Resource, error) {
	var resources []types.Resource
	
	for _, namespace := range namespaces {
		// Collect ServiceAccounts
		serviceAccounts, err := k.client.GetServiceAccounts(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get service accounts in namespace %s: %w", namespace, err)
		}
		for _, sa := range serviceAccounts {
			resource := k.normalizer.NormalizeServiceAccount(&sa)
			resources = append(resources, resource)
		}
		
		// Collect Roles
		roles, err := k.client.GetRoles(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get roles in namespace %s: %w", namespace, err)
		}
		for _, role := range roles {
			resource := k.normalizer.NormalizeRole(&role)
			resources = append(resources, resource)
		}
		
		// Collect RoleBindings
		roleBindings, err := k.client.GetRoleBindings(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to get role bindings in namespace %s: %w", namespace, err)
		}
		for _, rb := range roleBindings {
			resource := k.normalizer.NormalizeRoleBinding(&rb)
			resources = append(resources, resource)
		}
	}
	
	return resources, nil
}

// isKubernetesAuthError checks if an error is related to Kubernetes authentication
func isKubernetesAuthError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Common Kubernetes authentication error patterns
	authErrorPatterns := []string{
		"unauthorized",
		"forbidden",
		"authentication required",
		"invalid bearer token",
		"token has expired",
		"certificate has expired", 
		"x509: certificate",
		"unable to authenticate",
		"permission denied",
		"access denied",
		"credentials",
		"kubeconfig",
		"context",
		"authentication",
	}
	
	for _, pattern := range authErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	
	return false
}