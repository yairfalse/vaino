package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yairfalse/vaino/internal/errors"
	"github.com/yairfalse/vaino/pkg/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "vaino",
	Short:             "Infrastructure drift detection and monitoring",
	Long:              `vaino - infrastructure drift detection and monitoring`,
	DisableAutoGenTag: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
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
	// Set custom error handler for Cobra
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true

	err := rootCmd.Execute()
	if err != nil {
		// Use enhanced error display
		errors.DisplayError(err)

		// Use appropriate exit code based on error type
		exitCode := errors.GetExitCode(err)
		os.Exit(exitCode)
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

	// Essential commands only
	rootCmd.AddCommand(newScanCommand())
	rootCmd.AddCommand(newDiffCommand())
	rootCmd.AddCommand(newWatchCommand())
	rootCmd.AddCommand(newStatusCommand())
	rootCmd.AddCommand(newVersionCommand())
	rootCmd.AddCommand(newConfigureCommand())
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
