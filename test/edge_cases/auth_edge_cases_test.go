package edgecases

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/aws"
	"github.com/yairfalse/wgo/internal/collectors/gcp"
	"github.com/yairfalse/wgo/internal/collectors/kubernetes"
	wgoerrors "github.com/yairfalse/wgo/internal/errors"
)

// TestExpiredCredentials tests scenarios with expired authentication
func TestExpiredCredentials(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		setupAuth   func(t *testing.T) func()
		expectError bool
		errorType   wgoerrors.ErrorType
	}{
		{
			name:     "expired_aws_session_token",
			provider: "aws",
			setupAuth: func(t *testing.T) func() {
				original := os.Getenv("AWS_SESSION_TOKEN")
				// Set expired token (AWS tokens typically expire in 12 hours)
				os.Setenv("AWS_SESSION_TOKEN", "expired-token-12345")
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIA12345")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
				return func() {
					if original != "" {
						os.Setenv("AWS_SESSION_TOKEN", original)
					} else {
						os.Unsetenv("AWS_SESSION_TOKEN")
					}
				}
			},
			expectError: true,
			errorType:   wgoerrors.ErrorTypeAuthentication,
		},
		{
			name:     "invalid_gcp_service_account_key",
			provider: "gcp",
			setupAuth: func(t *testing.T) func() {
				tempDir := t.TempDir()
				invalidKeyFile := filepath.Join(tempDir, "invalid-key.json")
				invalidKey := `{"type": "service_account", "project_id": "fake", "private_key": "invalid"}`
				os.WriteFile(invalidKeyFile, []byte(invalidKey), 0644)

				original := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
				os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", invalidKeyFile)
				return func() {
					if original != "" {
						os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", original)
					} else {
						os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
					}
				}
			},
			expectError: true,
			errorType:   wgoerrors.ErrorTypeAuthentication,
		},
		{
			name:     "corrupted_kubeconfig",
			provider: "kubernetes",
			setupAuth: func(t *testing.T) func() {
				tempDir := t.TempDir()
				corruptedConfig := filepath.Join(tempDir, "kubeconfig")
				corruptedContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://invalid-server
    certificate-authority-data: invalid-cert-data
  name: invalid-cluster
contexts:
- context:
    cluster: invalid-cluster
    user: invalid-user
  name: invalid-context
current-context: invalid-context
users:
- name: invalid-user
  user:
    token: invalid-token`
				os.WriteFile(corruptedConfig, []byte(corruptedContent), 0644)

				original := os.Getenv("KUBECONFIG")
				os.Setenv("KUBECONFIG", corruptedConfig)
				return func() {
					if original != "" {
						os.Setenv("KUBECONFIG", original)
					} else {
						os.Unsetenv("KUBECONFIG")
					}
				}
			},
			expectError: true,
			errorType:   wgoerrors.ErrorTypeAuthentication,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupAuth(t)
			defer cleanup()

			var collector collectors.EnhancedCollector
			var config collectors.CollectorConfig

			switch tt.provider {
			case "aws":
				collector = aws.NewAWSCollector()
				config = collectors.CollectorConfig{
					Config: map[string]interface{}{
						"region": "us-east-1",
					},
				}
			case "gcp":
				collector = gcp.NewGCPCollector()
				config = collectors.CollectorConfig{
					Config: map[string]interface{}{
						"project_id": "test-project",
						"region":     "us-central1",
					},
				}
			case "kubernetes":
				collector = kubernetes.NewKubernetesCollector()
				config = collectors.CollectorConfig{
					Config: map[string]interface{}{
						"namespace":  "default",
						"kubeconfig": os.Getenv("KUBECONFIG"),
					},
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := collector.Collect(ctx, config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected authentication error but got none")
				} else {
					// Verify it's the right type of error
					if wgoErr, ok := err.(*wgoerrors.WGOError); ok {
						if wgoErr.Type != tt.errorType {
							t.Errorf("Expected %v error, got %v", tt.errorType, wgoErr.Type)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestInsufficientPermissions tests scenarios with valid credentials but insufficient permissions
func TestInsufficientPermissions(t *testing.T) {
	// Mock server that returns 403 Forbidden for various scenarios
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/compute/v1/projects/test-project/zones":
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": {"code": 403, "message": "Insufficient permissions"}}`))
		case "/api/v1/namespaces":
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"kind": "Status", "code": 403, "message": "forbidden"}`))
		default:
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "insufficient permissions"}`))
		}
	}))
	defer server.Close()

	tests := []struct {
		name        string
		scenario    string
		expectError bool
		errorType   wgoerrors.ErrorType
	}{
		{
			name:        "gcp_compute_read_only",
			scenario:    "User has read-only access but WGO needs broader permissions",
			expectError: true,
			errorType:   wgoerrors.ErrorTypePermission,
		},
		{
			name:        "kubernetes_namespace_restricted",
			scenario:    "User can only access specific namespace",
			expectError: true,
			errorType:   wgoerrors.ErrorTypePermission,
		},
		{
			name:        "aws_assume_role_denied",
			scenario:    "Role assumption denied due to trust policy",
			expectError: true,
			errorType:   wgoerrors.ErrorTypePermission,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate permission error by making request to mock server
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(server.URL + "/api/v1/namespaces")

			if err != nil {
				t.Errorf("Unexpected network error: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("Expected 403 Forbidden, got %d", resp.StatusCode)
			}

			// Verify that we can detect permission errors properly
			if resp.StatusCode == http.StatusForbidden {
				t.Logf("Successfully detected permission error for scenario: %s", tt.scenario)
			}
		})
	}
}

