package installer

import (
	"context"
	"fmt"
	"time"

	"github.com/yairfalse/wgo/cmd/install/platform"
)

// factory implements the Factory interface
type factory struct {
	platform *platform.Info
}

// NewFactory creates a new installer factory
func NewFactory(platformInfo *platform.Info) Factory {
	return &factory{
		platform: platformInfo,
	}
}

// Create returns an installer for the current platform
func (f *factory) Create(options ...Option) (Installer, error) {
	// Create base configuration
	config := &Config{
		Method:          "binary",
		InstallDir:      platform.InstallDir(),
		Version:         "latest",
		Timeout:         30 * time.Minute,
		RetryAttempts:   3,
		RetryBackoff:    5 * time.Second,
		ValidationLevel: "full",
		Features:        []string{},
		Environment:     make(map[string]string),
	}

	// Apply options
	for _, opt := range options {
		if err := opt(config); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// Create installer based on method
	switch config.Method {
	case "binary":
		return NewBinaryInstaller(config)
	case "container":
		return NewContainerInstaller(config)
	case "kubernetes":
		return NewKubernetesInstaller(config)
	default:
		return nil, fmt.Errorf("unsupported installation method: %s", config.Method)
	}
}

// Detect auto-detects the best installation method
func (f *factory) Detect(ctx context.Context) (Strategy, error) {
	// Priority order:
	// 1. Kubernetes (if running in k8s)
	// 2. Container (if docker/podman available)
	// 3. Binary (fallback)

	if f.platform.IsKubernetes {
		strategy := &KubernetesStrategy{platform: f.platform}
		if can, _ := strategy.CanInstall(ctx); can {
			return strategy, nil
		}
	}

	if f.platform.HasDocker && !f.platform.IsContainer {
		strategy := &ContainerStrategy{platform: f.platform}
		if can, _ := strategy.CanInstall(ctx); can {
			return strategy, nil
		}
	}

	// Default to binary installation
	return &BinaryStrategy{platform: f.platform}, nil
}

// Functional options

// WithConfig applies a custom configuration
func WithConfig(config *Config) Option {
	return func(c *Config) error {
		*c = *config
		return nil
	}
}

// WithMethod sets the installation method
func WithMethod(method string) Option {
	return func(c *Config) error {
		c.Method = method
		return nil
	}
}

// WithVersion sets the version to install
func WithVersion(version string) Option {
	return func(c *Config) error {
		c.Version = version
		return nil
	}
}

// WithInstallDir sets the installation directory
func WithInstallDir(dir string) Option {
	return func(c *Config) error {
		c.InstallDir = dir
		return nil
	}
}

// WithMirrors sets download mirrors
func WithMirrors(mirrors []string) Option {
	return func(c *Config) error {
		c.Mirrors = mirrors
		return nil
	}
}

// WithProgressTracker sets a custom progress tracker
func WithProgressTracker(tracker ProgressTracker) Option {
	return func(c *Config) error {
		// This would be used by the installer
		return nil
	}
}

// WithStateManager sets a custom state manager
func WithStateManager(manager StateManager) Option {
	return func(c *Config) error {
		// This would be used by the installer
		return nil
	}
}
