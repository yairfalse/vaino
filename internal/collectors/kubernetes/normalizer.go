package kubernetes

import (
	"fmt"

	"github.com/yairfalse/wgo/pkg/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// ResourceNormalizer converts Kubernetes resources to WGO Resource format
type ResourceNormalizer struct{}

// NewResourceNormalizer creates a new resource normalizer
func NewResourceNormalizer() *ResourceNormalizer {
	return &ResourceNormalizer{}
}

// NormalizeDeployment converts a Kubernetes Deployment to WGO Resource
func (n *ResourceNormalizer) NormalizeDeployment(dep *appsv1.Deployment) types.Resource {
	config := n.extractDeploymentConfig(dep)
	
	return types.Resource{
		ID:            fmt.Sprintf("deployment/%s", dep.Name),
		Type:          "deployment",
		Name:          dep.Name,
		Provider:      "kubernetes",
		Namespace:     dep.Namespace,
		Configuration: config,
		Tags:          dep.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: dep.CreationTimestamp.Time,
			Version:   dep.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":        string(dep.UID),
				"generation": dep.Generation,
				"replicas":   *dep.Spec.Replicas,
				"ready_replicas": dep.Status.ReadyReplicas,
			},
		},
	}
}

// NormalizeStatefulSet converts a Kubernetes StatefulSet to WGO Resource
func (n *ResourceNormalizer) NormalizeStatefulSet(ss *appsv1.StatefulSet) types.Resource {
	config := n.extractStatefulSetConfig(ss)
	
	return types.Resource{
		ID:            fmt.Sprintf("statefulset/%s", ss.Name),
		Type:          "statefulset",
		Name:          ss.Name,
		Provider:      "kubernetes",
		Namespace:     ss.Namespace,
		Configuration: config,
		Tags:          ss.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: ss.CreationTimestamp.Time,
			Version:   ss.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":        string(ss.UID),
				"generation": ss.Generation,
				"replicas":   *ss.Spec.Replicas,
				"ready_replicas": ss.Status.ReadyReplicas,
			},
		},
	}
}

// NormalizeDaemonSet converts a Kubernetes DaemonSet to WGO Resource
func (n *ResourceNormalizer) NormalizeDaemonSet(ds *appsv1.DaemonSet) types.Resource {
	config := n.extractDaemonSetConfig(ds)
	
	return types.Resource{
		ID:            fmt.Sprintf("daemonset/%s", ds.Name),
		Type:          "daemonset",
		Name:          ds.Name,
		Provider:      "kubernetes",
		Namespace:     ds.Namespace,
		Configuration: config,
		Tags:          ds.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: ds.CreationTimestamp.Time,
			Version:   ds.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":        string(ds.UID),
				"generation": ds.Generation,
				"desired_number_scheduled": ds.Status.DesiredNumberScheduled,
				"number_ready": ds.Status.NumberReady,
			},
		},
	}
}

// NormalizeService converts a Kubernetes Service to WGO Resource
func (n *ResourceNormalizer) NormalizeService(svc *corev1.Service) types.Resource {
	config := n.extractServiceConfig(svc)
	
	return types.Resource{
		ID:            fmt.Sprintf("service/%s", svc.Name),
		Type:          "service",
		Name:          svc.Name,
		Provider:      "kubernetes",
		Namespace:     svc.Namespace,
		Configuration: config,
		Tags:          svc.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: svc.CreationTimestamp.Time,
			Version:   svc.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":          string(svc.UID),
				"cluster_ip":   svc.Spec.ClusterIP,
				"service_type": string(svc.Spec.Type),
			},
		},
	}
}

// NormalizeIngress converts a Kubernetes Ingress to WGO Resource
func (n *ResourceNormalizer) NormalizeIngress(ing *networkingv1.Ingress) types.Resource {
	config := n.extractIngressConfig(ing)
	
	return types.Resource{
		ID:            fmt.Sprintf("ingress/%s", ing.Name),
		Type:          "ingress",
		Name:          ing.Name,
		Provider:      "kubernetes",
		Namespace:     ing.Namespace,
		Configuration: config,
		Tags:          ing.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: ing.CreationTimestamp.Time,
			Version:   ing.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid": string(ing.UID),
			},
		},
	}
}

