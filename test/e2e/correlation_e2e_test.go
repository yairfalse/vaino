package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

// TestE2ECorrelationWorkflow tests the complete end-to-end correlation workflow
func TestE2ECorrelationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup test environment
	env := setupE2EEnvironment(t)
	defer env.cleanup()

	// Test complete workflow: scan -> changes -> correlation
	t.Run("complete_workflow", func(t *testing.T) {
		testCompleteWorkflow(t, env)
	})

	// Test error handling
	t.Run("error_handling", func(t *testing.T) {
		testErrorHandling(t, env)
	})

	// Test different output formats
	t.Run("output_formats", func(t *testing.T) {
		testOutputFormats(t, env)
	})

	// Test performance with realistic data
	t.Run("realistic_performance", func(t *testing.T) {
		testRealisticPerformance(t, env)
	})
}

type e2eEnvironment struct {
	t         *testing.T
	tmpDir    string
	wgoBinary string
	dataDir   string
}

func setupE2EEnvironment(t *testing.T) *e2eEnvironment {
	tmpDir := t.TempDir()

	// Build WGO binary
	wgoBinary := filepath.Join(tmpDir, "wgo")
	buildCmd := exec.Command("go", "build", "-o", wgoBinary, "../../cmd/wgo")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build WGO: %v", err)
	}

	// Create test data directory
	dataDir := filepath.Join(tmpDir, "test-data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	env := &e2eEnvironment{
		t:         t,
		tmpDir:    tmpDir,
		wgoBinary: wgoBinary,
		dataDir:   dataDir,
	}

	// Create test snapshots
	env.createTestData()

	return env
}

func (env *e2eEnvironment) cleanup() {
	// Cleanup is automatic with t.TempDir()
}

func (env *e2eEnvironment) createTestData() {
	// Create realistic test scenarios
	env.createScalingScenario()
	env.createServiceDeploymentScenario()
	env.createConfigUpdateScenario()
	env.createMixedScenario()
}

func (env *e2eEnvironment) createScalingScenario() {
	baseline := &types.Snapshot{
		ID:        "scaling-baseline",
		Timestamp: time.Now().Add(-10 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "deployment/frontend",
				Type:      "deployment",
				Name:      "frontend",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"replicas": 3,
					"image":    "frontend:v2.1.0",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "200m",
							"memory": "256Mi",
						},
						"limits": map[string]interface{}{
							"cpu":    "500m",
							"memory": "512Mi",
						},
					},
				},
				Metadata: types.ResourceMetadata{
					Version:   "1234",
					CreatedAt: time.Now().Add(-2 * time.Hour),
				},
			},
			{
				ID:        "service/frontend-service",
				Type:      "service",
				Name:      "frontend-service",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"ports": []map[string]interface{}{
						{"port": 80, "targetPort": 8080, "protocol": "TCP"},
					},
					"selector": map[string]interface{}{
						"app": "frontend",
					},
					"type": "LoadBalancer",
				},
			},
			{
				ID:        "horizontalpodautoscaler/frontend-hpa",
				Type:      "horizontalpodautoscaler",
				Name:      "frontend-hpa",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"minReplicas":                    3,
					"maxReplicas":                    10,
					"targetCPUUtilizationPercentage": 70,
				},
			},
		},
	}

	scaled := &types.Snapshot{
		ID:        "scaling-after",
		Timestamp: time.Now().Add(-5 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "deployment/frontend",
				Type:      "deployment",
				Name:      "frontend",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"replicas": 7, // Scaled up by HPA
					"image":    "frontend:v2.1.0",
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"cpu":    "200m",
							"memory": "256Mi",
						},
						"limits": map[string]interface{}{
							"cpu":    "500m",
							"memory": "512Mi",
						},
					},
				},
				Metadata: types.ResourceMetadata{
					Version:   "1240", // Updated
					CreatedAt: time.Now().Add(-2 * time.Hour),
				},
			},
			{
				ID:        "service/frontend-service",
				Type:      "service",
				Name:      "frontend-service",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"ports": []map[string]interface{}{
						{"port": 80, "targetPort": 8080, "protocol": "TCP"},
					},
					"selector": map[string]interface{}{
						"app": "frontend",
					},
					"type": "LoadBalancer",
				},
			},
			{
				ID:        "horizontalpodautoscaler/frontend-hpa",
				Type:      "horizontalpodautoscaler",
				Name:      "frontend-hpa",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"minReplicas":                    3,
					"maxReplicas":                    10,
					"targetCPUUtilizationPercentage": 70,
					"currentReplicas":                7, // HPA triggered
				},
			},
			// Add new pods created by scaling
			{
				ID:        "pod/frontend-abc123",
				Type:      "pod",
				Name:      "frontend-abc123",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"phase": "Running",
					"containers": []map[string]interface{}{
						{"name": "frontend", "image": "frontend:v2.1.0"},
					},
				},
				Metadata: types.ResourceMetadata{
					CreatedAt: time.Now().Add(-4 * time.Minute),
				},
			},
			{
				ID:        "pod/frontend-def456",
				Type:      "pod",
				Name:      "frontend-def456",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"phase": "Running",
					"containers": []map[string]interface{}{
						{"name": "frontend", "image": "frontend:v2.1.0"},
					},
				},
				Metadata: types.ResourceMetadata{
					CreatedAt: time.Now().Add(-4 * time.Minute),
				},
			},
		},
	}

	env.saveSnapshot("scaling-baseline.json", baseline)
	env.saveSnapshot("scaling-after.json", scaled)
}

