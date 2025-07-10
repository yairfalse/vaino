package system

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

// TestCorrelationSystemWorkflow tests the complete correlation workflow
func TestCorrelationSystemWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	// Build vaino binary for testing
	wgoBinary := buildWGOBinary(t)
	defer os.Remove(wgoBinary)

	// Create test snapshots
	baselineSnapshot := createBaselineSnapshot(t)
	scalingSnapshot := createScalingSnapshot(t, baselineSnapshot)

	// Test correlation detection
	output, err := runWGOCommand(t, wgoBinary, "changes",
		"--from", baselineSnapshot,
		"--to", scalingSnapshot,
		"--correlated")

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	// Verify correlation output
	if !strings.Contains(output, "📊 Correlated Infrastructure Changes") {
		t.Error("Expected correlation header")
	}

	if !strings.Contains(output, "frontend Scaling") {
		t.Error("Expected scaling correlation to be detected")
	}

	if !strings.Contains(output, "● 🔗") {
		t.Error("Expected high confidence indicator")
	}

	if !strings.Contains(output, "Deployment scaling detected") {
		t.Error("Expected scaling reason")
	}
}

// TestTimelineSystemWorkflow tests the complete timeline workflow
func TestTimelineSystemWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	wgoBinary := buildWGOBinary(t)
	defer os.Remove(wgoBinary)

	// Create test snapshots with time progression
	snapshot1 := createTimelineSnapshot1(t)
	_ = createTimelineSnapshot2(t)
	snapshot3 := createTimelineSnapshot3(t)

	// Test timeline visualization
	output, err := runWGOCommand(t, wgoBinary, "changes",
		"--from", snapshot1,
		"--to", snapshot3,
		"--timeline")

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	// Verify timeline output
	if !strings.Contains(output, "📅 Change Timeline") {
		t.Error("Expected timeline header")
	}

	if !strings.Contains(output, "━") {
		t.Error("Expected timeline bar")
	}

	if !strings.Contains(output, "▲") {
		t.Error("Expected timeline markers")
	}

	// Should show multiple change groups on timeline
	markerCount := strings.Count(output, "▲")
	if markerCount < 2 {
		t.Errorf("Expected at least 2 timeline markers, got %d", markerCount)
	}
}

// TestCorrelationAccuracy tests correlation accuracy
func TestCorrelationAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	wgoBinary := buildWGOBinary(t)
	defer os.Remove(wgoBinary)

	// Test scenario: Scaling + Config change (should NOT be correlated)
	baselineSnapshot := createBaselineSnapshot(t)
	mixedChangesSnapshot := createMixedChangesSnapshot(t, baselineSnapshot)

	output, err := runWGOCommand(t, wgoBinary, "changes",
		"--from", baselineSnapshot,
		"--to", mixedChangesSnapshot,
		"--correlated")

	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}

	// Should have separate groups
	if !strings.Contains(output, "frontend Scaling") {
		t.Error("Expected scaling group")
	}

	if !strings.Contains(output, "Other Changes") {
		t.Error("Expected other changes group for unrelated changes")
	}

	// Should NOT incorrectly correlate unrelated changes
	scalingSection := extractSection(output, "frontend Scaling", "🔗")
	if strings.Contains(scalingSection, "configmap") {
		t.Error("Incorrectly correlated config change with scaling")
	}
}

// TestConfidenceLevels tests confidence level assignment
func TestConfidenceLevels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	wgoBinary := buildWGOBinary(t)
	defer os.Remove(wgoBinary)

	tests := []struct {
		name               string
		snapshotFunc       func(t *testing.T, baseline string) string
		expectedConfidence string
		expectedIndicator  string
	}{
		{
			name:               "scaling_high_confidence",
			snapshotFunc:       createScalingSnapshot,
			expectedConfidence: "high",
			expectedIndicator:  "●",
		},
		{
			name:               "service_deployment_medium_confidence",
			snapshotFunc:       createServiceDeploymentSnapshot,
			expectedConfidence: "medium",
			expectedIndicator:  "◐",
		},
		{
			name:               "unrelated_changes_low_confidence",
			snapshotFunc:       createUnrelatedChangesSnapshot,
			expectedConfidence: "low",
			expectedIndicator:  "○",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baselineSnapshot := createBaselineSnapshot(t)
			testSnapshot := tt.snapshotFunc(t, baselineSnapshot)

			output, err := runWGOCommand(t, wgoBinary, "changes",
				"--from", baselineSnapshot,
				"--to", testSnapshot,
				"--correlated")

			if err != nil {
				t.Fatalf("Command failed: %v", err)
			}

			if !strings.Contains(output, tt.expectedIndicator+" 🔗") {
				t.Errorf("Expected confidence indicator '%s'", tt.expectedIndicator)
			}
		})
	}
}

