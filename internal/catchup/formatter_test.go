package catchup

import (
	"strings"
	"testing"
	"time"

	"github.com/yairfalse/wgo/pkg/types"
)

func TestFormatter_Format(t *testing.T) {
	// Create test report
	report := &Report{
		Period: Period{
			Start:    time.Now().Add(-7 * 24 * time.Hour),
			End:      time.Now(),
			Duration: 7 * 24 * time.Hour,
		},
		Summary: Summary{
			TotalChanges:      10,
			PlannedCount:      6,
			UnplannedCount:    2,
			RoutineCount:      2,
			CriticalSystems:   "All stable",
			SecurityIncidents: 0,
			TeamEfficiency:    "Excellent",
		},
		PlannedChanges: []Change{
			{
				Timestamp:    time.Now().Add(-5 * 24 * time.Hour),
				Type:         ChangeTypePlanned,
				Provider:     "aws",
				Description:  "Deployed new API version v2.3.0",
				Impact:       "Improved response times by 20%",
				IsSuccessful: true,
			},
		},
		UnplannedChanges: []Change{
			{
				Timestamp:    time.Now().Add(-3 * 24 * time.Hour),
				Type:         ChangeTypeUnplanned,
				Provider:     "kubernetes",
				Description:  "Pod crash loop detected and resolved",
				Impact:       "5 minutes of degraded service",
				HandledBy:    "ops-team",
				IsSuccessful: true,
			},
		},
		RoutineChanges: []Change{
			{
				Timestamp:   time.Now().Add(-2 * 24 * time.Hour),
				Type:        ChangeTypeRoutine,
				Provider:    "aws",
				Description: "Auto-scaling adjusted capacity",
				Resource: types.Resource{
					Type: "autoscaling-group",
				},
			},
		},
		SecurityStatus: SecurityStatus{
			IncidentCount:   0,
			ComplianceScore: 98.5,
			LastAudit:       time.Now().Add(-30 * 24 * time.Hour),
		},
		TeamActivity: TeamActivity{
			TotalActions:     10,
			TopContributors:  []string{"Alice (5 actions)", "Bob (3 actions)"},
			IncidentHandling: "Excellent",
		},
		ComfortMetrics: ComfortMetrics{
			StabilityScore:    0.95,
			TeamPerformance:   0.98,
			SystemResilience:  1.0,
			OverallConfidence: 0.96,
		},
		Recommendations: []string{
			"Continue the excellent work maintaining infrastructure stability!",
		},
	}

	t.Run("comfort mode", func(t *testing.T) {
		formatter := NewFormatter(true)
		output := formatter.Format(report)

		// Check for key comfort mode elements
		if !strings.Contains(output, "Welcome back!") {
			t.Error("Expected comfort introduction")
		}

		if !strings.Contains(output, "Everything went smoothly") {
			t.Error("Expected positive messaging")
		}

		if !strings.Contains(output, "System Health Metrics") {
			t.Error("Expected comfort metrics section")
		}

		if !strings.Contains(output, "You're all caught up!") {
			t.Error("Expected positive closing message")
		}
	})

	t.Run("standard mode", func(t *testing.T) {
		formatter := NewFormatter(false)
		output := formatter.Format(report)

		// Check for key sections
		if !strings.Contains(output, "Infrastructure Catch-Up Report") {
			t.Error("Expected report header")
		}

		if !strings.Contains(output, "Executive Summary") {
			t.Error("Expected executive summary")
		}

		if !strings.Contains(output, "Security Status") {
			t.Error("Expected security status")
		}

		if !strings.Contains(output, "Team Activity") {
			t.Error("Expected team activity")
		}

		if !strings.Contains(output, "Changes Breakdown") {
			t.Error("Expected changes breakdown")
		}

		// Should not have comfort-specific elements
		if strings.Contains(output, "Welcome back!") {
			t.Error("Standard mode should not have comfort introduction")
		}
	})
}

func TestFormatter_SecurityStatus(t *testing.T) {
	formatter := NewFormatter(true)

	t.Run("no incidents", func(t *testing.T) {
		report := &Report{
			SecurityStatus: SecurityStatus{
				IncidentCount:   0,
				ComplianceScore: 100.0,
			},
		}

		output := formatter.Format(report)

		if !strings.Contains(output, "✅ No security incidents occurred") {
			t.Error("Expected positive security message")
		}
	})

	t.Run("with incidents", func(t *testing.T) {
		report := &Report{
			SecurityStatus: SecurityStatus{
				IncidentCount:   2,
				ComplianceScore: 85.0,
				Vulnerabilities: []string{"CVE-2024-1234 patched"},
			},
		}

		output := formatter.Format(report)

		if !strings.Contains(output, "2 security incident(s) were handled") {
			t.Error("Expected incident count")
		}

		if !strings.Contains(output, "CVE-2024-1234 patched") {
			t.Error("Expected vulnerability details")
		}
	})
}

func TestFormatter_MetricBar(t *testing.T) {
	formatter := NewFormatter(true)

	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{
			name:     "perfect score",
			value:    1.0,
			expected: "████████████████████",
		},
		{
			name:     "half score",
			value:    0.5,
			expected: "██████████",
		},
		{
			name:     "low score",
			value:    0.2,
			expected: "████",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			formatter.writeMetricBar(&output, "Test Metric", tt.value)

			result := output.String()
			if !strings.Contains(result, tt.expected) {
				t.Errorf("Expected bar to contain %s filled blocks", tt.expected)
			}
		})
	}
}

func TestFormatter_DurationFormat(t *testing.T) {
	formatter := NewFormatter(false)

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{
			duration: 7 * 24 * time.Hour,
			expected: "7 days",
		},
		{
			duration: 36 * time.Hour,
			expected: "1 days, 12 hours",
		},
		{
			duration: 5 * time.Hour,
			expected: "5 hours",
		},
		{
			duration: 30 * time.Minute,
			expected: "30 minutes",
		},
	}

	for _, tt := range tests {
		result := formatter.formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %v, want %v", tt.duration, result, tt.expected)
		}
	}
}
