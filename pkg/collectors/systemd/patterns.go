package systemd

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// RestartPattern tracks and analyzes service restart patterns
type RestartPattern struct {
	Restarts      []time.Time
	Pattern       string  // "stable", "flapping", "recovering", "degrading"
	Frequency     float64 // Restarts per hour
	Trend         string  // "increasing", "decreasing", "stable"
	NextPredicted *time.Time
	Confidence    float64
}

// PatternAnalysis contains detailed pattern analysis results
type PatternAnalysis struct {
	Pattern            string
	Confidence         float64
	RestartCount       int
	TimeSpan           time.Duration
	AverageInterval    time.Duration
	ShortestInterval   time.Duration
	LongestInterval    time.Duration
	StandardDeviation  time.Duration
	Trend              TrendInfo
	Predictions        []PredictedRestart
	AnomalousRestarts  []AnomalousRestart
	RecommendedActions []string
}

// TrendInfo contains trend analysis information
type TrendInfo struct {
	Direction  string  // "increasing", "decreasing", "stable", "volatile"
	Slope      float64 // Rate of change in restart frequency
	Confidence float64
	Projection string // Human-readable projection
}

// PredictedRestart represents a predicted future restart
type PredictedRestart struct {
	Time       time.Time
	Confidence float64
	Reason     string
}

// AnomalousRestart represents an unusual restart event
type AnomalousRestart struct {
	Time      time.Time
	Deviation float64 // Standard deviations from mean
	Type      string  // "too_soon", "too_late", "burst"
	Context   string
}

// RestartBurst represents a cluster of restarts
type RestartBurst struct {
	StartTime    time.Time
	EndTime      time.Time
	RestartCount int
	Duration     time.Duration
	Severity     string // "minor", "moderate", "severe"
}

// NewRestartPattern creates a new restart pattern tracker
func NewRestartPattern() *RestartPattern {
	return &RestartPattern{
		Restarts:   make([]time.Time, 0),
		Pattern:    "stable",
		Frequency:  0,
		Trend:      "stable",
		Confidence: 1.0,
	}
}

// RecordRestart records a new restart event
func (rp *RestartPattern) RecordRestart(timestamp time.Time) {
	rp.Restarts = append(rp.Restarts, timestamp)

	// Keep only last 100 restarts for analysis
	if len(rp.Restarts) > 100 {
		rp.Restarts = rp.Restarts[len(rp.Restarts)-100:]
	}

	// Update analysis
	rp.analyze()
}

// analyze performs pattern analysis on restart history
func (rp *RestartPattern) analyze() {
	if len(rp.Restarts) < 2 {
		rp.Pattern = "stable"
		rp.Frequency = 0
		rp.Confidence = 1.0
		return
	}

	// Sort restarts
	sort.Slice(rp.Restarts, func(i, j int) bool {
		return rp.Restarts[i].Before(rp.Restarts[j])
	})

	// Calculate frequency
	timeSpan := rp.Restarts[len(rp.Restarts)-1].Sub(rp.Restarts[0])
	if timeSpan > 0 {
		rp.Frequency = float64(len(rp.Restarts)-1) / timeSpan.Hours()
	}

	// Detect pattern
	rp.detectPattern()

	// Analyze trend
	rp.analyzeTrend()

	// Predict next restart
	rp.predictNext()
}