func (env *e2eEnvironment) createServiceDeploymentScenario() {
	before := &types.Snapshot{
		ID:        "service-before",
		Timestamp: time.Now().Add(-15 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "deployment/existing-app",
				Type:      "deployment",
				Name:      "existing-app",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 2,
					"image":    "existing:v1.0",
				},
			},
		},
	}

	after := &types.Snapshot{
		ID:        "service-after",
		Timestamp: time.Now().Add(-10 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "deployment/existing-app",
				Type:      "deployment",
				Name:      "existing-app",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 2,
					"image":    "existing:v1.0",
				},
			},
			// New service deployment
			{
				ID:        "service/analytics-service",
				Type:      "service",
				Name:      "analytics-service",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"ports": []map[string]interface{}{
						{"port": 9090, "targetPort": 9090},
					},
					"selector": map[string]interface{}{
						"app": "analytics",
					},
				},
			},
			{
				ID:        "deployment/analytics",
				Type:      "deployment",
				Name:      "analytics",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 3,
					"image":    "analytics:v1.0",
				},
			},
			{
				ID:        "configmap/analytics-config",
				Type:      "configmap",
				Name:      "analytics-config",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"data": map[string]string{
						"database_url": "postgres://analytics-db:5432/analytics",
						"log_level":    "info",
					},
				},
			},
			{
				ID:        "secret/analytics-secrets",
				Type:      "secret",
				Name:      "analytics-secrets",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"type": "Opaque",
					"data": map[string]string{
						"db_password": "encrypted_password",
						"api_key":     "encrypted_api_key",
					},
				},
			},
		},
	}

	env.saveSnapshot("service-before.json", before)
	env.saveSnapshot("service-after.json", after)
}

