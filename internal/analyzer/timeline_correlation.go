package analyzer

import (
	"fmt"
	"sort"
	"time"

	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/types"
)

// TimelineEvent represents a significant event in the infrastructure timeline
type TimelineEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	Type         string                 `json:"type"`     // "resource_change", "deployment", "scaling", "failure"
	Severity     string                 `json:"severity"` // "info", "warning", "critical"
	Provider     string                 `json:"provider"`
	Resource     string                 `json:"resource"`
	Description  string                 `json:"description"`
	Context      map[string]interface{} `json:"context"`
	Correlations []string               `json:"correlations"` // IDs of related events
}

// TimelineTrend represents a trend in resource changes over time
type TimelineTrend struct {
	Provider     string                 `json:"provider"`
	ResourceType string                 `json:"resource_type"`
	Trend        string                 `json:"trend"`      // "increasing", "decreasing", "stable", "volatile"
	Confidence   float64                `json:"confidence"` // 0.0 to 1.0
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	DataPoints   []TrendDataPoint       `json:"data_points"`
	Predictions  []TrendPrediction      `json:"predictions"`
	Context      map[string]interface{} `json:"context"`
}

// TrendDataPoint represents a single measurement in a trend
type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int       `json:"value"`   // resource count
	Change    int       `json:"change"`  // change from previous point
	Context   string    `json:"context"` // description of what caused this change
}

// TrendPrediction represents a predicted future trend
type TrendPrediction struct {
	FutureTime     time.Time `json:"future_time"`
	PredictedValue int       `json:"predicted_value"`
	Confidence     float64   `json:"confidence"`
	Reasoning      string    `json:"reasoning"`
}

// CorrelationAnalysis represents correlation between different providers/resources
type CorrelationAnalysis struct {
	Provider1     string                 `json:"provider1"`
	Provider2     string                 `json:"provider2"`
	ResourceType1 string                 `json:"resource_type1"`
	ResourceType2 string                 `json:"resource_type2"`
	Correlation   float64                `json:"correlation"`  // -1.0 to 1.0
	Confidence    float64                `json:"confidence"`   // 0.0 to 1.0
	Relationship  string                 `json:"relationship"` // "positive", "negative", "none"
	Examples      []CorrelationExample   `json:"examples"`
	Context       map[string]interface{} `json:"context"`
}

// CorrelationExample represents a specific example of correlation
type CorrelationExample struct {
	Timestamp   time.Time     `json:"timestamp"`
	Event1      string        `json:"event1"`
	Event2      string        `json:"event2"`
	TimeDiff    time.Duration `json:"time_diff"`
	Description string        `json:"description"`
}

// TimelineAnalyzer provides advanced timeline analysis capabilities
type TimelineAnalyzer struct {
	snapshots    []storage.SnapshotInfo
	events       []TimelineEvent
	trends       []TimelineTrend
	correlations []CorrelationAnalysis
}

// NewTimelineAnalyzer creates a new timeline analyzer
func NewTimelineAnalyzer(snapshots []storage.SnapshotInfo) *TimelineAnalyzer {
	return &TimelineAnalyzer{
		snapshots:    snapshots,
		events:       make([]TimelineEvent, 0),
		trends:       make([]TimelineTrend, 0),
		correlations: make([]CorrelationAnalysis, 0),
	}
}

// AnalyzeTimeline performs comprehensive timeline analysis
func (ta *TimelineAnalyzer) AnalyzeTimeline() error {
	// Sort snapshots by timestamp
	sort.Slice(ta.snapshots, func(i, j int) bool {
		return ta.snapshots[i].Timestamp.Before(ta.snapshots[j].Timestamp)
	})

	// Detect events
	if err := ta.detectEvents(); err != nil {
		return err
	}

	// Analyze trends
	if err := ta.analyzeTrends(); err != nil {
		return err
	}

	// Find correlations
	if err := ta.findCorrelations(); err != nil {
		return err
	}

	return nil
}

