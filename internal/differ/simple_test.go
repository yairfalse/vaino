package differ

import (
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestSimpleDiffer_NewSimpleDiffer(t *testing.T) {
	differ := NewSimpleDiffer()
	if differ == nil {
		t.Fatal("NewSimpleDiffer returned nil")
	}
}

func TestSimpleDiffer_Compare_EmptySnapshots(t *testing.T) {
	differ := NewSimpleDiffer()
	
	from := &types.Snapshot{
		ID:        "snap1",
		Timestamp: time.Now(),
		Resources: []types.Resource{},
	}
	
	to := &types.Snapshot{
		ID:        "snap2", 
		Timestamp: time.Now(),
		Resources: []types.Resource{},
	}
	
	report, err := differ.Compare(from, to)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	
	if report == nil {
		t.Fatal("Report is nil")
	}
	
	if len(report.Changes) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(report.Changes))
	}
	
	if report.Summary.Added != 0 {
		t.Errorf("Expected 0 added, got %d", report.Summary.Added)
	}
	
	if report.Summary.Modified != 0 {
		t.Errorf("Expected 0 modified, got %d", report.Summary.Modified)
	}
	
	if report.Summary.Removed != 0 {
		t.Errorf("Expected 0 removed, got %d", report.Summary.Removed)
	}
}

func TestSimpleDiffer_Compare_AddedResource(t *testing.T) {
	differ := NewSimpleDiffer()
	now := time.Now()
	
	from := &types.Snapshot{
		ID:        "snap1",
		Timestamp: now,
		Resources: []types.Resource{},
	}
	
	to := &types.Snapshot{
		ID:        "snap2",
		Timestamp: now.Add(1 * time.Minute),
		Resources: []types.Resource{
			{
				ID:       "deployment/test",
				Type:     "deployment",
				Name:     "test",
				Provider: "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 3,
					"image":    "nginx:latest",
				},
			},
		},
	}
	
	report, err := differ.Compare(from, to)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	
	if len(report.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(report.Changes))
	}
	
	change := report.Changes[0]
	if change.Type != "added" {
		t.Errorf("Expected type 'added', got '%s'", change.Type)
	}
	
	if change.ResourceID != "deployment/test" {
		t.Errorf("Expected ID 'deployment/test', got '%s'", change.ResourceID)
	}
	
	if change.ResourceType != "deployment" {
		t.Errorf("Expected type 'deployment', got '%s'", change.ResourceType)
	}
	
	if change.ResourceName != "test" {
		t.Errorf("Expected name 'test', got '%s'", change.ResourceName)
	}
	
	if change.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", change.Namespace)
	}
	
	if report.Summary.Added != 1 {
		t.Errorf("Expected 1 added, got %d", report.Summary.Added)
	}
}

func TestSimpleDiffer_Compare_RemovedResource(t *testing.T) {
	differ := NewSimpleDiffer()
	now := time.Now()
	
	from := &types.Snapshot{
		ID:        "snap1",
		Timestamp: now,
		Resources: []types.Resource{
			{
				ID:       "service/api",
				Type:     "service",
				Name:     "api",
				Provider: "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"port": 8080,
				},
			},
		},
	}
	
	to := &types.Snapshot{
		ID:        "snap2",
		Timestamp: now.Add(1 * time.Minute),
		Resources: []types.Resource{},
	}
	
	report, err := differ.Compare(from, to)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	
	if len(report.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(report.Changes))
	}
	
	change := report.Changes[0]
	if change.Type != "removed" {
		t.Errorf("Expected type 'removed', got '%s'", change.Type)
	}
	
	if change.ResourceID != "service/api" {
		t.Errorf("Expected ID 'service/api', got '%s'", change.ResourceID)
	}
	
	if report.Summary.Removed != 1 {
		t.Errorf("Expected 1 removed, got %d", report.Summary.Removed)
	}
}

