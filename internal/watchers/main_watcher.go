package watchers

import (
	"context"
	"fmt"
	"time"
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

// Watcher is the main infrastructure watcher
type Watcher struct {
	config   WatcherConfig
	watchers map[string]*ProviderWatcher
	events   chan *WatchEvent
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewWatcher creates a new infrastructure watcher
func NewWatcher(config WatcherConfig) (*Watcher, error) {
	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		config:   config,
		watchers: make(map[string]*ProviderWatcher),
		events:   make(chan *WatchEvent, 100),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Initialize provider watchers
	for _, provider := range config.Providers {
		pw := &ProviderWatcher{
			Provider: provider,
			Config: WatchConfig{
				Providers: []string{provider},
				PollingIntervals: map[string]time.Duration{
					provider: config.Interval,
				},
			},
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

// watchProvider watches a specific provider
func (w *Watcher) watchProvider(ctx context.Context, pw *ProviderWatcher) {
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// In a real implementation, this would check for changes
			// For now, we'll just simulate
		}
	}
}
