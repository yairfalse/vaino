package platform

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Info contains platform information
type Info struct {
	OS           string
	Arch         string
	Version      string
	Distribution string // For Linux
	HasDocker    bool
	HasKubectl   bool
	IsContainer  bool
	IsKubernetes bool
}

// Detector detects platform capabilities
type Detector interface {
	Detect() (*Info, error)
}

// detector implements the Detector interface
type detector struct{}

// NewDetector creates a new platform detector
func NewDetector() Detector {
	return &detector{}
}

// Detect gathers platform information
func (d *detector) Detect() (*Info, error) {
	info := &Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Detect if running in container
	info.IsContainer = d.detectContainer()

	// Detect if running in Kubernetes
	info.IsKubernetes = d.detectKubernetes()

	// Detect available tools
	info.HasDocker = d.checkCommand("docker")
	info.HasKubectl = d.checkCommand("kubectl")

	// Get platform-specific information
	if err := d.detectPlatformSpecific(info); err != nil {
		return nil, fmt.Errorf("failed to detect platform specifics: %w", err)
	}

	return info, nil
}

func (d *detector) detectContainer() bool {
	// Check for Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for containerd/k8s
	if _, err := os.Stat("/run/secrets/kubernetes.io"); err == nil {
		return true
	}

	// Check cgroup for container signatures
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "containerd") ||
			strings.Contains(content, "kubepods") {
			return true
		}
	}

	return false
}

func (d *detector) detectKubernetes() bool {
	// Check for Kubernetes service account
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		return true
	}

	// Check for Kubernetes environment variables
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}

func (d *detector) checkCommand(cmd string) bool {
	// This is a simplified check - in real implementation would use exec.LookPath
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, path := range paths {
		fullPath := path + string(os.PathSeparator) + cmd
		if runtime.GOOS == "windows" {
			fullPath += ".exe"
		}
		if _, err := os.Stat(fullPath); err == nil {
			return true
		}
	}
	return false
}

// SupportedPlatform checks if the current platform is supported
func SupportedPlatform(os, arch string) bool {
	supported := map[string][]string{
		"linux":   {"amd64", "arm64", "arm"},
		"darwin":  {"amd64", "arm64"},
		"windows": {"amd64", "arm64"},
		"freebsd": {"amd64", "arm64"},
	}

	archs, ok := supported[os]
	if !ok {
		return false
	}

	for _, a := range archs {
		if a == arch {
			return true
		}
	}

	return false
}

// BinaryName returns the platform-specific binary name
func BinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// InstallDir returns the default installation directory for the platform
func InstallDir() string {
	switch runtime.GOOS {
	case "windows":
		return os.Getenv("PROGRAMFILES") + "\\Tapio"
	case "darwin":
		return "/usr/local/bin"
	default:
		// Check if user has write access to /usr/local/bin
		if _, err := os.Stat("/usr/local/bin"); err == nil {
			testFile := "/usr/local/bin/.tapio-test"
			if f, err := os.Create(testFile); err == nil {
				f.Close()
				os.Remove(testFile)
				return "/usr/local/bin"
			}
		}
		// Fall back to user's home directory
		if home := os.Getenv("HOME"); home != "" {
			return home + "/.local/bin"
		}
		return "./bin"
	}
}