func (env *e2eEnvironment) createConfigUpdateScenario() {
	before := &types.Snapshot{
		ID:        "config-before",
		Timestamp: time.Now().Add(-8 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "configmap/app-config",
				Type:      "configmap",
				Name:      "app-config",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"data": map[string]string{
						"feature_flags": "feature_a=false,feature_b=true",
						"log_level":     "info",
						"timeout":       "30s",
					},
				},
				Metadata: types.ResourceMetadata{
					Version: "500",
				},
			},
			{
				ID:        "deployment/web-app",
				Type:      "deployment",
				Name:      "web-app",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"replicas": 4,
					"image":    "web-app:v3.2.1",
				},
				Metadata: types.ResourceMetadata{
					Version: "800",
				},
			},
		},
	}

	after := &types.Snapshot{
		ID:        "config-after",
		Timestamp: time.Now().Add(-6 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "configmap/app-config",
				Type:      "configmap",
				Name:      "app-config",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"data": map[string]string{
						"feature_flags": "feature_a=true,feature_b=true", // Updated
						"log_level":     "debug",                         // Updated
						"timeout":       "45s",                           // Updated
					},
				},
				Metadata: types.ResourceMetadata{
					Version: "501", // Incremented
				},
			},
			{
				ID:        "deployment/web-app",
				Type:      "deployment",
				Name:      "web-app",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"replicas": 4,
					"image":    "web-app:v3.2.1",
				},
				Metadata: types.ResourceMetadata{
					Version: "805", // Incremented due to restart
				},
			},
			// Pod restart due to config change
			{
				ID:        "pod/web-app-xyz789",
				Type:      "pod",
				Name:      "web-app-xyz789",
				Provider:  "kubernetes",
				Namespace: "production",
				Configuration: map[string]interface{}{
					"phase":        "Running",
					"restartCount": 1, // Restarted due to config
				},
				Metadata: types.ResourceMetadata{
					CreatedAt: time.Now().Add(-5 * time.Minute),
				},
			},
		},
	}

	env.saveSnapshot("config-before.json", before)
	env.saveSnapshot("config-after.json", after)
}

func (env *e2eEnvironment) createMixedScenario() {
	// Complex scenario with multiple changes happening
	baseline := &types.Snapshot{
		ID:        "mixed-baseline",
		Timestamp: time.Now().Add(-20 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "deployment/api",
				Type:      "deployment",
				Name:      "api",
				Provider:  "kubernetes",
				Namespace: "api",
				Configuration: map[string]interface{}{
					"replicas": 2,
					"image":    "api:v1.5.0",
				},
			},
			{
				ID:        "deployment/worker",
				Type:      "deployment",
				Name:      "worker",
				Provider:  "kubernetes",
				Namespace: "background",
				Configuration: map[string]interface{}{
					"replicas": 1,
					"image":    "worker:v1.2.0",
				},
			},
			{
				ID:        "secret/database-creds",
				Type:      "secret",
				Name:      "database-creds",
				Provider:  "kubernetes",
				Namespace: "api",
				Configuration: map[string]interface{}{
					"type": "Opaque",
				},
			},
		},
	}

	complex := &types.Snapshot{
		ID:        "mixed-complex",
		Timestamp: time.Now().Add(-10 * time.Minute),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			// API scaled up
			{
				ID:        "deployment/api",
				Type:      "deployment",
				Name:      "api",
				Provider:  "kubernetes",
				Namespace: "api",
				Configuration: map[string]interface{}{
					"replicas": 5, // Scaled
					"image":    "api:v1.5.0",
				},
			},
			// Worker updated to new version
			{
				ID:        "deployment/worker",
				Type:      "deployment",
				Name:      "worker",
				Provider:  "kubernetes",
				Namespace: "background",
				Configuration: map[string]interface{}{
					"replicas": 1,
					"image":    "worker:v1.3.0", // Updated
				},
			},
			// Secret rotated
			{
				ID:        "secret/database-creds",
				Type:      "secret",
				Name:      "database-creds",
				Provider:  "kubernetes",
				Namespace: "api",
				Configuration: map[string]interface{}{
					"type":    "Opaque",
					"rotated": "2023-01-01", // Rotated
				},
			},
			// New monitoring service added
			{
				ID:        "service/monitoring-service",
				Type:      "service",
				Name:      "monitoring-service",
				Provider:  "kubernetes",
				Namespace: "monitoring",
				Configuration: map[string]interface{}{
					"ports": []map[string]interface{}{
						{"port": 9090, "targetPort": 9090},
					},
				},
			},
			{
				ID:        "deployment/prometheus",
				Type:      "deployment",
				Name:      "prometheus",
				Provider:  "kubernetes",
				Namespace: "monitoring",
				Configuration: map[string]interface{}{
					"replicas": 1,
					"image":    "prometheus:v2.40.0",
				},
			},
		},
	}

	env.saveSnapshot("mixed-baseline.json", baseline)
	env.saveSnapshot("mixed-complex.json", complex)
}

