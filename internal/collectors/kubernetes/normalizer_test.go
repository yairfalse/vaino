package kubernetes

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestResourceNormalizer_NormalizeDeployment(t *testing.T) {
	normalizer := NewResourceNormalizer()

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
			UID:       "12345",
			Labels: map[string]string{
				"app":     "test",
				"version": "v1",
			},
			CreationTimestamp: metav1.Time{Time: time.Now()},
			ResourceVersion:   "1000",
			Generation:        2,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 2,
		},
	}

	resource := normalizer.NormalizeDeployment(deployment)

	// Verify basic fields
	if resource.ID != "deployment/test-deployment" {
		t.Errorf("Expected ID 'deployment/test-deployment', got %s", resource.ID)
	}

	if resource.Name != "test-deployment" {
		t.Errorf("Expected Name 'test-deployment', got %s", resource.Name)
	}

	if resource.Type != "deployment" {
		t.Errorf("Expected Type 'deployment', got %s", resource.Type)
	}

	if resource.Provider != "kubernetes" {
		t.Errorf("Expected Provider 'kubernetes', got %s", resource.Provider)
	}

	if resource.Namespace != "test-namespace" {
		t.Errorf("Expected Namespace 'test-namespace', got %s", resource.Namespace)
	}

	// Verify tags
	if len(resource.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(resource.Tags))
	}

	if resource.Tags["app"] != "test" {
		t.Errorf("Expected tag 'app' to be 'test', got %s", resource.Tags["app"])
	}

	// Verify metadata
	if resource.Metadata.Version != "1000" {
		t.Errorf("Expected Version '1000', got %s", resource.Metadata.Version)
	}

	// Verify additional data
	if replicas, ok := resource.Metadata.AdditionalData["replicas"].(int32); !ok || replicas != 3 {
		t.Error("Expected replicas to be 3")
	}

	if readyReplicas, ok := resource.Metadata.AdditionalData["ready_replicas"].(int32); !ok || readyReplicas != 2 {
		t.Error("Expected ready_replicas to be 2")
	}
}

func TestResourceNormalizer_NormalizeService(t *testing.T) {
	normalizer := NewResourceNormalizer()

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-namespace",
			UID:       "67890",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	resource := normalizer.NormalizeService(service)

	if resource.ID != "service/test-service" {
		t.Errorf("Expected ID 'service/test-service', got %s", resource.ID)
	}

	if resource.Type != "service" {
		t.Errorf("Expected Type 'service', got %s", resource.Type)
	}

	// Verify configuration
	if resource.Configuration["type"] != "LoadBalancer" {
		t.Errorf("Expected service type 'LoadBalancer', got %v", resource.Configuration["type"])
	}

	if resource.Configuration["cluster_ip"] != "10.0.0.1" {
		t.Errorf("Expected cluster_ip '10.0.0.1', got %v", resource.Configuration["cluster_ip"])
	}
}

func TestResourceNormalizer_NormalizeSecret(t *testing.T) {
	normalizer := NewResourceNormalizer()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret123"),
		},
	}

	resource := normalizer.NormalizeSecret(secret)

	// Verify that secret values are NOT exposed
	if resource.Configuration["data"] != nil {
		t.Error("Secret data should not be exposed in configuration")
	}

	// Verify that only keys are stored
	if dataKeys, ok := resource.Configuration["data_keys"].([]string); ok {
		if len(dataKeys) != 2 {
			t.Errorf("Expected 2 data keys, got %d", len(dataKeys))
		}
		// Keys might be in any order
		keyMap := make(map[string]bool)
		for _, key := range dataKeys {
			keyMap[key] = true
		}
		if !keyMap["username"] || !keyMap["password"] {
			t.Error("Expected data_keys to contain 'username' and 'password'")
		}
	} else {
		t.Error("Expected data_keys in configuration")
	}

	// Verify metadata
	if dataCount, ok := resource.Metadata.AdditionalData["data_keys"].(int); !ok || dataCount != 2 {
		t.Error("Expected data_keys count to be 2 in metadata")
	}
}

func TestResourceNormalizer_NormalizeConfigMap(t *testing.T) {
	normalizer := NewResourceNormalizer()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"config.yaml": "key: value",
			"app.conf":    "debug=true",
		},
		BinaryData: map[string][]byte{
			"binary.dat": []byte{0x01, 0x02, 0x03},
		},
	}

	resource := normalizer.NormalizeConfigMap(configMap)

	// Verify basic fields
	if resource.Type != "configmap" {
		t.Errorf("Expected Type 'configmap', got %s", resource.Type)
	}

	// Verify data keys
	if dataKeys, ok := resource.Configuration["data_keys"].([]string); ok {
		if len(dataKeys) != 2 {
			t.Errorf("Expected 2 data keys, got %d", len(dataKeys))
		}
	} else {
		t.Error("Expected data_keys in configuration")
	}

	// Verify binary data keys
	if binaryKeys, ok := resource.Configuration["binary_data_keys"].([]string); ok {
		if len(binaryKeys) != 1 {
			t.Errorf("Expected 1 binary data key, got %d", len(binaryKeys))
		}
		if binaryKeys[0] != "binary.dat" {
			t.Errorf("Expected binary key 'binary.dat', got %s", binaryKeys[0])
		}
	} else {
		t.Error("Expected binary_data_keys in configuration")
	}
}

func TestSanitizeLabels(t *testing.T) {
	normalizer := &ResourceNormalizer{}

	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil labels",
			input:    nil,
			expected: nil,
		},
		{
			name: "labels without sensitive data",
			input: map[string]string{
				"app":     "test",
				"version": "v1",
			},
			expected: map[string]string{
				"app":     "test",
				"version": "v1",
			},
		},
		{
			name: "labels with sensitive keys",
			input: map[string]string{
				"app":                  "test",
				"kubernetes.io/secret": "sensitive",
				"secrets":              "should-be-removed",
			},
			expected: map[string]string{
				"app": "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.sanitizeLabels(tt.input)

			if tt.expected == nil && result != nil {
				t.Error("Expected nil result for nil input")
			}

			if tt.expected != nil {
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d labels, got %d", len(tt.expected), len(result))
				}

				for k, v := range tt.expected {
					if result[k] != v {
						t.Errorf("Expected label %s=%s, got %s", k, v, result[k])
					}
				}

				// Ensure sensitive keys are removed
				if _, ok := result["kubernetes.io/secret"]; ok {
					t.Error("Sensitive key 'kubernetes.io/secret' should be removed")
				}
				if _, ok := result["secrets"]; ok {
					t.Error("Sensitive key 'secrets' should be removed")
				}
			}
		})
	}
}
