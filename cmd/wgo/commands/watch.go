package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/watcher"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Real-time infrastructure monitoring",
	Long: `Watch infrastructure for real-time changes with smart correlation.

WGO watch mode continuously monitors your infrastructure and displays
changes as they happen, with intelligent correlation to group related
changes together.

EXAMPLES:
  wgo watch                                    # Watch all providers, 30s interval  
  wgo watch --provider kubernetes              # Watch only Kubernetes
  wgo watch --interval 10s                     # Custom 10-second interval
  wgo watch --quiet                           # Only show changes (script-friendly)
  wgo watch --format json                     # JSON output for automation
  wgo watch --webhook https://hooks.slack.com # Send to Slack webhook
  wgo watch --high-confidence                 # Only show high-confidence correlations

INTEGRATION:
  • Uses smart correlation engine to group related changes
  • Supports webhooks for external notifications (Slack, Teams, etc.)
  • Multiple output formats for different use cases
  • Graceful shutdown with Ctrl+C
  • Low resource usage for continuous monitoring

OUTPUT FORMATS:
  table    Human-readable table format (default)
  json     JSON format for automation
  quiet    Minimal output for scripts`,
	RunE: runWatch,
}

var (
	watchProviders []string
	watchInterval  time.Duration
	watchFormat    string
	watchQuiet     bool
	watchWebhook   string
	watchHighConf  bool
)

func init() {
	// Provider selection
	watchCmd.Flags().StringSliceVarP(&watchProviders, "provider", "p", []string{},
		"providers to watch (kubernetes, terraform, aws, gcp)")

	// Watch configuration
	watchCmd.Flags().DurationVarP(&watchInterval, "interval", "i", 30*time.Second,
		"watch interval (minimum 5s)")

	// Output configuration
	watchCmd.Flags().StringVarP(&watchFormat, "format", "f", "table",
		"output format (table, json, quiet)")

	watchCmd.Flags().BoolVarP(&watchQuiet, "quiet", "q", false,
		"quiet mode - only show changes")

	watchCmd.Flags().BoolVar(&watchHighConf, "high-confidence", false,
		"only show high-confidence correlations")

	// Integration
	watchCmd.Flags().StringVar(&watchWebhook, "webhook", "",
		"webhook URL for notifications (supports Slack)")

	// Validation
	watchCmd.MarkFlagsMutuallyExclusive("quiet", "format")
}

func runWatch(cmd *cobra.Command, args []string) error {
	// Validate interval
	if watchInterval < 5*time.Second {
		return fmt.Errorf("minimum watch interval is 5 seconds")
	}

	// If quiet is set, override format
	if watchQuiet {
		watchFormat = "quiet"
	}

	// Default providers if none specified
	if len(watchProviders) == 0 {
		watchProviders = []string{"kubernetes", "terraform"}
	}

	// Create watcher configuration
	config := watcher.WatcherConfig{
		Providers:    watchProviders,
		Interval:     watchInterval,
		OutputFormat: watchFormat,
		Quiet:        watchQuiet,
		OnlyHighConf: watchHighConf,
		WebhookURL:   watchWebhook,
	}

	// Create watcher
	w, err := watcher.NewWatcher(config)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start signal handler
	go func() {
		<-sigChan
		if !watchQuiet {
			fmt.Printf("\n🛑 Received interrupt signal, stopping watch mode...\n")
		}
		cancel()
	}()

	// Start watching
	if err := w.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("watch failed: %w", err)
	}

	return nil
}

// newWatchCommand creates the watch command (for testing)
func newWatchCommand() *cobra.Command {
	return watchCmd
}
