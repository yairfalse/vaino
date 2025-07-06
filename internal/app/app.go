package app

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yairfalse/wgo/internal/cache"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/logger"
	"github.com/yairfalse/wgo/internal/storage"
)

// App represents the main application
type App struct {
	config   *Config
	storage  storage.Storage
	cache    cache.Manager
	logger   logger.Logger
	registry *collectors.CollectorRegistry
}

// Config holds application configuration
type Config struct {
	Version   string
	Commit    string
	BuildDate string
	Debug     bool
	Verbose   bool
}

// New creates a new application instance with all dependencies
func New(config Config) (*App, error) {
	factory := &AppFactory{}
	return factory.Create(config)
}

// Run executes the application
func (a *App) Run() error {
	rootCmd := a.CreateRootCommand()
	return rootCmd.Execute()
}

// newStatusCommand creates the primary status command - "What's cooking right now?"
func (a *App) newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current infrastructure status and health",
		Long: `Get instant visibility into your infrastructure status.
Shows what's running, what's changed recently, and what needs attention.
Perfect for the "what's cooking?" question.`,
		Example: `  # Quick status overview
  wgo status

  # Status for specific provider
  wgo status --provider aws

  # Detailed status with recent changes
  wgo status --detailed --since 24h

  # Live refreshing status
  wgo status --watch`,
		RunE: a.runStatusCommand,
	}

	// Flags
	cmd.Flags().StringP("provider", "p", "", "filter by provider (terraform, aws, kubernetes)")
	cmd.Flags().StringSlice("region", []string{}, "filter by regions")
	cmd.Flags().Bool("detailed", false, "show detailed resource information")
	cmd.Flags().Duration("since", 24*time.Hour, "show changes since duration (e.g., 1h, 24h, 7d)")
	cmd.Flags().Bool("watch", false, "continuously refresh status")
	cmd.Flags().Duration("refresh", 30*time.Second, "refresh interval for watch mode")

	return cmd
}

// newWatchCommand creates the live monitoring command
func (a *App) newWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Live monitoring of infrastructure changes",
		Long: `Monitor your infrastructure in real-time.
Watch for changes, alerts, and status updates as they happen.`,
		Example: `  # Watch all infrastructure
  wgo watch

  # Watch specific provider
  wgo watch --provider kubernetes --namespace production

  # Watch with custom refresh rate
  wgo watch --interval 10s`,
		RunE: a.runWatchCommand,
	}

	// Flags
	cmd.Flags().StringP("provider", "p", "", "watch specific provider")
	cmd.Flags().StringSlice("region", []string{}, "watch specific regions")
	cmd.Flags().StringSlice("namespace", []string{}, "watch specific Kubernetes namespaces")
	cmd.Flags().Duration("interval", 30*time.Second, "refresh interval")
	cmd.Flags().Bool("alerts-only", false, "show only alerts and warnings")

	return cmd
}

// newInspectCommand creates the deep-dive inspection command
func (a *App) newInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [resource-id]",
		Short: "Deep dive into specific resources",
		Long: `Get detailed information about specific infrastructure resources.
Shows configuration, history, relationships, and health status.`,
		Example: `  # Inspect a specific resource
  wgo inspect ec2-i-1234567890abcdef0

  # Inspect with history
  wgo inspect rds-prod-database --history 7d

  # Inspect with relationships
  wgo inspect vpc-12345 --show-related`,
		Args: cobra.MaximumNArgs(1),
		RunE: a.runInspectCommand,
	}

	// Flags
	cmd.Flags().Duration("history", 7*24*time.Hour, "show history for duration")
	cmd.Flags().Bool("show-related", false, "show related resources")
	cmd.Flags().String("format", "table", "output format (table, json, yaml)")

	return cmd
}

