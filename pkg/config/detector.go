package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProviderDetector detects available providers
type ProviderDetector struct{}

// NewProviderDetector creates a new provider detector
func NewProviderDetector() *ProviderDetector {
	return &ProviderDetector{}
}

// DetectionResult contains the result of provider detection
type DetectionResult struct {
	Available  bool
	Status     string
	Version    string
	StateFiles int      // For Terraform
	StatePaths []string // For Terraform
}

// DetectAll detects all available providers
func (d *ProviderDetector) DetectAll() map[string]DetectionResult {
	results := make(map[string]DetectionResult)

	results["terraform"] = d.DetectTerraform()
	results["gcp"] = d.DetectGCP()
	results["aws"] = d.DetectAWS()
	results["kubernetes"] = d.DetectKubernetes()

	return results
}

// DetectTerraform detects Terraform state files
func (d *ProviderDetector) DetectTerraform() DetectionResult {
	result := DetectionResult{
		Available: true, // Terraform provider is always available
		Status:    "ready",
	}

	// Look for state files
	patterns := []string{
		"terraform.tfstate",
		"*.tfstate",
		"terraform/*.tfstate",
		"**/*.tfstate",
	}

	statePaths := []string{}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			// Skip backup files
			if !strings.HasSuffix(match, ".backup") {
				statePaths = append(statePaths, match)
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniquePaths := []string{}
	for _, path := range statePaths {
		if !seen[path] {
			seen[path] = true
			uniquePaths = append(uniquePaths, path)
		}
	}

	result.StateFiles = len(uniquePaths)
	result.StatePaths = uniquePaths

	if result.StateFiles > 0 {
		result.Status = "ready"
	} else {
		result.Status = "no state files found"
	}

	return result
}

// DetectGCP detects Google Cloud SDK
func (d *ProviderDetector) DetectGCP() DetectionResult {
	result := DetectionResult{}

	// Check if gcloud is installed
	cmd := exec.Command("gcloud", "version", "--format=json")
	output, err := cmd.Output()

	if err != nil {
		result.Available = false
		result.Status = "gcloud CLI not found"
		return result
	}

	result.Available = true

	// Extract version
	outputStr := string(output)
	if strings.Contains(outputStr, "Google Cloud SDK") {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Google Cloud SDK") {
				parts := strings.Fields(line)
				if len(parts) >= 4 {
					result.Version = parts[3]
				}
				break
			}
		}
	}

	result.Status = "gcloud CLI found"
	if result.Version != "" {
		result.Status += " (v" + result.Version + ")"
	}

	return result
}

// DetectAWS detects AWS CLI
func (d *ProviderDetector) DetectAWS() DetectionResult {
	result := DetectionResult{}

	// Check if aws cli is installed
	cmd := exec.Command("aws", "--version")
	output, err := cmd.Output()

	if err != nil {
		result.Available = false
		result.Status = "AWS CLI not found"
		return result
	}

	result.Available = true

	// Extract version
	outputStr := string(output)
	parts := strings.Fields(outputStr)
	if len(parts) > 0 && strings.HasPrefix(parts[0], "aws-cli/") {
		result.Version = strings.TrimPrefix(parts[0], "aws-cli/")
	}

	result.Status = "AWS CLI found"
	if result.Version != "" {
		result.Status += " (v" + result.Version + ")"
	}

	return result
}

// DetectKubernetes detects kubectl
func (d *ProviderDetector) DetectKubernetes() DetectionResult {
	result := DetectionResult{}

	// Check if kubectl is installed
	cmd := exec.Command("kubectl", "version", "--client", "--short")
	output, err := cmd.Output()

	if err != nil {
		result.Available = false
		result.Status = "kubectl not found"
		return result
	}

	result.Available = true

	// Extract version
	outputStr := strings.TrimSpace(string(output))
	if strings.HasPrefix(outputStr, "Client Version:") {
		result.Version = strings.TrimSpace(strings.TrimPrefix(outputStr, "Client Version:"))
	}

	result.Status = "kubectl found"
	if result.Version != "" {
		result.Status += " (" + result.Version + ")"
	}

	return result
}

// AuthChecker checks authentication status for providers
type AuthChecker struct{}

// NewAuthChecker creates a new auth checker
func NewAuthChecker() *AuthChecker {
	return &AuthChecker{}
}

// AuthResult contains authentication check results
type AuthResult struct {
	Authenticated bool
	Message       string
	ProjectID     string // For GCP
	Region        string // For AWS
	Profile       string // For AWS
	Context       string // For Kubernetes
	Namespaces    int    // For Kubernetes
}

// CheckGCP checks GCP authentication
func (a *AuthChecker) CheckGCP() AuthResult {
	result := AuthResult{}

	// Check if authenticated
	cmd := exec.Command("gcloud", "auth", "application-default", "print-access-token")
	if err := cmd.Run(); err != nil {
		result.Authenticated = false
		result.Message = "not authenticated with gcloud"
		return result
	}

	result.Authenticated = true

	// Get current project
	cmd = exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err == nil {
		result.ProjectID = strings.TrimSpace(string(output))
		result.Message = "authenticated"
		if result.ProjectID != "" && result.ProjectID != "(unset)" {
			result.Message += " with project " + result.ProjectID
		}
	}

	return result
}

// CheckAWS checks AWS authentication
func (a *AuthChecker) CheckAWS() AuthResult {
	result := AuthResult{}

	// Check for credentials
	homeDir, _ := os.UserHomeDir()
	credFile := filepath.Join(homeDir, ".aws", "credentials")

	// Check environment variables first
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		result.Authenticated = true
		result.Message = "authenticated via environment variables"
		result.Region = os.Getenv("AWS_DEFAULT_REGION")
		if result.Region == "" {
			result.Region = "us-east-1"
		}
		return result
	}

	// Check credentials file
	if _, err := os.Stat(credFile); err == nil {
		result.Authenticated = true
		result.Message = "authenticated via credentials file"

		// Try to get default profile info
		cmd := exec.Command("aws", "configure", "get", "region")
		output, err := cmd.Output()
		if err == nil {
			result.Region = strings.TrimSpace(string(output))
		}

		// Get profile
		result.Profile = os.Getenv("AWS_PROFILE")
		if result.Profile == "" {
			result.Profile = "default"
		}

		return result
	}

	// Check for IAM role (EC2 instance)
	cmd := exec.Command("aws", "sts", "get-caller-identity")
	if err := cmd.Run(); err == nil {
		result.Authenticated = true
		result.Message = "authenticated via IAM role"
		return result
	}

	result.Authenticated = false
	result.Message = "no AWS credentials found"
	return result
}

// CheckKubernetes checks Kubernetes cluster access
func (a *AuthChecker) CheckKubernetes() AuthResult {
	result := AuthResult{}

	// Get current context
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		result.Authenticated = false
		result.Message = "no kubernetes context configured"
		return result
	}

	result.Context = strings.TrimSpace(string(output))

	// Try to access the cluster
	cmd = exec.Command("kubectl", "get", "namespaces", "--no-headers")
	output, err = cmd.Output()
	if err != nil {
		result.Authenticated = false
		result.Message = "cannot connect to kubernetes cluster"
		return result
	}

	result.Authenticated = true
	result.Message = "cluster accessible"

	// Count namespaces
	lines := strings.Split(string(output), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	result.Namespaces = count

	return result
}
