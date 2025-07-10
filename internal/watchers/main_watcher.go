package watchers

import (
	"context"
	"fmt"
	"time"

	"github.com/yairfalse/vaino/internal/analyzer"
	"github.com/yairfalse/vaino/internal/collectors"
	"github.com/yairfalse/vaino/internal/collectors/aws"
	"github.com/yairfalse/vaino/internal/collectors/kubernetes"
	"github.com/yairfalse/vaino/internal/collectors/terraform"
	"github.com/yairfalse/vaino/internal/differ"
	"github.com/yairfalse/vaino/pkg/types"
)

// WatcherConfig holds configuration for the main watcher
type WatcherConfig struct {
	Providers    []string
	Interval     time.Duration
	OutputFormat string
	Quiet        bool
	OnlyHighConf bool
	WebhookURL   string
}

// DisplayEvent represents a change event for display purposes
type DisplayEvent struct {
	Timestamp        time.Time
	Summary          differ.ChangeSummary
	CorrelatedGroups []analyzer.ChangeGroup
	RawChanges       []differ.SimpleChange
	Source           string
}

// Watcher is the main infrastructure watcher
type Watcher struct {
	config       WatcherConfig
	watchers     map[string]*ProviderWatcher
	events       chan *DisplayEvent
	ctx          context.Context
	cancel       context.CancelFunc
	outputFormat string
	quiet        bool
	onlyHighConf bool
	providers    []string
	interval     time.Duration
	webhookURL   string
}

// NewWatcher creates a new infrastructure watcher
func NewWatcher(config WatcherConfig) (*Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		config:       config,
		watchers:     make(map[string]*ProviderWatcher),
		events:       make(chan *DisplayEvent, 100),
		ctx:          ctx,
		cancel:       cancel,
		outputFormat: config.OutputFormat,
		quiet:        config.Quiet,
		onlyHighConf: config.OnlyHighConf,
		providers:    config.Providers,
		interval:     config.Interval,
		webhookURL:   config.WebhookURL,
	}

	// Initialize provider watchers with actual collectors
	for _, provider := range config.Providers {
		var collector collectors.EnhancedCollector
		var err error

		// Create the appropriate collector for each provider
		switch provider {
		case "kubernetes":
			collector = kubernetes.NewKubernetesCollector()
		case "terraform":
			collector = terraform.NewTerraformCollector()
		case "aws":
			collector = aws.NewAWSCollector()
		default:
			return nil, fmt.Errorf("unsupported provider: %s", provider)
		}

		// Get collector configuration
		collectorConfig, err := collector.AutoDiscover()
		if err != nil {
			// Log warning but continue - some providers may not be configured
			fmt.Printf("Warning: Failed to auto-discover config for %s: %v\n", provider, err)
			collectorConfig = collectors.CollectorConfig{}
		}

		pw := &ProviderWatcher{
			provider:        provider,
			collector:       collector,
			config:          collectorConfig,
			pollingInterval: config.Interval,
			resourceCache:   make(map[string]types.Resource),
			resourceHashes:  make(map[string]string),
			eventChannel:    make(chan WatchEvent, 100),
			running:         false,
		}
		w.watchers[provider] = pw
	}

	return w, nil
}

// Start begins watching for infrastructure changes
func (w *Watcher) Start(ctx context.Context) error {
	// Display header
	if !w.config.Quiet {
		w.displayWatchHeader()
	}

	// Start provider watchers
	for _, pw := range w.watchers {
		go w.watchProvider(ctx, pw)
	}

	// Process events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-w.events:
			w.displayChanges(event)

			if w.config.WebhookURL != "" {
				if err := w.SendWebhook(event); err != nil {
					fmt.Printf("Warning: Failed to send webhook: %v\n", err)
				}
			}
		}
	}
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	w.cancel()
}

// GetStatus returns the current status of the watcher
func (w *Watcher) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"providers": w.providers,
		"interval":  w.interval,
		"running":   w.ctx.Err() == nil,
	}
}

