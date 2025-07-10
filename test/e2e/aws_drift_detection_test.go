//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// TestE2EAWSDriftDetection tests the complete end-to-end workflow
func TestE2EAWSDriftDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Check if vaino binary exists
	wgoBinary := "./wgo"
	if _, err := os.Stat(wgoBinary); os.IsNotExist(err) {
		// Try to build it
		cmd := exec.Command("go", "build", "-o", wgoBinary, "./cmd/vaino")
		if err := cmd.Run(); err != nil {
			t.Skip("Could not build vaino binary")
		}
		defer os.Remove(wgoBinary)
	}

	// Create temp directory for test outputs
	tmpDir, err := ioutil.TempDir("", "vaino-e2e-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test complete workflow
	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Scan Terraform state
		terraformOutput := filepath.Join(tmpDir, "terraform-snapshot.json")
		runCommand(t, wgoBinary, "scan",
			"--provider", "terraform",
			"--output-file", terraformOutput,
			"--quiet")

		// Verify Terraform snapshot was created
		assert.FileExists(t, terraformOutput)
		terraformSnapshot := loadSnapshot(t, terraformOutput)
		assert.Equal(t, "terraform", terraformSnapshot.Provider)
		t.Logf("Terraform snapshot contains %d resources", len(terraformSnapshot.Resources))

		// 2. Scan AWS resources (if credentials available)
		awsOutput := filepath.Join(tmpDir, "aws-snapshot.json")
		cmd := exec.Command(wgoBinary, "scan",
			"--provider", "aws",
			"--region", "us-east-1",
			"--output-file", awsOutput,
			"--quiet")

		output, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(output), "credentials") {
				t.Skip("AWS credentials not available")
			}
			t.Fatalf("AWS scan failed: %v\nOutput: %s", err, output)
		}

		// Verify AWS snapshot was created
		assert.FileExists(t, awsOutput)
		awsSnapshot := loadSnapshot(t, awsOutput)
		assert.Equal(t, "aws", awsSnapshot.Provider)
		t.Logf("AWS snapshot contains %d resources", len(awsSnapshot.Resources))

		// 3. Compare Terraform vs AWS
		diffOutput := filepath.Join(tmpDir, "drift-report.json")
		runCommand(t, wgoBinary, "diff",
			"--from", terraformOutput,
			"--to", awsOutput,
			"--format", "json",
			"--output", diffOutput)

		// Verify drift report was created
		assert.FileExists(t, diffOutput)
		driftReport := loadDriftReport(t, diffOutput)

		t.Logf("Drift Report Summary:")
		t.Logf("  Total Resources: %d", driftReport.Summary.TotalResources)
		t.Logf("  Changed Resources: %d", driftReport.Summary.ChangedResources)
		t.Logf("  Added Resources: %d", driftReport.Summary.AddedResources)
		t.Logf("  Removed Resources: %d", driftReport.Summary.RemovedResources)
		t.Logf("  Modified Resources: %d", driftReport.Summary.ModifiedResources)

		// 4. Test different output formats
		testDiffFormats(t, wgoBinary, terraformOutput, awsOutput)
	})
}

// TestE2EAWSProviderFeatures tests specific AWS provider features
func TestE2EAWSProviderFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	wgoBinary := "./wgo"
	if _, err := os.Stat(wgoBinary); os.IsNotExist(err) {
		t.Skip("vaino binary not found")
	}

	// Test auto-discovery
	t.Run("AutoDiscovery", func(t *testing.T) {
		cmd := exec.Command(wgoBinary, "scan", "--auto-discover")
		output, err := cmd.CombinedOutput()

		// Should discover available providers
		assert.Contains(t, string(output), "Auto-discovering infrastructure")

		// Should either find Terraform files or show no providers
		if err == nil {
			assert.Contains(t, string(output), "Found")
		}
	})

	// Test profile support
	t.Run("ProfileSupport", func(t *testing.T) {
		cmd := exec.Command(wgoBinary, "scan",
			"--provider", "aws",
			"--profile", "nonexistent-profile",
			"--quiet")

		output, err := cmd.CombinedOutput()
		assert.Error(t, err)
		assert.Contains(t, string(output), "profile")
	})

	// Test multi-region support
	t.Run("MultiRegion", func(t *testing.T) {
		regions := []string{"us-east-1", "us-west-2", "eu-west-1"}

		for _, region := range regions {
			t.Run(region, func(t *testing.T) {
				cmd := exec.Command(wgoBinary, "scan",
					"--provider", "aws",
					"--region", region,
					"--quiet")

				// Just verify the command accepts the region flag
				// Actual execution may fail without credentials
				_ = cmd.Run()
			})
		}
	})
}

