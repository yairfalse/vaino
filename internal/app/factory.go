package app

import (
	"github.com/yairfalse/vaino/internal/cache"
	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/collectors/aws"
	"github.com/yairfalse/vaino/internal/collectors/kubernetes"
	"github.com/yairfalse/vaino/internal/collectors/terraform"
	"github.com/yairfalse/vaino/internal/logger"
	"github.com/yairfalse/vaino/internal/storage"
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
	enhancedRegistry := collectors.NewCollectorRegistry()

	// Register Terraform collector
	terraformCollector := terraform.NewTerraformCollector()
	enhancedRegistry.Register(terraformCollector)

	// Register Kubernetes collector
	kubernetesCollector := kubernetes.NewKubernetesCollector()
	enhancedRegistry.Register(kubernetesCollector)

	// Register AWS collector
	awsCollector := aws.NewAWSCollector()
	enhancedRegistry.Register(awsCollector)

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
