package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// InfraConfig represents infrastructure-specific configuration
type InfraConfig struct {
	Terraform  TerraformConfig  `mapstructure:"terraform"`
	AWS        AWSConfig        `mapstructure:"aws"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Git        GitConfig        `mapstructure:"git"`
}

// TerraformConfig holds Terraform-specific settings
type TerraformConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	StatePaths   []string `mapstructure:"state_paths"`
	Workspaces   []string `mapstructure:"workspaces"`
	AutoDiscover bool     `mapstructure:"auto_discover"`
}

// AWSConfig holds AWS-specific settings
type AWSConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Regions  []string `mapstructure:"regions"`
	Profiles []string `mapstructure:"profiles"`
}

// KubernetesConfig holds Kubernetes-specific settings
type KubernetesConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	Contexts   []string `mapstructure:"contexts"`
	Namespaces []string `mapstructure:"namespaces"`
}

// GitConfig holds Git integration settings
type GitConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	TrackCommits    bool     `mapstructure:"track_commits"`
	BaselineOnTag   bool     `mapstructure:"baseline_on_tag"`
	IgnoreBranches  []string `mapstructure:"ignore_branches"`
	AutoBaseline    bool     `mapstructure:"auto_baseline"`
	BaselineBranch  string   `mapstructure:"baseline_branch"`
}

// LoadInfraConfig loads infrastructure configuration
func LoadInfraConfig() (*InfraConfig, error) {
	// Set defaults
	setDefaults()
	
	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
	}
	
	var config InfraConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Auto-discover Terraform state files if enabled
	if config.Terraform.AutoDiscover {
		discovered, err := discoverTerraformStates()
		if err == nil && len(discovered) > 0 {
			config.Terraform.StatePaths = append(config.Terraform.StatePaths, discovered...)
		}
	}
	
	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Terraform defaults
	viper.SetDefault("terraform.enabled", true)
	viper.SetDefault("terraform.state_paths", []string{"./terraform.tfstate", "./terraform"})
	viper.SetDefault("terraform.workspaces", []string{"default"})
	viper.SetDefault("terraform.auto_discover", true)
	
	// AWS defaults
	viper.SetDefault("aws.enabled", false)
	viper.SetDefault("aws.regions", []string{"us-east-1"})
	viper.SetDefault("aws.profiles", []string{"default"})
	
	// Kubernetes defaults
	viper.SetDefault("kubernetes.enabled", false)
	viper.SetDefault("kubernetes.contexts", []string{"default"})
	viper.SetDefault("kubernetes.namespaces", []string{"default"})
	
	// Git defaults
	viper.SetDefault("git.enabled", true)
	viper.SetDefault("git.track_commits", true)
	viper.SetDefault("git.baseline_on_tag", true)
	viper.SetDefault("git.auto_baseline", false)
	viper.SetDefault("git.baseline_branch", "main")
	viper.SetDefault("git.ignore_branches", []string{"feature/*", "hotfix/*"})
}

// discoverTerraformStates automatically discovers Terraform state files
func discoverTerraformStates() ([]string, error) {
	var statePaths []string
	
	// Common patterns to search for
	patterns := []string{
		"terraform.tfstate",
		"**/terraform.tfstate",
		"**/*.tfstate",
		".terraform/terraform.tfstate",
	}
	
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		statePaths = append(statePaths, matches...)
	}
	
	return statePaths, nil
}

// ValidateConfig validates the configuration
func (c *InfraConfig) Validate() error {
	if c.Terraform.Enabled && len(c.Terraform.StatePaths) == 0 {
		return fmt.Errorf("terraform is enabled but no state paths configured")
	}
	
	if c.AWS.Enabled && len(c.AWS.Regions) == 0 {
		return fmt.Errorf("aws is enabled but no regions configured")
	}
	
	if c.Kubernetes.Enabled && len(c.Kubernetes.Contexts) == 0 {
		return fmt.Errorf("kubernetes is enabled but no contexts configured")
	}
	
	return nil
}

// GetEnabledProviders returns a list of enabled providers
func (c *InfraConfig) GetEnabledProviders() []string {
	var providers []string
	
	if c.Terraform.Enabled {
		providers = append(providers, "terraform")
	}
	if c.AWS.Enabled {
		providers = append(providers, "aws")
	}
	if c.Kubernetes.Enabled {
		providers = append(providers, "kubernetes")
	}
	
	return providers
}

// InitConfigFile creates a default config file if it doesn't exist
func InitConfigFile() error {
	configDir := filepath.Join(os.Getenv("HOME"), ".wgo")
	configFile := filepath.Join(configDir, "config.yaml")
	
	// Check if config file already exists
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists at %s", configFile)
	}
	
	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create default config content
	defaultConfig := `# WGO Configuration
# Generated configuration file

# Basic settings
verbose: false
debug: false

# Provider configurations
providers:
  terraform:
    enabled: true
    state_paths:
      - "./terraform.tfstate"
      - "./terraform"
    auto_discover: true
    
  aws:
    enabled: false
    regions:
      - "us-east-1"
    
  kubernetes:
    enabled: false
    contexts:
      - "default"

# Git integration  
git:
  enabled: true
  track_commits: true
  baseline_on_tag: true

# Output preferences
output:
  default_format: "table"
  color: true
`
	
	// Write config file
	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	fmt.Printf("✅ Created default config file at %s\n", configFile)
	fmt.Println("Edit this file to customize your WGO configuration.")
	
	return nil
}

// SetupGitIntegration configures Git integration
func SetupGitIntegration(gitConfig GitConfig) error {
	if !gitConfig.Enabled {
		return nil
	}
	
	// Check if we're in a Git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Git repository")
	}
	
	// TODO: Set up Git hooks for automatic baseline creation
	// This would install pre-commit or post-commit hooks
	
	return nil
}

// GetCurrentBranch returns the current Git branch
func GetCurrentBranch() (string, error) {
	// This is a simplified version - in practice you'd use go-git library
	// or exec git commands
	return "main", nil
}

// ShouldIgnoreBranch checks if the current branch should be ignored
func (c *GitConfig) ShouldIgnoreBranch(branch string) bool {
	for _, pattern := range c.IgnoreBranches {
		if matched, _ := filepath.Match(pattern, branch); matched {
			return true
		}
	}
	return false
}

// GetProjectRoot finds the project root directory
func GetProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	// Walk up directory tree looking for .git directory
	for {
		if _, err := os.Stat(filepath.Join(wd, ".git")); err == nil {
			return wd, nil
		}
		
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	
	return "", fmt.Errorf("not in a Git repository")
}