// NormalizeConfigMap converts a Kubernetes ConfigMap to WGO Resource
func (n *ResourceNormalizer) NormalizeConfigMap(cm *corev1.ConfigMap) types.Resource {
	config := n.extractConfigMapConfig(cm)
	
	return types.Resource{
		ID:            fmt.Sprintf("configmap/%s", cm.Name),
		Type:          "configmap",
		Name:          cm.Name,
		Provider:      "kubernetes",
		Namespace:     cm.Namespace,
		Configuration: config,
		Tags:          cm.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: cm.CreationTimestamp.Time,
			Version:   cm.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid": string(cm.UID),
				"data_keys": len(cm.Data),
			},
		},
	}
}

// NormalizeSecret converts a Kubernetes Secret to WGO Resource
func (n *ResourceNormalizer) NormalizeSecret(secret *corev1.Secret) types.Resource {
	config := n.extractSecretConfig(secret)
	
	return types.Resource{
		ID:            fmt.Sprintf("secret/%s", secret.Name),
		Type:          "secret",
		Name:          secret.Name,
		Provider:      "kubernetes",
		Namespace:     secret.Namespace,
		Configuration: config,
		Tags:          secret.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: secret.CreationTimestamp.Time,
			Version:   secret.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":         string(secret.UID),
				"secret_type": string(secret.Type),
				"data_keys":   len(secret.Data),
			},
		},
	}
}

// NormalizePersistentVolume converts a Kubernetes PersistentVolume to WGO Resource
func (n *ResourceNormalizer) NormalizePersistentVolume(pv *corev1.PersistentVolume) types.Resource {
	config := n.extractPersistentVolumeConfig(pv)
	
	return types.Resource{
		ID:            fmt.Sprintf("persistentvolume/%s", pv.Name),
		Type:          "persistentvolume",
		Name:          pv.Name,
		Provider:      "kubernetes",
		Configuration: config,
		Tags:          pv.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: pv.CreationTimestamp.Time,
			Version:   pv.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":    string(pv.UID),
				"phase":  string(pv.Status.Phase),
				"access_modes": pv.Spec.AccessModes,
			},
		},
	}
}

// NormalizePersistentVolumeClaim converts a Kubernetes PVC to WGO Resource
func (n *ResourceNormalizer) NormalizePersistentVolumeClaim(pvc *corev1.PersistentVolumeClaim) types.Resource {
	config := n.extractPersistentVolumeClaimConfig(pvc)
	
	return types.Resource{
		ID:            fmt.Sprintf("persistentvolumeclaim/%s", pvc.Name),
		Type:          "persistentvolumeclaim",
		Name:          pvc.Name,
		Provider:      "kubernetes",
		Namespace:     pvc.Namespace,
		Configuration: config,
		Tags:          pvc.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: pvc.CreationTimestamp.Time,
			Version:   pvc.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid":    string(pvc.UID),
				"phase":  string(pvc.Status.Phase),
				"volume_name": pvc.Spec.VolumeName,
			},
		},
	}
}

// NormalizeServiceAccount converts a Kubernetes ServiceAccount to WGO Resource
func (n *ResourceNormalizer) NormalizeServiceAccount(sa *corev1.ServiceAccount) types.Resource {
	config := n.extractServiceAccountConfig(sa)
	
	return types.Resource{
		ID:            fmt.Sprintf("serviceaccount/%s", sa.Name),
		Type:          "serviceaccount",
		Name:          sa.Name,
		Provider:      "kubernetes",
		Namespace:     sa.Namespace,
		Configuration: config,
		Tags:          sa.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: sa.CreationTimestamp.Time,
			Version:   sa.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid": string(sa.UID),
				"secrets_count": len(sa.Secrets),
			},
		},
	}
}

// NormalizeRole converts a Kubernetes Role to WGO Resource
func (n *ResourceNormalizer) NormalizeRole(role *rbacv1.Role) types.Resource {
	config := n.extractRoleConfig(role)
	
	return types.Resource{
		ID:            fmt.Sprintf("role/%s", role.Name),
		Type:          "role",
		Name:          role.Name,
		Provider:      "kubernetes",
		Namespace:     role.Namespace,
		Configuration: config,
		Tags:          role.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: role.CreationTimestamp.Time,
			Version:   role.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid": string(role.UID),
				"rules_count": len(role.Rules),
			},
		},
	}
}