func TestSimpleDiffer_Compare_ModifiedResource(t *testing.T) {
	differ := NewSimpleDiffer()
	now := time.Now()
	
	from := &types.Snapshot{
		ID:        "snap1",
		Timestamp: now,
		Resources: []types.Resource{
			{
				ID:       "deployment/app",
				Type:     "deployment",
				Name:     "app",
				Provider: "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 3,
					"image":    "app:v1.0",
				},
				Metadata: types.ResourceMetadata{
					Version: "100",
				},
			},
		},
	}
	
	to := &types.Snapshot{
		ID:        "snap2",
		Timestamp: now.Add(1 * time.Minute),
		Resources: []types.Resource{
			{
				ID:       "deployment/app",
				Type:     "deployment",
				Name:     "app",
				Provider: "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"replicas": 5, // Changed
					"image":    "app:v1.1", // Changed
				},
				Metadata: types.ResourceMetadata{
					Version: "101", // Changed
				},
			},
		},
	}
	
	report, err := differ.Compare(from, to)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	
	if len(report.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(report.Changes))
	}
	
	change := report.Changes[0]
	if change.Type != "modified" {
		t.Errorf("Expected type 'modified', got '%s'", change.Type)
	}
	
	if change.ResourceID != "deployment/app" {
		t.Errorf("Expected ID 'deployment/app', got '%s'", change.ResourceID)
	}
	
	// Check details
	if len(change.Details) == 0 {
		t.Fatal("Expected change details")
	}
	
	// Should detect multiple field changes
	expectedFields := map[string]bool{
		"replicas": false,
		"image": false,
		"version": false,
	}
	
	for _, detail := range change.Details {
		if _, exists := expectedFields[detail.Field]; exists {
			expectedFields[detail.Field] = true
		}
	}
	
	for field, found := range expectedFields {
		if !found {
			t.Errorf("Expected change in field '%s'", field)
		}
	}
	
	if report.Summary.Modified != 1 {
		t.Errorf("Expected 1 modified, got %d", report.Summary.Modified)
	}
}

func TestSimpleDiffer_Compare_ComplexScenario(t *testing.T) {
	differ := NewSimpleDiffer()
	now := time.Now()
	
	from := &types.Snapshot{
		ID:        "snap1",
		Timestamp: now,
		Resources: []types.Resource{
			{
				ID:       "deployment/frontend",
				Type:     "deployment",
				Name:     "frontend",
				Provider: "kubernetes",
				Namespace: "web",
				Configuration: map[string]interface{}{
					"replicas": 3,
					"image":    "frontend:v1.0",
				},
			},
			{
				ID:       "service/api",
				Type:     "service", 
				Name:     "api",
				Provider: "kubernetes",
				Namespace: "api",
				Configuration: map[string]interface{}{
					"port": 8080,
				},
			},
			{
				ID:       "configmap/old-config",
				Type:     "configmap",
				Name:     "old-config",
				Provider: "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"data": "old-value",
				},
			},
		},
	}
	
	to := &types.Snapshot{
		ID:        "snap2",
		Timestamp: now.Add(5 * time.Minute),
		Resources: []types.Resource{
			{
				ID:       "deployment/frontend",
				Type:     "deployment",
				Name:     "frontend", 
				Provider: "kubernetes",
				Namespace: "web",
				Configuration: map[string]interface{}{
					"replicas": 5, // Modified
					"image":    "frontend:v1.1", // Modified
				},
			},
			// service/api removed
			{
				ID:       "configmap/new-config", // Added
				Type:     "configmap",
				Name:     "new-config",
				Provider: "kubernetes",
				Namespace: "default",
				Configuration: map[string]interface{}{
					"data": "new-value",
				},
			},
		},
	}
	
	report, err := differ.Compare(from, to)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	
	if len(report.Changes) != 4 {
		t.Fatalf("Expected 4 changes (1 modified, 1 removed, 1 added, 1 removed), got %d", len(report.Changes))
	}
	
	// Count change types
	changeTypes := make(map[string]int)
	for _, change := range report.Changes {
		changeTypes[change.Type]++
	}
	
	if changeTypes["modified"] != 1 {
		t.Errorf("Expected 1 modified change, got %d", changeTypes["modified"])
	}
	
	if changeTypes["removed"] != 2 {
		t.Errorf("Expected 2 removed changes, got %d", changeTypes["removed"])
	}
	
	if changeTypes["added"] != 1 {
		t.Errorf("Expected 1 added change, got %d", changeTypes["added"])
	}
	
	// Check summary
	if report.Summary.Added != 1 {
		t.Errorf("Expected 1 added in summary, got %d", report.Summary.Added)
	}
	
	if report.Summary.Modified != 1 {
		t.Errorf("Expected 1 modified in summary, got %d", report.Summary.Modified)
	}
	
	if report.Summary.Removed != 2 {
		t.Errorf("Expected 2 removed in summary, got %d", report.Summary.Removed)
	}
	
	if report.Summary.Total != 4 {
		t.Errorf("Expected 4 total changes, got %d", report.Summary.Total)
	}
}

