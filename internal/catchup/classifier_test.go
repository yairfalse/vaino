package catchup

import (
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestClassifier_Classify(t *testing.T) {
	classifier := NewClassifier()

	tests := []struct {
		name     string
		change   Change
		expected ChangeType
	}{
		{
			name: "planned deployment",
			change: Change{
				Description: "Scheduled deployment of v2.1.0",
				Timestamp:   time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC), // Monday afternoon
				Tags:        []string{"deployment", "planned"},
			},
			expected: ChangeTypePlanned,
		},
		{
			name: "emergency hotfix",
			change: Change{
				Description: "Emergency hotfix for production issue",
				Timestamp:   time.Date(2024, 1, 15, 23, 30, 0, 0, time.UTC), // Late night
				Tags:        []string{"production", "urgent"},
			},
			expected: ChangeTypeUnplanned,
		},
		{
			name: "routine backup",
			change: Change{
				Description: "Automated nightly backup completed",
				Timestamp:   time.Date(2024, 1, 15, 2, 0, 0, 0, time.UTC), // 2 AM
				Resource: types.Resource{
					Type: "backup",
				},
			},
			expected: ChangeTypeRoutine,
		},
		{
			name: "incident response",
			change: Change{
				Description: "Response to database connection failure",
				Timestamp:   time.Date(2024, 1, 13, 15, 0, 0, 0, time.UTC), // Saturday afternoon
				Tags:        []string{"incident"},
			},
			expected: ChangeTypeUnplanned,
		},
		{
			name: "auto-scaling event",
			change: Change{
				Description: "Auto-scaling group adjusted capacity",
				Timestamp:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
				Resource: types.Resource{
					Type: "autoscaling-group",
				},
			},
			expected: ChangeTypeRoutine,
		},
		{
			name: "weekend maintenance",
			change: Change{
				Description: "Database maintenance window",
				Timestamp:   time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC), // Sunday morning
			},
			expected: ChangeTypePlanned,
		},
		{
			name: "business hours change without keywords",
			change: Change{
				Description: "Modified security group rules",
				Timestamp:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), // Monday 3 PM
			},
			expected: ChangeTypePlanned,
		},
		{
			name: "after hours change without keywords",
			change: Change{
				Description: "Modified security group rules",
				Timestamp:   time.Date(2024, 1, 15, 22, 0, 0, 0, time.UTC), // Monday 10 PM
			},
			expected: ChangeTypeUnplanned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.change)
			if result != tt.expected {
				t.Errorf("Classify() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifier_CustomPatterns(t *testing.T) {
	classifier := NewClassifier()

	// Add custom patterns
	classifier.AddPlannedPattern("schema migration")
	classifier.AddUnplannedPattern("service down")
	classifier.AddRoutinePattern("cache refresh")

	tests := []struct {
		name     string
		change   Change
		expected ChangeType
	}{
		{
			name: "custom planned pattern",
			change: Change{
				Description: "Running schema migration for user table",
				Timestamp:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			expected: ChangeTypePlanned,
		},
		{
			name: "custom unplanned pattern",
			change: Change{
				Description: "API service down - restarting",
				Timestamp:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			expected: ChangeTypeUnplanned,
		},
		{
			name: "custom routine pattern",
			change: Change{
				Description: "Redis cache refresh completed",
				Timestamp:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			},
			expected: ChangeTypeRoutine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.change)
			if result != tt.expected {
				t.Errorf("Classify() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifier_BusinessHours(t *testing.T) {
	classifier := NewClassifier()

	// Test custom business hours
	classifier.SetBusinessHours(8, 18, []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday,
	})

	// 8 AM Monday - should be business hours
	monday8am := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
	if !classifier.isDuringBusinessHours(monday8am) {
		t.Error("Expected 8 AM Monday to be during business hours")
	}

	// 7 AM Monday - should not be business hours
	monday7am := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	if classifier.isDuringBusinessHours(monday7am) {
		t.Error("Expected 7 AM Monday to not be during business hours")
	}

	// 5 PM Monday - should be business hours (before 6 PM)
	monday5pm := time.Date(2024, 1, 15, 17, 0, 0, 0, time.UTC)
	if !classifier.isDuringBusinessHours(monday5pm) {
		t.Error("Expected 5 PM Monday to be during business hours")
	}

	// 6 PM Monday - should not be business hours
	monday6pm := time.Date(2024, 1, 15, 18, 0, 0, 0, time.UTC)
	if classifier.isDuringBusinessHours(monday6pm) {
		t.Error("Expected 6 PM Monday to not be during business hours")
	}

	// Saturday - should not be business hours
	saturday := time.Date(2024, 1, 13, 12, 0, 0, 0, time.UTC)
	if classifier.isDuringBusinessHours(saturday) {
		t.Error("Expected Saturday to not be during business hours")
	}
}
