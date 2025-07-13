package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/yairfalse/vaino/cmd/install/platform"
)

// BinaryStrategy implements the binary installation strategy
type BinaryStrategy struct {
	platform *platform.Info
}

func (s *BinaryStrategy) Name() string {
	return "binary"
}

func (s *BinaryStrategy) CanInstall(ctx context.Context) (bool, error) {
	// Check if platform is supported
	if !platform.SupportedPlatform(s.platform.OS, s.platform.Arch) {
		return false, fmt.Errorf("unsupported platform: %s/%s", s.platform.OS, s.platform.Arch)
	}

	// Check if we have write access to install directory
	installDir := platform.InstallDir()
	if err := checkWriteAccess(installDir); err != nil {
		// Try user directory
		homeDir, _ := os.UserHomeDir()
		userBinDir := homeDir + "/.local/bin"
		if err := checkWriteAccess(userBinDir); err != nil {
			return false, fmt.Errorf("no write access to install directories")
		}
	}

	return true, nil
}

func (s *BinaryStrategy) PreInstall(ctx context.Context) error {
	// Ensure install directory exists
	installDir := platform.InstallDir()
	return os.MkdirAll(installDir, 0755)
}

func (s *BinaryStrategy) Install(ctx context.Context) error {
	// Installation is handled by BinaryInstaller
	return nil
}

func (s *BinaryStrategy) PostInstall(ctx context.Context) error {
	// Post-installation tasks are handled by steps
	return nil
}

func (s *BinaryStrategy) Rollback(ctx context.Context) error {
	// Rollback is handled by BinaryInstaller
	return nil
}

// ContainerStrategy implements the container installation strategy
type ContainerStrategy struct {
	platform *platform.Info
}

func (s *ContainerStrategy) Name() string {
	return "container"
}

func (s *ContainerStrategy) CanInstall(ctx context.Context) (bool, error) {
	if !s.platform.HasDocker {
		return false, fmt.Errorf("Docker not found")
	}

	if s.platform.IsContainer {
		return false, fmt.Errorf("already running in a container")
	}

	return true, nil
}

func (s *ContainerStrategy) PreInstall(ctx context.Context) error {
	// Verify Docker daemon is running
	// This would check docker info in real implementation
	return nil
}

func (s *ContainerStrategy) Install(ctx context.Context) error {
	// Container installation logic
	return fmt.Errorf("container installation not yet implemented")
}

func (s *ContainerStrategy) PostInstall(ctx context.Context) error {
	return nil
}

func (s *ContainerStrategy) Rollback(ctx context.Context) error {
	return nil
}

// KubernetesStrategy implements the Kubernetes installation strategy
type KubernetesStrategy struct {
	platform *platform.Info
}

func (s *KubernetesStrategy) Name() string {
	return "kubernetes"
}

func (s *KubernetesStrategy) CanInstall(ctx context.Context) (bool, error) {
	if !s.platform.HasKubectl {
		return false, fmt.Errorf("kubectl not found")
	}

	// Would check kubectl access in real implementation
	return true, nil
}

func (s *KubernetesStrategy) PreInstall(ctx context.Context) error {
	// Verify Kubernetes access
	return nil
}

func (s *KubernetesStrategy) Install(ctx context.Context) error {
	// Kubernetes installation logic
	return fmt.Errorf("Kubernetes installation not yet implemented")
}

func (s *KubernetesStrategy) PostInstall(ctx context.Context) error {
	return nil
}

func (s *KubernetesStrategy) Rollback(ctx context.Context) error {
	return nil
}

// Helper functions

func checkWriteAccess(dir string) error {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		// Try to create parent directory
		parent := filepath.Dir(dir)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("cannot create directory: %w", err)
		}
		return nil
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	// Check write permission by creating a temp file
	testFile := filepath.Join(dir, ".tapio-write-test")
	file, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("no write access to %s: %w", dir, err)
	}
	file.Close()
	os.Remove(testFile)

	return nil
}

// Placeholder installer implementations

func NewContainerInstaller(config *Config) (Installer, error) {
	return nil, fmt.Errorf("container installer not yet implemented")
}

func NewKubernetesInstaller(config *Config) (Installer, error) {
	return nil, fmt.Errorf("Kubernetes installer not yet implemented")
}
