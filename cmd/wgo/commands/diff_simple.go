package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/analyzer"
	"github.com/yairfalse/wgo/internal/differ"
	"github.com/yairfalse/wgo/pkg/types"
)

func newSimpleDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changes",
		Short: "Show infrastructure changes",
		Long:  `Show what changed in your infrastructure between two points in time.`,
		Example: `  # Show changes in the last hour
  wgo changes --since 1h

  # Show changes between two snapshots
  wgo changes --from snapshot1.json --to snapshot2.json

  # Show changes for a specific provider
  wgo changes --provider kubernetes --since 30m`,
		RunE: runSimpleDiff,
	}

	// Flags
	cmd.Flags().String("since", "", "show changes since duration ago (e.g., 1h, 30m)")
	cmd.Flags().String("from", "", "compare from this snapshot")
	cmd.Flags().String("to", "", "compare to this snapshot (default: now)")
	cmd.Flags().StringP("provider", "p", "", "filter by provider")
	cmd.Flags().String("namespace", "", "filter by namespace")
	cmd.Flags().StringP("output", "o", "text", "output format (text, json)")
	cmd.Flags().Bool("correlated", false, "show correlated changes (group related changes)")
	cmd.Flags().Bool("timeline", false, "show visual timeline of changes")

	return cmd
}

func runSimpleDiff(cmd *cobra.Command, args []string) error {
	since, _ := cmd.Flags().GetString("since")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	provider, _ := cmd.Flags().GetString("provider")
	outputFormat, _ := cmd.Flags().GetString("output")
	correlated, _ := cmd.Flags().GetBool("correlated")
	timeline, _ := cmd.Flags().GetBool("timeline")

	var fromSnapshot, toSnapshot *types.Snapshot
	var err error

	// Handle --since flag
	if since != "" {
		duration, err := time.ParseDuration(since)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}

		// Find snapshot from 'duration' ago
		fromSnapshot, err = findSnapshotFromTime(time.Now().Add(-duration), provider)
		if err != nil {
			return fmt.Errorf("no snapshot found from %s ago", since)
		}

		// Get current state
		if to == "" {
			toSnapshot, err = getCurrentSnapshot(provider)
			if err != nil {
				return err
			}
		}
	} else if from != "" {
		// Load specific snapshots
		fromSnapshot, err = loadSnapshot(from)
		if err != nil {
			return fmt.Errorf("failed to load 'from' snapshot: %w", err)
		}

		if to != "" {
			toSnapshot, err = loadSnapshot(to)
			if err != nil {
				return fmt.Errorf("failed to load 'to' snapshot: %w", err)
			}
		} else {
			// Get current state
			toSnapshot, err = getCurrentSnapshot(provider)
			if err != nil {
				return err
			}
		}
	} else {
		// Default: compare last two snapshots
		snapshots, err := findRecentSnapshots(provider, 2)
		if err != nil || len(snapshots) < 2 {
			return fmt.Errorf("need at least 2 snapshots to compare")
		}
		fromSnapshot = snapshots[1]
		toSnapshot = snapshots[0]
	}

	// Compare snapshots
	simpleDiffer := differ.NewSimpleDiffer()
	report, err := simpleDiffer.Compare(fromSnapshot, toSnapshot)
	if err != nil {
		return fmt.Errorf("comparison failed: %w", err)
	}

	// Output results
	switch outputFormat {
	case "json":
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
	default:
		if timeline {
			// Show timeline view
			correlator := analyzer.NewCorrelator()
			groups := correlator.GroupChanges(report.Changes)
			
			// Calculate duration from report
			var duration time.Duration
			if fromSnapshot != nil && toSnapshot != nil {
				duration = toSnapshot.Timestamp.Sub(fromSnapshot.Timestamp)
			}
			
			fmt.Print(analyzer.FormatChangeTimeline(groups, duration))
		} else if correlated {
			// Group related changes
			correlator := analyzer.NewCorrelator()
			groups := correlator.GroupChanges(report.Changes)
			fmt.Print(analyzer.FormatCorrelatedChanges(groups))
		} else {
			// Simple flat list
			fmt.Print(differ.FormatChangeReport(report))
		}
	}

	return nil
}

// Helper to get current infrastructure state
func getCurrentSnapshot(provider string) (*types.Snapshot, error) {
	fmt.Println("📸 Capturing current state...")
	
	// Create temp file for scan output
	tempFile, err := os.CreateTemp("", "wgo-current-*.json")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	// Run scan
	scanCmd := newScanCommand()
	args := []string{"--output-file", tempPath}
	if provider != "" {
		args = append(args, "--provider", provider)
	}
	scanCmd.SetArgs(args)
	
	if err := scanCmd.Execute(); err != nil {
		return nil, fmt.Errorf("failed to scan current state: %w", err)
	}

	// Load the snapshot
	return loadSnapshot(tempPath)
}

// Helper to load a snapshot from file
func loadSnapshot(path string) (*types.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// Helper to find snapshot from a specific time
func findSnapshotFromTime(targetTime time.Time, provider string) (*types.Snapshot, error) {
	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(homeDir, ".wgo", "history")

	// Find all snapshot files
	pattern := "*.json"
	if provider != "" {
		pattern = fmt.Sprintf("*-%s-*.json", provider)
	}

	matches, _ := filepath.Glob(filepath.Join(historyDir, pattern))
	
	// Find the closest snapshot to target time
	var closestPath string
	var closestDiff time.Duration
	
	for _, path := range matches {
		snapshot, err := loadSnapshot(path)
		if err != nil {
			continue
		}
		
		diff := targetTime.Sub(snapshot.Timestamp).Abs()
		if closestPath == "" || diff < closestDiff {
			closestPath = path
			closestDiff = diff
		}
	}

	if closestPath == "" {
		return nil, fmt.Errorf("no snapshots found")
	}

	return loadSnapshot(closestPath)
}

// Helper to find recent snapshots
func findRecentSnapshots(provider string, count int) ([]*types.Snapshot, error) {
	homeDir, _ := os.UserHomeDir()
	historyDir := filepath.Join(homeDir, ".wgo", "history")

	// Find all snapshot files
	pattern := "*.json"
	if provider != "" {
		pattern = fmt.Sprintf("*-%s-*.json", provider)
	}

	matches, _ := filepath.Glob(filepath.Join(historyDir, pattern))
	
	// Load and sort by timestamp
	var snapshots []*types.Snapshot
	for _, path := range matches {
		snapshot, err := loadSnapshot(path)
		if err == nil {
			snapshots = append(snapshots, snapshot)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	// Return requested count
	if len(snapshots) > count {
		snapshots = snapshots[:count]
	}

	return snapshots, nil
}