package watcher

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yairfalse/wgo/internal/analyzer"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/pkg/types"
)

// Watcher provides real-time infrastructure monitoring
type Watcher struct {
	providers      []string
	interval       time.Duration
	lastSnapshot   *types.Snapshot
	correlator     *analyzer.Correlator
	differ         *differ.SimpleDiffer
	quiet          bool
	outputFormat   string
	webhookURL     string
	registry       *collectors.EnhancedRegistry
	onlyHighConf   bool
	changeCallback func(*WatchEvent)
}

// WatchEvent represents a real-time change event
type WatchEvent struct {
	Timestamp        time.Time                  `json:"timestamp"`
	CorrelatedGroups []analyzer.ChangeGroup     `json:"correlated_groups"`
	RawChanges       []differ.SimpleChange      `json:"raw_changes"`
	Summary          differ.ChangeSummary       `json:"summary"`
	Source           string                     `json:"source"`
}

// WatcherConfig holds configuration for the watcher
type WatcherConfig struct {
	Providers      []string
	Interval       time.Duration
	OutputFormat   string
	Quiet          bool
	OnlyHighConf   bool
	WebhookURL     string
	ChangeCallback func(*WatchEvent)
}

// NewWatcher creates a new infrastructure watcher
func NewWatcher(config WatcherConfig) (*Watcher, error) {
	if config.Interval < 5*time.Second {
		return nil, fmt.Errorf("minimum watch interval is 5 seconds")
	}

	if config.OutputFormat == "" {
		config.OutputFormat = "table"
	}

	correlator := analyzer.NewCorrelator()
	differ := differ.NewSimpleDiffer()
	
	// Get registry (this would normally be injected)
	registry := collectors.NewEnhancedRegistry()

	return &Watcher{
		providers:      config.Providers,
		interval:       config.Interval,
		correlator:     correlator,
		differ:         differ,
		quiet:          config.Quiet,
		outputFormat:   config.OutputFormat,
		webhookURL:     config.WebhookURL,
		onlyHighConf:   config.OnlyHighConf,
		changeCallback: config.ChangeCallback,
		registry:       registry,
	}, nil
}

// Start begins real-time monitoring
func (w *Watcher) Start(ctx context.Context) error {
	if !w.quiet {
		fmt.Printf("🔍 Starting infrastructure watch mode\n")
		fmt.Printf("   Providers: %v\n", w.providers)
		fmt.Printf("   Interval: %v\n", w.interval)
		fmt.Printf("   Format: %s\n", w.outputFormat)
		if w.webhookURL != "" {
			fmt.Printf("   Webhook: enabled\n")
		}
		fmt.Printf("   Press Ctrl+C to stop\n\n")
	}

	// Take initial snapshot
	if err := w.takeInitialSnapshot(ctx); err != nil {
		return fmt.Errorf("failed to take initial snapshot: %w", err)
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.checkForChanges(ctx); err != nil {
				if !w.quiet {
					log.Printf("Watch error: %v", err)
				}
				// Continue watching even if there are errors
			}
		case <-ctx.Done():
			if !w.quiet {
				fmt.Printf("\n🛑 Watch mode stopped\n")
			}
			return ctx.Err()
		}
	}
}

// takeInitialSnapshot captures the first snapshot
func (w *Watcher) takeInitialSnapshot(ctx context.Context) error {
	snapshot, err := w.takeSnapshot(ctx)
	if err != nil {
		return err
	}

	w.lastSnapshot = snapshot
	
	if !w.quiet {
		fmt.Printf("Initial snapshot captured (%d resources)\n", len(snapshot.Resources))
		fmt.Printf("Watching for changes every %v...\n\n", w.interval)
	}

	return nil
}

// checkForChanges compares current state with last snapshot
func (w *Watcher) checkForChanges(ctx context.Context) error {
	// Take new snapshot
	currentSnapshot, err := w.takeSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to take snapshot: %w", err)
	}

	if w.lastSnapshot == nil {
		w.lastSnapshot = currentSnapshot
		return nil
	}

	// Detect changes using simple differ
	report, err := w.differ.Compare(w.lastSnapshot, currentSnapshot)
	if err != nil {
		return fmt.Errorf("failed to compare snapshots: %w", err)
	}

	// If no changes, continue silently
	if len(report.Changes) == 0 {
		w.lastSnapshot = currentSnapshot
		return nil
	}

	// Correlate changes using the brilliant correlation engine
	correlatedGroups := w.correlator.GroupChanges(report.Changes)

	// Create watch event
	event := &WatchEvent{
		Timestamp:        time.Now(),
		CorrelatedGroups: correlatedGroups,
		RawChanges:       report.Changes,
		Summary:          report.Summary,
		Source:           "wgo-watch",
	}

	// Display changes
	w.displayChanges(event)

	// Send webhook if configured
	if w.webhookURL != "" {
		go w.sendWebhook(event) // Non-blocking
	}

	// Call custom callback if provided
	if w.changeCallback != nil {
		go w.changeCallback(event) // Non-blocking
	}

	// Update last snapshot
	w.lastSnapshot = currentSnapshot

	return nil
}

// takeSnapshot captures current infrastructure state
func (w *Watcher) takeSnapshot(ctx context.Context) (*types.Snapshot, error) {
	// This is a simplified version - in reality we'd iterate through providers
	// and use the collector registry to gather resources
	
	snapshot := &types.Snapshot{
		ID:        fmt.Sprintf("watch-%d", time.Now().Unix()),
		Timestamp: time.Now(),
		Provider:  "multi", // Multiple providers
		Resources: []types.Resource{},
	}

	// In a real implementation, we'd:
	// 1. Iterate through w.providers
	// 2. Use w.registry.GetCollector(provider) 
	// 3. Collect resources from each provider
	// 4. Merge results into snapshot.Resources
	
	// For now, return empty snapshot to avoid build errors
	// This would be replaced with actual collection logic
	
	return snapshot, nil
}

// GetStatus returns current watcher status
func (w *Watcher) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"active":         true,
		"interval":       w.interval.String(),
		"providers":      w.providers,
		"output_format":  w.outputFormat,
		"webhook_url":    w.webhookURL != "",
		"quiet_mode":     w.quiet,
		"only_high_conf": w.onlyHighConf,
	}

	if w.lastSnapshot != nil {
		status["last_snapshot_time"] = w.lastSnapshot.Timestamp
		status["last_resource_count"] = len(w.lastSnapshot.Resources)
	}

	return status
}

// Stop gracefully stops the watcher (used for testing)
func (w *Watcher) Stop() {
	// This method exists for testing purposes
	// The actual stopping is handled by context cancellation in Start()
}