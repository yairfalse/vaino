package visualization

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/yairfalse/vaino/internal/storage"
)

// TimelineGraph represents a terminal-based timeline visualization
type TimelineGraph struct {
	width        int
	snapshots    []storage.SnapshotInfo
	changeEvents []ChangeEvent
	startTime    time.Time
	endTime      time.Time
}

// ChangeEvent represents a change event on the timeline
type ChangeEvent struct {
	Timestamp   time.Time
	ChangeCount int
	Provider    string
	Description string
}

// NewTimelineGraph creates a new timeline graph
func NewTimelineGraph(width int) *TimelineGraph {
	if width <= 0 {
		width = 80 // default terminal width
	}
	return &TimelineGraph{
		width: width,
	}
}

// SetSnapshots sets the snapshots to visualize
func (tg *TimelineGraph) SetSnapshots(snapshots []storage.SnapshotInfo) {
	tg.snapshots = snapshots
	if len(snapshots) > 0 {
		// Sort by timestamp
		sort.Slice(tg.snapshots, func(i, j int) bool {
			return tg.snapshots[i].Timestamp.Before(tg.snapshots[j].Timestamp)
		})
		tg.startTime = tg.snapshots[0].Timestamp
		tg.endTime = tg.snapshots[len(tg.snapshots)-1].Timestamp
	}
}

// SetChangeEvents sets the change events to visualize
func (tg *TimelineGraph) SetChangeEvents(events []ChangeEvent) {
	tg.changeEvents = events
	if len(events) > 0 && len(tg.snapshots) == 0 {
		// If no snapshots, use events to determine time range
		sort.Slice(tg.changeEvents, func(i, j int) bool {
			return tg.changeEvents[i].Timestamp.Before(tg.changeEvents[j].Timestamp)
		})
		tg.startTime = tg.changeEvents[0].Timestamp
		tg.endTime = tg.changeEvents[len(tg.changeEvents)-1].Timestamp
	}
}

// Render generates the beautiful timeline graph
func (tg *TimelineGraph) Render() string {
	if len(tg.snapshots) == 0 && len(tg.changeEvents) == 0 {
		return "No data to display"
	}

	var output strings.Builder

	// Calculate timeline parameters
	duration := tg.endTime.Sub(tg.startTime)
	if duration == 0 {
		duration = time.Hour * 24 // default to 1 day if all events at same time
		tg.startTime = tg.startTime.Add(-duration / 2)
		tg.endTime = tg.endTime.Add(duration / 2)
	}

	// Header with color
	headerColor := color.New(color.FgWhite, color.Bold)
	dateColor := color.New(color.FgCyan)
	output.WriteString(headerColor.Sprint("Infrastructure Timeline "))
	output.WriteString(fmt.Sprintf("(%s to %s)\n",
		dateColor.Sprint(tg.startTime.Format("Jan 2")),
		dateColor.Sprint(tg.endTime.Format("Jan 2"))))

	// Create the timeline line
	timelineWidth := tg.width - 10 // leave space for margins
	timeline := tg.renderTimelineLine(timelineWidth)
	output.WriteString(timeline + "\n")

	// Add date labels
	dateLabels := tg.renderDateLabels(timelineWidth)
	output.WriteString(dateLabels + "\n")

	// Add change summary
	if len(tg.changeEvents) > 0 || len(tg.snapshots) > 0 {
		output.WriteString("\n")
		output.WriteString(headerColor.Sprint("Changes:") + "\n")
		summary := tg.renderChangeSummary()
		output.WriteString(summary)
	}

	return output.String()
}

// renderTimelineLine creates the beautiful ────•──────• timeline
func (tg *TimelineGraph) renderTimelineLine(width int) string {
	line := make([]rune, width)

	// Fill with horizontal line characters
	for i := range line {
		line[i] = '─'
	}

	// Place markers for snapshots
	for _, snapshot := range tg.snapshots {
		pos := tg.calculatePosition(snapshot.Timestamp, width)
		if pos >= 0 && pos < width {
			line[pos] = '•'
		}
	}

	// Place different markers for change events if no snapshots at that position
	for _, event := range tg.changeEvents {
		pos := tg.calculatePosition(event.Timestamp, width)
		if pos >= 0 && pos < width && line[pos] == '─' {
			// Use different marker based on change magnitude
			if event.ChangeCount > 10 {
				line[pos] = '●' // major change
			} else if event.ChangeCount > 5 {
				line[pos] = '◉' // medium change
			} else {
				line[pos] = '○' // minor change
			}
		}
	}

	return string(line)
}

