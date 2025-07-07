package analyzer

import (
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/wgo/internal/differ"
)

func TestFormatCorrelatedChanges_EmptyGroups(t *testing.T) {
	groups := []ChangeGroup{}
	result := FormatCorrelatedChanges(groups)
	
	expected := "📊 Correlated Infrastructure Changes\n====================================\n\n"
	if result != expected {
		t.Errorf("Expected empty groups output, got:\n%s", result)
	}
}

func TestFormatCorrelatedChanges_SingleGroup(t *testing.T) {
	now := time.Now()
	groups := []ChangeGroup{
		{
			Timestamp:   now,
			Title:       "frontend Scaling",
			Description: "Scaled from 3 to 5 replicas",
			Reason:      "Deployment scaling detected",
			Confidence:  "high",
			Changes: []differ.SimpleChange{
				{
					Type:         "modified",
					ResourceName: "frontend",
					ResourceType: "deployment",
					Details: []differ.SimpleFieldChange{
						{Field: "replicas", OldValue: 3, NewValue: 5},
					},
				},
			},
		},
	}
	
	result := FormatCorrelatedChanges(groups)
	
	// Check for confidence indicator
	if !strings.Contains(result, "● 🔗 frontend Scaling") {
		t.Error("Expected high confidence indicator (●)")
	}
	
	// Check for description
	if !strings.Contains(result, "Scaled from 3 to 5 replicas") {
		t.Error("Expected description to be included")
	}
	
	// Check for reason
	if !strings.Contains(result, "Reason: Deployment scaling detected") {
		t.Error("Expected reason to be included")
	}
	
	// Check for time
	if !strings.Contains(result, "Time: "+now.Format("15:04:05")) {
		t.Error("Expected time to be included")
	}
	
	// Check for change details
	if !strings.Contains(result, "~ frontend (deployment)") {
		t.Error("Expected change details")
	}
	
	if !strings.Contains(result, "• replicas: 3 → 5") {
		t.Error("Expected field change details")
	}
}

func TestFormatCorrelatedChanges_MultipleGroups(t *testing.T) {
	now := time.Now()
	groups := []ChangeGroup{
		{
			Timestamp:   now,
			Title:       "frontend Scaling",
			Description: "Scaled from 3 to 5 replicas",
			Confidence:  "high",
			Changes: []differ.SimpleChange{
				{
					Type:         "modified",
					ResourceName: "frontend",
					ResourceType: "deployment",
				},
			},
		},
		{
			Timestamp:   now.Add(1 * time.Minute),
			Title:       "Other Changes",
			Description: "Individual resource changes",
			Confidence:  "low",
			Changes: []differ.SimpleChange{
				{
					Type:         "added",
					ResourceName: "new-service",
					ResourceType: "service",
				},
			},
		},
	}
	
	result := FormatCorrelatedChanges(groups)
	
	// Check for both confidence indicators
	if !strings.Contains(result, "● 🔗 frontend Scaling") {
		t.Error("Expected high confidence indicator (●) for first group")
	}
	
	if !strings.Contains(result, "○ 🔗 Other Changes") {
		t.Error("Expected low confidence indicator (○) for second group")
	}
	
	// Check for separation between groups
	lines := strings.Split(result, "\n")
	emptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLines++
		}
	}
	
	if emptyLines < 2 {
		t.Error("Expected proper separation between groups")
	}
}

func TestFormatCorrelatedChanges_ConfidenceIndicators(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name               string
		confidence         string
		expectedIndicator  string
	}{
		{"high confidence", "high", "●"},
		{"medium confidence", "medium", "◐"},
		{"low confidence", "low", "○"},
		{"unknown confidence", "unknown", "○"},
		{"empty confidence", "", "○"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := []ChangeGroup{
				{
					Timestamp:   now,
					Title:       "Test Group",
					Description: "Test description",
					Confidence:  tt.confidence,
					Changes: []differ.SimpleChange{
						{
							Type:         "modified",
							ResourceName: "test",
							ResourceType: "deployment",
						},
					},
				},
			}
			
			result := FormatCorrelatedChanges(groups)
			expectedLine := tt.expectedIndicator + " 🔗 Test Group"
			
			if !strings.Contains(result, expectedLine) {
				t.Errorf("Expected '%s' in output for confidence '%s', got:\n%s", 
					expectedLine, tt.confidence, result)
			}
		})
	}
}

func TestFormatCorrelatedChanges_ChangeTypes(t *testing.T) {
	now := time.Now()
	
	groups := []ChangeGroup{
		{
			Timestamp:   now,
			Title:       "Mixed Changes",
			Description: "Various change types",
			Confidence:  "medium",
			Changes: []differ.SimpleChange{
				{
					Type:         "added",
					ResourceName: "new-service",
					ResourceType: "service",
				},
				{
					Type:         "removed",
					ResourceName: "old-config",
					ResourceType: "configmap",
				},
				{
					Type:         "modified",
					ResourceName: "updated-deploy",
					ResourceType: "deployment",
					Details: []differ.SimpleFieldChange{
						{Field: "image", OldValue: "v1.0", NewValue: "v1.1"},
					},
				},
			},
		},
	}
	
	result := FormatCorrelatedChanges(groups)
	
	// Check for different change type indicators
	if !strings.Contains(result, "+ new-service (service)") {
		t.Error("Expected added resource indicator (+)")
	}
	
	if !strings.Contains(result, "- old-config (configmap)") {
		t.Error("Expected removed resource indicator (-)")
	}
	
	if !strings.Contains(result, "~ updated-deploy (deployment)") {
		t.Error("Expected modified resource indicator (~)")
	}
	
	// Check for modification details
	if !strings.Contains(result, "• image: v1.0 → v1.1") {
		t.Error("Expected modification details")
	}
}

