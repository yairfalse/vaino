package kubernetes

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMain(m *testing.M) {
	// Set CI environment for all tests
	os.Setenv("CI", "true")
	code := m.Run()
	os.Unsetenv("CI")
	os.Exit(code)
}

func TestKubernetesCollector_Name(t *testing.T) {
	collector := NewKubernetesCollector()
	if collector.Name() != "kubernetes" {
		t.Errorf("Expected Name() to return 'kubernetes', got %s", collector.Name())
	}
}

func TestKubernetesCollector_Status(t *testing.T) {
	collector := NewKubernetesCollector()
	status := collector.Status()
	// In CI mode, should return ready
	if status != "ready (CI mode)" {
		t.Errorf("Expected Status() to return 'ready (CI mode)', got %s", status)
	}
}

func TestKubernetesCollector_Validate(t *testing.T) {
	collector := NewKubernetesCollector()

	tests := []struct {
		name    string
		config  collectors.CollectorConfig
		wantErr bool
	}{
		{
			name:    "empty config should validate",
			config:  collectors.CollectorConfig{},
			wantErr: false,
		},
		{
			name: "config with namespaces should validate",
			config: collectors.CollectorConfig{
				Namespaces: []string{"default", "kube-system"},
			},
			wantErr: false,
		},
		{
			name: "config with contexts should validate",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"contexts": []string{"prod", "staging"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collector.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKubernetesCollector_SupportedRegions(t *testing.T) {
	collector := NewKubernetesCollector()
	regions := collector.SupportedRegions()
	if len(regions) != 0 {
		t.Errorf("Expected SupportedRegions() to return empty slice for Kubernetes, got %v", regions)
	}
}

func TestKubernetesCollector_AutoDiscover(t *testing.T) {
	collector := NewKubernetesCollector()
	config, err := collector.AutoDiscover()

	// AutoDiscover might fail without valid kubeconfig, but should not panic
	if err == nil {
		if config.Config == nil {
			t.Error("Expected AutoDiscover() to return config with non-nil Config map")
		}
	}
}

func TestKubernetesCollector_CollectWithFakeClient(t *testing.T) {
	// Create fake Kubernetes client with test data
	fakeClient := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 3,
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: "10.0.0.1",
			},
		},
	)

	// Create collector with fake client
	collector := &KubernetesCollector{
		client: &KubernetesClient{
			clientset: fakeClient,
		},
		normalizer: NewResourceNormalizer(),
		version:    "1.0.0",
	}

	// Test collection
	config := collectors.CollectorConfig{
		Namespaces: []string{"default"},
	}

	ctx := context.Background()
	snapshot, err := collector.Collect(ctx, config)

	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	if snapshot.Provider != "kubernetes" {
		t.Errorf("Expected Provider to be 'kubernetes', got %s", snapshot.Provider)
	}

	// Should have collected at least the deployment and service
	if len(snapshot.Resources) < 2 {
		t.Errorf("Expected at least 2 resources, got %d", len(snapshot.Resources))
	}

	// Verify resource types
	resourceTypes := make(map[string]int)
	for _, r := range snapshot.Resources {
		resourceTypes[r.Type]++
	}

	if resourceTypes["deployment"] != 1 {
		t.Errorf("Expected 1 deployment, got %d", resourceTypes["deployment"])
	}

	if resourceTypes["service"] != 1 {
		t.Errorf("Expected 1 service, got %d", resourceTypes["service"])
	}
}

func TestExtractKubernetesConfig(t *testing.T) {
	collector := &KubernetesCollector{}

	tests := []struct {
		name        string
		config      collectors.CollectorConfig
		expectedNS  []string
		expectedCtx string
	}{
		{
			name:        "empty config uses defaults",
			config:      collectors.CollectorConfig{},
			expectedNS:  []string{""},
			expectedCtx: "",
		},
		{
			name: "config with namespaces",
			config: collectors.CollectorConfig{
				Namespaces: []string{"default", "kube-system"},
			},
			expectedNS:  []string{"default", "kube-system"},
			expectedCtx: "",
		},
		{
			name: "config with context",
			config: collectors.CollectorConfig{
				Config: map[string]interface{}{
					"context": "production",
				},
			},
			expectedNS:  []string{""},
			expectedCtx: "production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeConfig := collector.extractKubernetesConfig(tt.config)

			if len(kubeConfig.Namespaces) != len(tt.expectedNS) {
				t.Errorf("Expected %d namespaces, got %d", len(tt.expectedNS), len(kubeConfig.Namespaces))
			}

			if kubeConfig.Context != tt.expectedCtx {
				t.Errorf("Expected context %s, got %s", tt.expectedCtx, kubeConfig.Context)
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}