// TestLargeScaleCorrelation tests correlation with many changes
func TestLargeScaleCorrelation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	wgoBinary := buildWGOBinary(t)
	defer os.Remove(wgoBinary)

	// Create snapshot with 50+ resources
	baselineSnapshot := createLargeBaselineSnapshot(t, 50)
	largeChangesSnapshot := createLargeChangesSnapshot(t, baselineSnapshot, 20)

	start := time.Now()
	output, err := runWGOCommand(t, wgoBinary, "changes",
		"--from", baselineSnapshot,
		"--to", largeChangesSnapshot,
		"--correlated")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Performance check - should complete within reasonable time
	if duration > 10*time.Second {
		t.Errorf("Correlation took too long: %v", duration)
	}

	// Should still produce meaningful groups
	if !strings.Contains(output, "📊 Correlated Infrastructure Changes") {
		t.Error("Expected correlation output for large dataset")
	}

	// Should group some changes
	groupCount := strings.Count(output, "🔗")
	if groupCount == 0 {
		t.Error("Expected at least some correlation groups")
	}
}

// TestJSONOutput tests correlation with JSON output
func TestJSONOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping system test in short mode")
	}

	wgoBinary := buildWGOBinary(t)
	defer os.Remove(wgoBinary)

	baselineSnapshot := createBaselineSnapshot(t)
	scalingSnapshot := createScalingSnapshot(t, baselineSnapshot)

	output, err := runWGOCommand(t, wgoBinary, "changes",
		"--from", baselineSnapshot,
		"--to", scalingSnapshot,
		"--output", "json")

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify valid JSON
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Check for expected fields
	if _, exists := report["changes"]; !exists {
		t.Error("Expected 'changes' field in JSON output")
	}

	if _, exists := report["summary"]; !exists {
		t.Error("Expected 'summary' field in JSON output")
	}
}

// Helper functions

func buildWGOBinary(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	binary := filepath.Join(tmpDir, "wgo")

	cmd := exec.Command("go", "build", "-o", binary, "../../cmd/vaino")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build wgo: %v", err)
	}

	return binary
}

func runWGOCommand(t *testing.T, binary string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func createBaselineSnapshot(t *testing.T) string {
	t.Helper()

	snapshot := &types.Snapshot{
		ID:        "baseline",
		Timestamp: time.Now(),
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:        "deployment/frontend",
				Type:      "deployment",
				Name:      "frontend",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 3,
					"image":    "frontend:v1.0",
				},
				Metadata: types.ResourceMetadata{
					Version: "100",
				},
			},
			{
				ID:        "service/frontend-service",
				Type:      "service",
				Name:      "frontend-service",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"port": 80,
				},
			},
			{
				ID:        "configmap/app-config",
				Type:      "configmap",
				Name:      "app-config",
				Provider:  "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"data": map[string]string{
						"env": "production",
					},
				},
			},
		},
	}

	file := createTempSnapshot(t, snapshot)
	return file
}

func createScalingSnapshot(t *testing.T, baselineFile string) string {
	t.Helper()

	// Load baseline
	baseline := loadSnapshot(t, baselineFile)

	// Create modified version with scaling
	scaled := *baseline
	scaled.ID = "scaled"
	scaled.Timestamp = baseline.Timestamp.Add(1 * time.Minute)

	// Scale frontend deployment
	for i := range scaled.Resources {
		if scaled.Resources[i].ID == "deployment/frontend" {
			config := scaled.Resources[i].Configuration
			config["replicas"] = 5 // Scale up
			scaled.Resources[i].Metadata.Version = "101"
		}
	}

	return createTempSnapshot(t, &scaled)
}

func createServiceDeploymentSnapshot(t *testing.T, baselineFile string) string {
	t.Helper()

	baseline := loadSnapshot(t, baselineFile)
	modified := *baseline
	modified.ID = "service-deployment"
	modified.Timestamp = baseline.Timestamp.Add(2 * time.Minute)

	// Add new service and related resources
	newService := types.Resource{
		ID:        "service/api-service",
		Type:      "service",
		Name:      "api-service",
		Provider:  "kubernetes",
		Namespace: "default",
		Configuration: map[string]interface{}{
			"port": 8080,
		},
	}

	newDeployment := types.Resource{
		ID:        "deployment/api",
		Type:      "deployment",
		Name:      "api",
		Provider:  "kubernetes",
		Namespace: "default",
		Configuration: map[string]interface{}{
			"replicas": 2,
			"image":    "api:latest",
		},
	}

	modified.Resources = append(modified.Resources, newService, newDeployment)

	return createTempSnapshot(t, &modified)
}

func createMixedChangesSnapshot(t *testing.T, baselineFile string) string {
	t.Helper()

	baseline := loadSnapshot(t, baselineFile)
	mixed := *baseline
	mixed.ID = "mixed-changes"
	mixed.Timestamp = baseline.Timestamp.Add(3 * time.Minute)

	// Scale deployment AND modify unrelated config
	for i := range mixed.Resources {
		if mixed.Resources[i].ID == "deployment/frontend" {
			config := mixed.Resources[i].Configuration
			config["replicas"] = 5
			mixed.Resources[i].Metadata.Version = "101"
		}
		if mixed.Resources[i].ID == "configmap/app-config" {
			config := mixed.Resources[i].Configuration
			if data, ok := config["data"].(map[string]string); ok {
				data["version"] = "2.0" // Unrelated config change
			}
		}
	}

	return createTempSnapshot(t, &mixed)
}