// renderDateLabels creates date labels under the timeline
func (tg *TimelineGraph) renderDateLabels(width int) string {
	labels := make([]rune, width)
	for i := range labels {
		labels[i] = ' '
	}

	// Only show labels at the start, end, and where snapshots exist
	labelPositions := []labelPosition{}

	// Always show start and end
	labelPositions = append(labelPositions, labelPosition{position: 0, priority: 100})
	labelPositions = append(labelPositions, labelPosition{position: width - 1, priority: 100})

	// Add positions for each unique day with snapshots
	dayPositions := make(map[string]int)
	for _, snapshot := range tg.snapshots {
		day := snapshot.Timestamp.Format("Jan2")
		pos := tg.calculatePosition(snapshot.Timestamp, width)
		if pos >= 0 && pos < width {
			if existingPos, exists := dayPositions[day]; !exists || pos < existingPos {
				dayPositions[day] = pos
			}
		}
	}

	// Add unique day positions
	for _, pos := range dayPositions {
		if pos > 5 && pos < width-5 { // Avoid overlapping with start/end
			labelPositions = append(labelPositions, labelPosition{position: pos, priority: 50})
		}
	}

	// Place labels ensuring no overlap
	placedLabels := make(map[int]bool)
	for _, pos := range labelPositions {
		timestamp := tg.calculateTimestamp(pos.position, width)
		label := timestamp.Format("Jan2")

		// Find a position that doesn't overlap
		start := pos.position - len(label)/2
		if start < 0 {
			start = 0
		}
		if start+len(label) > width {
			start = width - len(label)
		}

		// Check for overlap
		canPlace := true
		for i := start; i < start+len(label)+1 && i < width; i++ {
			if placedLabels[i] {
				canPlace = false
				break
			}
		}

		if canPlace {
			for i, ch := range label {
				if start+i < width {
					labels[start+i] = ch
					placedLabels[start+i] = true
				}
			}
		}
	}

	return string(labels)
}

// renderChangeSummary creates the change details section
func (tg *TimelineGraph) renderChangeSummary() string {
	var output strings.Builder

	// Color definitions
	majorChangeColor := color.New(color.FgRed, color.Bold)
	mediumChangeColor := color.New(color.FgYellow)
	minorChangeColor := color.New(color.FgGreen)
	snapshotColor := color.New(color.FgCyan)
	dateColor := color.New(color.FgWhite, color.Bold)
	providerColors := map[string]*color.Color{
		"terraform":  color.New(color.FgMagenta),
		"aws":        color.New(color.FgYellow),
		"gcp":        color.New(color.FgBlue),
		"kubernetes": color.New(color.FgCyan),
	}

	// Combine snapshots and change events
	type timelineItem struct {
		timestamp   time.Time
		description string
		changeCount int
		provider    string
		isSnapshot  bool
	}

	var items []timelineItem

	// Add snapshots
	for _, snapshot := range tg.snapshots {
		items = append(items, timelineItem{
			timestamp:   snapshot.Timestamp,
			description: fmt.Sprintf("%s snapshot", snapshot.Provider),
			changeCount: snapshot.ResourceCount,
			provider:    snapshot.Provider,
			isSnapshot:  true,
		})
	}

	// Add change events
	for _, event := range tg.changeEvents {
		items = append(items, timelineItem{
			timestamp:   event.Timestamp,
			description: event.Description,
			changeCount: event.ChangeCount,
			provider:    event.Provider,
			isSnapshot:  false,
		})
	}

	// Sort by timestamp
	sort.Slice(items, func(i, j int) bool {
		return items[i].timestamp.Before(items[j].timestamp)
	})

	// Format items
	for _, item := range items {
		var marker string
		var changeColor *color.Color

		if item.isSnapshot {
			marker = snapshotColor.Sprint("•")
			changeColor = snapshotColor
		} else {
			if item.changeCount > 10 {
				marker = majorChangeColor.Sprint("●")
				changeColor = majorChangeColor
			} else if item.changeCount > 5 {
				marker = mediumChangeColor.Sprint("◉")
				changeColor = mediumChangeColor
			} else {
				marker = minorChangeColor.Sprint("○")
				changeColor = minorChangeColor
			}
		}

		// Format date
		dateStr := dateColor.Sprintf("%s:", item.timestamp.Format("Jan 2"))

		// Get provider color
		providerColor := providerColors[item.provider]
		if providerColor == nil {
			providerColor = color.New(color.FgWhite)
		}

		// Format the line
		output.WriteString(fmt.Sprintf("%s %s ", marker, dateStr))

		if item.isSnapshot {
			output.WriteString(changeColor.Sprintf("%d resources ", item.changeCount))
			output.WriteString(fmt.Sprintf("(%s)\n", providerColor.Sprint(item.description)))
		} else {
			output.WriteString(changeColor.Sprintf("%d changes ", item.changeCount))
			output.WriteString(fmt.Sprintf("(%s)\n", item.description))
		}
	}

	return output.String()
}

