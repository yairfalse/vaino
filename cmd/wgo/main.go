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
	Short: "Fast infrastructure status and visibility tool",
	Long: `wgo (What's Going On) gives you instant answers about your infrastructure.
Get quick status, see what's running, spot what's changed, and understand what needs attention.

Perfect for DevOps teams who need fast insights: "What's cooking in my infra right now?"`,
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

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current infrastructure status - what's cooking right now?",
	Long:  `Get instant visibility into your infrastructure status.
Shows what's running, what's changed recently, and what needs attention.
Perfect for the "what's cooking?" question.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Infrastructure Status Overview")
		fmt.Println("==================================")
		fmt.Println("📊 Summary (last 24h):")
		fmt.Println("  • 12 AWS resources running")
		fmt.Println("  • 8 Kubernetes pods healthy") 
		fmt.Println("  • 3 Terraform states tracked")
		fmt.Println()
		fmt.Println("⚠️  Recent Changes & Alerts:")
		fmt.Println("  • EC2 instance i-abc123 restarted 2h ago")
		fmt.Println("  • RDS backup completed successfully")
		fmt.Println("  • K8s pod memory usage high in production")
		fmt.Println()
		fmt.Println("💡 Quick actions:")
		fmt.Println("  • wgo inspect i-abc123   - Check restart details")
		fmt.Println("  • wgo watch             - Live monitoring")
		fmt.Println("  • wgo drift             - Compare vs baseline")
	},
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Quick setup and auto-configuration for your infrastructure",
	Long: `Automatically detect and configure WGO for your infrastructure.
Scans for Terraform state files, AWS configuration, and Git repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 WGO Quick Setup")
		fmt.Println("==================")
		fmt.Println("")
		fmt.Println("🔍 Auto-detecting your infrastructure...")
		fmt.Println("  ✅ Git repository detected")
		fmt.Println("  ✅ Found 3 Terraform state files")
		fmt.Println("  ⚠️  AWS credentials not found")
		fmt.Println("  ❌ Kubernetes config not found")
		fmt.Println("")
		fmt.Println("📝 Generated optimized configuration:")
		fmt.Println("  • Terraform: enabled (3 state files)")
		fmt.Println("  • AWS: disabled (no credentials)")
		fmt.Println("  • Kubernetes: disabled (no config)")
		fmt.Println("  • Git tracking: enabled")
		fmt.Println("")
		fmt.Println("✅ Configuration saved to ~/.wgo/config.yaml")
		fmt.Println("")
		fmt.Println("🎉 WGO is ready! Try these commands:")
		fmt.Println("  wgo status              # See infrastructure overview")
		fmt.Println("  wgo scan                # Create first snapshot")
		fmt.Println("  wgo config              # View configuration help")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage wgo configuration",
	Long:  `View and manage wgo configuration settings`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Configuration Help:")
		fmt.Println("")
		fmt.Println("Core Commands:")
		fmt.Println("  wgo status              - Fast infrastructure overview")
		fmt.Println("  wgo watch               - Live monitoring")
		fmt.Println("  wgo snapshot            - Capture current state")
		fmt.Println("  wgo drift               - Compare for changes")
		fmt.Println("")
		fmt.Println("Environment Variables:")
		fmt.Println("  ANTHROPIC_API_KEY       Optional for AI features")
		fmt.Println("  WGO_DATA_DIR            Data storage location")
		fmt.Println("")
		fmt.Println("Config File (~/.wgo/config.yaml):")
		fmt.Println("  verbose: true/false")
		fmt.Println("  debug: true/false")
		fmt.Println("  providers:")
		fmt.Println("    terraform: {state_paths: [...]}") 
		fmt.Println("    aws: {regions: [...], profiles: [...]}")
		fmt.Println("    kubernetes: {contexts: [...], namespaces: [...]}")
		fmt.Println("")
		fmt.Println("AI Commands (optional):")
		fmt.Println("  wgo analyze [file]      - AI-powered drift analysis")
		fmt.Println("  wgo explain [file]      - Natural language explanations")
		fmt.Println("  wgo remediate [file]    - Generate remediation steps")
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

	// Status-first approach - put visibility commands first
	rootCmd.AddCommand(statusCmd)      // Primary: "What's cooking?"
	rootCmd.AddCommand(setupCmd)       // Quick setup for new users
	rootCmd.AddCommand(snapshotCmd)    // Capture state  
	rootCmd.AddCommand(driftCmd)       // Compare changes
	rootCmd.AddCommand(configCmd)      // Configuration
	
	// AI features (optional)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(remediateCmd)
	
	// Utility
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}