package errors

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayError(t *testing.T) {
	// Save original stderr
	oldStderr := os.Stderr

	// Test various error types
	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name: "Authentication Error",
			err:  GCPAuthenticationError(fmt.Errorf("could not find default credentials")),
			contains: []string{
				"GCP authentication failed",
				"Application Default Credentials not found",
				"gcloud auth application-default login",
				"gcloud auth list",
			},
		},
		{
			name: "Network Error",
			err: NetworkError(ProviderAWS, "API endpoint unreachable").
				WithCause("Connection timeout after 30s").
				WithSolutions(
					"Check your internet connection",
					"Verify firewall settings",
					"Try using a VPN if behind corporate proxy",
				),
			contains: []string{
				"API endpoint unreachable",
				"Connection timeout",
				"Check your internet connection",
			},
		},
		{
			name: "Configuration Error",
			err: New(ErrorTypeConfiguration, ProviderTerraform, "Invalid Terraform configuration").
				WithCause("Backend configuration missing").
				WithSolutions("Add backend configuration to main.tf"),
			contains: []string{
				"Invalid Terraform configuration",
				"Backend configuration missing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipe to capture stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Display the error
			DisplayError(tt.err)

			// Close writer and read output
			w.Close()
			buf := &bytes.Buffer{}
			buf.ReadFrom(r)
			output := buf.String()

			// Restore stderr
			os.Stderr = oldStderr

			// Check that expected strings are present
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Output should contain: %s", expected)
			}
		})
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "Authentication Error",
			err:      GCPAuthenticationError(nil),
			expected: 77, // EX_NOPERM
		},
		{
			name:     "Configuration Error",
			err:      New(ErrorTypeConfiguration, ProviderAWS, "Invalid config"),
			expected: 78, // EX_CONFIG
		},
		{
			name:     "Network Error",
			err:      NetworkError(ProviderAWS, "Connection failed"),
			expected: 69, // EX_UNAVAILABLE
		},
		{
			name:     "Generic Error",
			err:      fmt.Errorf("some generic error"),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := GetExitCode(tt.err)
			assert.Equal(t, tt.expected, exitCode)
		})
	}
}

func TestFormatErrorWithContext(t *testing.T) {
	err := AWSCredentialsError(fmt.Errorf("no credentials found")).
		WithCause("No credentials found in environment").
		WithSolutions("Run 'aws configure'", "Set AWS_ACCESS_KEY_ID")

	context := map[string]string{
		"Region":  "us-east-1",
		"Profile": "default",
		"CI":      "true",
	}

	output := FormatErrorWithContext(err, context)

	// Check plain text formatting (no colors)
	assert.Contains(t, output, "AWS credentials not found")
	assert.Contains(t, output, "Type: Authentication/AWS")
	assert.Contains(t, output, "Context:")
	assert.Contains(t, output, "Region: us-east-1")
	assert.Contains(t, output, "1. Run 'aws configure'")
}