func (env *e2eEnvironment) saveSnapshot(filename string, snapshot *types.Snapshot) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		env.t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	filepath := filepath.Join(env.dataDir, filename)
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		env.t.Fatalf("Failed to write snapshot: %v", err)
	}
}

func (env *e2eEnvironment) runWGO(args ...string) (string, error) {
	cmd := exec.Command(env.wgoBinary, args...)
	cmd.Dir = env.tmpDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func testCompleteWorkflow(t *testing.T, env *e2eEnvironment) {
	// Test scaling scenario
	t.Run("scaling_correlation", func(t *testing.T) {
		baselineFile := filepath.Join(env.dataDir, "scaling-baseline.json")
		scaledFile := filepath.Join(env.dataDir, "scaling-after.json")

		// Test correlation detection
		output, err := env.runWGO("changes", "--from", baselineFile, "--to", scaledFile, "--correlated")
		if err != nil {
			t.Fatalf("Correlation command failed: %v\nOutput: %s", err, output)
		}

		// Verify scaling correlation
		if !strings.Contains(output, "frontend Scaling") {
			t.Error("Expected frontend scaling correlation")
		}

		if !strings.Contains(output, "● 🔗") {
			t.Error("Expected high confidence indicator")
		}

		if !strings.Contains(output, "Scaled from 3 to 7 replicas") {
			t.Error("Expected scaling description")
		}

		// Should correlate HPA trigger
		if !strings.Contains(output, "HPA triggered") ||
			!strings.Contains(output, "horizontalpodautoscaler") {
			t.Error("Expected HPA correlation")
		}

		// Should include new pods in scaling group
		scalingSection := extractSectionBetween(output, "frontend Scaling", "🔗")
		if !strings.Contains(scalingSection, "pod/frontend") {
			t.Error("Expected pods to be correlated with scaling")
		}
	})

	// Test service deployment correlation
	t.Run("service_deployment_correlation", func(t *testing.T) {
		beforeFile := filepath.Join(env.dataDir, "service-before.json")
		afterFile := filepath.Join(env.dataDir, "service-after.json")

		output, err := env.runWGO("changes", "--from", beforeFile, "--to", afterFile, "--correlated")
		if err != nil {
			t.Fatalf("Service correlation command failed: %v", err)
		}

		// Should detect service deployment pattern
		if !strings.Contains(output, "New Service: analytics-service") {
			t.Error("Expected service deployment correlation")
		}

		// Should group related resources
		serviceSection := extractSectionBetween(output, "analytics-service", "🔗")
		if !strings.Contains(serviceSection, "deployment/analytics") {
			t.Error("Expected deployment to be correlated with service")
		}

		if !strings.Contains(serviceSection, "configmap/analytics-config") {
			t.Error("Expected configmap to be correlated with service")
		}
	})

	// Test config update correlation
	t.Run("config_update_correlation", func(t *testing.T) {
		beforeFile := filepath.Join(env.dataDir, "config-before.json")
		afterFile := filepath.Join(env.dataDir, "config-after.json")

		output, err := env.runWGO("changes", "--from", beforeFile, "--to", afterFile, "--correlated")
		if err != nil {
			t.Fatalf("Config correlation command failed: %v", err)
		}

		// Should detect config update pattern
		if !strings.Contains(output, "app-config Update") {
			t.Error("Expected config update correlation")
		}

		// Should correlate with deployment restart
		configSection := extractSectionBetween(output, "app-config Update", "🔗")
		if !strings.Contains(configSection, "triggered") &&
			!strings.Contains(configSection, "restart") {
			t.Error("Expected restart correlation with config change")
		}
	})
}

func testErrorHandling(t *testing.T, env *e2eEnvironment) {
	// Test invalid file paths
	t.Run("invalid_files", func(t *testing.T) {
		_, err := env.runWGO("changes", "--from", "nonexistent.json", "--to", "alsononexistent.json")
		if err == nil {
			t.Error("Expected error for nonexistent files")
		}
	})

	// Test invalid JSON
	t.Run("invalid_json", func(t *testing.T) {
		invalidFile := filepath.Join(env.dataDir, "invalid.json")
		if err := os.WriteFile(invalidFile, []byte("invalid json"), 0644); err != nil {
			t.Fatal(err)
		}

		validFile := filepath.Join(env.dataDir, "scaling-baseline.json")
		_, err := env.runWGO("changes", "--from", invalidFile, "--to", validFile)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	// Test missing required flags
	t.Run("missing_flags", func(t *testing.T) {
		_, err := env.runWGO("changes", "--correlated")
		if err == nil {
			t.Error("Expected error for missing snapshot files")
		}
	})
}

func testOutputFormats(t *testing.T, env *e2eEnvironment) {
	baselineFile := filepath.Join(env.dataDir, "scaling-baseline.json")
	scaledFile := filepath.Join(env.dataDir, "scaling-after.json")

	// Test JSON output
	t.Run("json_output", func(t *testing.T) {
		output, err := env.runWGO("changes", "--from", baselineFile, "--to", scaledFile, "--output", "json")
		if err != nil {
			t.Fatalf("JSON output failed: %v", err)
		}

		// Verify valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// Check for expected fields
		if _, exists := result["changes"]; !exists {
			t.Error("Expected 'changes' field in JSON")
		}

		if _, exists := result["summary"]; !exists {
			t.Error("Expected 'summary' field in JSON")
		}
	})

	// Test timeline output
	t.Run("timeline_output", func(t *testing.T) {
		output, err := env.runWGO("changes", "--from", baselineFile, "--to", scaledFile, "--timeline")
		if err != nil {
			t.Fatalf("Timeline output failed: %v", err)
		}

		// Verify timeline format
		if !strings.Contains(output, "📅 Change Timeline") {
			t.Error("Expected timeline header")
		}

		if !strings.Contains(output, "━") {
			t.Error("Expected timeline bar")
		}

		if !strings.Contains(output, "▲") {
			t.Error("Expected timeline markers")
		}
	})

	// Test regular output
	t.Run("regular_output", func(t *testing.T) {
		output, err := env.runWGO("changes", "--from", baselineFile, "--to", scaledFile)
		if err != nil {
			t.Fatalf("Regular output failed: %v", err)
		}

		// Should show standard change format
		if !strings.Contains(output, "📊 Infrastructure Changes") {
			t.Error("Expected standard change header")
		}

		if !strings.Contains(output, "Summary:") {
			t.Error("Expected summary section")
		}
	})
}

func testRealisticPerformance(t *testing.T, env *e2eEnvironment) {
	// Test with complex mixed scenario
	baselineFile := filepath.Join(env.dataDir, "mixed-baseline.json")
	complexFile := filepath.Join(env.dataDir, "mixed-complex.json")

	start := time.Now()
	output, err := env.runWGO("changes", "--from", baselineFile, "--to", complexFile, "--correlated")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	// Should complete quickly
	if duration > 5*time.Second {
		t.Errorf("Correlation took too long: %v", duration)
	}

	// Should produce meaningful correlations
	if !strings.Contains(output, "📊 Correlated Infrastructure Changes") {
		t.Error("Expected correlation output")
	}

	// Should separate different types of changes
	groupCount := strings.Count(output, "🔗")
	if groupCount < 2 {
		t.Errorf("Expected multiple correlation groups, got %d", groupCount)
	}

	t.Logf("Processed complex scenario in %v, produced %d groups", duration, groupCount)
}

// Helper functions

func extractSectionBetween(text, start, end string) string {
	startIdx := strings.Index(text, start)
	if startIdx == -1 {
		return ""
	}

	searchText := text[startIdx:]
	endIdx := strings.Index(searchText, end)
	if endIdx == -1 {
		return searchText
	}

	return searchText[:endIdx]
}
