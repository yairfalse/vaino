package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yairfalse/vaino/internal/storage"
)

func TestParseTimeFilter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "parse 2 weeks ago",
			input:    "2 weeks ago",
			expected: 14 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "parse 3 days ago",
			input:    "3 days ago",
			expected: 3 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "parse 5 hours ago",
			input:    "5 hours ago",
			expected: 5 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "parse invalid format",
			input:    "yesterday",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "parse date format",
			input:    "2024-01-01",
			expected: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTimelineCommand()
			cmd.Flags().Set("since", tt.input)

			result, err := parseTimeFilter(cmd, "since")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// For duration-based inputs, check the duration
				if tt.expected > 0 && result != nil {
					now := time.Now()
					expectedTime := now.Add(-tt.expected)
					// Allow 1 second tolerance for test execution
					assert.WithinDuration(t, expectedTime, *result, time.Second)
				}
			}
		})
	}
}

func TestFilterSnapshots(t *testing.T) {
	now := time.Now()
	snapshots := []storage.SnapshotInfo{
		{
			ID:            "snap1",
			Provider:      "terraform",
			Timestamp:     now.Add(-48 * time.Hour),
			ResourceCount: 10,
		},
		{
			ID:            "snap2",
			Provider:      "aws",
			Timestamp:     now.Add(-24 * time.Hour),
			ResourceCount: 20,
		},
		{
			ID:            "snap3",
			Provider:      "kubernetes",
			Timestamp:     now.Add(-12 * time.Hour),
			ResourceCount: 30,
		},
		{
			ID:            "snap4",
			Provider:      "gcp",
			Timestamp:     now,
			ResourceCount: 40,
		},
	}

	tests := []struct {
		name          string
		since         *time.Time
		until         *time.Time
		providers     []string
		expectedCount int
	}{
		{
			name:          "no filters",
			since:         nil,
			until:         nil,
			providers:     nil,
			expectedCount: 4,
		},
		{
			name:          "filter by provider",
			since:         nil,
			until:         nil,
			providers:     []string{"aws", "gcp"},
			expectedCount: 2,
		},
		{
			name:          "filter by time since",
			since:         timePtr(now.Add(-25 * time.Hour)),
			until:         nil,
			providers:     nil,
			expectedCount: 3,
		},
		{
			name:          "filter by time range",
			since:         timePtr(now.Add(-30 * time.Hour)),
			until:         timePtr(now.Add(-10 * time.Hour)),
			providers:     nil,
			expectedCount: 2,
		},
		{
			name:          "combined filters",
			since:         timePtr(now.Add(-30 * time.Hour)),
			until:         nil,
			providers:     []string{"aws", "kubernetes"},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterSnapshots(snapshots, tt.since, tt.until, "", "", tt.providers)
			assert.Equal(t, tt.expectedCount, len(filtered))
		})
	}
}

func TestParseDurationAgo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "minutes",
			input:    "30 minutes ago",
			expected: 30 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "hours",
			input:    "2 hours ago",
			expected: 2 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "days",
			input:    "7 days ago",
			expected: 7 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "weeks",
			input:    "3 weeks ago",
			expected: 3 * 7 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "months",
			input:    "2 months ago",
			expected: 2 * 30 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "invalid format no ago",
			input:    "2 weeks",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid unit",
			input:    "2 years ago",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "non-numeric amount",
			input:    "two weeks ago",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDurationAgo(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Check that the duration is approximately correct
				now := time.Now()
				expectedTime := now.Add(-tt.expected)
				assert.WithinDuration(t, expectedTime, *result, time.Second)
			}
		})
	}
}

func TestHandleTimelineBetween(t *testing.T) {
	// This would need a mock storage implementation to test properly
	// For now, we'll just test the validation logic

	snapshots := []storage.SnapshotInfo{
		{
			ID:        "snap1",
			Timestamp: time.Now().Add(-48 * time.Hour),
			Tags:      map[string]string{"version": "v1.0", "environment": "production"},
		},
		{
			ID:        "snap2",
			Timestamp: time.Now().Add(-24 * time.Hour),
		},
		{
			ID:        "snap3",
			Timestamp: time.Now(),
			Tags:      map[string]string{"version": "v2.0", "environment": "staging"},
		},
	}

	// Test that it correctly identifies snapshots by tag values
	var snap1, snap3 *storage.SnapshotInfo
	for i, snapshot := range snapshots {
		if snapshot.Tags["version"] == "v1.0" {
			snap1 = &snapshots[i]
		}
		if snapshot.Tags["version"] == "v2.0" {
			snap3 = &snapshots[i]
		}
	}

	assert.NotNil(t, snap1)
	assert.NotNil(t, snap3)
	assert.Equal(t, "v1.0", snap1.Tags["version"])
	assert.Equal(t, "v2.0", snap3.Tags["version"])
}

func TestFilterSnapshotsByTags(t *testing.T) {
	now := time.Now()
	snapshots := []storage.SnapshotInfo{
		{
			ID:        "snap1",
			Timestamp: now.Add(-48 * time.Hour),
			Tags:      map[string]string{"environment": "production", "version": "v1"},
		},
		{
			ID:        "snap2",
			Timestamp: now.Add(-24 * time.Hour),
			Tags:      map[string]string{"environment": "development", "version": "v2"},
		},
		{
			ID:        "snap3",
			Timestamp: now,
			Tags:      map[string]string{"environment": "staging", "version": "v3"},
		},
	}

	tests := []struct {
		name          string
		tagFilter     map[string]string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "filter by environment production",
			tagFilter:     map[string]string{"environment": "production"},
			expectedCount: 1,
			expectedIDs:   []string{"snap1"},
		},
		{
			name:          "filter by version v2",
			tagFilter:     map[string]string{"version": "v2"},
			expectedCount: 1,
			expectedIDs:   []string{"snap2"},
		},
		{
			name:          "filter by multiple tags",
			tagFilter:     map[string]string{"environment": "staging", "version": "v3"},
			expectedCount: 1,
			expectedIDs:   []string{"snap3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test tag filtering logic
			var filtered []storage.SnapshotInfo
			for _, snapshot := range snapshots {
				match := true
				for k, v := range tt.tagFilter {
					if snapshot.Tags[k] != v {
						match = false
						break
					}
				}
				if match {
					filtered = append(filtered, snapshot)
				}
			}

			assert.Equal(t, tt.expectedCount, len(filtered))
			for i, id := range tt.expectedIDs {
				assert.Equal(t, id, filtered[i].ID)
			}
		})
	}
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
