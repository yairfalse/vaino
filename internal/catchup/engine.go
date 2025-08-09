package catchup

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/internal/logger"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/config"
	"github.com/yairfalse/vaino/pkg/types"
)

// Options configures the catch-up analysis
type Options struct {
	Since       time.Time
	ComfortMode bool
	SyncState   bool
	Providers   []string
}

// Report contains the catch-up analysis results
type Report struct {
	Period           Period
	Summary          Summary
	PlannedChanges   []Change
	UnplannedChanges []Change
	RoutineChanges   []Change
	SecurityStatus   SecurityStatus
	TeamActivity     TeamActivity
	Recommendations  []string
	ComfortMetrics   ComfortMetrics
}

// Period represents the time period analyzed
type Period struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// Summary provides high-level statistics
type Summary struct {
	TotalChanges      int
	PlannedCount      int
	UnplannedCount    int
	RoutineCount      int
	CriticalSystems   string
	SecurityIncidents int
	TeamEfficiency    string
}

// Change represents a single infrastructure change
type Change struct {
	Timestamp    time.Time
	Type         ChangeType
	Provider     string
	Resource     types.Resource
	Description  string
	Impact       string
	HandledBy    string
	IsSuccessful bool
	Tags         []string
}

// ChangeType categorizes changes
type ChangeType string

const (
	ChangeTypePlanned   ChangeType = "planned"
	ChangeTypeUnplanned ChangeType = "unplanned"
	ChangeTypeRoutine   ChangeType = "routine"
)

// SecurityStatus summarizes security-related information
type SecurityStatus struct {
	IncidentCount   int
	Vulnerabilities []string
	ComplianceScore float64
	LastAudit       time.Time
}

// TeamActivity shows what the team did
type TeamActivity struct {
	TotalActions     int
	TopContributors  []string
	KeyDecisions     []string
	IncidentHandling string
}

// ComfortMetrics provides emotional reassurance data
type ComfortMetrics struct {
	StabilityScore    float64
	TeamPerformance   float64
	SystemResilience  float64
	OverallConfidence float64
}

// Engine performs catch-up analysis
type Engine struct {
	storage    storage.Storage
	config     *config.Config
	collectors map[string]collectors.Collector
	classifier *Classifier
	logger     logger.Logger
}

// NewEngine creates a new catch-up engine
func NewEngine(storage storage.Storage, config *config.Config) *Engine {
	return &Engine{
		storage:    storage,
		config:     config,
		collectors: make(map[string]collectors.Collector),
		classifier: NewClassifier(),
		logger:     logger.NewSimple(),
	}
}

// GenerateReport creates a comprehensive catch-up report
func (e *Engine) GenerateReport(ctx context.Context, options Options) (*Report, error) {
	e.logger.WithFields(map[string]interface{}{
		"since":        options.Since,
		"providers":    options.Providers,
		"comfort_mode": options.ComfortMode,
	}).Info("Generating catch-up report")

	// Initialize report
	report := &Report{
		Period: Period{
			Start:    options.Since,
			End:      time.Now(),
			Duration: time.Since(options.Since),
		},
		PlannedChanges:   []Change{},
		UnplannedChanges: []Change{},
		RoutineChanges:   []Change{},
		Recommendations:  []string{},
	}

	// Collect changes from all providers
	allChanges, err := e.collectChanges(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to collect changes: %w", err)
	}

	// Classify changes
	e.classifyChanges(allChanges, report)

	// Analyze security status
	report.SecurityStatus = e.analyzeSecurityStatus(allChanges)

	// Analyze team activity
	report.TeamActivity = e.analyzeTeamActivity(allChanges)

	// Calculate comfort metrics
	report.ComfortMetrics = e.calculateComfortMetrics(report)

	// Generate summary
	report.Summary = e.generateSummary(report)

	// Add recommendations
	report.Recommendations = e.generateRecommendations(report)

	return report, nil
}

