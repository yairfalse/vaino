package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yairfalse/wgo/internal/app"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	builtBy   = "unknown"

	configFile string
	verbose    bool
	debug      bool
)

var rootCmd = &cobra.Command{
	Use:   "wgo",
	Short: "A powerful infrastructure drift detection tool",
	Long: `wgo is a tool for detecting and managing infrastructure drift.
It helps you track changes in your infrastructure over time and identify
discrepancies between expected and actual states.`,
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if configFile != "" {
			viper.SetConfigFile(configFile)
		} else {
			home, err := os.UserHomeDir()
			if err == nil {
				viper.AddConfigPath(filepath.Join(home, ".wgo"))
				viper.SetConfigName("config")
				viper.SetConfigType("yaml")
			}
		}

		viper.AutomaticEnv()
		viper.SetEnvPrefix("WGO")

		if err := viper.ReadInConfig(); err == nil {
			if verbose {
				fmt.Println("Using config file:", viper.ConfigFileUsed())
			}
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.wgo/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")

	// Create app factory
	factory := app.NewAppFactory()

	// Create app configuration
	config := app.Config{
		Verbose:   verbose,
		Debug:     debug,
		Version:   version,
		Commit:    commit,
		BuildDate: buildTime,
	}

	// Create app instance
	appInstance, err := factory.Create(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create app: %v\n", err)
		os.Exit(1)
	}

	// Add commands from app
	rootCmd.AddCommand(appInstance.GetCommands()...)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}