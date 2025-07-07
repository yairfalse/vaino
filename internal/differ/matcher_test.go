package differ

import (
	"fmt"
	"testing"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestDefaultResourceMatcher_Match(t *testing.T) {
	matcher := &DefaultResourceMatcher{}
	
	baseline := []types.Resource{
		{
			ID:       "resource-1",
			Type:     "instance",
			Name:     "web-server",
			Provider: "aws",
			Region:   "us-east-1",
		},
		{
			ID:       "resource-2", 
			Type:     "security_group",
			Name:     "web-sg",
			Provider: "aws",
			Region:   "us-east-1",
		},
	}
	
	current := []types.Resource{
		{
			ID:       "resource-1",
			Type:     "instance", 
			Name:     "web-server",
			Provider: "aws",
			Region:   "us-east-1",
		},
		{
			ID:       "resource-3",
			Type:     "database",
			Name:     "db-server",
			Provider: "aws", 
			Region:   "us-east-1",
		},
	}
	
	matches, added, removed := matcher.Match(baseline, current)
	
	// Should match resource-1 by ID
	if matches["resource-1"] != "resource-1" {
		t.Errorf("expected resource-1 to match itself")
	}
	
	// resource-2 should be in removed list
	if len(removed) != 1 || removed[0].ID != "resource-2" {
		t.Errorf("expected resource-2 to be removed, got %v", removed)
	}
	
	// resource-3 should be in added list  
	if len(added) != 1 || added[0].ID != "resource-3" {
		t.Errorf("expected resource-3 to be added, got %v", added)
	}
}

func TestSmartResourceMatcher_Match(t *testing.T) {
	matcher := &SmartResourceMatcher{}
	
	baseline := []types.Resource{
		{
			ID:       "old-id-1",
			Type:     "instance",
			Name:     "web-server",
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "production",
				"Application": "web",
			},
		},
		{
			ID:       "old-id-2",
			Type:     "database", 
			Name:     "db-server",
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "production",
				"Application": "database",
			},
		},
	}
	
	current := []types.Resource{
		{
			ID:       "new-id-1",
			Type:     "instance",
			Name:     "web-server", // Same name
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "production",
				"Application": "web",
			},
		},
		{
			ID:       "new-id-3",
			Type:     "storage",
			Name:     "file-storage",
			Provider: "aws",
			Region:   "us-east-1",
		},
	}
	
	matches, added, removed := matcher.Match(baseline, current)
	
	// Should match old-id-1 to new-id-1 by name and tags
	if matches["old-id-1"] != "new-id-1" {
		t.Errorf("expected old-id-1 to match new-id-1 by name/tags, got %s", matches["old-id-1"])
	}
	
	// old-id-2 should be in removed (no match found)
	if len(removed) != 1 || removed[0].ID != "old-id-2" {
		t.Errorf("expected old-id-2 to be removed, got %v", removed)
	}
	
	// new-id-3 should be in added (no match found)
	if len(added) != 1 || added[0].ID != "new-id-3" {
		t.Errorf("expected new-id-3 to be added, got %v", added)
	}
}

func TestSmartResourceMatcher_TagMatching(t *testing.T) {
	matcher := &SmartResourceMatcher{}
	
	// Test with complex tag matching scenario
	resource1 := types.Resource{
		ID:       "id-1",
		Type:     "instance",
		Name:     "web-server",
		Provider: "aws",
		Region:   "us-east-1",
		Tags: map[string]string{
			"Environment": "production",
			"Application": "web",
			"Team":        "backend",
		},
	}
	
	// Similar resource with overlapping tags
	resource2 := types.Resource{
		ID:       "id-2",
		Type:     "instance",
		Name:     "web-server", // Same name
		Provider: "aws",
		Region:   "us-east-1",
		Tags: map[string]string{
			"Environment": "production", // Same
			"Application": "web",        // Same
			"Team":        "frontend",   // Different
		},
	}
	
	baseline := []types.Resource{resource1}
	current := []types.Resource{resource2}
	
	matches, _, _ := matcher.Match(baseline, current)
	
	// Should match by name and most tags
	if matches["id-1"] != "id-2" {
		t.Errorf("expected resources to match by name and tags")
	}
}

func TestSmartResourceMatcher_EdgeCases(t *testing.T) {
	matcher := &SmartResourceMatcher{}
	
	// Test empty slices
	matches, added, removed := matcher.Match([]types.Resource{}, []types.Resource{})
	if len(matches) != 0 || len(added) != 0 || len(removed) != 0 {
		t.Errorf("expected empty results for empty inputs")
	}
	
	// Test nil tags
	baseline := []types.Resource{
		{
			ID:       "resource-1",
			Type:     "instance",
			Name:     "server",
			Provider: "aws",
			Tags:     nil,
		},
	}
	
	current := []types.Resource{
		{
			ID:       "resource-2",
			Type:     "instance", 
			Name:     "server",
			Provider: "aws",
			Tags:     map[string]string{},
		},
	}
	
	matches2, _, _ := matcher.Match(baseline, current)
	
	// Should still match by name even with nil/empty tags
	if matches2["resource-1"] != "resource-2" {
		t.Errorf("expected resources to match by name despite nil tags")
	}
}

func TestResourceMatcher_Performance(t *testing.T) {
	// Test with larger datasets to ensure reasonable performance
	baseline := make([]types.Resource, 100)
	current := make([]types.Resource, 100)
	
	for i := 0; i < 100; i++ {
		baseline[i] = types.Resource{
			ID:       fmt.Sprintf("baseline-%d", i),
			Type:     "instance",
			Name:     fmt.Sprintf("server-%d", i),
			Provider: "aws",
			Region:   "us-east-1",
		}
		
		current[i] = types.Resource{
			ID:       fmt.Sprintf("current-%d", i),
			Type:     "instance",
			Name:     fmt.Sprintf("server-%d", i), // Same names for matching
			Provider: "aws",
			Region:   "us-east-1",
		}
	}
	
	matcher := &SmartResourceMatcher{}
	
	// This should complete reasonably quickly
	matches, _, _ := matcher.Match(baseline, current)
	
	// Should match all 100 resources
	if len(matches) != 100 {
		t.Errorf("expected 100 matches, got %d", len(matches))
	}
}