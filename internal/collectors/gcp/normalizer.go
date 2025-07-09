package gcp

import (
	"fmt"
	"time"

	"github.com/yairfalse/vaino/pkg/types"
)

type ResourceNormalizer struct{}

func NewResourceNormalizer() *ResourceNormalizer {
	return &ResourceNormalizer{}
}

func (n *ResourceNormalizer) NormalizeComputeInstance(instance interface{}) types.Resource {
	// This would normalize actual GCP compute instance data
	// For now, return a placeholder
	return types.Resource{
		ID:       "placeholder-instance",
		Type:     "compute_instance",
		Name:     "placeholder",
		Provider: "gcp",
		Configuration: map[string]interface{}{
			"machine_type": "e2-micro",
			"status":       "running",
		},
		Metadata: types.ResourceMetadata{
			CreatedAt: time.Now(),
			Version:   "1",
		},
	}
}

func (n *ResourceNormalizer) NormalizeResource(resourceType string, data interface{}) (types.Resource, error) {
	switch resourceType {
	case "compute_instance":
		return n.NormalizeComputeInstance(data), nil
	default:
		return types.Resource{}, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
