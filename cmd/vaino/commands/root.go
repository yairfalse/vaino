package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yairfalse/vaino/pkg/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vaino",
	Short: "Infrastructure drift detection and monitoring",
	Long: `VAINO - Infrastructure drift detection and monitoring tool.

Named after Väinö from Finnish mythology.

VAINO helps you detect and monitor infrastructure changes across multiple 
cloud providers and Infrastructure as Code tools. Think of it as "git diff" 
for your infrastructure.

CORE FEATURES:
- Multi-provider support (AWS, GCP, Kubernetes, Terraform)
- Real-time drift detection
- Unix-style output for automation
- Multiple output formats (JSON, YAML, table, markdown)
- Continuous monitoring capabilities

QUICK START:
  vaino scan              # Scan your infrastructure
  vaino diff              # Show changes since last scan
  vaino diff --stat       # Show change statistics
  vaino diff --quiet      # Silent mode for scripting

SUPPORTED PROVIDERS:
  Terraform, AWS, Kubernetes, GCP`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --version flag
		if showVersion, _ := cmd.Flags().GetBool("version"); showVersion {
			runVersion(cmd, []string{})
			return nil
		}
		// If no subcommand is provided and no --version flag, show help
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vaino/config.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("output", "table", "output format (table, json, yaml, markdown)")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	rootCmd.PersistentFlags().Bool("version", false, "show version information")

	// Bind flags to viper
	viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("output.format", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("output.no_color", rootCmd.PersistentFlags().Lookup("no-color"))

	// Add subcommands
	rootCmd.AddCommand(newScanCommand())
	rootCmd.AddCommand(newCheckCommand())
	rootCmd.AddCommand(newExplainCommand())
	rootCmd.AddCommand(newDiffCommand())
	rootCmd.AddCommand(newSimpleDiffCommand()) // New simple changes command
	rootCmd.AddCommand(newWatchCommand())      // Real-time watch mode
	rootCmd.AddCommand(catchUpCmd)             // Empathetic catch-up summary
	rootCmd.AddCommand(newTimelineCommand())   // Timeline view of snapshots
	rootCmd.AddCommand(newHistoryCommand())    // History browsing
	rootCmd.AddCommand(newAuthCommand())
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newConfigureCommand())   // Configuration wizard
	rootCmd.AddCommand(newStatusCommand())      // System status
	rootCmd.AddCommand(newCheckConfigCommand()) // Configuration validation
	rootCmd.AddCommand(newHelpCommand())        // Help topics
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Expand paths like ~ to home directory
	if err := cfg.ExpandPaths(); err != nil {
		return fmt.Errorf("failed to expand config paths: %w", err)
	}

	return nil
}

// GetConfig returns the loaded configuration
func GetConfig() *config.Config {
	return cfg
}
