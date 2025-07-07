package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesClient handles Kubernetes API interactions
type KubernetesClient struct {
	clientset kubernetes.Interface
	config    *rest.Config
}

// NewKubernetesClient creates a new Kubernetes client
func NewKubernetesClient() *KubernetesClient {
	return &KubernetesClient{}
}

// Initialize initializes the Kubernetes client with given context and kubeconfig
func (k *KubernetesClient) Initialize(context, kubeconfig string) error {
	config, err := k.GetConfig(context, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	k.config = config
	k.clientset = clientset
	return nil
}

// GetConfig returns Kubernetes configuration
func (k *KubernetesClient) GetConfig(context, kubeconfig string) (*rest.Config, error) {
	// If kubeconfig is not specified, use default paths
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// First try in-cluster config
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	// Then try kubeconfig file
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		configLoadingRules.ExplicitPath = kubeconfig
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		configOverrides.CurrentContext = context
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		configLoadingRules,
		configOverrides,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config, nil
}

// GetAvailableContexts returns available Kubernetes contexts
func (k *KubernetesClient) GetAvailableContexts(kubeconfig string) ([]string, error) {
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		configLoadingRules.ExplicitPath = kubeconfig
	}

	config, err := configLoadingRules.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	var contexts []string
	for contextName := range config.Contexts {
		contexts = append(contexts, contextName)
	}

	return contexts, nil
}

// GetAvailableNamespaces returns available namespaces in the cluster
func (k *KubernetesClient) GetAvailableNamespaces() ([]string, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	namespaces, err := k.clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var nsNames []string
	for _, ns := range namespaces.Items {
		nsNames = append(nsNames, ns.Name)
	}

	return nsNames, nil
}

// Workload resources

// GetDeployments returns all deployments in the specified namespace
func (k *KubernetesClient) GetDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	deployments, err := k.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	return deployments.Items, nil
}

// GetStatefulSets returns all statefulsets in the specified namespace
func (k *KubernetesClient) GetStatefulSets(ctx context.Context, namespace string) ([]appsv1.StatefulSet, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	statefulSets, err := k.clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets: %w", err)
	}

	return statefulSets.Items, nil
}

// GetDaemonSets returns all daemonsets in the specified namespace
func (k *KubernetesClient) GetDaemonSets(ctx context.Context, namespace string) ([]appsv1.DaemonSet, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	daemonSets, err := k.clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets: %w", err)
	}

	return daemonSets.Items, nil
}

// Service resources

// GetServices returns all services in the specified namespace
func (k *KubernetesClient) GetServices(ctx context.Context, namespace string) ([]corev1.Service, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	services, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	return services.Items, nil
}

// GetIngresses returns all ingresses in the specified namespace
func (k *KubernetesClient) GetIngresses(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	ingresses, err := k.clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	return ingresses.Items, nil
}

// Config resources

// GetConfigMaps returns all configmaps in the specified namespace
func (k *KubernetesClient) GetConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	configMaps, err := k.clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps: %w", err)
	}

	return configMaps.Items, nil
}

// GetSecrets returns all secrets in the specified namespace
func (k *KubernetesClient) GetSecrets(ctx context.Context, namespace string) ([]corev1.Secret, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	secrets, err := k.clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secrets.Items, nil
}

// Storage resources

// GetPersistentVolumes returns all persistent volumes (cluster-wide)
func (k *KubernetesClient) GetPersistentVolumes(ctx context.Context) ([]corev1.PersistentVolume, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	pvs, err := k.clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volumes: %w", err)
	}

	return pvs.Items, nil
}

// GetPersistentVolumeClaims returns all persistent volume claims in the specified namespace
func (k *KubernetesClient) GetPersistentVolumeClaims(ctx context.Context, namespace string) ([]corev1.PersistentVolumeClaim, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	pvcs, err := k.clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}

	return pvcs.Items, nil
}

// Security resources

// GetServiceAccounts returns all service accounts in the specified namespace
func (k *KubernetesClient) GetServiceAccounts(ctx context.Context, namespace string) ([]corev1.ServiceAccount, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	serviceAccounts, err := k.clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list service accounts: %w", err)
	}

	return serviceAccounts.Items, nil
}

// GetRoles returns all roles in the specified namespace
func (k *KubernetesClient) GetRoles(ctx context.Context, namespace string) ([]rbacv1.Role, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	roles, err := k.clientset.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	return roles.Items, nil
}

// GetRoleBindings returns all role bindings in the specified namespace
func (k *KubernetesClient) GetRoleBindings(ctx context.Context, namespace string) ([]rbacv1.RoleBinding, error) {
	if k.clientset == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	roleBindings, err := k.clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	return roleBindings.Items, nil
}

// homeDir returns the user's home directory
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // Windows
}