// calculatePosition calculates the position on the timeline for a given timestamp
func (tg *TimelineGraph) calculatePosition(timestamp time.Time, width int) int {
	if timestamp.Before(tg.startTime) || timestamp.After(tg.endTime) {
		return -1
	}

	elapsed := timestamp.Sub(tg.startTime)
	total := tg.endTime.Sub(tg.startTime)

	position := float64(elapsed) / float64(total) * float64(width-1)
	return int(math.Round(position))
}

// calculateTimestamp calculates the timestamp for a given position
func (tg *TimelineGraph) calculateTimestamp(position, width int) time.Time {
	ratio := float64(position) / float64(width-1)
	duration := tg.endTime.Sub(tg.startTime)
	offset := time.Duration(float64(duration) * ratio)
	return tg.startTime.Add(offset)
}

// labelPosition represents a label position on the timeline
type labelPosition struct {
	position int
	priority int
}

// calculateLabelPositions determines optimal label positions
func (tg *TimelineGraph) calculateLabelPositions(maxLabels, width int) []labelPosition {
	var positions []labelPosition

	// Always include start and end
	positions = append(positions, labelPosition{position: 0, priority: 100})
	positions = append(positions, labelPosition{position: width - 1, priority: 100})

	// Add positions for significant events
	for _, snapshot := range tg.snapshots {
		pos := tg.calculatePosition(snapshot.Timestamp, width)
		if pos > 0 && pos < width-1 {
			positions = append(positions, labelPosition{position: pos, priority: 50})
		}
	}

	// Add evenly spaced positions if needed
	if len(positions) < maxLabels {
		interval := width / (maxLabels - 1)
		for i := 1; i < maxLabels-1; i++ {
			pos := i * interval
			positions = append(positions, labelPosition{position: pos, priority: 10})
		}
	}

	// Sort by position and deduplicate
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].position < positions[j].position
	})

	// Remove positions that are too close together
	minDistance := width / maxLabels / 2
	filtered := []labelPosition{positions[0]}
	for i := 1; i < len(positions); i++ {
		if positions[i].position-filtered[len(filtered)-1].position >= minDistance {
			filtered = append(filtered, positions[i])
		}
	}

	// Limit to maxLabels
	if len(filtered) > maxLabels {
		// Sort by priority and take top maxLabels
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].priority > filtered[j].priority
		})
		filtered = filtered[:maxLabels]

		// Re-sort by position
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].position < filtered[j].position
		})
	}

	return filtered
}

// CreateSimpleTimeline creates a simple timeline for a date range
func CreateSimpleTimeline(snapshots []storage.SnapshotInfo, width int) string {
	graph := NewTimelineGraph(width)
	graph.SetSnapshots(snapshots)
	return graph.Render()
}

// CreateChangeTimeline creates a timeline showing both snapshots and changes
func CreateChangeTimeline(snapshots []storage.SnapshotInfo, events []ChangeEvent, width int) string {
	graph := NewTimelineGraph(width)
	graph.SetSnapshots(snapshots)
	graph.SetChangeEvents(events)
	return graph.Render()
}
