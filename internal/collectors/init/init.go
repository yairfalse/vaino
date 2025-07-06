package init

import (
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/collectors/terraform"
)

// RegisterAllCollectors registers all built-in collectors to the default registry
func RegisterAllCollectors() {
	// Register terraform collector
	terraformCollector := terraform.NewCollector()
	if err := collectors.DefaultRegistry.Register(terraformCollector); err != nil {
		panic("failed to register terraform collector: " + err.Error())
	}
}

// init automatically registers collectors when the package is imported
func init() {
	RegisterAllCollectors()
}
