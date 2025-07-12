package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/internal/catchup"
	"github.com/yairfalse/vaino/internal/logger"
	"github.com/yairfalse/vaino/internal/storage"
	"github.com/yairfalse/vaino/pkg/config"
)

var (
	catchUpSince       string
	catchUpComfortMode bool
	catchUpSyncState   bool
	catchUpProviders   []string
)

// catchUpCmd represents the catch-up command
var catchUpCmd = &cobra.Command{
	Use:   "catch-up",
	Short: "Show infrastructure changes over time",
	Long: `Show a summary of infrastructure changes that occurred during a specified period.

Examples:
  vaino catch-up                      # Auto-detect period
  vaino catch-up --since "2 weeks ago"  # Changes from 2 weeks ago
  vaino catch-up --sync-state         # Update baselines after review`,
	RunE: runCatchUp,
}

func init() {
	catchUpCmd.Flags().StringVar(&catchUpSince, "since", "", "Time period to catch up from (e.g., '2 weeks ago', '2024-01-01')")
	catchUpCmd.Flags().BoolVar(&catchUpComfortMode, "comfort-mode", true, "Use friendly output format")
	catchUpCmd.Flags().BoolVar(&catchUpSyncState, "sync-state", false, "Update baselines after reviewing changes")
	catchUpCmd.Flags().StringSliceVar(&catchUpProviders, "providers", []string{}, "Specific providers to check (default: all)")
}

func runCatchUp(cmd *cobra.Command, args []string) error {
	// Initialize logger
	log := logger.NewSimple()
	log.Info("Starting catch-up analysis...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize storage
	store := storage.NewLocal(cfg.Storage.BaseDir)
	if store == nil {
		return fmt.Errorf("failed to initialize storage")
	}

	// Parse time period
	sinceTime, err := parseSinceTime(catchUpSince)
	if err != nil {
		return fmt.Errorf("failed to parse time period: %w", err)
	}

	// If no providers specified, use all configured providers
	if len(catchUpProviders) == 0 {
		// Get all enabled providers from config
		if cfg.Collectors.Terraform.Enabled {
			catchUpProviders = append(catchUpProviders, "terraform")
		}
		if cfg.Collectors.AWS.Enabled {
			catchUpProviders = append(catchUpProviders, "aws")
		}
		if cfg.Collectors.Kubernetes.Enabled {
			catchUpProviders = append(catchUpProviders, "kubernetes")
		}
	}

	// Create catch-up engine
	engine := catchup.NewEngine(store, cfg)

	// Configure options
	options := catchup.Options{
		Since:       sinceTime,
		ComfortMode: catchUpComfortMode,
		SyncState:   catchUpSyncState,
		Providers:   catchUpProviders,
	}

	// Run catch-up analysis
	report, err := engine.GenerateReport(cmd.Context(), options)
	if err != nil {
		return fmt.Errorf("failed to generate catch-up report: %w", err)
	}

	// Format and display report
	formatter := catchup.NewFormatter(catchUpComfortMode)
	output := formatter.Format(report)
	fmt.Println(output)

	// Sync state if requested
	if catchUpSyncState {
		log.Info("Updating baselines with current state...")
		if err := engine.SyncState(cmd.Context(), options); err != nil {
			return fmt.Errorf("failed to sync state: %w", err)
		}
		fmt.Println("\nBaselines updated successfully!")
	}

	return nil
}

// parseSinceTime parses the --since flag into a time.Time
func parseSinceTime(since string) (time.Time, error) {
	if since == "" {
		// Auto-detect: default to last login or 1 week ago
		return autoDetectAbsencePeriod(), nil
	}

	// Handle relative time strings
	since = strings.ToLower(strings.TrimSpace(since))
	now := time.Now()

	// Common relative time patterns
	switch {
	case strings.Contains(since, "hour"):
		hours := extractNumber(since, 1)
		return now.Add(-time.Duration(hours) * time.Hour), nil
	case strings.Contains(since, "day"):
		days := extractNumber(since, 1)
		return now.AddDate(0, 0, -days), nil
	case strings.Contains(since, "week"):
		weeks := extractNumber(since, 1)
		return now.AddDate(0, 0, -weeks*7), nil
	case strings.Contains(since, "month"):
		months := extractNumber(since, 1)
		return now.AddDate(0, -months, 0), nil
	case strings.Contains(since, "year"):
		years := extractNumber(since, 1)
		return now.AddDate(-years, 0, 0), nil
	default:
		// Try parsing as absolute date
		layouts := []string{
			"2006-01-02",
			"2006-01-02 15:04:05",
			"Jan 2, 2006",
			"January 2, 2006",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, since); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("unable to parse time: %s", since)
	}
}

// autoDetectAbsencePeriod tries to intelligently detect when the user was last active
func autoDetectAbsencePeriod() time.Time {
	// For now, default to 1 week ago
	// In a real implementation, this could check:
	// - Last command execution time
	// - Last baseline update
	// - Last system login
	// - Git commit history
	return time.Now().AddDate(0, 0, -7)
}

// extractNumber extracts a number from a string, with a default value
func extractNumber(s string, defaultVal int) int {
	parts := strings.Fields(s)
	for _, part := range parts {
		var n int
		if _, err := fmt.Sscanf(part, "%d", &n); err == nil {
			return n
		}
	}
	return defaultVal
}