// newScanCommand creates the scan command
func (a *App) newScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Discover and collect infrastructure state",
		Long: `Scan discovers and collects the current state of your infrastructure
from various providers (Terraform, AWS, Kubernetes) and creates a snapshot.

This snapshot can be used as a baseline for future drift detection or 
compared against existing baselines to identify changes.`,
		Example: `  # Scan Terraform state
  wgo scan --provider terraform --path ./terraform

  # Scan AWS resources in multiple regions
  wgo scan --provider aws --region us-east-1,us-west-2

  # Scan Kubernetes cluster
  wgo scan --provider kubernetes --context prod --namespace default,kube-system

  # Scan all providers and save with custom name
  wgo scan --all --output-file my-snapshot.json`,
		RunE: a.runScanCommand,
	}

	// Flags
	cmd.Flags().StringP("provider", "p", "", "infrastructure provider (terraform, aws, kubernetes)")
	cmd.Flags().Bool("all", false, "scan all configured providers")
	cmd.Flags().StringSlice("region", []string{}, "AWS regions to scan (comma-separated)")
	cmd.Flags().String("path", ".", "path to Terraform files")
	cmd.Flags().StringSlice("context", []string{}, "Kubernetes contexts to scan")
	cmd.Flags().StringSlice("namespace", []string{}, "Kubernetes namespaces to scan")
	cmd.Flags().StringP("output-file", "o", "", "save snapshot to file")
	cmd.Flags().Bool("no-cache", false, "disable caching for this scan")
	cmd.Flags().Duration("timeout", 10*time.Minute, "scan timeout")

	return cmd
}

// newCheckCommand creates the check command
func (a *App) newCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Compare current state against baseline to detect drift",
		Long: `Check compares the current infrastructure state against a baseline
to detect configuration drift. It can automatically scan the current state
or use a previously captured snapshot for comparison.`,
		Example: `  # Check against latest baseline
  wgo check

  # Check against specific baseline
  wgo check --baseline prod-baseline-2025-01-15.json

  # Check with current scan and AI analysis
  wgo check --scan --explain

  # Check specific provider only
  wgo check --provider aws --region us-east-1`,
		RunE: a.runCheckCommand,
	}

	// Flags
	cmd.Flags().StringP("baseline", "b", "", "baseline file to compare against")
	cmd.Flags().Bool("scan", false, "perform current scan before comparison")
	cmd.Flags().StringP("provider", "p", "", "limit check to specific provider")
	cmd.Flags().StringSlice("region", []string{}, "limit check to specific regions")
	cmd.Flags().Bool("explain", false, "get AI analysis of detected drift")
	cmd.Flags().String("output-file", "", "save drift report to file")
	cmd.Flags().Float64("risk-threshold", 0.7, "minimum risk score to report (0.0-1.0)")

	return cmd
}

// newBaselineCommand creates the baseline command
func (a *App) newBaselineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Manage infrastructure baselines",
		Long: `Manage infrastructure baselines used for drift detection.
Baselines represent known good states of your infrastructure.`,
		Example: `  # Create baseline from current state
  wgo baseline create --name prod-v1.0 --description "Production baseline v1.0"

  # Create baseline from existing snapshot
  wgo baseline create --from-snapshot snapshot-123.json --name staging-v2.1

  # List all baselines
  wgo baseline list

  # Show baseline details
  wgo baseline show prod-v1.0

  # Delete baseline
  wgo baseline delete old-baseline`,
	}

	// Subcommands
	cmd.AddCommand(a.newBaselineCreateCommand())
	cmd.AddCommand(a.newBaselineListCommand())
	cmd.AddCommand(a.newBaselineShowCommand())
	cmd.AddCommand(a.newBaselineDeleteCommand())

	return cmd
}

// newBaselineCreateCommand creates the baseline create subcommand
func (a *App) newBaselineCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new baseline",
		RunE:  a.runBaselineCreateCommand,
	}

	cmd.Flags().StringP("name", "n", "", "baseline name (required)")
	cmd.Flags().StringP("description", "d", "", "baseline description")
	cmd.Flags().String("from-snapshot", "", "create baseline from existing snapshot")
	cmd.Flags().StringSlice("tags", []string{}, "baseline tags (key=value)")
	cmd.MarkFlagRequired("name")

	return cmd
}

// newBaselineListCommand creates the baseline list subcommand
func (a *App) newBaselineListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all baselines",
		RunE:  a.runBaselineListCommand,
	}
}

// newBaselineShowCommand creates the baseline show subcommand
func (a *App) newBaselineShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [baseline-id]",
		Short: "Show baseline details",
		Args:  cobra.ExactArgs(1),
		RunE:  a.runBaselineShowCommand,
	}

	return cmd
}

