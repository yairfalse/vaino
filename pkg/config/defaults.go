package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultsManager handles smart defaults for WGO configuration
type DefaultsManager struct {
	workingDir string
}

// NewDefaultsManager creates a new defaults manager
func NewDefaultsManager() *DefaultsManager {
	wd, _ := os.Getwd()
	return &DefaultsManager{
		workingDir: wd,
	}
}

// GenerateSmartDefaults creates a configuration with smart defaults
func (dm *DefaultsManager) GenerateSmartDefaults() (*Config, error) {
	config := &Config{
		Storage: StorageConfig{
			BasePath: dm.getDefaultStoragePath(),
			Backend:  "local",
		},
		Cache: CacheConfig{
			DefaultTTL: time.Hour,
			MaxSize:    100,
			Enabled:    true,
		},
		Output: OutputConfig{
			Format:    "table",
			Pretty:    true,
			NoColor:   false,
			Timestamp: true,
		},
		Collectors: CollectorsConfig{},
	}

	// Auto-detect and configure available providers
	if err := dm.configureProviders(config); err != nil {
		return nil, fmt.Errorf("failed to configure providers: %w", err)
	}

	return config, nil
}

// getDefaultStoragePath returns a smart default for storage
func (dm *DefaultsManager) getDefaultStoragePath() string {
	// Check if we're in a project directory (has .git, go.mod, etc.)
	projectMarkers := []string{".git", "go.mod", "package.json", "terraform.tf", "main.tf"}
	
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(dm.workingDir, marker)); err == nil {
			// Use project-local storage
			return filepath.Join(dm.workingDir, ".wgo")
		}
	}
	
	// Use user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./.wgo" // Fallback to current directory
	}
	
	return filepath.Join(homeDir, ".wgo")
}

// getDefaultCachePath returns a smart default for cache
func (dm *DefaultsManager) getDefaultCachePath() string {
	storagePath := dm.getDefaultStoragePath()
	return filepath.Join(storagePath, "cache")
}

// configureProviders auto-detects and configures available providers
func (dm *DefaultsManager) configureProviders(config *Config) error {
	// Configure Terraform if detected
	if dm.isTerraformProject() {
		config.Collectors.Terraform = TerraformConfig{
			Enabled:    true,
			StatePaths: []string{},
		}
	}

	// Configure AWS if credentials are available
	if dm.isAWSAvailable() {
		config.Collectors.AWS = AWSConfig{
			Enabled: true,
			Regions: dm.getDefaultAWSRegions(),
		}
	}

	// Configure Kubernetes if config is available
	if dm.isKubernetesAvailable() {
		config.Collectors.Kubernetes = KubernetesConfig{
			Enabled:    true,
			Contexts:   []string{},
			Namespaces: []string{"default"},
		}
	}

	return nil
}

// isTerraformProject checks if current directory appears to be a Terraform project
func (dm *DefaultsManager) isTerraformProject() bool {
	terraformFiles := []string{
		"main.tf", "terraform.tf", "*.tf",
		"terraform.tfstate", ".terraform",
		"terraform.tfstate.d",
	}

	for _, pattern := range terraformFiles {
		if matches, _ := filepath.Glob(filepath.Join(dm.workingDir, pattern)); len(matches) > 0 {
			return true
		}
	}

	return false
}

// isAWSAvailable checks if AWS credentials are configured
func (dm *DefaultsManager) isAWSAvailable() bool {
	// Check for AWS credentials in environment
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		return true
	}

	// Check for AWS config files
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	awsConfigPaths := []string{
		filepath.Join(homeDir, ".aws", "credentials"),
		filepath.Join(homeDir, ".aws", "config"),
	}

	for _, path := range awsConfigPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// isKubernetesAvailable checks if kubectl is configured
func (dm *DefaultsManager) isKubernetesAvailable() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); err == nil {
		return true
	}

	// Check KUBECONFIG environment variable
	if os.Getenv("KUBECONFIG") != "" {
		return true
	}

	return false
}

// getDefaultAWSRegions returns commonly used AWS regions
func (dm *DefaultsManager) getDefaultAWSRegions() []string {
	return []string{"us-east-1", "us-west-2"}
}

// GenerateAutoName creates a meaningful name based on context
func (dm *DefaultsManager) GenerateAutoName(prefix string) string {
	// Get directory name
	dirName := filepath.Base(dm.workingDir)
	
	// Clean up the name
	if dirName == "." || dirName == "/" {
		dirName = "wgo"
	}

	// Add timestamp
	timestamp := time.Now().Format("2006-01-02-15-04")
	
	return fmt.Sprintf("%s-%s-%s", prefix, dirName, timestamp)
}

// GetRecommendedStoragePath suggests storage path based on project context
func (dm *DefaultsManager) GetRecommendedStoragePath() string {
	return dm.getDefaultStoragePath()
}

// ValidateDefaults checks if the generated defaults are valid
func (dm *DefaultsManager) ValidateDefaults(config *Config) error {
	// Check if storage path is writable
	if err := os.MkdirAll(config.Storage.BasePath, 0755); err != nil {
		return fmt.Errorf("cannot create storage directory %s: %w", config.Storage.BasePath, err)
	}

	// Check if cache path is writable - use storage path since cache doesn't have PersistDir
	cachePath := filepath.Join(config.Storage.BasePath, "cache")
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return fmt.Errorf("cannot create cache directory %s: %w", cachePath, err)
	}

	return nil
}

// GetUserFriendlyFeedback returns helpful information about the generated defaults
func (dm *DefaultsManager) GetUserFriendlyFeedback(config *Config) []string {
	var feedback []string

	feedback = append(feedback, fmt.Sprintf("📁 Storage location: %s", config.Storage.BasePath))
	
	var enabledProviders []string
	if config.Collectors.Terraform.Enabled {
		enabledProviders = append(enabledProviders, "terraform")
	}
	if config.Collectors.AWS.Enabled {
		enabledProviders = append(enabledProviders, "aws")
	}
	if config.Collectors.Kubernetes.Enabled {
		enabledProviders = append(enabledProviders, "kubernetes")
	}
	
	if len(enabledProviders) > 0 {
		feedback = append(feedback, "🔍 Auto-detected providers:")
		for _, provider := range enabledProviders {
			feedback = append(feedback, fmt.Sprintf("  • %s", provider))
		}
	} else {
		feedback = append(feedback, "⚠️  No providers auto-detected")
	}

	if dm.isTerraformProject() {
		feedback = append(feedback, "🏗️  Terraform project detected")
	}

	return feedback
}