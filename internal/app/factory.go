package app

import (
	"github.com/yairfalse/wgo/internal/cache"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/gcp"
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

	// Create registry and register collectors
	registry := collectors.NewRegistry()
	
	// Register GCP collector
	gcpCollector := gcp.NewGCPCollector()
	registry.Register(gcpCollector)

	return &App{
		config:   config,
		storage:  stor,
		cache:    cacheManager,
		logger:   log,
		registry: registry,
	}, nil
}