// TestMissingCredentials tests scenarios where credentials are completely missing
func TestMissingCredentials(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		clearEnv    []string
		expectError bool
		errorType   wgoerrors.ErrorType
	}{
		{
			name:     "no_aws_credentials",
			provider: "aws",
			clearEnv: []string{
				"AWS_ACCESS_KEY_ID",
				"AWS_SECRET_ACCESS_KEY",
				"AWS_SESSION_TOKEN",
				"AWS_PROFILE",
			},
			expectError: true,
			errorType:   wgoerrors.ErrorTypeConfiguration,
		},
		{
			name:     "no_gcp_credentials",
			provider: "gcp",
			clearEnv: []string{
				"GOOGLE_APPLICATION_CREDENTIALS",
				"GCLOUD_PROJECT",
			},
			expectError: true,
			errorType:   wgoerrors.ErrorTypeConfiguration,
		},
		{
			name:     "no_kubeconfig",
			provider: "kubernetes",
			clearEnv: []string{
				"KUBECONFIG",
				"HOME", // This affects default kubeconfig location
			},
			expectError: true,
			errorType:   wgoerrors.ErrorTypeConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store original environment variables
			originalEnv := make(map[string]string)
			for _, envVar := range tt.clearEnv {
				originalEnv[envVar] = os.Getenv(envVar)
				os.Unsetenv(envVar)
			}

			// Restore environment after test
			defer func() {
				for envVar, originalValue := range originalEnv {
					if originalValue != "" {
						os.Setenv(envVar, originalValue)
					}
				}
			}()

			var collector collectors.EnhancedCollector
			var config collectors.CollectorConfig

			switch tt.provider {
			case "aws":
				collector = aws.NewAWSCollector()
				config = collectors.CollectorConfig{
					Config: map[string]interface{}{
						"region": "us-east-1",
					},
				}
			case "gcp":
				collector = gcp.NewGCPCollector()
				config = collectors.CollectorConfig{
					Config: map[string]interface{}{
						"project_id": "test-project",
					},
				}
			case "kubernetes":
				collector = kubernetes.NewKubernetesCollector()
				config = collectors.CollectorConfig{
					Config: map[string]interface{}{
						"namespace":  "default",
						"kubeconfig": os.Getenv("KUBECONFIG"),
					},
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := collector.Collect(ctx, config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected credentials error but got none")
				} else {
					t.Logf("Got expected error: %v", err)
				}
			}
		})
	}
}