// newBaselineDeleteCommand creates the baseline delete subcommand
func (a *App) newBaselineDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [baseline-id]",
		Short: "Delete a baseline",
		Args:  cobra.ExactArgs(1),
		RunE:  a.runBaselineDeleteCommand,
	}

	cmd.Flags().Bool("force", false, "force deletion without confirmation")

	return cmd
}

// newExplainCommand creates the explain command
func (a *App) newExplainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Get AI analysis of infrastructure drift or changes",
		Long: `Get detailed AI-powered analysis and recommendations for infrastructure
drift, changes, or specific resources. Uses Claude AI to provide insights
about root causes, risks, and remediation steps.`,
		Example: `  # Explain latest drift report
  wgo explain

  # Explain specific drift report
  wgo explain --report drift-report-123.json

  # Explain changes in a resource
  wgo explain --resource ec2-instance-123 --provider aws

  # Get security-focused analysis
  wgo explain --focus security --report latest`,
		RunE: a.runExplainCommand,
	}

	// Flags
	cmd.Flags().String("report", "", "drift report file to analyze")
	cmd.Flags().String("resource", "", "specific resource to analyze")
	cmd.Flags().StringP("provider", "p", "", "resource provider")
	cmd.Flags().String("focus", "", "analysis focus (security, cost, performance)")
	cmd.Flags().String("output-file", "", "save analysis to file")

	return cmd
}

// newCacheCommand creates the cache command
func (a *App) newCacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cache operations",
		Long:  `Manage WGO's caching system for improved performance.`,
	}

	// Subcommands
	cmd.AddCommand(a.newCacheStatsCommand())
	cmd.AddCommand(a.newCacheClearCommand())
	cmd.AddCommand(a.newCacheWarmCommand())

	return cmd
}

// newCacheStatsCommand creates the cache stats subcommand
func (a *App) newCacheStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show cache statistics",
		RunE:  a.runCacheStatsCommand,
	}
}

// newCacheClearCommand creates the cache clear subcommand
func (a *App) newCacheClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear cache",
		RunE:  a.runCacheClearCommand,
	}

	cmd.Flags().Bool("all", false, "clear all cache types")
	cmd.Flags().Bool("memory", false, "clear memory cache")
	cmd.Flags().Bool("disk", false, "clear disk cache")

	return cmd
}

// newCacheWarmCommand creates the cache warm subcommand
func (a *App) newCacheWarmCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "warm",
		Short: "Warm up cache with frequently used data",
		RunE:  a.runCacheWarmCommand,
	}
}

// newVersionCommand creates the version command
func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("WGO version %s (%s) built on %s\n", 
				a.config.Version, a.config.Commit, a.config.BuildDate)
		},
	}
}

// CreateRootCommand creates and returns the root command with all subcommands
func (a *App) CreateRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "wgo",
		Short: "Fast infrastructure status and visibility tool",
		Long: `WGO (What's Going On) gives you instant answers about your infrastructure.
Get quick status, see what's running, spot what's changed, and understand what needs attention.

Perfect for DevOps teams who need fast insights: "What's cooking in my infra right now?"`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize viper configuration
			configFile := viper.GetString("config")
			if configFile != "" {
				viper.SetConfigFile(configFile)
			} else {
				home, err := os.UserHomeDir()
				if err == nil {
					viper.AddConfigPath(home + "/.wgo")
					viper.SetConfigName("config")
					viper.SetConfigType("yaml")
				}
			}
			
			viper.AutomaticEnv()
			
			if err := viper.ReadInConfig(); err == nil && a.config.Verbose {
				fmt.Println("Using config file:", viper.ConfigFileUsed())
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.wgo/config.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")

	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

	// Add commands (status-first approach)
	rootCmd.AddCommand(a.newStatusCommand())     // Primary: "What's my status right now?"
	rootCmd.AddCommand(a.newWatchCommand())      // Live monitoring
	rootCmd.AddCommand(a.newInspectCommand())    // Deep dive into resources
	rootCmd.AddCommand(a.newScanCommand())       // Discovery & collection
	rootCmd.AddCommand(a.newCheckCommand())      // Drift detection
	rootCmd.AddCommand(a.newBaselineCommand())   // Baseline management
	rootCmd.AddCommand(a.newExplainCommand())    // AI analysis
	rootCmd.AddCommand(a.newCacheCommand())      // Cache management
	rootCmd.AddCommand(a.newVersionCommand())    // Version info

	return rootCmd
}