package errors

import (
	"fmt"
	"os"
	"strings"
)

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeAuthentication ErrorType = "Authentication"
	ErrorTypeConfiguration  ErrorType = "Configuration"
	ErrorTypeProvider       ErrorType = "Provider"
	ErrorTypeFileSystem     ErrorType = "FileSystem"
	ErrorTypeNetwork        ErrorType = "Network"
	ErrorTypePermission     ErrorType = "Permission"
	ErrorTypeValidation     ErrorType = "Validation"
)

// Provider represents infrastructure provider
type Provider string

const (
	ProviderGCP        Provider = "GCP"
	ProviderAWS        Provider = "AWS"
	ProviderKubernetes Provider = "Kubernetes"
	ProviderTerraform  Provider = "Terraform"
	ProviderUnknown    Provider = "Unknown"
)

// WGOError represents a user-friendly error with actionable guidance
type WGOError struct {
	Type        ErrorType
	Provider    Provider
	Message     string
	Cause       string
	Solutions   []string
	Verify      string
	Help        string
	Environment string
}

// Error implements the error interface
func (e *WGOError) Error() string {
	var sb strings.Builder
	
	// Main error message
	sb.WriteString(fmt.Sprintf("\nError: %s\n", e.Message))
	
	// Cause if available
	if e.Cause != "" {
		sb.WriteString(fmt.Sprintf("Cause: %s\n", e.Cause))
	}
	
	// Environment context
	if e.Environment != "" {
		sb.WriteString(fmt.Sprintf("Environment: %s\n", e.Environment))
	}
	
	// Solutions
	if len(e.Solutions) > 0 {
		sb.WriteString("\nSolutions:\n")
		for _, solution := range e.Solutions {
			sb.WriteString(fmt.Sprintf("  %s\n", solution))
		}
	}
	
	// Verification step
	if e.Verify != "" {
		sb.WriteString(fmt.Sprintf("\nVerify: %s\n", e.Verify))
	}
	
	// Help command
	if e.Help != "" {
		sb.WriteString(fmt.Sprintf("Help: %s\n", e.Help))
	}
	
	return sb.String()
}

// Format implements fmt.Formatter for custom formatting
func (e *WGOError) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		fmt.Fprintf(f, "%s", e.Error())
	case 'v':
		if f.Flag('+') {
			// Verbose mode includes type and provider
			fmt.Fprintf(f, "[%s/%s] %s", e.Type, e.Provider, e.Error())
		} else {
			fmt.Fprintf(f, "%s", e.Error())
		}
	}
}

// New creates a new WGOError
func New(errType ErrorType, provider Provider, message string) *WGOError {
	return &WGOError{
		Type:        errType,
		Provider:    provider,
		Message:     message,
		Environment: detectEnvironment(),
	}
}

// WithCause adds cause information
func (e *WGOError) WithCause(cause string) *WGOError {
	e.Cause = cause
	return e
}

// WithSolutions adds solution steps
func (e *WGOError) WithSolutions(solutions ...string) *WGOError {
	e.Solutions = append(e.Solutions, solutions...)
	return e
}

// WithVerify adds verification command
func (e *WGOError) WithVerify(verify string) *WGOError {
	e.Verify = verify
	return e
}

// WithHelp adds help command
func (e *WGOError) WithHelp(help string) *WGOError {
	e.Help = help
	return e
}

// detectEnvironment detects the current environment
func detectEnvironment() string {
	// Check for CI/CD environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_HOME"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return "CI/CD detected"
		}
	}
	
	// Check for container environment
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "Container environment detected"
	}
	
	// Check for cloud shell
	if os.Getenv("CLOUD_SHELL") == "true" || os.Getenv("GOOGLE_CLOUD_SHELL") == "true" {
		return "Cloud Shell detected"
	}
	
	// Default to development workstation
	return "Development workstation detected"
}

// IsUserError checks if error requires user action
func IsUserError(err error) bool {
	_, ok := err.(*WGOError)
	return ok
}

// GetExitCode returns appropriate exit code for error type
func GetExitCode(err error) int {
	wgoErr, ok := err.(*WGOError)
	if !ok {
		return 1 // Generic error
	}
	
	switch wgoErr.Type {
	case ErrorTypeAuthentication:
		return 77 // EX_NOPERM
	case ErrorTypeConfiguration:
		return 78 // EX_CONFIG
	case ErrorTypePermission:
		return 77 // EX_NOPERM
	case ErrorTypeFileSystem:
		return 66 // EX_NOINPUT
	case ErrorTypeNetwork:
		return 69 // EX_UNAVAILABLE
	default:
		return 1
	}
}