// detectEvents identifies significant events in the timeline
func (ta *TimelineAnalyzer) detectEvents() error {
	resourceCounts := make(map[string]map[string]int) // provider -> resource_type -> count

	for i, snapshot := range ta.snapshots {
		// Initialize if first snapshot
		if i == 0 {
			resourceCounts[snapshot.Provider] = make(map[string]int)
			continue
		}

		prevSnapshot := ta.snapshots[i-1]

		// Resource count change
		countDiff := 0
		if snapshot.Provider == prevSnapshot.Provider {
			countDiff = snapshot.ResourceCount - prevSnapshot.ResourceCount

			var eventType, severity string
			if countDiff > 0 {
				eventType = "resource_addition"
				if countDiff > 10 {
					severity = "warning"
				} else {
					severity = "info"
				}
			} else if countDiff < 0 {
				eventType = "resource_removal"
				if countDiff < -5 {
					severity = "critical"
				} else {
					severity = "warning"
				}
			}

			if countDiff != 0 {
				event := TimelineEvent{
					Timestamp:   snapshot.Timestamp,
					Type:        eventType,
					Severity:    severity,
					Provider:    snapshot.Provider,
					Resource:    "total_resources",
					Description: ta.generateEventDescription(eventType, countDiff, snapshot.Provider),
					Context: map[string]interface{}{
						"count_change":   countDiff,
						"previous_count": prevSnapshot.ResourceCount,
						"current_count":  snapshot.ResourceCount,
						"time_diff":      snapshot.Timestamp.Sub(prevSnapshot.Timestamp),
					},
				}
				ta.events = append(ta.events, event)
			}
		}

		// Detect deployment patterns
		timeDiff := snapshot.Timestamp.Sub(prevSnapshot.Timestamp)
		if timeDiff < time.Hour && snapshot.ResourceCount > prevSnapshot.ResourceCount {
			event := TimelineEvent{
				Timestamp:   snapshot.Timestamp,
				Type:        "deployment",
				Severity:    "info",
				Provider:    snapshot.Provider,
				Resource:    "deployment_activity",
				Description: "Rapid resource deployment detected",
				Context: map[string]interface{}{
					"deployment_speed": timeDiff,
					"resources_added":  snapshot.ResourceCount - prevSnapshot.ResourceCount,
				},
			}
			ta.events = append(ta.events, event)
		}

		// Detect large-scale changes (potential incidents)
		if snapshot.Provider == prevSnapshot.Provider && abs(float64(countDiff)) > float64(prevSnapshot.ResourceCount)*0.2 {
			event := TimelineEvent{
				Timestamp: snapshot.Timestamp,
				Type:      "infrastructure_change",
				Severity:  "critical",
				Provider:  snapshot.Provider,
				Resource:  "infrastructure_scale",
				Description: fmt.Sprintf("Large-scale infrastructure change: %+d resources (%.1f%% change)",
					countDiff, float64(countDiff)/float64(prevSnapshot.ResourceCount)*100),
				Context: map[string]interface{}{
					"count_change":      countDiff,
					"percentage_change": float64(countDiff) / float64(prevSnapshot.ResourceCount) * 100,
					"scale":             "large",
				},
			}
			ta.events = append(ta.events, event)
		}

		// Detect sustained growth/decline patterns
		if i >= 2 {
			prevPrevSnapshot := ta.snapshots[i-2]
			if snapshot.Provider == prevSnapshot.Provider && prevSnapshot.Provider == prevPrevSnapshot.Provider {
				trend1 := snapshot.ResourceCount - prevSnapshot.ResourceCount
				trend2 := prevSnapshot.ResourceCount - prevPrevSnapshot.ResourceCount

				// Check for sustained growth (3 consecutive increases)
				if trend1 > 0 && trend2 > 0 {
					event := TimelineEvent{
						Timestamp:   snapshot.Timestamp,
						Type:        "sustained_growth",
						Severity:    "info",
						Provider:    snapshot.Provider,
						Resource:    "growth_pattern",
						Description: "Sustained infrastructure growth detected over multiple scans",
						Context: map[string]interface{}{
							"total_growth": (snapshot.ResourceCount - prevPrevSnapshot.ResourceCount),
							"periods":      3,
						},
					}
					ta.events = append(ta.events, event)
				}

				// Check for sustained decline (3 consecutive decreases)
				if trend1 < 0 && trend2 < 0 {
					event := TimelineEvent{
						Timestamp:   snapshot.Timestamp,
						Type:        "sustained_decline",
						Severity:    "warning",
						Provider:    snapshot.Provider,
						Resource:    "decline_pattern",
						Description: "Sustained infrastructure decline detected over multiple scans",
						Context: map[string]interface{}{
							"total_decline": (prevPrevSnapshot.ResourceCount - snapshot.ResourceCount),
							"periods":       3,
						},
					}
					ta.events = append(ta.events, event)
				}
			}
		}
	}

	return nil
}

