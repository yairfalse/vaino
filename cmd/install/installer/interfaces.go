package installer

import (
	"context"
	"io"
	"time"
)

// InstallationType represents different installation types with generics support
type InstallationType interface {
	Binary | Container | Kubernetes
}

// Binary represents a binary installation
type Binary struct {
	Platform string
	Arch     string
	Version  string
	Path     string
}

// Container represents a container-based installation
type Container struct {
	Image   string
	Tag     string
	Runtime string // docker, podman, etc.
}

// Kubernetes represents a Kubernetes deployment
type Kubernetes struct {
	Namespace  string
	Deployment string
	ConfigMap  string
	Secret     string
}

// Step represents a generic installation step
type Step[T InstallationType] interface {
	Name() string
	Execute(ctx context.Context, target T) error
	Rollback(ctx context.Context, target T) error
	Validate(ctx context.Context, target T) error
}

// StepResult captures the result of an installation step
type StepResult struct {
	StepName  string
	Success   bool
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Metadata  map[string]interface{}
}

// Installer is the core installer interface
type Installer interface {
	// Install performs the installation
	Install(ctx context.Context) error
	// Rollback reverses the installation
	Rollback(ctx context.Context) error
	// Validate checks the installation
	Validate(ctx context.Context) error
	// Progress returns a channel for progress updates
	Progress() <-chan Progress
}

// Progress represents installation progress
type Progress struct {
	CurrentStep     string
	TotalSteps      int
	CompletedSteps  int
	BytesDownloaded int64
	TotalBytes      int64
	Speed           float64 // bytes per second
	EstimatedTime   time.Duration
	Message         string
}

// Strategy defines the installation strategy pattern
type Strategy interface {
	// Name returns the strategy name
	Name() string
	// CanInstall checks if this strategy can be used
	CanInstall(ctx context.Context) (bool, error)
	// PreInstall prepares for installation
	PreInstall(ctx context.Context) error
	// Install performs the actual installation
	Install(ctx context.Context) error
	// PostInstall performs post-installation tasks
	PostInstall(ctx context.Context) error
	// Rollback reverses the installation
	Rollback(ctx context.Context) error
}

// Factory creates platform-specific installers
type Factory interface {
	// Create returns an installer for the current platform
	Create(options ...Option) (Installer, error)
	// Detect auto-detects the best installation method
	Detect(ctx context.Context) (Strategy, error)
}

// Option represents a functional option for installer configuration
type Option func(*Config) error

// Config holds installer configuration
type Config struct {
	// Installation method (binary, container, kubernetes)
	Method string
	// Installation directory
	InstallDir string
	// Version to install
	Version string
	// Download mirrors
	Mirrors []string
	// Timeout for operations
	Timeout time.Duration
	// Number of retry attempts
	RetryAttempts int
	// Retry backoff configuration
	RetryBackoff time.Duration
	// Enable debug logging
	Debug bool
	// Custom HTTP client for downloads
	HTTPClient HTTPClient
	// Progress writer
	ProgressWriter io.Writer
	// Validation level (basic, full)
	ValidationLevel string
	// Environment variables to set
	Environment map[string]string
	// Features to enable
	Features []string
}

// HTTPClient defines the interface for HTTP operations
type HTTPClient interface {
	Do(ctx context.Context, req *Request) (*Response, error)
}

// Request represents an HTTP request
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    io.Reader
	// For resumable downloads
	RangeStart int64
	RangeEnd   int64
}

// Response represents an HTTP response
type Response struct {
	StatusCode    int
	Headers       map[string]string
	Body          io.ReadCloser
	ContentLength int64
}

// StateManager manages installation state for recovery
type StateManager interface {
	// SaveState saves the current installation state
	SaveState(ctx context.Context, state State) error
	// LoadState loads the saved state
	LoadState(ctx context.Context) (State, error)
	// ClearState removes the saved state
	ClearState(ctx context.Context) error
}

// State represents the installation state
type State struct {
	ID            string
	Method        string
	Version       string
	Steps         []StepResult
	StartTime     time.Time
	LastUpdate    time.Time
	DownloadState DownloadState
	Metadata      map[string]interface{}
}

// DownloadState tracks download progress for resumption
type DownloadState struct {
	URL             string
	LocalPath       string
	BytesDownloaded int64
	TotalBytes      int64
	Checksum        string
	LastModified    time.Time
}

// Validator performs post-installation validation
type Validator interface {
	// Validate performs validation checks
	Validate(ctx context.Context) ValidationResult
}

// ValidationResult contains validation results
type ValidationResult struct {
	Success bool
	Checks  []ValidationCheck
	Summary string
}

// ValidationCheck represents a single validation check
type ValidationCheck struct {
	Name        string
	Description string
	Success     bool
	Error       error
	Duration    time.Duration
	Metadata    map[string]interface{}
}

// RollbackHandler handles installation rollback
type RollbackHandler interface {
	// CanRollback checks if rollback is possible
	CanRollback(ctx context.Context, state State) bool
	// Rollback performs the rollback
	Rollback(ctx context.Context, state State) error
}

// ProgressTracker tracks installation progress
type ProgressTracker interface {
	// Start begins tracking
	Start(totalSteps int)
	// Update updates the current progress
	Update(step string, metadata map[string]interface{})
	// Complete marks a step as complete
	Complete(step string)
	// Fail marks a step as failed
	Fail(step string, err error)
	// Finish completes tracking
	Finish()
}

// CircuitBreaker provides circuit breaker functionality for network operations
type CircuitBreaker interface {
	// Execute runs a function with circuit breaker protection
	Execute(ctx context.Context, fn func() error) error
	// State returns the current circuit state
	State() CircuitState
	// Reset resets the circuit breaker
	Reset()
}

// CircuitState represents the circuit breaker state
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// Middleware represents installation middleware
type Middleware func(Step[InstallationType]) Step[InstallationType]

// Pipeline represents the installation pipeline
type Pipeline interface {
	// AddStep adds a step to the pipeline
	AddStep(step Step[InstallationType])
	// Execute runs the pipeline
	Execute(ctx context.Context) error
	// AddMiddleware adds middleware to the pipeline
	AddMiddleware(middleware Middleware)
}
