package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	verbose   bool
	format    string
)

var rootCmd = &cobra.Command{
	Use:   "wgo",
	Short: "AI-powered infrastructure drift detection",
	Long: `WGO is an AI-powered infrastructure drift detection CLI tool.

It helps you discover, baseline, and monitor infrastructure changes across 
your environment, providing intelligent analysis of detected drift to help 
maintain infrastructure consistency and compliance.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildTime),
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.wgo/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "output format (text, json, yaml)")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding verbose flag: %v\n", err)
	}
	if err := viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding format flag: %v\n", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		wgoDir := filepath.Join(home, ".wgo")
		if err := os.MkdirAll(wgoDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .wgo directory: %v\n", err)
		}

		viper.AddConfigPath(wgoDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("WGO")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
	}
}