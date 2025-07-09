package analyzer

import (
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
)

func TestCorrelator_NewCorrelator(t *testing.T) {
	correlator := NewCorrelator()
	if correlator == nil {
		t.Fatal("NewCorrelator returned nil")
	}

	expected := 30 * time.Second
	if correlator.timeWindow != expected {
		t.Errorf("Expected time window %v, got %v", expected, correlator.timeWindow)
	}
}

func TestCorrelator_GroupChanges_EmptyInput(t *testing.T) {
	correlator := NewCorrelator()
	changes := []differ.SimpleChange{}

	groups := correlator.GroupChanges(changes)

	if groups != nil {
		t.Errorf("Expected nil for empty input, got %v", groups)
	}
}

func TestCorrelator_GroupChanges_SingleChange(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "deployment/test",
			ResourceType: "deployment",
			ResourceName: "test",
			Namespace:    "default",
			Timestamp:    now,
			Details: []differ.SimpleFieldChange{
				{Field: "image", OldValue: "v1.0", NewValue: "v1.1"},
			},
		},
	}

	groups := correlator.GroupChanges(changes)

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}

	group := groups[0]
	if group.Title != "Other Changes" {
		t.Errorf("Expected 'Other Changes', got '%s'", group.Title)
	}

	if group.Confidence != "low" {
		t.Errorf("Expected confidence 'low', got '%s'", group.Confidence)
	}

	if len(group.Changes) != 1 {
		t.Errorf("Expected 1 change in group, got %d", len(group.Changes))
	}
}

func TestCorrelator_DetectScalingGroup(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "deployment/frontend",
			ResourceType: "deployment",
			ResourceName: "frontend",
			Namespace:    "default",
			Timestamp:    now,
			Details: []differ.SimpleFieldChange{
				{Field: "replicas", OldValue: 3, NewValue: 5},
			},
		},
		{
			Type:         "modified",
			ResourceID:   "pod/frontend-abc123",
			ResourceType: "pod",
			ResourceName: "frontend-abc123",
			Namespace:    "default",
			Timestamp:    now.Add(5 * time.Second),
		},
	}

	groups := correlator.GroupChanges(changes)

	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}

	group := groups[0]
	if group.Title != "frontend Scaling" {
		t.Errorf("Expected 'frontend Scaling', got '%s'", group.Title)
	}

	if group.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", group.Confidence)
	}

	if group.Description != "Scaled from 3 to 5 replicas" {
		t.Errorf("Unexpected description: %s", group.Description)
	}

	if len(group.Changes) != 2 {
		t.Errorf("Expected 2 changes in scaling group, got %d", len(group.Changes))
	}
}

func TestCorrelator_DetectServiceGroup(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	changes := []differ.SimpleChange{
		{
			Type:         "added",
			ResourceID:   "service/api-service",
			ResourceType: "service",
			ResourceName: "api-service",
			Namespace:    "default",
			Timestamp:    now,
		},
		{
			Type:         "added",
			ResourceID:   "deployment/api",
			ResourceType: "deployment",
			ResourceName: "api",
			Namespace:    "default",
			Timestamp:    now.Add(2 * time.Second),
		},
		{
			Type:         "added",
			ResourceID:   "configmap/api-config",
			ResourceType: "configmap",
			ResourceName: "api-config",
			Namespace:    "default",
			Timestamp:    now.Add(3 * time.Second),
		},
	}

	groups := correlator.GroupChanges(changes)

	// Should find service group
	var serviceGroup *ChangeGroup
	for _, group := range groups {
		if group.Title == "New Service: api-service" {
			serviceGroup = &group
			break
		}
	}

	if serviceGroup == nil {
		t.Fatal("Service group not found")
	}

	if serviceGroup.Confidence != "medium" {
		t.Errorf("Expected confidence 'medium', got '%s'", serviceGroup.Confidence)
	}

	if len(serviceGroup.Changes) < 2 {
		t.Errorf("Expected at least 2 changes in service group, got %d", len(serviceGroup.Changes))
	}
}

func TestCorrelator_DetectConfigUpdateGroup(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "configmap/app-config",
			ResourceType: "configmap",
			ResourceName: "app-config",
			Namespace:    "default",
			Timestamp:    now,
			Details: []differ.SimpleFieldChange{
				{Field: "data.version", OldValue: "1.0", NewValue: "1.1"},
			},
		},
		{
			Type:         "modified",
			ResourceID:   "deployment/app",
			ResourceType: "deployment",
			ResourceName: "app",
			Namespace:    "default",
			Timestamp:    now.Add(30 * time.Second),
			Details: []differ.SimpleFieldChange{
				{Field: "generation", OldValue: 5, NewValue: 6},
			},
		},
	}

	groups := correlator.GroupChanges(changes)

	// Should find config update group
	var configGroup *ChangeGroup
	for _, group := range groups {
		if group.Title == "app-config Update" {
			configGroup = &group
			break
		}
	}

	if configGroup == nil {
		t.Fatal("Config update group not found")
	}

	if configGroup.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", configGroup.Confidence)
	}

	if len(configGroup.Changes) != 2 {
		t.Errorf("Expected 2 changes in config group, got %d", len(configGroup.Changes))
	}
}

