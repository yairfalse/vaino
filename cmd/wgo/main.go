package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yairfalse/wgo/internal/ai"
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

		if err := viper.ReadInConfig(); err == nil {
			if verbose {
				fmt.Println("Using config file:", viper.ConfigFileUsed())
			}
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of wgo",
	Long:  `All software has versions. This is wgo's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wgo version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", buildTime)
		fmt.Printf("  built by: %s\n", builtBy)
	},
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage infrastructure snapshots",
	Long:  `Create, list, and manage snapshots of your infrastructure state`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Snapshot command - use 'wgo snapshot --help' for subcommands")
	},
}

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect infrastructure drift",
	Long:  `Compare snapshots and detect drift in your infrastructure`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Drift command - use 'wgo drift --help' for subcommands")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage wgo configuration",
	Long:  `View and manage wgo configuration settings`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Configuration Help:")
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  ANTHROPIC_API_KEY    Required for AI-powered commands (analyze, explain, remediate)")
		fmt.Println("")
		fmt.Println("Config File (~/.wgo/config.yaml):")
		fmt.Println("  verbose: true/false")
		fmt.Println("  debug: true/false")
		fmt.Println("  default_snapshot_dir: /path/to/snapshots")
		fmt.Println("")
		fmt.Println("AI Commands:")
		fmt.Println("  wgo analyze [file]    - AI-powered drift analysis")
		fmt.Println("  wgo explain [file]    - Natural language explanations")
		fmt.Println("  wgo remediate [file]  - Generate remediation steps")
		fmt.Println("")
		fmt.Println("To get your Claude API key, visit: https://console.anthropic.com/")
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [file]",
	Short: "AI-powered analysis of infrastructure drift",
	Long:  `Use Claude AI to analyze infrastructure drift data and provide insights, risk assessment, and recommendations`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		claudeClient, err := ai.NewClaudeClient()
		if err != nil {
			return fmt.Errorf("failed to initialize Claude client: %w", err)
		}

		var driftData string
		if len(args) == 1 {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", args[0], err)
			}
			driftData = string(content)
		} else {
			fmt.Print("Enter drift data (press Ctrl+D when done):\n")
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			driftData = string(content)
		}

		if strings.TrimSpace(driftData) == "" {
			return fmt.Errorf("no drift data provided")
		}

		fmt.Println("🤖 Analyzing drift data with Claude AI...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		analysis, err := claudeClient.AnalyzeDrift(ctx, driftData)
		if err != nil {
			return fmt.Errorf("failed to analyze drift: %w", err)
		}

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("🔍 DRIFT ANALYSIS RESULTS")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println(analysis)
		fmt.Println(strings.Repeat("=", 80))

		return nil
	},
}

var explainCmd = &cobra.Command{
	Use:   "explain [file]",
	Short: "Get natural language explanations of infrastructure changes",
	Long:  `Use Claude AI to explain infrastructure changes in simple, understandable terms`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		claudeClient, err := ai.NewClaudeClient()
		if err != nil {
			return fmt.Errorf("failed to initialize Claude client: %w", err)
		}

		var changeData string
		if len(args) == 1 {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", args[0], err)
			}
			changeData = string(content)
		} else {
			fmt.Print("Enter change data (press Ctrl+D when done):\n")
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			changeData = string(content)
		}

		if strings.TrimSpace(changeData) == "" {
			return fmt.Errorf("no change data provided")
		}

		fmt.Println("🤖 Explaining changes with Claude AI...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		explanation, err := claudeClient.ExplainChange(ctx, changeData)
		if err != nil {
			return fmt.Errorf("failed to explain change: %w", err)
		}

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("📝 CHANGE EXPLANATION")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println(explanation)
		fmt.Println(strings.Repeat("=", 80))

		return nil
	},
}

var remediateCmd = &cobra.Command{
	Use:   "remediate [file]",
	Short: "Generate AI-powered remediation steps for drift",
	Long:  `Use Claude AI to generate specific remediation steps and commands to fix infrastructure drift`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		claudeClient, err := ai.NewClaudeClient()
		if err != nil {
			return fmt.Errorf("failed to initialize Claude client: %w", err)
		}

		var driftData string
		if len(args) == 1 {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", args[0], err)
			}
			driftData = string(content)
		} else {
			fmt.Print("Enter drift data (press Ctrl+D when done):\n")
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			driftData = string(content)
		}

		if strings.TrimSpace(driftData) == "" {
			return fmt.Errorf("no drift data provided")
		}

		fmt.Println("🤖 Generating remediation steps with Claude AI...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		remediation, err := claudeClient.GenerateRemediation(ctx, driftData)
		if err != nil {
			return fmt.Errorf("failed to generate remediation: %w", err)
		}

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("🔧 REMEDIATION STEPS")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println(remediation)
		fmt.Println(strings.Repeat("=", 80))

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.wgo/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(driftCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(remediateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}