package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestWatchCommandE2E tests the watch command end-to-end
func TestWatchCommandE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build the vaino binary
	binPath := buildWGO(t)

	// Test basic watch functionality
	t.Run("BasicWatch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath, "watch", "--interval", "5s", "--quiet")
		output, err := cmd.CombinedOutput()

		// Should exit cleanly when context times out
		if err != nil && ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("Watch command failed: %v\nOutput: %s", err, output)
		}
	})

	// Test JSON output format
	t.Run("JSONOutput", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath, "watch", "--format", "json", "--interval", "5s")

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatalf("Failed to create stdout pipe: %v", err)
		}

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start command: %v", err)
		}

		// Read output
		var jsonOutput bytes.Buffer
		go io.Copy(&jsonOutput, stdout)

		// Let it run for a bit
		time.Sleep(2 * time.Second)
		cancel()
		cmd.Wait()

		// Verify JSON output format
		output := jsonOutput.String()
		if output != "" && !strings.Contains(output, "{") {
			t.Errorf("Expected JSON output, got: %s", output)
		}
	})

	// Test provider filtering
	t.Run("ProviderFilter", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath, "watch", "--provider", "kubernetes", "--quiet")
		output, err := cmd.CombinedOutput()

		if err != nil && ctx.Err() != context.DeadlineExceeded {
			t.Fatalf("Watch command failed: %v\nOutput: %s", err, output)
		}
	})

	// Test help output
	t.Run("HelpOutput", func(t *testing.T) {
		cmd := exec.Command(binPath, "watch", "--help")
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Fatalf("Help command failed: %v\nOutput: %s", err, output)
		}

		// Verify help contains expected content
		helpText := string(output)
		expectedStrings := []string{
			"Real-time infrastructure monitoring",
			"--interval",
			"--provider",
			"--format",
			"--webhook",
			"--high-confidence",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(helpText, expected) {
				t.Errorf("Help output missing expected string: %s", expected)
			}
		}
	})
}

// TestWatchWithMockInfrastructure tests watch with simulated infrastructure changes
func TestWatchWithMockInfrastructure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binPath := buildWGO(t)
	testDir := t.TempDir()

	// Create initial Terraform state
	initialState := createTerraformState("t2.micro", 1)
	stateFile := filepath.Join(testDir, "terraform.tfstate")
	if err := os.WriteFile(stateFile, []byte(initialState), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Start watch in background
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "watch",
		"--provider", "terraform",
		"--interval", "5s",
		"--format", "json")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start watch command: %v", err)
	}

	// Capture output
	var outputMu sync.Mutex
	var outputs []string
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputMu.Lock()
			outputs = append(outputs, line)
			outputMu.Unlock()
		}
	}()

	// Wait for initial snapshot
	time.Sleep(2 * time.Second)

	// Modify infrastructure
	modifiedState := createTerraformState("t2.large", 2)
	if err := os.WriteFile(stateFile, []byte(modifiedState), 0644); err != nil {
		t.Fatalf("Failed to update state file: %v", err)
	}

	// Wait for change detection
	time.Sleep(10 * time.Second)

	// Stop watch
	cancel()
	cmd.Wait()

	// Verify changes were detected
	outputMu.Lock()
	defer outputMu.Unlock()

	changeDetected := false
	for _, output := range outputs {
		if strings.Contains(output, "modified") || strings.Contains(output, "changes") {
			changeDetected = true
			break
		}
	}

	if !changeDetected {
		t.Error("Expected changes to be detected")
		t.Logf("Outputs: %v", outputs)
	}
}

// TestWatchGracefulShutdown tests Ctrl+C handling
func TestWatchGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binPath := buildWGO(t)

	cmd := exec.Command(binPath, "watch", "--interval", "5s")

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start command: %v", err)
	}

	// Let it run briefly
	time.Sleep(2 * time.Second)

	// Send interrupt signal
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to send interrupt: %v", err)
	}

	// Wait for graceful shutdown
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Should exit cleanly
		if err != nil && !strings.Contains(err.Error(), "interrupt") {
			t.Errorf("Expected clean shutdown, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Command did not shut down gracefully")
		cmd.Process.Kill()
	}
}

// TestWatchWithWebhook tests webhook integration
func TestWatchWithWebhook(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	binPath := buildWGO(t)

	// Create a simple webhook receiver
	webhookReceived := make(chan bool, 1)
	webhookServer := startWebhookServer(t, webhookReceived)
	defer webhookServer.Close()

	// Run watch with webhook
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "watch",
		"--webhook", webhookServer.URL,
		"--interval", "5s",
		"--quiet")

	output, err := cmd.CombinedOutput()
	if err != nil && ctx.Err() != context.DeadlineExceeded {
		t.Fatalf("Watch command failed: %v\nOutput: %s", err, output)
	}

	// In a real scenario, we would trigger changes and verify webhook
	// For now, just verify the command accepts webhook parameter
}

// Helper functions

func buildWGO(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "wgo")
	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/vaino")

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build wgo: %v\nOutput: %s", err, output)
	}

	return binPath
}

func createTerraformState(instanceType string, count int) string {
	instances := make([]string, count)
	for i := 0; i < count; i++ {
		instances[i] = fmt.Sprintf(`{
			"attributes": {
				"id": "i-%d",
				"instance_type": "%s",
				"tags": {
					"Name": "WebServer-%d"
				}
			}
		}`, i, instanceType, i)
	}

	return fmt.Sprintf(`{
		"version": 4,
		"terraform_version": "1.0.0",
		"resources": [
			{
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [%s]
			}
		]
	}`, strings.Join(instances, ","))
}

func startWebhookServer(t *testing.T, received chan<- bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify content type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected application/json, got %s", ct)
		}

		// Read body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode webhook: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Signal receipt
		select {
		case received <- true:
		default:
		}

		w.WriteHeader(http.StatusOK)
	}))
}