func createUnrelatedChangesSnapshot(t *testing.T, baselineFile string) string {
	t.Helper()

	baseline := loadSnapshot(t, baselineFile)
	unrelated := *baseline
	unrelated.ID = "unrelated-changes"
	unrelated.Timestamp = baseline.Timestamp.Add(4 * time.Minute)

	// Make unrelated changes across different namespaces/times
	for i := range unrelated.Resources {
		if unrelated.Resources[i].ID == "configmap/app-config" {
			config := unrelated.Resources[i].Configuration
			if data, ok := config["data"].(map[string]string); ok {
				data["debug"] = "true"
			}
		}
	}

	// Add unrelated resource
	unrelatedResource := types.Resource{
		ID:        "secret/other-secret",
		Type:      "secret",
		Name:      "other-secret",
		Provider:  "kubernetes",
		Namespace: "other-namespace", // Different namespace
		Configuration: map[string]interface{}{
			"data": "encrypted",
		},
	}

	unrelated.Resources = append(unrelated.Resources, unrelatedResource)

	return createTempSnapshot(t, &unrelated)
}

func createLargeBaselineSnapshot(t *testing.T, count int) string {
	t.Helper()

	snapshot := &types.Snapshot{
		ID:        "large-baseline",
		Timestamp: time.Now(),
		Provider:  "kubernetes",
		Resources: make([]types.Resource, count),
	}

	for i := 0; i < count; i++ {
		snapshot.Resources[i] = types.Resource{
			ID:        fmt.Sprintf("deployment/app-%d", i),
			Type:      "deployment",
			Name:      fmt.Sprintf("app-%d", i),
			Provider:  "kubernetes",
			Namespace: fmt.Sprintf("namespace-%d", i%5), // 5 namespaces
			Configuration: map[string]interface{}{
				"replicas": 3,
				"image":    fmt.Sprintf("app:v1.%d", i),
			},
			Metadata: types.ResourceMetadata{
				Version: fmt.Sprintf("%d", 100+i),
			},
		}
	}

	return createTempSnapshot(t, snapshot)
}

func createLargeChangesSnapshot(t *testing.T, baselineFile string, changeCount int) string {
	t.Helper()

	baseline := loadSnapshot(t, baselineFile)
	changed := *baseline
	changed.ID = "large-changes"
	changed.Timestamp = baseline.Timestamp.Add(5 * time.Minute)

	// Modify first N resources
	for i := 0; i < changeCount && i < len(changed.Resources); i++ {
		config := changed.Resources[i].Configuration
		config["replicas"] = 5 // Scale all
		changed.Resources[i].Metadata.Version = fmt.Sprintf("%d", 200+i)
	}

	return createTempSnapshot(t, &changed)
}

func createTimelineSnapshot1(t *testing.T) string {
	t.Helper()

	base := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

	snapshot := &types.Snapshot{
		ID:        "timeline-1",
		Timestamp: base,
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:   "deployment/app",
				Type: "deployment",
				Name: "app",
				Configuration: map[string]interface{}{
					"replicas": 3,
				},
			},
		},
	}

	return createTempSnapshot(t, snapshot)
}

func createTimelineSnapshot2(t *testing.T) string {
	t.Helper()

	base := time.Date(2023, 1, 1, 10, 2, 0, 0, time.UTC)

	snapshot := &types.Snapshot{
		ID:        "timeline-2",
		Timestamp: base,
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:   "deployment/app",
				Type: "deployment",
				Name: "app",
				Configuration: map[string]interface{}{
					"replicas": 5, // Scaled
				},
			},
		},
	}

	return createTempSnapshot(t, snapshot)
}

func createTimelineSnapshot3(t *testing.T) string {
	t.Helper()

	base := time.Date(2023, 1, 1, 10, 5, 0, 0, time.UTC)

	snapshot := &types.Snapshot{
		ID:        "timeline-3",
		Timestamp: base,
		Provider:  "kubernetes",
		Resources: []types.Resource{
			{
				ID:   "deployment/app",
				Type: "deployment",
				Name: "app",
				Configuration: map[string]interface{}{
					"replicas": 5,
				},
			},
			{
				ID:   "service/new-service", // Added
				Type: "service",
				Name: "new-service",
				Configuration: map[string]interface{}{
					"port": 8080,
				},
			},
		},
	}

	return createTempSnapshot(t, snapshot)
}

func createTempSnapshot(t *testing.T, snapshot *types.Snapshot) string {
	t.Helper()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "snapshot-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		t.Fatalf("Failed to write snapshot: %v", err)
	}

	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	return tmpFile.Name()
}

func loadSnapshot(t *testing.T, filename string) *types.Snapshot {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read snapshot: %v", err)
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("Failed to unmarshal snapshot: %v", err)
	}

	return &snapshot
}

func extractSection(text, startMarker, endMarker string) string {
	startIdx := strings.Index(text, startMarker)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(text[startIdx+len(startMarker):], endMarker)
	if endIdx == -1 {
		return text[startIdx:]
	}

	return text[startIdx : startIdx+len(startMarker)+endIdx]
}