// detectPattern identifies the restart pattern
func (rp *RestartPattern) detectPattern() {
	if len(rp.Restarts) < 3 {
		rp.Pattern = "stable"
		return
	}

	intervals := rp.calculateIntervals()
	if len(intervals) == 0 {
		return
	}

	// Calculate statistics
	mean, stdDev := rp.calculateStats(intervals)
	cv := stdDev / mean // Coefficient of variation

	// Detect bursts
	bursts := rp.detectBursts()

	// Classify pattern
	if rp.Frequency > 10 { // More than 10 restarts per hour
		rp.Pattern = "flapping"
		rp.Confidence = 0.9
	} else if len(bursts) > 0 && float64(len(bursts))/float64(len(rp.Restarts)) > 0.3 {
		rp.Pattern = "burst"
		rp.Confidence = 0.8
	} else if cv < 0.5 {
		// Low variation - regular pattern
		if rp.Frequency < 1 {
			rp.Pattern = "stable"
			rp.Confidence = 0.9
		} else {
			rp.Pattern = "periodic"
			rp.Confidence = 0.8
		}
	} else if cv > 1.5 {
		rp.Pattern = "erratic"
		rp.Confidence = 0.7
	} else {
		// Check if recovering or degrading
		recentFreq := rp.calculateRecentFrequency()
		overallFreq := rp.Frequency

		if recentFreq < overallFreq*0.5 {
			rp.Pattern = "recovering"
			rp.Confidence = 0.8
		} else if recentFreq > overallFreq*1.5 {
			rp.Pattern = "degrading"
			rp.Confidence = 0.8
		} else {
			rp.Pattern = "unstable"
			rp.Confidence = 0.6
		}
	}
}

// analyzeTrend analyzes the restart frequency trend
func (rp *RestartPattern) analyzeTrend() {
	if len(rp.Restarts) < 5 {
		rp.Trend = "stable"
		return
	}

	// Split into time windows
	windows := rp.createTimeWindows(5)
	if len(windows) < 2 {
		return
	}

	// Calculate restart rates for each window
	rates := make([]float64, len(windows))
	for i, window := range windows {
		if window.Duration > 0 {
			rates[i] = float64(window.Count) / window.Duration.Hours()
		}
	}

	// Perform linear regression on rates
	slope, _ := rp.linearRegression(rates)

	// Classify trend
	meanRate := rp.Frequency
	relativeSlope := slope / meanRate

	if math.Abs(relativeSlope) < 0.1 {
		rp.Trend = "stable"
	} else if relativeSlope > 0.3 {
		rp.Trend = "increasing"
	} else if relativeSlope < -0.3 {
		rp.Trend = "decreasing"
	} else {
		rp.Trend = "volatile"
	}
}

// predictNext predicts the next restart time
func (rp *RestartPattern) predictNext() {
	if len(rp.Restarts) < 3 {
		rp.NextPredicted = nil
		return
	}

	intervals := rp.calculateIntervals()
	if len(intervals) == 0 {
		return
	}

	lastRestart := rp.Restarts[len(rp.Restarts)-1]

	switch rp.Pattern {
	case "periodic", "stable":
		// Use average interval
		mean, _ := rp.calculateStats(intervals)
		predicted := lastRestart.Add(time.Duration(mean))
		rp.NextPredicted = &predicted

	case "flapping":
		// Very short interval expected
		predicted := lastRestart.Add(time.Minute)
		rp.NextPredicted = &predicted

	case "degrading":
		// Use recent trend to predict
		recentIntervals := intervals
		if len(intervals) > 5 {
			recentIntervals = intervals[len(intervals)-5:]
		}
		mean, _ := rp.calculateStats(recentIntervals)
		predicted := lastRestart.Add(time.Duration(mean))
		rp.NextPredicted = &predicted

	default:
		// Less predictable patterns
		rp.NextPredicted = nil
	}
}

// GetAnalysis returns detailed pattern analysis
func (rp *RestartPattern) GetAnalysis() *PatternAnalysis {
	analysis := &PatternAnalysis{
		Pattern:            rp.Pattern,
		Confidence:         rp.Confidence,
		RestartCount:       len(rp.Restarts),
		Predictions:        make([]PredictedRestart, 0),
		AnomalousRestarts:  make([]AnomalousRestart, 0),
		RecommendedActions: make([]string, 0),
	}

	if len(rp.Restarts) < 2 {
		analysis.TimeSpan = 0
		return analysis
	}

	// Calculate time span
	analysis.TimeSpan = rp.Restarts[len(rp.Restarts)-1].Sub(rp.Restarts[0])

	// Calculate intervals
	intervals := rp.calculateIntervals()
	if len(intervals) > 0 {
		mean, stdDev := rp.calculateStats(intervals)
		analysis.AverageInterval = time.Duration(mean)
		analysis.StandardDeviation = time.Duration(stdDev)

		// Find shortest and longest intervals
		shortest, longest := rp.findExtremes(intervals)
		analysis.ShortestInterval = time.Duration(shortest)
		analysis.LongestInterval = time.Duration(longest)

		// Detect anomalies
		analysis.AnomalousRestarts = rp.detectAnomalies(intervals, mean, stdDev)
	}

	// Add trend info
	analysis.Trend = rp.getTrendInfo()

	// Add predictions
	if rp.NextPredicted != nil {
		analysis.Predictions = append(analysis.Predictions, PredictedRestart{
			Time:       *rp.NextPredicted,
			Confidence: rp.Confidence,
			Reason:     fmt.Sprintf("Based on %s pattern", rp.Pattern),
		})
	}

	// Generate recommendations
	analysis.RecommendedActions = rp.generateRecommendations()

	return analysis
}