func TestCorrelator_DetectSecretRotation(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "secret/api-keys",
			ResourceType: "secret",
			ResourceName: "api-keys",
			Namespace:    "default",
			Timestamp:    now,
		},
		{
			Type:         "modified",
			ResourceID:   "secret/db-credentials",
			ResourceType: "secret",
			ResourceName: "db-credentials",
			Namespace:    "default",
			Timestamp:    now.Add(5 * time.Second),
		},
		{
			Type:         "modified",
			ResourceID:   "secret/tls-certs",
			ResourceType: "secret",
			ResourceName: "tls-certs",
			Namespace:    "default",
			Timestamp:    now.Add(10 * time.Second),
		},
	}

	groups := correlator.GroupChanges(changes)

	// Should find secret rotation group
	var secretGroup *ChangeGroup
	for _, group := range groups {
		if group.Title == "Secret Rotation in default" {
			secretGroup = &group
			break
		}
	}

	if secretGroup == nil {
		t.Fatal("Secret rotation group not found")
	}

	if secretGroup.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", secretGroup.Confidence)
	}

	if len(secretGroup.Changes) != 3 {
		t.Errorf("Expected 3 changes in secret rotation group, got %d", len(secretGroup.Changes))
	}
}

func TestCorrelator_AvoidFalseCorrelations(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	// Changes that should NOT be correlated
	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "deployment/frontend",
			ResourceType: "deployment",
			ResourceName: "frontend",
			Namespace:    "web",
			Timestamp:    now,
			Details: []differ.SimpleFieldChange{
				{Field: "replicas", OldValue: 3, NewValue: 5},
			},
		},
		{
			Type:         "modified",
			ResourceID:   "configmap/backend-config",
			ResourceType: "configmap",
			ResourceName: "backend-config",
			Namespace:    "api", // Different namespace
			Timestamp:    now.Add(5 * time.Second),
		},
		{
			Type:         "modified",
			ResourceID:   "secret/unrelated-secret",
			ResourceType: "secret",
			ResourceName: "unrelated-secret",
			Namespace:    "other",                  // Different namespace
			Timestamp:    now.Add(2 * time.Minute), // Outside time window
		},
	}

	groups := correlator.GroupChanges(changes)

	// Should have one scaling group and one other changes group
	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups (scaling + other), got %d", len(groups))
	}

	// Check that scaling is detected correctly
	var scalingGroup *ChangeGroup
	var otherGroup *ChangeGroup

	for _, group := range groups {
		if group.Title == "frontend Scaling" {
			scalingGroup = &group
		} else if group.Title == "Other Changes" {
			otherGroup = &group
		}
	}

	if scalingGroup == nil {
		t.Error("Scaling group should be detected")
	} else if len(scalingGroup.Changes) != 1 {
		t.Errorf("Scaling group should have 1 change, got %d", len(scalingGroup.Changes))
	}

	if otherGroup == nil {
		t.Error("Other changes group should exist")
	} else if len(otherGroup.Changes) != 2 {
		t.Errorf("Other changes should have 2 changes, got %d", len(otherGroup.Changes))
	}
}

func TestCorrelator_TimeWindowRespected(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	changes := []differ.SimpleChange{
		{
			Type:         "modified",
			ResourceID:   "configmap/app-config",
			ResourceType: "configmap",
			ResourceName: "app-config",
			Namespace:    "default",
			Timestamp:    now,
		},
		{
			Type:         "modified",
			ResourceID:   "deployment/app",
			ResourceType: "deployment",
			ResourceName: "app",
			Namespace:    "default",
			Timestamp:    now.Add(45 * time.Second), // Outside 30s window
			Details: []differ.SimpleFieldChange{
				{Field: "generation", OldValue: 5, NewValue: 6},
			},
		},
	}

	groups := correlator.GroupChanges(changes)

	// Should NOT correlate due to time window
	if len(groups) != 1 {
		t.Fatalf("Expected 1 group (other changes), got %d", len(groups))
	}

	if groups[0].Title != "Other Changes" {
		t.Errorf("Expected 'Other Changes', got '%s'", groups[0].Title)
	}

	if len(groups[0].Changes) != 2 {
		t.Errorf("Expected 2 uncorrelated changes, got %d", len(groups[0].Changes))
	}
}

func TestCorrelator_isWithinTimeWindow(t *testing.T) {
	correlator := NewCorrelator()
	now := time.Now()

	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected bool
	}{
		{
			name:     "same time",
			t1:       now,
			t2:       now,
			expected: true,
		},
		{
			name:     "within window",
			t1:       now,
			t2:       now.Add(15 * time.Second),
			expected: true,
		},
		{
			name:     "within window reverse",
			t1:       now.Add(15 * time.Second),
			t2:       now,
			expected: true,
		},
		{
			name:     "outside window",
			t1:       now,
			t2:       now.Add(45 * time.Second),
			expected: false,
		},
		{
			name:     "exactly at boundary",
			t1:       now,
			t2:       now.Add(30 * time.Second),
			expected: true,
		},
		{
			name:     "just outside boundary",
			t1:       now,
			t2:       now.Add(31 * time.Second),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := correlator.isWithinTimeWindow(tt.t1, tt.t2)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for times %v and %v", tt.expected, result, tt.t1, tt.t2)
			}
		})
	}
}
