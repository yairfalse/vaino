package app

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/yairfalse/wgo/internal/cache"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/logger"
	"github.com/yairfalse/wgo/internal/storage"
)

// AppFactory creates and configures the application with all dependencies
type AppFactory struct{}

// Create builds a fully configured App instance
func (f *AppFactory) Create(config Config) (*App, error) {
	// Create logger
	loggerImpl := logger.NewLogrus()
	
	// Create storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	storagePath := filepath.Join(homeDir, ".wgo")
	storageImpl, err := storage.NewLocal(storagePath)
	if err != nil {
		return nil, err
	}
	
	// Create cache
	cacheImpl, err := cache.NewManager(cache.Config{
		MemorySizeMB:    256,
		DiskSizeGB:      1,
		DefaultTTL:      "1h",
		CleanupInterval: "5m",
	})
	if err != nil {
		return nil, err
	}
	
	// Create collector registry
	registry := collectors.NewRegistry()
	
	// Create app with interfaces
	app := &App{
		config:   &config,
		storage:  storageImpl,
		cache:    cacheImpl,
		logger:   loggerImpl,
		registry: registry,
	}
	
	return app, nil
}

// setupLogging configures the logger based on flags
func (a *App) setupLogging() {
	logrusLogger, ok := a.logger.(*logger.LogrusLogger)
	if !ok {
		return
	}
	
	if viper.GetBool("debug") {
		logrusLogger.SetLevel(logrus.DebugLevel)
		a.logger.Info("Debug logging enabled")
	} else if viper.GetBool("verbose") {
		logrusLogger.SetLevel(logrus.InfoLevel)
	} else {
		logrusLogger.SetLevel(logrus.WarnLevel)
	}

	// Use JSON formatter for debug mode
	if viper.GetBool("debug") {
		logrusLogger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrusLogger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
			DisableColors:    false,
		})
	}
}

// setupConfig initializes configuration
func (a *App) setupConfig() {
	// Set default config file location
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(filepath.Join(home, ".wgo"))
			viper.SetConfigType("yaml")
			viper.SetConfigName("config")
		}
	}

	// Set environment variable prefix
	viper.SetEnvPrefix("WGO")
	viper.AutomaticEnv()

	// Set defaults
	a.setDefaultConfig()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			a.logger.Error("Failed to read config file", err)
		}
	} else {
		a.logger.WithField("config", viper.ConfigFileUsed()).Info("Using config file")
	}
}

// setDefaultConfig sets default configuration values
func (a *App) setDefaultConfig() {
	// Claude settings
	viper.SetDefault("claude.model", "claude-sonnet-4-20250514")
	viper.SetDefault("claude.max_tokens", 1000)

	// Cache settings
	viper.SetDefault("cache.enabled", true)
	viper.SetDefault("cache.memory_size", "256MB")
	viper.SetDefault("cache.disk_size", "1GB")
	viper.SetDefault("cache.default_ttl", "1h")
	viper.SetDefault("cache.cleanup_interval", "5m")

	// Collector settings
	viper.SetDefault("collectors.aws.regions", []string{"us-east-1"})
	viper.SetDefault("collectors.kubernetes.contexts", []string{"default"})
	viper.SetDefault("collectors.kubernetes.namespaces", []string{"default"})

	// Storage settings
	viper.SetDefault("storage.max_history", 30)
	viper.SetDefault("storage.compress_snapshots", true)

	// Output settings
	viper.SetDefault("output.default_format", "table")
	viper.SetDefault("output.color", true)
}