// NormalizeRoleBinding converts a Kubernetes RoleBinding to WGO Resource
func (n *ResourceNormalizer) NormalizeRoleBinding(rb *rbacv1.RoleBinding) types.Resource {
	config := n.extractRoleBindingConfig(rb)
	
	return types.Resource{
		ID:            fmt.Sprintf("rolebinding/%s", rb.Name),
		Type:          "rolebinding",
		Name:          rb.Name,
		Provider:      "kubernetes",
		Namespace:     rb.Namespace,
		Configuration: config,
		Tags:          rb.Labels,
		Metadata: types.ResourceMetadata{
			CreatedAt: rb.CreationTimestamp.Time,
			Version:   rb.ResourceVersion,
			AdditionalData: map[string]interface{}{
				"uid": string(rb.UID),
				"subjects_count": len(rb.Subjects),
			},
		},
	}
}

// Configuration extraction methods

func (n *ResourceNormalizer) extractDeploymentConfig(dep *appsv1.Deployment) map[string]interface{} {
	config := make(map[string]interface{})
	
	// Basic deployment info
	config["replicas"] = dep.Spec.Replicas
	config["selector"] = dep.Spec.Selector
	config["strategy"] = dep.Spec.Strategy
	
	// Template spec (without data values for security)
	if dep.Spec.Template.Spec.Containers != nil {
		var containers []map[string]interface{}
		for _, container := range dep.Spec.Template.Spec.Containers {
			containerConfig := map[string]interface{}{
				"name":  container.Name,
				"image": container.Image,
				"ports": container.Ports,
				"resources": container.Resources,
			}
			containers = append(containers, containerConfig)
		}
		config["containers"] = containers
	}
	
	return config
}

func (n *ResourceNormalizer) extractStatefulSetConfig(ss *appsv1.StatefulSet) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["replicas"] = ss.Spec.Replicas
	config["selector"] = ss.Spec.Selector
	config["service_name"] = ss.Spec.ServiceName
	config["update_strategy"] = ss.Spec.UpdateStrategy
	
	if ss.Spec.Template.Spec.Containers != nil {
		var containers []map[string]interface{}
		for _, container := range ss.Spec.Template.Spec.Containers {
			containerConfig := map[string]interface{}{
				"name":  container.Name,
				"image": container.Image,
				"ports": container.Ports,
				"resources": container.Resources,
			}
			containers = append(containers, containerConfig)
		}
		config["containers"] = containers
	}
	
	return config
}

func (n *ResourceNormalizer) extractDaemonSetConfig(ds *appsv1.DaemonSet) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["selector"] = ds.Spec.Selector
	config["update_strategy"] = ds.Spec.UpdateStrategy
	
	if ds.Spec.Template.Spec.Containers != nil {
		var containers []map[string]interface{}
		for _, container := range ds.Spec.Template.Spec.Containers {
			containerConfig := map[string]interface{}{
				"name":  container.Name,
				"image": container.Image,
				"ports": container.Ports,
				"resources": container.Resources,
			}
			containers = append(containers, containerConfig)
		}
		config["containers"] = containers
	}
	
	return config
}

func (n *ResourceNormalizer) extractServiceConfig(svc *corev1.Service) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["type"] = string(svc.Spec.Type)
	config["cluster_ip"] = svc.Spec.ClusterIP
	config["ports"] = svc.Spec.Ports
	config["selector"] = svc.Spec.Selector
	
	if svc.Spec.ExternalIPs != nil {
		config["external_ips"] = svc.Spec.ExternalIPs
	}
	
	return config
}

func (n *ResourceNormalizer) extractIngressConfig(ing *networkingv1.Ingress) map[string]interface{} {
	config := make(map[string]interface{})
	
	if ing.Spec.IngressClassName != nil {
		config["ingress_class_name"] = *ing.Spec.IngressClassName
	}
	
	config["rules"] = ing.Spec.Rules
	
	if ing.Spec.TLS != nil {
		config["tls"] = ing.Spec.TLS
	}
	
	return config
}

func (n *ResourceNormalizer) extractConfigMapConfig(cm *corev1.ConfigMap) map[string]interface{} {
	config := make(map[string]interface{})
	
	// Store keys but not values for security
	if cm.Data != nil {
		var keys []string
		for key := range cm.Data {
			keys = append(keys, key)
		}
		config["data_keys"] = keys
	}
	
	if cm.BinaryData != nil {
		var binaryKeys []string
		for key := range cm.BinaryData {
			binaryKeys = append(binaryKeys, key)
		}
		config["binary_data_keys"] = binaryKeys
	}
	
	return config
}

