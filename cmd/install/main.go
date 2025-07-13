package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yairfalse/vaino/cmd/install/installer"
	"github.com/yairfalse/vaino/cmd/install/platform"
	"github.com/yairfalse/vaino/cmd/install/progress"
)

var (
	// Version is set during build
	Version = "dev"
	// BuildTime is set during build
	BuildTime = "unknown"
)

var (
	installMethod   string
	installDir      string
	version         string
	mirrors         []string
	timeout         time.Duration
	retryAttempts   int
	debug           bool
	validationLevel string
	noProgress      bool
	resume          bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "install",
	Short: "Tapio installer - Enterprise-grade installation system",
	Long: `Tapio installer provides a sophisticated, plugin-based installation system
with support for multiple installation methods including binary, container, and Kubernetes deployments.

Features:
- Automatic platform detection
- Resumable downloads with progress tracking
- Atomic operations with rollback support
- Comprehensive validation and health checks
- Circuit breaker for network resilience`,
	RunE: runInstall,
}

func init() {
	rootCmd.Flags().StringVarP(&installMethod, "method", "m", "auto", "Installation method (auto, binary, container, kubernetes)")
	rootCmd.Flags().StringVarP(&installDir, "dir", "d", "", "Installation directory (defaults to system-specific)")
	rootCmd.Flags().StringVarP(&version, "version", "v", "latest", "Version to install")
	rootCmd.Flags().StringSliceVar(&mirrors, "mirrors", nil, "Download mirrors to use")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Minute, "Installation timeout")
	rootCmd.Flags().IntVar(&retryAttempts, "retry", 3, "Number of retry attempts")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
	rootCmd.Flags().StringVar(&validationLevel, "validation", "full", "Validation level (basic, full)")
	rootCmd.Flags().BoolVar(&noProgress, "no-progress", false, "Disable progress indicators")
	rootCmd.Flags().BoolVar(&resume, "resume", false, "Resume interrupted installation")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(rollbackCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show installer version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Tapio Installer %s (built %s)\n", Version, BuildTime)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Tapio",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("uninstall not yet implemented")
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Tapio installation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("validate not yet implemented")
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback failed installation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("rollback not yet implemented")
	},
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nInstallation interrupted. Cleaning up...")
		cancel()
	}()

	// Apply timeout if specified
	if timeout > 0 {
		var timeoutCancel context.CancelFunc
		ctx, timeoutCancel = context.WithTimeout(ctx, timeout)
		defer timeoutCancel()
	}

	// Create installer configuration
	config := &installer.Config{
		Method:          installMethod,
		InstallDir:      installDir,
		Version:         version,
		Mirrors:         mirrors,
		Timeout:         timeout,
		RetryAttempts:   retryAttempts,
		Debug:           debug,
		ValidationLevel: validationLevel,
	}

	// Setup progress tracking
	var progressTracker progress.Tracker
	if !noProgress {
		progressTracker = progress.NewTerminalTracker(os.Stdout)
	} else {
		progressTracker = progress.NewSilentTracker()
	}

	// Detect platform and create appropriate installer
	detector := platform.NewDetector()
	platformInfo, err := detector.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect platform: %w", err)
	}

	if debug {
		fmt.Printf("Detected platform: %s/%s\n", platformInfo.OS, platformInfo.Arch)
	}

	// Create installer factory
	factory := installer.NewFactory(platformInfo)

	// Auto-detect installation method if needed
	if installMethod == "auto" {
		strategy, err := factory.Detect(ctx)
		if err != nil {
			return fmt.Errorf("failed to detect installation method: %w", err)
		}
		config.Method = strategy.Name()
		fmt.Printf("Auto-detected installation method: %s\n", config.Method)
	}

	// Create installer with functional options
	installerInstance, err := factory.Create(
		installer.WithConfig(config),
		installer.WithProgressTracker(progressTracker),
		installer.WithStateManager(installer.NewFileStateManager()),
	)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	// Check for resumable installation
	if resume {
		stateManager := installer.NewFileStateManager()
		state, err := stateManager.LoadState(ctx)
		if err == nil && state.ID != "" {
			fmt.Printf("Resuming installation from step: %s\n", state.Steps[len(state.Steps)-1].StepName)
			// Resume logic would go here
		}
	}

	// Start installation
	fmt.Printf("Installing Tapio %s using %s method...\n", version, config.Method)

	// Monitor progress in a separate goroutine
	progressChan := installerInstance.Progress()
	go func() {
		for progress := range progressChan {
			if !noProgress {
				progressTracker.Update(progress)
			}
		}
	}()

	// Perform installation
	if err := installerInstance.Install(ctx); err != nil {
		// Attempt rollback on failure
		fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
		fmt.Println("Attempting rollback...")

		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer rollbackCancel()

		if rollbackErr := installerInstance.Rollback(rollbackCtx); rollbackErr != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", rollbackErr)
		} else {
			fmt.Println("Rollback completed successfully")
		}
		return err
	}

	// Validate installation
	fmt.Println("Validating installation...")
	if err := installerInstance.Validate(ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println("✓ Tapio installed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add Tapio to your PATH (if not done automatically)")
	fmt.Println("  2. Run 'tapio --help' to get started")
	fmt.Println("  3. Run 'tapio configure' to set up your environment")

	return nil
}
