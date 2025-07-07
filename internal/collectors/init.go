package collectors

import (
	"github.com/yairfalse/wgo/internal/collectors/terraform"
)

// InitializeCollectors registers all available collectors
func InitializeCollectors() *EnhancedRegistry {
	registry := NewEnhancedRegistry()
	
	// Register Terraform collector
	terraformCollector := terraform.NewTerraformCollector()
	registry.RegisterEnhanced(terraformCollector)
	
	// TODO: Register other collectors here
	// - AWS collector
	// - Kubernetes collector
	// - Azure collector
	// - GCP collector
	
	return registry
}

// InitializeDefaultRegistry initializes the default enhanced registry
func InitializeDefaultRegistry() {
	defaultEnhancedRegistry = InitializeCollectors()
}