package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCLIBasicCommands tests basic CLI functionality
func TestCLIBasicCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:        "help command",
			args:        []string{"--help"},
			expectError: false,
			expectContains: []string{
				"VAINO (What's Going On)",
				"infrastructure drift detection tool",
				"Available Commands:",
			},
		},
		{
			name:        "version command",
			args:        []string{"version"},
			expectError: false,
			expectContains: []string{
				"vaino version",
			},
		},
		{
			name:        "scan help",
			args:        []string{"scan", "--help"},
			expectError: false,
			expectContains: []string{
				"Scan infrastructure for current state",
				"--provider",
				"terraform",
				"kubernetes",
				"gcp",
			},
		},
		{
			name:        "baseline help",
			args:        []string{"baseline", "--help"},
			expectError: false,
			expectContains: []string{
				"Manage infrastructure baselines",
				"create",
				"list",
				"show",
				"delete",
			},
		},
		{
			name:        "diff help",
			args:        []string{"diff", "--help"},
			expectError: false,
			expectContains: []string{
				"Compare infrastructure states",
				"--baseline",
				"--from",
				"--to",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWGOCLI(tt.args...)

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v\nstderr: %s", tt.expectError, err, stderr)
			}

			output := stdout
			if output == "" {
				output = stderr
			}

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, output)
				}
			}
		})
	}
}

// TestScanWorkflow tests the complete scan workflow
func TestScanWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tempDir := t.TempDir()

	// Create a simple terraform state file for testing
	tfState := createTestTerraformState(t, tempDir)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:        "scan without provider shows available",
			args:        []string{"scan"},
			expectError: false,
			expectContains: []string{
				"Auto-discovering infrastructure",
			},
		},
		{
			name:        "scan terraform with state file",
			args:        []string{"scan", "--provider", "terraform", "--state-file", tfState},
			expectError: false,
			expectContains: []string{
				"Infrastructure Scan",
				"Collecting resources from terraform",
				"Collection completed",
				"Resources found:",
			},
		},
		{
			name: "scan terraform and save output",
			args: []string{
				"scan",
				"--provider", "terraform",
				"--state-file", tfState,
				"--output-file", filepath.Join(tempDir, "snapshot.json"),
			},
			expectError: false,
			expectContains: []string{
				"Output saved to:",
			},
		},
		{
			name:        "scan kubernetes without config",
			args:        []string{"scan", "--provider", "kubernetes"},
			expectError: true,
			expectContains: []string{
				"configuration validation failed",
			},
		},
		{
			name:        "scan gcp without project",
			args:        []string{"scan", "--provider", "gcp"},
			expectError: true,
			expectContains: []string{
				"project_id is required",
			},
		},
		{
			name:        "scan gcp with project",
			args:        []string{"scan", "--provider", "gcp", "--project", "test-project"},
			expectError: true,
			expectContains: []string{
				"Collecting resources from gcp",
				"failed to initialize collector",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWGOCLI(tt.args...)

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v\nstderr: %s", tt.expectError, err, stderr)
			}

			output := stdout + stderr

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, output)
				}
			}
		})
	}
}

// TestBaselineWorkflow tests baseline management workflow
func TestBaselineWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	// Create config with temp storage path
	createTestConfig(t, configFile, tempDir)

	// Create a snapshot first
	snapshotFile := filepath.Join(tempDir, "test-snapshot.json")
	createTestSnapshot(t, snapshotFile)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
	}{
		{
			name:        "list baselines empty",
			args:        []string{"baseline", "list", "--config", configFile},
			expectError: false,
			expectContains: []string{
				"Infrastructure Baselines",
			},
		},
		{
			name: "create baseline from snapshot",
			args: []string{
				"baseline", "create",
				"--name", "test-baseline",
				"--from", snapshotFile,
				"--config", configFile,
			},
			expectError: false,
			expectContains: []string{
				"Creating Baseline",
				"test-baseline",
			},
		},
		{
			name:        "list baselines after create",
			args:        []string{"baseline", "list", "--config", configFile},
			expectError: false,
			expectContains: []string{
				"Infrastructure Baselines",
			},
		},
		{
			name: "show specific baseline",
			args: []string{
				"baseline", "show",
				"test-baseline",
				"--config", configFile,
			},
			expectError: false,
			expectContains: []string{
				"Baseline Details",
				"test-baseline",
			},
		},
		{
			name: "delete baseline",
			args: []string{
				"baseline", "delete",
				"test-baseline",
				"--config", configFile,
			},
			expectError: false,
			expectContains: []string{
				"Deleting baseline",
				"test-baseline",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWGOCLI(tt.args...)

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v\nstderr: %s", tt.expectError, err, stderr)
			}

			output := stdout + stderr

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, output)
				}
			}
		})
	}
}