// analyzeTrends identifies trends in resource changes over time
func (ta *TimelineAnalyzer) analyzeTrends() error {
	providerTrends := make(map[string][]TrendDataPoint)

	// Group snapshots by provider
	for _, snapshot := range ta.snapshots {
		if providerTrends[snapshot.Provider] == nil {
			providerTrends[snapshot.Provider] = make([]TrendDataPoint, 0)
		}

		var change int
		if len(providerTrends[snapshot.Provider]) > 0 {
			lastPoint := providerTrends[snapshot.Provider][len(providerTrends[snapshot.Provider])-1]
			change = snapshot.ResourceCount - lastPoint.Value
		}

		dataPoint := TrendDataPoint{
			Timestamp: snapshot.Timestamp,
			Value:     snapshot.ResourceCount,
			Change:    change,
			Context:   ta.generateTrendContext(change),
		}
		providerTrends[snapshot.Provider] = append(providerTrends[snapshot.Provider], dataPoint)
	}

	// Analyze trends for each provider
	for provider, dataPoints := range providerTrends {
		if len(dataPoints) < 3 {
			continue // Need at least 3 points to detect a trend
		}

		trend := ta.calculateTrend(dataPoints)
		trendAnalysis := TimelineTrend{
			Provider:     provider,
			ResourceType: "total_resources",
			Trend:        trend.Type,
			Confidence:   trend.Confidence,
			StartTime:    dataPoints[0].Timestamp,
			EndTime:      dataPoints[len(dataPoints)-1].Timestamp,
			DataPoints:   dataPoints,
			Predictions:  ta.generatePredictions(dataPoints, trend),
			Context: map[string]interface{}{
				"data_points_count": len(dataPoints),
				"time_span":         dataPoints[len(dataPoints)-1].Timestamp.Sub(dataPoints[0].Timestamp),
				"total_change":      dataPoints[len(dataPoints)-1].Value - dataPoints[0].Value,
			},
		}
		ta.trends = append(ta.trends, trendAnalysis)
	}

	return nil
}

// findCorrelations identifies correlations between different providers/resources
func (ta *TimelineAnalyzer) findCorrelations() error {
	// Group data by provider
	providerData := make(map[string][]int)
	timestamps := make([]time.Time, 0)

	// Build time-aligned data
	for _, snapshot := range ta.snapshots {
		if len(timestamps) == 0 || !snapshot.Timestamp.Equal(timestamps[len(timestamps)-1]) {
			timestamps = append(timestamps, snapshot.Timestamp)
		}

		if providerData[snapshot.Provider] == nil {
			providerData[snapshot.Provider] = make([]int, 0)
		}

		// Align data by timestamp
		for len(providerData[snapshot.Provider]) < len(timestamps) {
			if len(providerData[snapshot.Provider]) == len(timestamps)-1 {
				providerData[snapshot.Provider] = append(providerData[snapshot.Provider], snapshot.ResourceCount)
			} else {
				providerData[snapshot.Provider] = append(providerData[snapshot.Provider], 0)
			}
		}
	}

	// Calculate correlations between providers
	providers := make([]string, 0, len(providerData))
	for provider := range providerData {
		providers = append(providers, provider)
	}

	for i := 0; i < len(providers); i++ {
		for j := i + 1; j < len(providers); j++ {
			provider1, provider2 := providers[i], providers[j]
			correlation := ta.calculateCorrelation(providerData[provider1], providerData[provider2])

			if abs(correlation.Value) > 0.3 { // Only include meaningful correlations
				analysis := CorrelationAnalysis{
					Provider1:     provider1,
					Provider2:     provider2,
					ResourceType1: "total_resources",
					ResourceType2: "total_resources",
					Correlation:   correlation.Value,
					Confidence:    correlation.Confidence,
					Relationship:  ta.classifyRelationship(correlation.Value),
					Examples:      ta.findCorrelationExamples(provider1, provider2, timestamps),
					Context: map[string]interface{}{
						"sample_size": len(providerData[provider1]),
						"time_span":   timestamps[len(timestamps)-1].Sub(timestamps[0]),
					},
				}
				ta.correlations = append(ta.correlations, analysis)
			}
		}
	}

	return nil
}

// Helper types and functions