// Helper methods

func (rp *RestartPattern) calculateIntervals() []float64 {
	if len(rp.Restarts) < 2 {
		return []float64{}
	}

	intervals := make([]float64, len(rp.Restarts)-1)
	for i := 1; i < len(rp.Restarts); i++ {
		intervals[i-1] = float64(rp.Restarts[i].Sub(rp.Restarts[i-1]))
	}

	return intervals
}

func (rp *RestartPattern) calculateStats(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(values))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func (rp *RestartPattern) findExtremes(values []float64) (min, max float64) {
	if len(values) == 0 {
		return 0, 0
	}

	min, max = values[0], values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return min, max
}

func (rp *RestartPattern) detectBursts() []RestartBurst {
	bursts := make([]RestartBurst, 0)

	if len(rp.Restarts) < 3 {
		return bursts
	}

	// Define burst as 3+ restarts within 5 minutes
	burstWindow := 5 * time.Minute
	burstThreshold := 3

	i := 0
	for i < len(rp.Restarts) {
		j := i
		count := 1

		// Count restarts within window
		for j+1 < len(rp.Restarts) && rp.Restarts[j+1].Sub(rp.Restarts[i]) <= burstWindow {
			j++
			count++
		}

		if count >= burstThreshold {
			burst := RestartBurst{
				StartTime:    rp.Restarts[i],
				EndTime:      rp.Restarts[j],
				RestartCount: count,
				Duration:     rp.Restarts[j].Sub(rp.Restarts[i]),
			}

			// Classify severity
			if count >= 10 {
				burst.Severity = "severe"
			} else if count >= 5 {
				burst.Severity = "moderate"
			} else {
				burst.Severity = "minor"
			}

			bursts = append(bursts, burst)
			i = j + 1
		} else {
			i++
		}
	}

	return bursts
}

func (rp *RestartPattern) calculateRecentFrequency() float64 {
	if len(rp.Restarts) < 2 {
		return 0
	}

	// Calculate frequency for last 25% of restarts
	recentCount := len(rp.Restarts) / 4
	if recentCount < 2 {
		recentCount = 2
	}

	recentRestarts := rp.Restarts[len(rp.Restarts)-recentCount:]
	timeSpan := recentRestarts[len(recentRestarts)-1].Sub(recentRestarts[0])

	if timeSpan > 0 {
		return float64(len(recentRestarts)-1) / timeSpan.Hours()
	}

	return 0
}

type TimeWindow struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
	Count    int
}

func (rp *RestartPattern) createTimeWindows(count int) []TimeWindow {
	if len(rp.Restarts) < 2 || count < 1 {
		return []TimeWindow{}
	}

	totalSpan := rp.Restarts[len(rp.Restarts)-1].Sub(rp.Restarts[0])
	windowSize := totalSpan / time.Duration(count)

	windows := make([]TimeWindow, count)
	for i := 0; i < count; i++ {
		windows[i].Start = rp.Restarts[0].Add(time.Duration(i) * windowSize)
		windows[i].End = rp.Restarts[0].Add(time.Duration(i+1) * windowSize)
		windows[i].Duration = windowSize
	}

	// Count restarts in each window
	for _, restart := range rp.Restarts {
		for i := range windows {
			if restart.After(windows[i].Start) && restart.Before(windows[i].End) {
				windows[i].Count++
			}
		}
	}

	return windows
}

