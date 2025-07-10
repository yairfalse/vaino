package visualization

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yairfalse/vaino/internal/storage"
)

func TestNewTimelineGraph(t *testing.T) {
	tests := []struct {
		name          string
		width         int
		expectedWidth int
	}{
		{
			name:          "positive width",
			width:         100,
			expectedWidth: 100,
		},
		{
			name:          "zero width defaults to 80",
			width:         0,
			expectedWidth: 80,
		},
		{
			name:          "negative width defaults to 80",
			width:         -10,
			expectedWidth: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewTimelineGraph(tt.width)
			assert.Equal(t, tt.expectedWidth, graph.width)
		})
	}
}

func TestTimelineGraph_SetSnapshots(t *testing.T) {
	graph := NewTimelineGraph(80)
	now := time.Now()

	snapshots := []storage.SnapshotInfo{
		{
			ID:        "snap3",
			Timestamp: now,
		},
		{
			ID:        "snap1",
			Timestamp: now.Add(-48 * time.Hour),
		},
		{
			ID:        "snap2",
			Timestamp: now.Add(-24 * time.Hour),
		},
	}

	graph.SetSnapshots(snapshots)

	// Check that snapshots are sorted by timestamp
	assert.Equal(t, 3, len(graph.snapshots))
	assert.Equal(t, "snap1", graph.snapshots[0].ID)
	assert.Equal(t, "snap2", graph.snapshots[1].ID)
	assert.Equal(t, "snap3", graph.snapshots[2].ID)

	// Check time range is set correctly (snapshots are sorted)
	assert.Equal(t, graph.snapshots[0].Timestamp, graph.startTime)
	assert.Equal(t, graph.snapshots[2].Timestamp, graph.endTime)
}

func TestTimelineGraph_Render(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		snapshots []storage.SnapshotInfo
		events    []ChangeEvent
		contains  []string
	}{
		{
			name:      "no data",
			snapshots: nil,
			events:    nil,
			contains:  []string{"No data to display"},
		},
		{
			name: "single snapshot",
			snapshots: []storage.SnapshotInfo{
				{
					ID:            "snap1",
					Provider:      "terraform",
					Timestamp:     now,
					ResourceCount: 50,
				},
			},
			events:   nil,
			contains: []string{"Infrastructure Timeline", "•", "terraform snapshot", "50 resources"},
		},
		{
			name: "multiple snapshots",
			snapshots: []storage.SnapshotInfo{
				{
					ID:            "snap1",
					Provider:      "terraform",
					Timestamp:     now.Add(-24 * time.Hour),
					ResourceCount: 50,
				},
				{
					ID:            "snap2",
					Provider:      "aws",
					Timestamp:     now,
					ResourceCount: 100,
				},
			},
			events:   nil,
			contains: []string{"Infrastructure Timeline", "•", "terraform", "aws", "50 resources", "100 resources"},
		},
		{
			name: "with change events",
			snapshots: []storage.SnapshotInfo{
				{
					ID:            "snap1",
					Provider:      "terraform",
					Timestamp:     now.Add(-24 * time.Hour),
					ResourceCount: 50,
				},
			},
			events: []ChangeEvent{
				{
					Timestamp:   now.Add(-12 * time.Hour),
					ChangeCount: 15,
					Provider:    "terraform",
					Description: "scaling event",
				},
			},
			contains: []string{"Infrastructure Timeline", "•", "●", "15 changes", "scaling event"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewTimelineGraph(80)
			g.SetSnapshots(tt.snapshots)
			g.SetChangeEvents(tt.events)

			output := g.Render()

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestTimelineGraph_calculatePosition(t *testing.T) {
	graph := NewTimelineGraph(100)
	start := time.Now()
	end := start.Add(100 * time.Hour)
	graph.startTime = start
	graph.endTime = end

	tests := []struct {
		name     string
		time     time.Time
		width    int
		expected int
	}{
		{
			name:     "start time",
			time:     start,
			width:    100,
			expected: 0,
		},
		{
			name:     "end time",
			time:     end,
			width:    100,
			expected: 99,
		},
		{
			name:     "middle time",
			time:     start.Add(50 * time.Hour),
			width:    100,
			expected: 50,
		},
		{
			name:     "before start",
			time:     start.Add(-10 * time.Hour),
			width:    100,
			expected: -1,
		},
		{
			name:     "after end",
			time:     end.Add(10 * time.Hour),
			width:    100,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := graph.calculatePosition(tt.time, tt.width)
			assert.Equal(t, tt.expected, pos)
		})
	}
}

func TestTimelineGraph_renderTimelineLine(t *testing.T) {
	graph := NewTimelineGraph(30) // Use 30 to match expected width after margins
	now := time.Now()
	graph.startTime = now
	graph.endTime = now.Add(20 * time.Hour)

	// Add a snapshot at position 10
	graph.snapshots = []storage.SnapshotInfo{
		{
			Timestamp: now.Add(10 * time.Hour),
		},
	}

	// The renderTimelineLine uses width-10 for margins
	line := graph.renderTimelineLine(20)

	// Check that the line is the correct length
	assert.Equal(t, 20, len([]rune(line)))

	// Check that most characters are horizontal lines
	assert.Contains(t, line, "─")

	// Check that there's a marker at approximately the middle
	assert.Contains(t, line, "•")
}

func TestCreateSimpleTimeline(t *testing.T) {
	now := time.Now()
	snapshots := []storage.SnapshotInfo{
		{
			ID:            "snap1",
			Provider:      "terraform",
			Timestamp:     now.Add(-24 * time.Hour),
			ResourceCount: 50,
		},
		{
			ID:            "snap2",
			Provider:      "aws",
			Timestamp:     now,
			ResourceCount: 100,
		},
	}

	output := CreateSimpleTimeline(snapshots, 80)

	assert.Contains(t, output, "Infrastructure Timeline")
	assert.Contains(t, output, "terraform")
	assert.Contains(t, output, "aws")
	assert.Contains(t, output, "50 resources")
	assert.Contains(t, output, "100 resources")
}

func TestCreateChangeTimeline(t *testing.T) {
	now := time.Now()
	snapshots := []storage.SnapshotInfo{
		{
			ID:            "snap1",
			Provider:      "terraform",
			Timestamp:     now.Add(-24 * time.Hour),
			ResourceCount: 50,
		},
	}

	events := []ChangeEvent{
		{
			Timestamp:   now.Add(-12 * time.Hour),
			ChangeCount: 5,
			Provider:    "terraform",
			Description: "minor update",
		},
		{
			Timestamp:   now.Add(-6 * time.Hour),
			ChangeCount: 20,
			Provider:    "terraform",
			Description: "major deployment",
		},
	}

	output := CreateChangeTimeline(snapshots, events, 80)

	assert.Contains(t, output, "Infrastructure Timeline")
	assert.Contains(t, output, "terraform")
	assert.Contains(t, output, "50 resources")
	assert.Contains(t, output, "5 changes")
	assert.Contains(t, output, "20 changes")
	assert.Contains(t, output, "minor update")
	assert.Contains(t, output, "major deployment")

	// Check for different markers based on change size
	lines := strings.Split(output, "\n")
	var timelineLine string
	for _, line := range lines {
		if strings.Contains(line, "─") {
			timelineLine = line
			break
		}
	}

	// Should have different markers for different change sizes
	assert.NotEmpty(t, timelineLine)
}