type TrendResult struct {
	Type       string // "increasing", "decreasing", "stable", "volatile"
	Confidence float64
	Slope      float64
	Variance   float64
}

type CorrelationValue struct {
	Value      float64
	Confidence float64
}

func (ta *TimelineAnalyzer) calculateTrend(dataPoints []TrendDataPoint) TrendResult {
	if len(dataPoints) < 2 {
		return TrendResult{Type: "stable", Confidence: 0.0}
	}

	// Calculate slope using linear regression
	n := float64(len(dataPoints))
	var sumX, sumY, sumXY, sumX2 float64

	for i, point := range dataPoints {
		x := float64(i)
		y := float64(point.Value)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Calculate variance to determine volatility
	mean := sumY / n
	var variance float64
	for _, point := range dataPoints {
		diff := float64(point.Value) - mean
		variance += diff * diff
	}
	variance /= n

	// Classify trend
	var trendType string
	var confidence float64

	if abs(slope) < 0.1 {
		trendType = "stable"
		confidence = 1.0 - (variance / (mean * mean))
	} else if slope > 0 {
		trendType = "increasing"
		confidence = min(abs(slope)/mean, 1.0)
	} else {
		trendType = "decreasing"
		confidence = min(abs(slope)/mean, 1.0)
	}

	// Check for volatility
	if variance > mean*mean*0.25 {
		trendType = "volatile"
		confidence = variance / (mean * mean)
	}

	return TrendResult{
		Type:       trendType,
		Confidence: max(0.0, min(1.0, confidence)),
		Slope:      slope,
		Variance:   variance,
	}
}

func (ta *TimelineAnalyzer) calculateCorrelation(data1, data2 []int) CorrelationValue {
	if len(data1) != len(data2) || len(data1) < 2 {
		return CorrelationValue{Value: 0.0, Confidence: 0.0}
	}

	n := float64(len(data1))
	var sum1, sum2, sum1_2, sum2_2, sumProduct float64

	for i := 0; i < len(data1); i++ {
		x, y := float64(data1[i]), float64(data2[i])
		sum1 += x
		sum2 += y
		sum1_2 += x * x
		sum2_2 += y * y
		sumProduct += x * y
	}

	numerator := n*sumProduct - sum1*sum2
	denominator1 := n*sum1_2 - sum1*sum1
	denominator2 := n*sum2_2 - sum2*sum2

	if denominator1 <= 0 || denominator2 <= 0 {
		return CorrelationValue{Value: 0.0, Confidence: 0.0}
	}

	correlation := numerator / (sqrt(denominator1) * sqrt(denominator2))
	confidence := min(n/10.0, 1.0) // Confidence increases with sample size

	return CorrelationValue{
		Value:      correlation,
		Confidence: confidence,
	}
}

func (ta *TimelineAnalyzer) generatePredictions(dataPoints []TrendDataPoint, trend TrendResult) []TrendPrediction {
	if len(dataPoints) < 2 {
		return []TrendPrediction{}
	}

	predictions := make([]TrendPrediction, 0)
	lastPoint := dataPoints[len(dataPoints)-1]

	// Predict next 3 time periods
	for i := 1; i <= 3; i++ {
		// Simple linear prediction based on slope
		predictedValue := int(float64(lastPoint.Value) + trend.Slope*float64(i))

		// Adjust confidence based on prediction distance
		confidence := trend.Confidence * (1.0 - float64(i)*0.1)

		// Estimate future time (assume regular intervals)
		var avgInterval time.Duration
		if len(dataPoints) > 1 {
			totalDuration := lastPoint.Timestamp.Sub(dataPoints[0].Timestamp)
			avgInterval = totalDuration / time.Duration(len(dataPoints)-1)
		} else {
			avgInterval = time.Hour * 24 // Default to daily
		}

		futureTime := lastPoint.Timestamp.Add(avgInterval * time.Duration(i))

		prediction := TrendPrediction{
			FutureTime:     futureTime,
			PredictedValue: max(0, predictedValue), // Can't have negative resources
			Confidence:     max(0.0, confidence),
			Reasoning:      ta.generatePredictionReasoning(trend, i),
		}
		predictions = append(predictions, prediction)
	}

	return predictions
}

func (ta *TimelineAnalyzer) generateEventDescription(eventType string, countDiff int, provider string) string {
	switch eventType {
	case "resource_addition":
		if countDiff > 10 {
			return fmt.Sprintf("Large-scale resource deployment: %d new %s resources added", countDiff, provider)
		}
		return fmt.Sprintf("Resource growth: %d new %s resources added", countDiff, provider)
	case "resource_removal":
		if countDiff < -5 {
			return fmt.Sprintf("Significant resource cleanup: %d %s resources removed", -countDiff, provider)
		}
		return fmt.Sprintf("Resource cleanup: %d %s resources removed", -countDiff, provider)
	default:
		return fmt.Sprintf("Resource change detected in %s", provider)
	}
}

func (ta *TimelineAnalyzer) generateTrendContext(change int) string {
	if change > 0 {
		return "Resource growth"
	} else if change < 0 {
		return "Resource reduction"
	}
	return "No change"
}

func (ta *TimelineAnalyzer) classifyRelationship(correlation float64) string {
	if correlation > 0.3 {
		return "positive"
	} else if correlation < -0.3 {
		return "negative"
	}
	return "none"
}

func (ta *TimelineAnalyzer) findCorrelationExamples(provider1, provider2 string, timestamps []time.Time) []CorrelationExample {
	examples := make([]CorrelationExample, 0)

	// Find events that happened close together for both providers
	provider1Events := make([]TimelineEvent, 0)
	provider2Events := make([]TimelineEvent, 0)

	for _, event := range ta.events {
		if event.Provider == provider1 {
			provider1Events = append(provider1Events, event)
		} else if event.Provider == provider2 {
			provider2Events = append(provider2Events, event)
		}
	}

	// Look for events within 1 hour of each other
	correlationWindow := time.Hour

	for _, event1 := range provider1Events {
		for _, event2 := range provider2Events {
			timeDiff := event2.Timestamp.Sub(event1.Timestamp)
			absTimeDiff := timeDiff
			if absTimeDiff < 0 {
				absTimeDiff = -absTimeDiff
			}

			if absTimeDiff <= correlationWindow {
				example := CorrelationExample{
					Timestamp:   event1.Timestamp,
					Event1:      fmt.Sprintf("%s: %s", event1.Provider, event1.Description),
					Event2:      fmt.Sprintf("%s: %s", event2.Provider, event2.Description),
					TimeDiff:    timeDiff,
					Description: ta.generateCorrelationDescription(event1, event2, timeDiff),
				}
				examples = append(examples, example)

				// Limit to 3 examples to avoid overwhelming output
				if len(examples) >= 3 {
					return examples
				}
			}
		}
	}

	return examples
}

func (ta *TimelineAnalyzer) generatePredictionReasoning(trend TrendResult, periodAhead int) string {
	base := fmt.Sprintf("Based on %s trend", trend.Type)
	if periodAhead == 1 {
		return base + " (short-term prediction)"
	} else if periodAhead == 2 {
		return base + " (medium-term prediction)"
	}
	return base + " (long-term prediction with lower confidence)"
}

// Getter methods
func (ta *TimelineAnalyzer) GetEvents() []TimelineEvent {
	return ta.events
}

func (ta *TimelineAnalyzer) GetTrends() []TimelineTrend {
	return ta.trends
}

func (ta *TimelineAnalyzer) GetCorrelations() []CorrelationAnalysis {
	return ta.correlations
}

// Helper math functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Simple Newton's method for square root
	guess := x / 2
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}

// generateCorrelationDescription creates a human-readable description of the correlation
func (ta *TimelineAnalyzer) generateCorrelationDescription(event1, event2 TimelineEvent, timeDiff time.Duration) string {
	direction := "after"
	if timeDiff < 0 {
		direction = "before"
		timeDiff = -timeDiff
	}

	if timeDiff < time.Minute {
		return fmt.Sprintf("%s change occurred %s seconds %s %s change",
			event2.Provider, formatDuration(timeDiff), direction, event1.Provider)
	} else if timeDiff < time.Hour {
		return fmt.Sprintf("%s change occurred %d minutes %s %s change",
			event2.Provider, int(timeDiff.Minutes()), direction, event1.Provider)
	} else {
		return fmt.Sprintf("%s change occurred %d hours %s %s change",
			event2.Provider, int(timeDiff.Hours()), direction, event1.Provider)
	}
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0f", d.Seconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.0f", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0f", d.Minutes())
	} else {
		return fmt.Sprintf("%.1f", d.Hours())
	}
}
