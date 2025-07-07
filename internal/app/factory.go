package app

import (
	"github.com/yairfalse/wgo/internal/cache"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/kubernetes"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
	"github.com/yairfalse/wgo/internal/logger"
	"github.com/yairfalse/wgo/internal/storage"
)

type AppFactory struct{}

func NewAppFactory() *AppFactory {
	return &AppFactory{}
}

func (f *AppFactory) Create(config Config) (*App, error) {
	// Create logger
	var log logger.Logger
	if config.Debug {
		log = logger.NewLogrus()
	} else {
		log = logger.NewSimple()
	}

	// Create storage
	stor := storage.NewLocal("./snapshots")

	// Create cache
	cacheManager := cache.NewManager()

	// Create enhanced registry and initialize collectors
	enhancedRegistry := collectors.NewEnhancedRegistry()
	
	// Register Terraform collector
	terraformCollector := terraform.NewTerraformCollector()
	enhancedRegistry.RegisterEnhanced(terraformCollector)
	
	// Register Kubernetes collector
	kubernetesCollector := kubernetes.NewKubernetesCollector()
	enhancedRegistry.RegisterEnhanced(kubernetesCollector)
	
	// Create legacy registry for compatibility
	registry := collectors.NewRegistry()

	return &App{
		config:           config,
		storage:          stor,
		cache:            cacheManager,
		logger:           log,
		registry:         registry,
		enhancedRegistry: enhancedRegistry,
	}, nil
}