// TestDiffWorkflow tests the diff comparison workflow
func TestDiffWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tempDir := t.TempDir()

	// Create two different snapshots
	snapshot1 := filepath.Join(tempDir, "snapshot1.json")
	snapshot2 := filepath.Join(tempDir, "snapshot2.json")
	createTestSnapshot(t, snapshot1)
	createModifiedSnapshot(t, snapshot2)

	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectContains []string
		expectFormats  map[string][]string
	}{
		{
			name: "diff two snapshots",
			args: []string{
				"diff",
				"--from", snapshot1,
				"--to", snapshot2,
			},
			expectError: false,
			expectContains: []string{
				"Comparing Infrastructure States",
				"Loading snapshots",
				"Comparison completed",
			},
		},
		{
			name: "diff with json output",
			args: []string{
				"diff",
				"--from", snapshot1,
				"--to", snapshot2,
				"--format", "json",
			},
			expectError: false,
			expectFormats: map[string][]string{
				"json": {"{", "\"total_changes\":", "\"added\":", "\"modified\":", "\"removed\":"},
			},
		},
		{
			name: "diff with output file",
			args: []string{
				"diff",
				"--from", snapshot1,
				"--to", snapshot2,
				"--output", filepath.Join(tempDir, "diff-report.json"),
				"--format", "json",
			},
			expectError: false,
			expectContains: []string{
				"Report exported to:",
			},
		},
		{
			name: "diff with missing files",
			args: []string{
				"diff",
				"--from", "/non/existent/file1.json",
				"--to", "/non/existent/file2.json",
			},
			expectError: true,
			expectContains: []string{
				"failed to load",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runWGOCLI(tt.args...)

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v\nstderr: %s", tt.expectError, err, stderr)
			}

			output := stdout + stderr

			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', but it didn't.\nOutput: %s", expected, output)
				}
			}

			// Check format-specific expectations
			if tt.expectFormats != nil {
				for _, expectations := range tt.expectFormats {
					for _, expected := range expectations {
						if !strings.Contains(stdout, expected) {
							t.Errorf("Expected formatted output to contain '%s', but it didn't", expected)
						}
					}
				}
			}
		})
	}
}

// TestProviderIntegration tests provider-specific integration
func TestProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tests := []struct {
		name        string
		provider    string
		setupFunc   func(t *testing.T) (args []string, cleanup func())
		expectError bool
	}{
		{
			name:     "terraform provider",
			provider: "terraform",
			setupFunc: func(t *testing.T) ([]string, func()) {
				tempDir := t.TempDir()
				tfState := createTestTerraformState(t, tempDir)
				return []string{"scan", "--provider", "terraform", "--state-file", tfState}, func() {}
			},
			expectError: false,
		},
		{
			name:     "kubernetes provider",
			provider: "kubernetes",
			setupFunc: func(t *testing.T) ([]string, func()) {
				return []string{"scan", "--provider", "kubernetes", "--namespace", "default"}, func() {}
			},
			expectError: true, // Expected to fail without kubeconfig
		},
		{
			name:     "gcp provider",
			provider: "gcp",
			setupFunc: func(t *testing.T) ([]string, func()) {
				return []string{"scan", "--provider", "gcp", "--project", "test-project", "--region", "us-central1"}, func() {}
			},
			expectError: true, // Expected to fail without credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, cleanup := tt.setupFunc(t)
			defer cleanup()

			stdout, stderr, err := runWGOCLI(args...)

			if (err != nil) != tt.expectError {
				t.Fatalf("Expected error: %v, got: %v\nstderr: %s", tt.expectError, err, stderr)
			}

			// Verify provider name appears in output
			output := stdout + stderr
			if !strings.Contains(output, tt.provider) {
				t.Errorf("Expected output to contain provider name '%s'", tt.provider)
			}
		})
	}
}

// Helper functions

func runWGOCLI(args ...string) (string, string, error) {
	cmd := exec.Command("./vaino", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func createTestTerraformState(t *testing.T, dir string) string {
	t.Helper()

	stateFile := filepath.Join(dir, "terraform.tfstate")
	state := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.0.0",
		"resources": []interface{}{
			map[string]interface{}{
				"type": "aws_instance",
				"name": "test",
				"instances": []interface{}{
					map[string]interface{}{
						"attributes": map[string]interface{}{
							"id":            "i-1234567890",
							"instance_type": "t2.micro",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		t.Fatalf("Failed to write test state: %v", err)
	}

	return stateFile
}

func createTestSnapshot(t *testing.T, file string) {
	t.Helper()

	snapshot := map[string]interface{}{
		"id":        "test-snapshot-1",
		"timestamp": time.Now().Format(time.RFC3339),
		"provider":  "terraform",
		"resources": []interface{}{
			map[string]interface{}{
				"id":       "aws_instance.test",
				"type":     "aws_instance",
				"name":     "test",
				"provider": "terraform",
				"configuration": map[string]interface{}{
					"instance_type": "t2.micro",
				},
			},
		},
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatalf("Failed to write snapshot: %v", err)
	}
}

func createModifiedSnapshot(t *testing.T, file string) {
	t.Helper()

	snapshot := map[string]interface{}{
		"id":        "test-snapshot-2",
		"timestamp": time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		"provider":  "terraform",
		"resources": []interface{}{
			map[string]interface{}{
				"id":       "aws_instance.test",
				"type":     "aws_instance",
				"name":     "test",
				"provider": "terraform",
				"configuration": map[string]interface{}{
					"instance_type": "t2.small", // Changed
				},
			},
			map[string]interface{}{
				"id":       "aws_s3_bucket.new",
				"type":     "aws_s3_bucket",
				"name":     "new-bucket",
				"provider": "terraform",
				"configuration": map[string]interface{}{
					"bucket": "my-new-bucket",
				},
			},
		},
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal snapshot: %v", err)
	}

	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatalf("Failed to write snapshot: %v", err)
	}
}

func createTestConfig(t *testing.T, file string, storageDir string) {
	t.Helper()

	config := fmt.Sprintf(`
storage:
  base_path: %s
output:
  format: table
  pretty: true
`, storageDir)

	if err := os.WriteFile(file, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
}