// collectChanges gathers all changes from the specified time period
func (e *Engine) collectChanges(ctx context.Context, options Options) ([]Change, error) {
	var allChanges []Change

	// Get historical snapshots
	// Note: We need to implement time-filtered listing in storage interface
	allSnapshots, err := e.storage.ListSnapshots()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Filter snapshots by time range
	var snapshots []*types.Snapshot
	for _, info := range allSnapshots {
		if info.Timestamp.After(options.Since) && info.Timestamp.Before(time.Now()) {
			// Load the actual snapshot
			snapshot, err := e.storage.LoadSnapshot(info.ID)
			if err != nil {
				e.logger.WithField("snapshot_id", info.ID).Error("Failed to load snapshot", err)
				continue
			}
			snapshots = append(snapshots, snapshot)
		}
	}

	// Group snapshots by provider
	providerSnapshots := make(map[string][]*types.Snapshot)
	for _, snapshot := range snapshots {
		if shouldIncludeProvider(snapshot.Provider, options.Providers) {
			providerSnapshots[snapshot.Provider] = append(providerSnapshots[snapshot.Provider], snapshot)
		}
	}

	// Analyze changes for each provider
	for provider, snaps := range providerSnapshots {
		if len(snaps) < 2 {
			continue // Need at least 2 snapshots to detect changes
		}

		// Sort snapshots by timestamp
		sort.Slice(snaps, func(i, j int) bool {
			return snaps[i].Timestamp.Before(snaps[j].Timestamp)
		})

		// Compare consecutive snapshots
		for i := 1; i < len(snaps); i++ {
			previous := snaps[i-1]
			current := snaps[i]

			// Use differ to find changes
			driftReport := e.detectChanges(previous, current)

			// Convert to catch-up changes
			if driftReport != nil {
				for _, resourceDiff := range driftReport.ResourceChanges {
					// Create a generic description from the resource diff
					description := fmt.Sprintf("%s %s: %s", resourceDiff.DriftType, resourceDiff.ResourceType, resourceDiff.Description)

					// Find the resource in current snapshot
					var resource types.Resource
					for _, r := range current.Resources {
						if r.ID == resourceDiff.ResourceID {
							resource = r
							break
						}
					}

					allChanges = append(allChanges, Change{
						Timestamp:    current.Timestamp,
						Type:         ChangeTypeRoutine, // Will be classified later
						Provider:     provider,
						Resource:     resource,
						Description:  description,
						Impact:       resourceDiff.Description,
						IsSuccessful: true,
					})
				}
			}
		}
	}

	// Sort changes by timestamp
	sort.Slice(allChanges, func(i, j int) bool {
		return allChanges[i].Timestamp.Before(allChanges[j].Timestamp)
	})

	return allChanges, nil
}

// detectChanges compares two snapshots and returns the drift report
func (e *Engine) detectChanges(previous, current *types.Snapshot) *differ.DriftReport {
	engine := differ.NewDifferEngine()
	drift, _ := engine.Compare(previous, current)
	return drift
}

// classifyChanges categorizes changes as planned, unplanned, or routine
func (e *Engine) classifyChanges(changes []Change, report *Report) {
	for _, change := range changes {
		classification := e.classifier.Classify(change)
		change.Type = classification

		switch classification {
		case ChangeTypePlanned:
			report.PlannedChanges = append(report.PlannedChanges, change)
		case ChangeTypeUnplanned:
			report.UnplannedChanges = append(report.UnplannedChanges, change)
		case ChangeTypeRoutine:
			report.RoutineChanges = append(report.RoutineChanges, change)
		}
	}
}

// analyzeSecurityStatus checks for security-related changes
func (e *Engine) analyzeSecurityStatus(changes []Change) SecurityStatus {
	status := SecurityStatus{
		IncidentCount:   0,
		Vulnerabilities: []string{},
		ComplianceScore: 100.0,
		LastAudit:       time.Now().AddDate(0, -1, 0), // Default to 1 month ago
	}

	for _, change := range changes {
		// Check for security-related tags
		for _, tag := range change.Tags {
			if tag == "security" || tag == "incident" {
				status.IncidentCount++
			}
			if tag == "vulnerability" {
				status.Vulnerabilities = append(status.Vulnerabilities, change.Description)
			}
		}

		// Check for compliance impact
		if change.Impact == "compliance" {
			status.ComplianceScore -= 5.0
		}
	}

	// Ensure compliance score doesn't go below 0
	if status.ComplianceScore < 0 {
		status.ComplianceScore = 0
	}

	return status
}