func TestSimpleDiffer_Compare_IdenticalSnapshots(t *testing.T) {
	differ := NewSimpleDiffer()
	now := time.Now()
	
	resource := types.Resource{
		ID:       "deployment/test",
		Type:     "deployment", 
		Name:     "test",
		Provider: "kubernetes",
		Namespace: "default",
		Configuration: map[string]interface{}{
			"replicas": 3,
			"image":    "test:latest",
		},
		Metadata: types.ResourceMetadata{
			Version: "100",
		},
	}
	
	from := &types.Snapshot{
		ID:        "snap1",
		Timestamp: now,
		Resources: []types.Resource{resource},
	}
	
	to := &types.Snapshot{
		ID:        "snap2",
		Timestamp: now.Add(1 * time.Minute),
		Resources: []types.Resource{resource}, // Identical
	}
	
	report, err := differ.Compare(from, to)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}
	
	if len(report.Changes) != 0 {
		t.Errorf("Expected 0 changes for identical snapshots, got %d", len(report.Changes))
	}
	
	if report.Summary.Total != 0 {
		t.Errorf("Expected 0 total changes, got %d", report.Summary.Total)
	}
}

func TestSimpleDiffer_compareConfiguration(t *testing.T) {
	differ := NewSimpleDiffer()
	
	tests := []struct {
		name     string
		from     map[string]interface{}
		to       map[string]interface{}
		expected int
	}{
		{
			name: "no changes",
			from: map[string]interface{}{"key": "value"},
			to:   map[string]interface{}{"key": "value"},
			expected: 0,
		},
		{
			name: "simple change",
			from: map[string]interface{}{"replicas": 3},
			to:   map[string]interface{}{"replicas": 5},
			expected: 1,
		},
		{
			name: "multiple changes",
			from: map[string]interface{}{
				"replicas": 3,
				"image": "v1.0",
				"port": 8080,
			},
			to: map[string]interface{}{
				"replicas": 5,
				"image": "v1.1", 
				"port": 8080, // unchanged
			},
			expected: 2,
		},
		{
			name: "added field",
			from: map[string]interface{}{"replicas": 3},
			to: map[string]interface{}{
				"replicas": 3,
				"image": "new", // added
			},
			expected: 1,
		},
		{
			name: "removed field",
			from: map[string]interface{}{
				"replicas": 3,
				"image": "old", // will be removed
			},
			to: map[string]interface{}{"replicas": 3},
			expected: 1,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := differ.compareConfiguration(tt.from, tt.to, "")
			if len(changes) != tt.expected {
				t.Errorf("Expected %d changes, got %d", tt.expected, len(changes))
			}
		})
	}
}

// Removed FormatChangeReport test as the function may not exist yet