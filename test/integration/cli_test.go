package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	vainoBinary = "../../vaino"
)

func TestMain(m *testing.M) {
	// Build the binary before running tests
	cmd := exec.Command("go", "build", "-o", vainoBinary, "../../cmd/vaino")
	if err := cmd.Run(); err != nil {
		panic("Failed to build vaino binary: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(vainoBinary)

	os.Exit(code)
}

func runVainoIntegration(args ...string) (string, string, error) {
	cmd := exec.Command(vainoBinary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestVAINO_Help(t *testing.T) {
	stdout, stderr, err := runVainoIntegration("--help")
	if err != nil {
		t.Fatalf("vaino --help failed: %v\nstderr: %s", err, stderr)
	}

	// Check that help output contains expected content
	expectedContent := []string{
		"VAINO (What's Going On)",
		"infrastructure drift detection tool",
		"Available Commands:",
		"baseline",
		"check",
		"scan",
		"version",
	}

	for _, content := range expectedContent {
		if !strings.Contains(stdout, content) {
			t.Errorf("Expected help output to contain '%s', but it didn't", content)
		}
	}
}

func TestVAINO_Version(t *testing.T) {
	stdout, stderr, err := runVainoIntegration("version")
	if err != nil {
		t.Fatalf("vaino version failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "vaino version") {
		t.Errorf("Expected version output to contain 'vaino version', got: %s", stdout)
	}
}

func TestVAINO_BaselineHelp(t *testing.T) {
	stdout, stderr, err := runVainoIntegration("baseline", "--help")
	if err != nil {
		t.Fatalf("vaino baseline --help failed: %v\nstderr: %s", err, stderr)
	}

	expectedContent := []string{
		"Manage infrastructure baselines",
		"Available Commands:",
		"create",
		"delete",
		"list",
		"show",
	}

	for _, content := range expectedContent {
		if !strings.Contains(stdout, content) {
			t.Errorf("Expected baseline help to contain '%s', but it didn't", content)
		}
	}
}

func TestVAINO_ScanHelp(t *testing.T) {
	stdout, stderr, err := runVainoIntegration("scan", "--help")
	if err != nil {
		t.Fatalf("vaino scan --help failed: %v\nstderr: %s", err, stderr)
	}

	expectedContent := []string{
		"Scan discovers and collects",
		"infrastructure state",
		"--provider",
		"--region",
		"terraform",
		"aws",
		"kubernetes",
	}

	for _, content := range expectedContent {
		if !strings.Contains(stdout, content) {
			t.Errorf("Expected scan help to contain '%s', but it didn't", content)
		}
	}
}

func TestVAINO_CheckHelp(t *testing.T) {
	stdout, stderr, err := runVainoIntegration("check", "--help")
	if err != nil {
		t.Fatalf("vaino check --help failed: %v\nstderr: %s", err, stderr)
	}

	expectedContent := []string{
		"Check for infrastructure drift",
		"--baseline",
		"--explain",
		"--risk-threshold",
	}

	for _, content := range expectedContent {
		if !strings.Contains(stdout, content) {
			t.Errorf("Expected check help to contain '%s', but it didn't", content)
		}
	}
}

func TestVAINO_BaselineCreate(t *testing.T) {
	tmpDir := t.TempDir()

	// Test baseline create command
	stdout, stderr, err := runVainoIntegration("baseline", "create", "--name", "test-baseline", "--config", filepath.Join(tmpDir, "config.yaml"))

	// Should show "not implemented" message but not error
	if err != nil {
		t.Logf("baseline create stderr: %s", stderr)
		// Command should run but show not implemented message
	}

	if !strings.Contains(stdout, "Creating Baseline") {
		t.Errorf("Expected output to contain 'Creating Baseline', got: %s", stdout)
	}

	if !strings.Contains(stdout, "test-baseline") {
		t.Errorf("Expected output to contain baseline name 'test-baseline', got: %s", stdout)
	}
}

func TestVAINO_BaselineList(t *testing.T) {
	tmpDir := t.TempDir()

	stdout, stderr, err := runVainoIntegration("baseline", "list", "--config", filepath.Join(tmpDir, "config.yaml"))

	// Should show "not implemented" message but not error
	if err != nil {
		t.Logf("baseline list stderr: %s", stderr)
	}

	if !strings.Contains(stdout, "Infrastructure Baselines") {
		t.Errorf("Expected output to contain 'Infrastructure Baselines', got: %s", stdout)
	}
}

func TestVAINO_ScanTerraform(t *testing.T) {
	tmpDir := t.TempDir()

	stdout, stderr, err := runVainoIntegration("scan", "--provider", "terraform", "--path", tmpDir, "--config", filepath.Join(tmpDir, "config.yaml"))

	// Should show scan output but may error due to no state files
	if err != nil {
		t.Logf("scan terraform stderr: %s", stderr)
	}

	if !strings.Contains(stdout, "Scanning infrastructure") {
		t.Errorf("Expected output to contain 'Scanning infrastructure', got: %s", stdout)
	}
}

func TestVAINO_InvalidCommand(t *testing.T) {
	_, stderr, err := runVainoIntegration("invalid-command")

	if err == nil {
		t.Error("Expected invalid command to return error")
	}

	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("Expected error about unknown command, got: %s", stderr)
	}
}

func TestVAINO_RequiredFlags(t *testing.T) {
	tmpDir := t.TempDir()

	// Test baseline create without required name flag
	_, stderr, err := runVainoIntegration("baseline", "create", "--config", filepath.Join(tmpDir, "config.yaml"))

	if err == nil {
		t.Error("Expected baseline create without name to return error")
	}

	if !strings.Contains(stderr, "required flag") {
		t.Errorf("Expected error about required flag, got: %s", stderr)
	}
}

func TestVAINO_OutputFormats(t *testing.T) {
	tmpDir := t.TempDir()
	formats := []string{"table", "json", "yaml", "markdown"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			stdout, stderr, err := runVainoIntegration("baseline", "list", "--output", format, "--config", filepath.Join(tmpDir, "config.yaml"))

			// Should not error due to invalid format
			if err != nil {
				t.Logf("baseline list with format %s stderr: %s", format, stderr)
			}

			// Should contain some output
			if stdout == "" {
				t.Errorf("Expected some output for format %s", format)
			}
		})
	}
}

func TestVAINO_ConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	// Create a basic config file
	configContent := `
storage:
  base_path: ` + tmpDir + `
output:
  format: json
  pretty: true
logging:
  level: debug
`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	stdout, stderr, err := runVainoIntegration("baseline", "list", "--config", configFile)

	if err != nil {
		t.Logf("baseline list with config stderr: %s", stderr)
	}

	// Should process without config errors
	if strings.Contains(stderr, "config") && strings.Contains(stderr, "error") {
		t.Errorf("Unexpected config error: %s", stderr)
	}

	if stdout == "" {
		t.Error("Expected some output when using config file")
	}
}

func TestVAINO_GlobalFlags(t *testing.T) {
	tmpDir := t.TempDir()

	// Test verbose flag
	stdout, stderr, err := runVainoIntegration("version", "--verbose", "--config", filepath.Join(tmpDir, "config.yaml"))

	if err != nil {
		t.Logf("version with verbose stderr: %s", stderr)
	}

	if !strings.Contains(stdout, "version") {
		t.Errorf("Expected version output with verbose flag, got: %s", stdout)
	}

	// Test debug flag
	stdout, stderr, err = runVainoIntegration("version", "--debug", "--config", filepath.Join(tmpDir, "config.yaml"))

	if err != nil {
		t.Logf("version with debug stderr: %s", stderr)
	}

	if !strings.Contains(stdout, "version") {
		t.Errorf("Expected version output with debug flag, got: %s", stdout)
	}
}