func (rp *RestartPattern) linearRegression(values []float64) (slope, intercept float64) {
	n := float64(len(values))
	if n < 2 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	return slope, intercept
}

func (rp *RestartPattern) detectAnomalies(intervals []float64, mean, stdDev float64) []AnomalousRestart {
	anomalies := make([]AnomalousRestart, 0)

	if stdDev == 0 || len(intervals) == 0 {
		return anomalies
	}

	for i, interval := range intervals {
		deviation := math.Abs(interval-mean) / stdDev

		if deviation > 2 { // More than 2 standard deviations
			anomaly := AnomalousRestart{
				Time:      rp.Restarts[i+1], // Interval is between i and i+1
				Deviation: deviation,
			}

			if interval < mean {
				anomaly.Type = "too_soon"
				anomaly.Context = fmt.Sprintf("Restarted %.1f times faster than average", mean/interval)
			} else {
				anomaly.Type = "too_late"
				anomaly.Context = fmt.Sprintf("Restarted %.1f times slower than average", interval/mean)
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	// Also check for burst anomalies
	bursts := rp.detectBursts()
	for _, burst := range bursts {
		if burst.Severity != "minor" {
			anomaly := AnomalousRestart{
				Time:      burst.StartTime,
				Deviation: float64(burst.RestartCount),
				Type:      "burst",
				Context:   fmt.Sprintf("%d restarts in %v", burst.RestartCount, burst.Duration),
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (rp *RestartPattern) getTrendInfo() TrendInfo {
	info := TrendInfo{
		Direction:  rp.Trend,
		Confidence: 0.7,
	}

	if len(rp.Restarts) < 5 {
		info.Confidence = 0.3
		info.Projection = "Insufficient data for trend analysis"
		return info
	}

	// Calculate slope of restart frequency over time
	windows := rp.createTimeWindows(5)
	rates := make([]float64, len(windows))
	for i, window := range windows {
		if window.Duration > 0 {
			rates[i] = float64(window.Count) / window.Duration.Hours()
		}
	}

	slope, _ := rp.linearRegression(rates)
	info.Slope = slope

	// Generate projection
	switch rp.Trend {
	case "increasing":
		info.Projection = fmt.Sprintf("Restart frequency increasing by %.1f per hour", math.Abs(slope))
		info.Confidence = 0.8
	case "decreasing":
		info.Projection = fmt.Sprintf("Restart frequency decreasing by %.1f per hour", math.Abs(slope))
		info.Confidence = 0.8
	case "stable":
		info.Projection = "Restart frequency remains stable"
		info.Confidence = 0.9
	default:
		info.Projection = "Restart pattern is unpredictable"
		info.Confidence = 0.5
	}

	return info
}

func (rp *RestartPattern) generateRecommendations() []string {
	recommendations := make([]string, 0)

	switch rp.Pattern {
	case "flapping":
		recommendations = append(recommendations,
			"Service is flapping - consider increasing restart delay",
			"Check service logs for immediate crash causes",
			"Review resource limits and dependencies")

	case "degrading":
		recommendations = append(recommendations,
			"Service stability is degrading - investigate recent changes",
			"Monitor system resources (CPU, memory, disk)",
			"Check for dependency service issues")

	case "burst":
		recommendations = append(recommendations,
			"Restart bursts detected - check for external triggers",
			"Review system logs during burst periods",
			"Consider implementing circuit breaker pattern")

	case "erratic":
		recommendations = append(recommendations,
			"Unpredictable restart pattern - comprehensive investigation needed",
			"Enable debug logging to capture failure details",
			"Consider health check implementation")
	}

	// Add frequency-based recommendations
	if rp.Frequency > 5 {
		recommendations = append(recommendations,
			"High restart frequency - consider disabling automatic restart",
			"Implement exponential backoff for restarts")
	}

	return recommendations
}