// analyzeTeamActivity summarizes what the team accomplished
func (e *Engine) analyzeTeamActivity(changes []Change) TeamActivity {
	activity := TeamActivity{
		TotalActions:     len(changes),
		TopContributors:  []string{},
		KeyDecisions:     []string{},
		IncidentHandling: "Excellent",
	}

	// Count contributors
	contributorCount := make(map[string]int)
	for _, change := range changes {
		if change.HandledBy != "" {
			contributorCount[change.HandledBy]++
		}
	}

	// Find top contributors
	type contributor struct {
		name  string
		count int
	}
	var contributors []contributor
	for name, count := range contributorCount {
		contributors = append(contributors, contributor{name, count})
	}
	sort.Slice(contributors, func(i, j int) bool {
		return contributors[i].count > contributors[j].count
	})

	// Take top 3 contributors
	for i := 0; i < len(contributors) && i < 3; i++ {
		activity.TopContributors = append(activity.TopContributors,
			fmt.Sprintf("%s (%d actions)", contributors[i].name, contributors[i].count))
	}

	// Extract key decisions (planned changes with high impact)
	for _, change := range changes {
		if change.Type == ChangeTypePlanned && change.Impact != "" {
			activity.KeyDecisions = append(activity.KeyDecisions, change.Description)
		}
	}

	// Assess incident handling
	unplannedCount := 0
	successfulCount := 0
	for _, change := range changes {
		if change.Type == ChangeTypeUnplanned {
			unplannedCount++
			if change.IsSuccessful {
				successfulCount++
			}
		}
	}

	if unplannedCount > 0 {
		successRate := float64(successfulCount) / float64(unplannedCount)
		switch {
		case successRate >= 0.95:
			activity.IncidentHandling = "Excellent"
		case successRate >= 0.80:
			activity.IncidentHandling = "Good"
		case successRate >= 0.60:
			activity.IncidentHandling = "Satisfactory"
		default:
			activity.IncidentHandling = "Needs improvement"
		}
	}

	return activity
}

// calculateComfortMetrics generates reassuring metrics
func (e *Engine) calculateComfortMetrics(report *Report) ComfortMetrics {
	metrics := ComfortMetrics{}

	// Stability score based on unplanned vs planned changes
	totalChanges := len(report.PlannedChanges) + len(report.UnplannedChanges) + len(report.RoutineChanges)
	if totalChanges > 0 {
		plannedRatio := float64(len(report.PlannedChanges)) / float64(totalChanges)
		metrics.StabilityScore = 0.7 + (plannedRatio * 0.3) // Base 70% + up to 30%
	} else {
		metrics.StabilityScore = 1.0 // No changes = perfect stability
	}

	// Team performance based on successful handling
	successCount := 0
	for _, change := range append(report.PlannedChanges, report.UnplannedChanges...) {
		if change.IsSuccessful {
			successCount++
		}
	}
	if len(report.PlannedChanges)+len(report.UnplannedChanges) > 0 {
		metrics.TeamPerformance = float64(successCount) / float64(len(report.PlannedChanges)+len(report.UnplannedChanges))
	} else {
		metrics.TeamPerformance = 1.0
	}

	// System resilience based on incident recovery
	if report.Summary.SecurityIncidents == 0 {
		metrics.SystemResilience = 1.0
	} else {
		// Decrease by 10% per incident, minimum 50%
		metrics.SystemResilience = max(0.5, 1.0-(float64(report.Summary.SecurityIncidents)*0.1))
	}

	// Overall confidence is weighted average
	metrics.OverallConfidence = (metrics.StabilityScore*0.4 +
		metrics.TeamPerformance*0.4 +
		metrics.SystemResilience*0.2)

	return metrics
}

// generateSummary creates a high-level summary
func (e *Engine) generateSummary(report *Report) Summary {
	summary := Summary{
		TotalChanges:      len(report.PlannedChanges) + len(report.UnplannedChanges) + len(report.RoutineChanges),
		PlannedCount:      len(report.PlannedChanges),
		UnplannedCount:    len(report.UnplannedChanges),
		RoutineCount:      len(report.RoutineChanges),
		SecurityIncidents: report.SecurityStatus.IncidentCount,
	}

	// Assess critical systems
	if report.ComfortMetrics.StabilityScore >= 0.9 {
		summary.CriticalSystems = "All stable"
	} else if report.ComfortMetrics.StabilityScore >= 0.7 {
		summary.CriticalSystems = "Mostly stable"
	} else {
		summary.CriticalSystems = "Some instability"
	}

	// Team efficiency
	if report.ComfortMetrics.TeamPerformance >= 0.95 {
		summary.TeamEfficiency = "Excellent"
	} else if report.ComfortMetrics.TeamPerformance >= 0.85 {
		summary.TeamEfficiency = "Good"
	} else {
		summary.TeamEfficiency = "Adequate"
	}

	return summary
}