// watchProvider watches a specific provider
func (w *Watcher) watchProvider(ctx context.Context, pw *ProviderWatcher) {
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	// Perform initial scan to establish baseline
	if err := w.performProviderScan(ctx, pw); err != nil {
		if !w.quiet {
			fmt.Printf("Warning: Initial scan failed for %s: %v\n", pw.provider, err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Perform scan and detect changes
			if err := w.performProviderScan(ctx, pw); err != nil {
				if !w.quiet {
					fmt.Printf("Warning: Scan failed for %s: %v\n", pw.provider, err)
				}
			}
		}
	}
}

// performProviderScan performs a scan for a specific provider and detects changes
func (w *Watcher) performProviderScan(ctx context.Context, pw *ProviderWatcher) error {
	if pw.collector == nil {
		return fmt.Errorf("no collector configured for provider %s", pw.provider)
	}

	// Create scan context with timeout
	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Perform collection
	snapshot, err := pw.collector.Collect(scanCtx, pw.config)
	if err != nil {
		return fmt.Errorf("failed to collect from %s: %w", pw.provider, err)
	}

	// Check for changes by comparing with cached resources
	changes := w.detectChanges(pw, snapshot.Resources)
	if len(changes) > 0 {
		// Create correlation groups
		correlator := analyzer.NewCorrelator()
		groups := correlator.GroupChanges(changes)

		// Create summary
		summary := w.createSummary(changes)

		// Create display event
		event := &DisplayEvent{
			Timestamp:        time.Now(),
			Summary:          summary,
			CorrelatedGroups: groups,
			RawChanges:       changes,
			Source:           pw.provider,
		}

		// Send event to display channel
		select {
		case w.events <- event:
			// Event sent successfully
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel is full, drop event
			if !w.quiet {
				fmt.Printf("Warning: Event channel full, dropping event for %s\n", pw.provider)
			}
		}
	}

	// Update resource cache
	w.updateResourceCache(pw, snapshot.Resources)

	return nil
}

// detectChanges compares current resources with cached resources and returns changes
func (w *Watcher) detectChanges(pw *ProviderWatcher, currentResources []types.Resource) []differ.SimpleChange {
	var changes []differ.SimpleChange
	currentMap := make(map[string]types.Resource)

	// Build map of current resources
	for _, resource := range currentResources {
		currentMap[resource.ID] = resource
	}

	// Check for removed and modified resources
	for id, cachedResource := range pw.resourceCache {
		if currentResource, exists := currentMap[id]; exists {
			// Resource still exists - check for modifications
			if w.hasResourceChanged(cachedResource, currentResource) {
				changes = append(changes, differ.SimpleChange{
					Type:         "modified",
					ResourceID:   id,
					ResourceType: currentResource.Type,
					ResourceName: currentResource.Name,
					Namespace:    currentResource.Namespace,
					Timestamp:    time.Now(),
				})
			}
		} else {
			// Resource was removed
			changes = append(changes, differ.SimpleChange{
				Type:         "removed",
				ResourceID:   id,
				ResourceType: cachedResource.Type,
				ResourceName: cachedResource.Name,
				Namespace:    cachedResource.Namespace,
				Timestamp:    time.Now(),
			})
		}
	}

	// Check for added resources
	for id, currentResource := range currentMap {
		if _, exists := pw.resourceCache[id]; !exists {
			// Resource was added
			changes = append(changes, differ.SimpleChange{
				Type:         "added",
				ResourceID:   id,
				ResourceType: currentResource.Type,
				ResourceName: currentResource.Name,
				Namespace:    currentResource.Namespace,
				Timestamp:    time.Now(),
			})
		}
	}

	return changes
}

// hasResourceChanged checks if a resource has changed by comparing hashes
func (w *Watcher) hasResourceChanged(cached, current types.Resource) bool {
	return cached.Hash() != current.Hash()
}

// updateResourceCache updates the resource cache with current resources
func (w *Watcher) updateResourceCache(pw *ProviderWatcher, currentResources []types.Resource) {
	// Clear existing cache
	pw.resourceCache = make(map[string]types.Resource)
	pw.resourceHashes = make(map[string]string)

	// Update with current resources
	for _, resource := range currentResources {
		pw.resourceCache[resource.ID] = resource
		pw.resourceHashes[resource.ID] = resource.Hash()
	}
}

// createSummary creates a summary of changes
func (w *Watcher) createSummary(changes []differ.SimpleChange) differ.ChangeSummary {
	summary := differ.ChangeSummary{}

	for _, change := range changes {
		switch change.Type {
		case "added":
			summary.Added++
		case "removed":
			summary.Removed++
		case "modified":
			summary.Modified++
		}
	}

	summary.Total = summary.Added + summary.Removed + summary.Modified
	return summary
}