func (n *ResourceNormalizer) extractSecretConfig(secret *corev1.Secret) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["type"] = string(secret.Type)
	
	// Store keys but not values for security
	if secret.Data != nil {
		var keys []string
		for key := range secret.Data {
			keys = append(keys, key)
		}
		config["data_keys"] = keys
	}
	
	return config
}

func (n *ResourceNormalizer) extractPersistentVolumeConfig(pv *corev1.PersistentVolume) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["capacity"] = pv.Spec.Capacity
	config["access_modes"] = pv.Spec.AccessModes
	config["reclaim_policy"] = string(pv.Spec.PersistentVolumeReclaimPolicy)
	config["volume_mode"] = pv.Spec.VolumeMode
	
	// Sanitize volume source (remove sensitive data)
	if pv.Spec.PersistentVolumeSource.AWSElasticBlockStore != nil {
		config["volume_source"] = map[string]interface{}{
			"type": "aws_ebs",
			"volume_id": pv.Spec.PersistentVolumeSource.AWSElasticBlockStore.VolumeID,
		}
	} else if pv.Spec.PersistentVolumeSource.GCEPersistentDisk != nil {
		config["volume_source"] = map[string]interface{}{
			"type": "gce_pd",
			"pd_name": pv.Spec.PersistentVolumeSource.GCEPersistentDisk.PDName,
		}
	}
	
	return config
}

func (n *ResourceNormalizer) extractPersistentVolumeClaimConfig(pvc *corev1.PersistentVolumeClaim) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["access_modes"] = pvc.Spec.AccessModes
	config["resources"] = pvc.Spec.Resources
	config["volume_name"] = pvc.Spec.VolumeName
	config["storage_class_name"] = pvc.Spec.StorageClassName
	config["volume_mode"] = pvc.Spec.VolumeMode
	
	return config
}

func (n *ResourceNormalizer) extractServiceAccountConfig(sa *corev1.ServiceAccount) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["automount_service_account_token"] = sa.AutomountServiceAccountToken
	
	if sa.ImagePullSecrets != nil {
		var imagePullSecrets []string
		for _, secret := range sa.ImagePullSecrets {
			imagePullSecrets = append(imagePullSecrets, secret.Name)
		}
		config["image_pull_secrets"] = imagePullSecrets
	}
	
	if sa.Secrets != nil {
		var secrets []string
		for _, secret := range sa.Secrets {
			secrets = append(secrets, secret.Name)
		}
		config["secrets"] = secrets
	}
	
	return config
}

func (n *ResourceNormalizer) extractRoleConfig(role *rbacv1.Role) map[string]interface{} {
	config := make(map[string]interface{})
	
	// Sanitize rules - remove sensitive information
	var rules []map[string]interface{}
	for _, rule := range role.Rules {
		ruleConfig := map[string]interface{}{
			"api_groups": rule.APIGroups,
			"resources": rule.Resources,
			"verbs": rule.Verbs,
		}
		if rule.ResourceNames != nil {
			ruleConfig["resource_names"] = rule.ResourceNames
		}
		rules = append(rules, ruleConfig)
	}
	config["rules"] = rules
	
	return config
}

func (n *ResourceNormalizer) extractRoleBindingConfig(rb *rbacv1.RoleBinding) map[string]interface{} {
	config := make(map[string]interface{})
	
	config["role_ref"] = rb.RoleRef
	
	var subjects []map[string]interface{}
	for _, subject := range rb.Subjects {
		subjectConfig := map[string]interface{}{
			"kind": subject.Kind,
			"name": subject.Name,
		}
		if subject.Namespace != "" {
			subjectConfig["namespace"] = subject.Namespace
		}
		subjects = append(subjects, subjectConfig)
	}
	config["subjects"] = subjects
	
	return config
}

// sanitizeLabels removes sensitive labels
func (n *ResourceNormalizer) sanitizeLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	
	sanitized := make(map[string]string)
	for key, value := range labels {
		// Skip sensitive label keys
		if key != "kubernetes.io/secret" && key != "secrets" {
			sanitized[key] = value
		}
	}
	
	return sanitized
}