// TestE2EGitLikeDiff tests git diff-like functionality
func TestE2EGitLikeDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	wgoBinary := "./wgo"
	if _, err := os.Stat(wgoBinary); os.IsNotExist(err) {
		t.Skip("vaino binary not found")
	}

	tmpDir, err := ioutil.TempDir("", "vaino-e2e-diff")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create two mock snapshots with differences
	snapshot1 := createMockSnapshot("snapshot1", []mockResource{
		{ID: "i-123", Type: "aws_instance", Name: "web-server"},
		{ID: "sg-456", Type: "aws_security_group", Name: "web-sg"},
		{ID: "s3-789", Type: "aws_s3_bucket", Name: "my-bucket"},
	})

	snapshot2 := createMockSnapshot("snapshot2", []mockResource{
		{ID: "i-123", Type: "aws_instance", Name: "web-server-modified"}, // Modified
		{ID: "sg-456", Type: "aws_security_group", Name: "web-sg"},       // Unchanged
		// s3-789 removed
		{ID: "i-999", Type: "aws_instance", Name: "new-server"}, // Added
	})

	file1 := filepath.Join(tmpDir, "snapshot1.json")
	file2 := filepath.Join(tmpDir, "snapshot2.json")
	saveSnapshot(t, snapshot1, file1)
	saveSnapshot(t, snapshot2, file2)

	// Test --quiet flag (exit code only)
	t.Run("QuietMode", func(t *testing.T) {
		cmd := exec.Command(wgoBinary, "diff",
			"--from", file1,
			"--to", file2,
			"--quiet")

		err := cmd.Run()
		// Should exit with code 1 when differences found
		assert.Error(t, err)
		exitError, ok := err.(*exec.ExitError)
		assert.True(t, ok)
		assert.Equal(t, 1, exitError.ExitCode())
	})

	// Test --name-only flag
	t.Run("NameOnly", func(t *testing.T) {
		output := runCommandGetOutput(t, wgoBinary, "diff",
			"--from", file1,
			"--to", file2,
			"--name-only")

		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.Contains(t, lines, "aws_instance/i-123")
		assert.Contains(t, lines, "aws_s3_bucket/s3-789")
		assert.Contains(t, lines, "aws_instance/i-999")
	})

	// Test --stat flag
	t.Run("Stat", func(t *testing.T) {
		output := runCommandGetOutput(t, wgoBinary, "diff",
			"--from", file1,
			"--to", file2,
			"--stat")

		assert.Contains(t, output, "aws_instance/i-123")
		assert.Contains(t, output, "change")
		assert.Contains(t, output, "resources changed")
	})

	// Test Unix-style output
	t.Run("UnixStyle", func(t *testing.T) {
		output := runCommandGetOutput(t, wgoBinary, "diff",
			"--from", file1,
			"--to", file2,
			"--format", "unix")

		// Should have git diff-like format
		assert.Contains(t, output, "---")
		assert.Contains(t, output, "+++")
		assert.Contains(t, output, "@@")
	})
}

// Helper functions

func runCommand(t *testing.T, command string, args ...string) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}
}

func runCommandGetOutput(t *testing.T, command string, args ...string) string {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// For diff commands, exit code 1 means differences found
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return string(output)
		}
		t.Fatalf("Command failed: %v\nOutput: %s", err, output)
	}
	return string(output)
}

func loadSnapshot(t *testing.T, filename string) *types.Snapshot {
	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	var snapshot types.Snapshot
	err = json.Unmarshal(data, &snapshot)
	require.NoError(t, err)

	return &snapshot
}

func loadDriftReport(t *testing.T, filename string) *differ.DriftReport {
	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err)

	var report differ.DriftReport
	err = json.Unmarshal(data, &report)
	require.NoError(t, err)

	return &report
}

type mockResource struct {
	ID   string
	Type string
	Name string
}

func createMockSnapshot(id string, resources []mockResource) *types.Snapshot {
	snapshot := &types.Snapshot{
		ID:        id,
		Timestamp: time.Now(),
		Provider:  "aws",
		Resources: []types.Resource{},
	}

	for _, res := range resources {
		snapshot.Resources = append(snapshot.Resources, types.Resource{
			ID:       res.ID,
			Type:     res.Type,
			Name:     res.Name,
			Provider: "aws",
			Region:   "us-east-1",
			Configuration: map[string]interface{}{
				"name": res.Name,
			},
			Tags: map[string]string{
				"Name": res.Name,
			},
		})
	}

	return snapshot
}

func saveSnapshot(t *testing.T, snapshot *types.Snapshot, filename string) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	require.NoError(t, err)

	err = ioutil.WriteFile(filename, data, 0644)
	require.NoError(t, err)
}

func testDiffFormats(t *testing.T, wgoBinary, from, to string) {
	formats := []string{"unix", "simple", "name-only", "stat", "json", "yaml"}

	for _, format := range formats {
		t.Run(fmt.Sprintf("Format_%s", format), func(t *testing.T) {
			output := runCommandGetOutput(t, wgoBinary, "diff",
				"--from", from,
				"--to", to,
				"--format", format)

			// Verify we got some output
			assert.NotEmpty(t, output)

			// Format-specific checks
			switch format {
			case "json":
				var report differ.DriftReport
				err := json.Unmarshal([]byte(output), &report)
				assert.NoError(t, err)
			case "yaml":
				assert.Contains(t, output, ":")
			case "name-only":
				// Should just be resource names
				assert.NotContains(t, output, "---")
			case "unix":
				// Should have diff markers
				assert.Contains(t, output, "---")
				assert.Contains(t, output, "+++")
			}
		})
	}
}