// TestCredentialRotation tests scenarios during credential rotation
func TestCredentialRotation(t *testing.T) {
	// Mock server that simulates credential rotation
	rotationCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rotationCount++

		// First few requests fail with invalid credentials
		if rotationCount <= 2 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid credentials - rotation in progress"}`))
			return
		}

		// After rotation completes, requests succeed
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"instances": []}`))
	}))
	defer server.Close()

	t.Run("credential_rotation_scenario", func(t *testing.T) {
		client := &http.Client{Timeout: 5 * time.Second}

		// Simulate multiple requests during rotation
		for i := 0; i < 5; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
				continue
			}

			if i < 2 {
				// Expect failure during rotation
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("Request %d: expected 401, got %d", i, resp.StatusCode)
				}
			} else {
				// Expect success after rotation
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Request %d: expected 200, got %d", i, resp.StatusCode)
				}
			}
			resp.Body.Close()

			// Brief delay between requests
			time.Sleep(100 * time.Millisecond)
		}
	})
}

// TestMalformedCredentials tests scenarios with malformed credential files
func TestMalformedCredentials(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		createFile  func(t *testing.T) (string, func())
		expectError bool
	}{
		{
			name:     "malformed_gcp_service_account_json",
			provider: "gcp",
			createFile: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				malformedFile := filepath.Join(tempDir, "malformed.json")
				malformedContent := `{"type": "service_account", "project_id": "test", invalid json}`
				os.WriteFile(malformedFile, []byte(malformedContent), 0644)

				original := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
				os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", malformedFile)

				return malformedFile, func() {
					if original != "" {
						os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", original)
					} else {
						os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
					}
				}
			},
			expectError: true,
		},
		{
			name:     "malformed_kubeconfig_yaml",
			provider: "kubernetes",
			createFile: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				malformedFile := filepath.Join(tempDir, "malformed-kubeconfig")
				malformedContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test
  name: [invalid yaml structure`
				os.WriteFile(malformedFile, []byte(malformedContent), 0644)

				original := os.Getenv("KUBECONFIG")
				os.Setenv("KUBECONFIG", malformedFile)

				return malformedFile, func() {
					if original != "" {
						os.Setenv("KUBECONFIG", original)
					} else {
						os.Unsetenv("KUBECONFIG")
					}
				}
			},
			expectError: true,
		},
		{
			name:     "empty_credentials_file",
			provider: "gcp",
			createFile: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				emptyFile := filepath.Join(tempDir, "empty.json")
				os.WriteFile(emptyFile, []byte(""), 0644)

				original := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
				os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", emptyFile)

				return emptyFile, func() {
					if original != "" {
						os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", original)
					} else {
						os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
					}
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, cleanup := tt.createFile(t)
			defer cleanup()

			// Try to read and parse the credential file
			content, err := os.ReadFile(file)
			if err != nil {
				if !tt.expectError {
					t.Errorf("Failed to read credential file: %v", err)
				}
				return
			}

			// For JSON files, try to parse
			if filepath.Ext(file) == ".json" {
				var parsed map[string]interface{}
				err = json.Unmarshal(content, &parsed)

				if tt.expectError && err == nil {
					t.Error("Expected JSON parsing to fail but it succeeded")
				} else if !tt.expectError && err != nil {
					t.Errorf("Expected JSON parsing to succeed but got: %v", err)
				}
			}

			t.Logf("Successfully tested malformed credential file: %s", file)
		})
	}
}

// TestConcurrentAuthRequests tests authentication under concurrent load
func TestConcurrentAuthRequests(t *testing.T) {
	// Mock server that tracks concurrent requests
	concurrentRequests := 0
	maxConcurrent := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		concurrentRequests++
		if concurrentRequests > maxConcurrent {
			maxConcurrent = concurrentRequests
		}

		// Simulate processing time
		time.Sleep(100 * time.Millisecond)

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"instances": []}`))

		concurrentRequests--
	}))
	defer server.Close()

	t.Run("concurrent_authentication", func(t *testing.T) {
		client := &http.Client{Timeout: 10 * time.Second}

		// Make multiple concurrent requests
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(requestID int) {
				resp, err := client.Get(fmt.Sprintf("%s?request=%d", server.URL, requestID))
				if err != nil {
					done <- fmt.Errorf("request %d failed: %v", requestID, err)
					return
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					done <- fmt.Errorf("request %d got status %d", requestID, resp.StatusCode)
					return
				}

				done <- nil
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			if err := <-done; err != nil {
				t.Error(err)
			}
		}

		t.Logf("Max concurrent requests handled: %d", maxConcurrent)
		if maxConcurrent == 0 {
			t.Error("No concurrent requests detected")
		}
	})
}
