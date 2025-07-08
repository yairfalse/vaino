package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete WGO configuration
type Config struct {
	Claude     ClaudeConfig     `mapstructure:"claude"`
	Cache      CacheConfig      `mapstructure:"cache"`
	Collectors CollectorsConfig `mapstructure:"collectors"`
	Providers  ProvidersConfig  `mapstructure:"providers"`
	Storage    StorageConfig    `mapstructure:"storage"`
	Output     OutputConfig     `mapstructure:"output"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// ClaudeConfig contains Claude AI configuration
type ClaudeConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

// CacheConfig contains caching configuration
type CacheConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	DefaultTTL time.Duration `mapstructure:"default_ttl"`
	MaxSize    int64         `mapstructure:"max_size"`
}

// CollectorsConfig contains all collector configurations
type CollectorsConfig struct {
	Terraform  TerraformConfig  `mapstructure:"terraform"`
	AWS        AWSConfig        `mapstructure:"aws"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
}

// TerraformConfig contains Terraform collector configuration
type TerraformConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	StatePaths []string `mapstructure:"state_paths"`
}

// AWSConfig contains AWS collector configuration
type AWSConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Regions  []string `mapstructure:"regions"`
	Profiles []string `mapstructure:"profiles"`
}

// KubernetesConfig contains Kubernetes collector configuration
type KubernetesConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	Contexts   []string `mapstructure:"contexts"`
	Namespaces []string `mapstructure:"namespaces"`
}

// StorageConfig contains storage configuration
type StorageConfig struct {
	BasePath string `mapstructure:"base_path"`
	BaseDir  string `mapstructure:"base_dir"`
	Backend  string `mapstructure:"backend"`
}

// OutputConfig contains output formatting configuration
type OutputConfig struct {
	Format    string `mapstructure:"format"`
	Pretty    bool   `mapstructure:"pretty"`
	NoColor   bool   `mapstructure:"no_color"`
	Timestamp bool   `mapstructure:"timestamp"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

// ProvidersConfig contains provider-specific configuration
type ProvidersConfig struct {
	Terraform  TerraformProviderConfig  `mapstructure:"terraform"`
	GCP        GCPProviderConfig        `mapstructure:"gcp"`
	AWS        AWSProviderConfig        `mapstructure:"aws"`
	Kubernetes KubernetesProviderConfig `mapstructure:"kubernetes"`
}

// TerraformProviderConfig contains Terraform-specific configuration
type TerraformProviderConfig struct {
	AutoDiscover bool     `mapstructure:"auto_discover"`
	StatePaths   []string `mapstructure:"state_paths"`
}

// GCPProviderConfig contains GCP-specific configuration
type GCPProviderConfig struct {
	Project string   `mapstructure:"project"`
	Regions []string `mapstructure:"regions"`
}

// AWSProviderConfig contains AWS-specific configuration
type AWSProviderConfig struct {
	Profile       string `mapstructure:"profile"`
	DefaultRegion string `mapstructure:"default_region"`
}

// KubernetesProviderConfig contains Kubernetes-specific configuration
type KubernetesProviderConfig struct {
	Namespaces []string `mapstructure:"namespaces"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Claude: ClaudeConfig{
			APIKey: "",
			Model:  "claude-sonnet-4-20250514",
		},
		Cache: CacheConfig{
			Enabled:    true,
			DefaultTTL: time.Hour,
			MaxSize:    100,
		},
		Collectors: CollectorsConfig{
			Terraform: TerraformConfig{
				Enabled:    true,
				StatePaths: []string{"./terraform/*.tfstate"},
			},
			AWS: AWSConfig{
				Enabled: true,
				Regions: []string{"us-east-1", "us-west-2"},
			},
			Kubernetes: KubernetesConfig{
				Enabled:    true,
				Contexts:   []string{},
				Namespaces: []string{"default"},
			},
		},
		Providers: ProvidersConfig{
			Terraform: TerraformProviderConfig{
				AutoDiscover: true,
				StatePaths:   []string{"."},
			},
			GCP: GCPProviderConfig{
				Project: "",
				Regions: []string{"us-central1"},
			},
			AWS: AWSProviderConfig{
				Profile:       "",
				DefaultRegion: "",
			},
			Kubernetes: KubernetesProviderConfig{
				Namespaces: []string{"default", "kube-system"},
			},
		},
		Storage: StorageConfig{
			BasePath: "~/.wgo",
			BaseDir:  filepath.Join(homeDir, ".wgo", "storage"),
			Backend:  "local",
		},
		Output: OutputConfig{
			Format:    "table",
			Pretty:    true,
			NoColor:   false,
			Timestamp: true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
			File:   "",
		},
	}
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	config := DefaultConfig()

	// Set configuration file paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add configuration paths
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".wgo"))
	}
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set environment variable support
	viper.SetEnvPrefix("WGO")
	viper.AutomaticEnv()

	// Map environment variables to config keys
	viper.BindEnv("claude.api_key", "CLAUDE_API_KEY", "ANTHROPIC_API_KEY")
	viper.BindEnv("logging.level", "LOG_LEVEL")
	viper.BindEnv("cache.enabled", "CACHE_ENABLED")

	// Read configuration file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is not an error - we'll use defaults
	}

	// Unmarshal into our config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Claude API key is optional - only required for AI features
	// This makes WGO work without needing AI setup
	
	if c.Storage.BasePath == "" {
		return fmt.Errorf("storage base path is required")
	}

	if c.Cache.DefaultTTL <= 0 {
		return fmt.Errorf("cache default TTL must be positive")
	}

	return nil
}

// HasAIFeatures checks if AI features are available
func (c *Config) HasAIFeatures() bool {
	return c.Claude.APIKey != ""
}

// ExpandPaths expands home directory paths
func (c *Config) ExpandPaths() error {
	var err error
	c.Storage.BasePath, err = expandPath(c.Storage.BasePath)
	if err != nil {
		return fmt.Errorf("failed to expand storage base path: %w", err)
	}

	// Expand Terraform state paths
	for i, path := range c.Collectors.Terraform.StatePaths {
		c.Collectors.Terraform.StatePaths[i], err = expandPath(path)
		if err != nil {
			return fmt.Errorf("failed to expand terraform state path %s: %w", path, err)
		}
	}

	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path, err
	}

	if len(path) == 1 {
		return home, nil
	}

	return filepath.Join(home, path[1:]), nil
}