func TestFormatChangeTimeline_EmptyGroups(t *testing.T) {
	groups := []ChangeGroup{}
	duration := 5 * time.Minute
	
	result := FormatChangeTimeline(groups, duration)
	
	if !strings.Contains(result, "📅 Change Timeline") {
		t.Error("Expected timeline header")
	}
	
	if !strings.Contains(result, "No changes in this time period") {
		t.Error("Expected no changes message")
	}
}

func TestFormatChangeTimeline_SingleGroup(t *testing.T) {
	now := time.Now()
	groups := []ChangeGroup{
		{
			Timestamp: now,
			Title:     "Test Change",
			Changes: []differ.SimpleChange{
				{Type: "modified", ResourceName: "test"},
			},
		},
	}
	duration := 5 * time.Minute
	
	result := FormatChangeTimeline(groups, duration)
	
	// Check for timeline header
	if !strings.Contains(result, "📅 Change Timeline") {
		t.Error("Expected timeline header")
	}
	
	// Check for time markers
	timeStr := now.Format("15:04")
	if !strings.Contains(result, timeStr) {
		t.Error("Expected time markers")
	}
	
	// Check for timeline bar
	if !strings.Contains(result, "━") {
		t.Error("Expected timeline bar")
	}
	
	// Check for change marker
	if !strings.Contains(result, "▲") {
		t.Error("Expected change marker")
	}
	
	// Check for change description
	if !strings.Contains(result, "Test Change (1 changes)") {
		t.Error("Expected change description")
	}
}

func TestFormatChangeTimeline_MultipleGroups(t *testing.T) {
	now := time.Now()
	groups := []ChangeGroup{
		{
			Timestamp: now,
			Title:     "First Change",
			Changes: []differ.SimpleChange{
				{Type: "modified", ResourceName: "test1"},
			},
		},
		{
			Timestamp: now.Add(2 * time.Minute),
			Title:     "Second Change",
			Changes: []differ.SimpleChange{
				{Type: "added", ResourceName: "test2"},
				{Type: "modified", ResourceName: "test3"},
			},
		},
		{
			Timestamp: now.Add(4 * time.Minute),
			Title:     "Third Change",
			Changes: []differ.SimpleChange{
				{Type: "removed", ResourceName: "test4"},
			},
		},
	}
	duration := 5 * time.Minute
	
	result := FormatChangeTimeline(groups, duration)
	
	// Check for all three groups
	if !strings.Contains(result, "First Change (1 changes)") {
		t.Error("Expected first change group")
	}
	
	if !strings.Contains(result, "Second Change (2 changes)") {
		t.Error("Expected second change group")
	}
	
	if !strings.Contains(result, "Third Change (1 changes)") {
		t.Error("Expected third change group")
	}
	
	// Check for multiple markers
	markerCount := strings.Count(result, "▲")
	if markerCount != 3 {
		t.Errorf("Expected 3 markers, got %d", markerCount)
	}
}

func TestFormatChangeTimeline_TimeRangeCalculation(t *testing.T) {
	// Test with zero time range (all changes at same time)
	now := time.Now()
	groups := []ChangeGroup{
		{
			Timestamp: now,
			Title:     "Change 1",
			Changes:   []differ.SimpleChange{{Type: "modified"}},
		},
		{
			Timestamp: now, // Same time
			Title:     "Change 2", 
			Changes:   []differ.SimpleChange{{Type: "added"}},
		},
	}
	
	result := FormatChangeTimeline(groups, 0)
	
	// Should handle zero time range gracefully
	if !strings.Contains(result, "📅 Change Timeline") {
		t.Error("Expected timeline header even with zero time range")
	}
	
	// Should still show both changes
	if !strings.Contains(result, "Change 1") {
		t.Error("Expected first change")
	}
	
	if !strings.Contains(result, "Change 2") {
		t.Error("Expected second change")
	}
}

func TestFormatChangeTimeline_PositionCalculation(t *testing.T) {
	// Test timeline positioning with specific time intervals
	start := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	
	groups := []ChangeGroup{
		{
			Timestamp: start, // Beginning
			Title:     "Start Change",
			Changes:   []differ.SimpleChange{{Type: "added"}},
		},
		{
			Timestamp: start.Add(30 * time.Second), // Middle
			Title:     "Middle Change",
			Changes:   []differ.SimpleChange{{Type: "modified"}},
		},
		{
			Timestamp: start.Add(60 * time.Second), // End
			Title:     "End Change", 
			Changes:   []differ.SimpleChange{{Type: "removed"}},
		},
	}
	
	result := FormatChangeTimeline(groups, time.Minute)
	
	lines := strings.Split(result, "\n")
	
	// Find timeline and marker lines
	var timelineLine string
	var markerLines []string
	
	for _, line := range lines {
		if strings.Contains(line, "━") {
			timelineLine = line
		}
		if strings.Contains(line, "▲") {
			markerLines = append(markerLines, line)
		}
	}
	
	if timelineLine == "" {
		t.Error("Timeline line not found")
	}
	
	if len(markerLines) != 3 {
		t.Errorf("Expected 3 marker lines, got %d", len(markerLines))
	}
	
	// Verify markers are positioned differently (not all at same position)
	positions := make(map[int]bool)
	for _, line := range markerLines {
		pos := strings.Index(line, "▲")
		if pos >= 0 {
			positions[pos] = true
		}
	}
	
	// Should have markers at different positions (at least 2 different positions)
	if len(positions) < 2 {
		t.Error("Expected markers at different positions on timeline")
	}
}