// generateRecommendations provides actionable next steps
func (e *Engine) generateRecommendations(report *Report) []string {
	var recommendations []string

	// Check for security issues
	if report.SecurityStatus.IncidentCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review %d security incident(s) that occurred", report.SecurityStatus.IncidentCount))
	}

	// Check for unplanned changes
	if len(report.UnplannedChanges) > 5 {
		recommendations = append(recommendations,
			"Consider implementing better change planning processes to reduce unplanned changes")
	}

	// Check stability
	if report.ComfortMetrics.StabilityScore < 0.8 {
		recommendations = append(recommendations,
			"Focus on system stability improvements")
	}

	// Always add a positive recommendation
	if report.ComfortMetrics.OverallConfidence >= 0.9 {
		recommendations = append(recommendations,
			"Continue the excellent work maintaining infrastructure stability!")
	} else {
		recommendations = append(recommendations,
			"Schedule a team sync to discuss improvement opportunities")
	}

	return recommendations
}

// SyncState updates baselines with current state
func (e *Engine) SyncState(ctx context.Context, options Options) error {
	e.logger.Info("Syncing state with catch-up baseline")

	// Create a new baseline snapshot tagged with catch-up date
	timestamp := time.Now()
	tag := fmt.Sprintf("catch-up-%s", timestamp.Format("2006-01-02"))

	// Collect current state from all providers
	// Use the providers from the report options if available
	providers := options.Providers
	if len(providers) == 0 {
		// Get all enabled providers from config
		var enabledProviders []string
		if e.config.Collectors.Terraform.Enabled {
			enabledProviders = append(enabledProviders, "terraform")
		}
		if e.config.Collectors.AWS.Enabled {
			enabledProviders = append(enabledProviders, "aws")
		}
		if e.config.Collectors.Kubernetes.Enabled {
			enabledProviders = append(enabledProviders, "kubernetes")
		}
		providers = enabledProviders
	}
	for _, provider := range providers {
		// Get collector for provider
		collector, err := e.getCollector(provider)
		if err != nil {
			e.logger.WithField("provider", provider).Error("Failed to get collector", err)
			continue
		}

		// Collect current state
		enhancedCollector, ok := collector.(collectors.Collector)
		if !ok {
			e.logger.WithField("provider", provider).Error("Collector does not implement Collector", fmt.Errorf("type assertion failed"))
			continue
		}
		snapshot, err := enhancedCollector.Collect(ctx, collectors.CollectorConfig{})
		if err != nil {
			e.logger.WithField("provider", provider).Error("Failed to collect snapshot", err)
			continue
		}

		// Tag the snapshot
		if snapshot.Metadata.Tags == nil {
			snapshot.Metadata.Tags = make(map[string]string)
		}
		snapshot.Metadata.Tags["catch-up"] = tag

		// Store the snapshot
		if err := e.storage.SaveSnapshot(snapshot); err != nil {
			e.logger.WithField("provider", provider).Error("Failed to store snapshot", err)
			continue
		}
	}

	e.logger.WithField("tag", tag).Info("State sync completed")
	return nil
}

// getCollector returns a collector for the specified provider
func (e *Engine) getCollector(provider string) (collectors.Collector, error) {
	if collector, exists := e.collectors[provider]; exists {
		return collector, nil
	}

	// Initialize collector based on provider type
	registry := collectors.DefaultRegistry()
	collector, exists := registry.Get(provider)
	if !exists {
		return nil, fmt.Errorf("collector %s not found", provider)
	}

	e.collectors[provider] = collector
	return collector, nil
}

// shouldIncludeProvider checks if a provider should be included in the analysis
func shouldIncludeProvider(provider string, filter []string) bool {
	if len(filter) == 0 {
		return true // Include all if no filter specified
	}

	for _, p := range filter {
		if p == provider {
			return true
		}
	}

	return false
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
