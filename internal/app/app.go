package app

import (
	"github.com/spf13/cobra"
	"github.com/yairfalse/wgo/internal/cache"
	"github.com/yairfalse/wgo/internal/collectors"
	"github.com/yairfalse/wgo/internal/logger"
	"github.com/yairfalse/wgo/internal/storage"
)

type Config struct {
	Verbose   bool
	Debug     bool
	Version   string
	Commit    string
	BuildDate string
}

type App struct {
	config   Config
	storage  storage.Storage
	cache    cache.Manager
	logger   logger.Logger
	registry *collectors.CollectorRegistry
}

func (a *App) GetCommands() []*cobra.Command {
	return []*cobra.Command{
		a.createVersionCommand(),
		a.createStatusCommand(),
		a.createScanCommand(),
		a.createCheckCommand(),
		a.createDiffCommand(),
		a.createBaselineCommand(),
		a.createExplainCommand(),
		a.createCacheCommand(),
		a.createConfigCommand(),
		a.createSetupCommand(),
	}
}

func (a *App) createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of wgo",
		Long:  `All software has versions. This is wgo's`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runVersionCommand(cmd, args)
		},
	}
}

func (a *App) createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show infrastructure status",
		Long:  `Display current infrastructure status and health`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runStatusCommand(cmd, args)
		},
	}
}

func (a *App) createScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan infrastructure for changes",
		Long:  `Scan your infrastructure providers for changes`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runScanCommand(cmd, args)
		},
	}
}

func (a *App) createCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check for infrastructure drift",
		Long:  `Check for drift between current state and baseline`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runCheckCommand(cmd, args)
		},
	}
}

func (a *App) createBaselineCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "baseline",
		Short: "Manage infrastructure baselines",
		Long:  `Create and manage infrastructure baselines`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runBaselineCommand(cmd, args)
		},
	}
}

func (a *App) createExplainCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "explain",
		Short: "Explain infrastructure changes",
		Long:  `Get AI-powered explanations of infrastructure changes`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runExplainCommand(cmd, args)
		},
	}
}

func (a *App) createCacheCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cache",
		Short: "Manage cache",
		Long:  `Manage the application cache`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runCacheCommand(cmd, args)
		},
	}
}

func (a *App) createConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and manage wgo configuration`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runConfigCommand(cmd, args)
		},
	}
}

func (a *App) createDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare infrastructure states (like 'git diff' for infrastructure)",
		Long: `Compare infrastructure states to see what changed - just like 'git diff' but for your infrastructure.

Shows changes in a familiar, easy-to-read format that works great with Unix tools.

QUICK START:
  wgo diff              # See what changed (git diff style)
  wgo diff --name-only  # Just list what changed
  wgo diff --stat       # Show change summary
  wgo diff --quiet      # Silent mode for scripts`,
		Example: `  # See infrastructure changes (like git diff)
  wgo diff

  # List just the names of changed resources
  wgo diff --name-only

  # Show statistics about changes
  wgo diff --stat

  # Silent mode - check exit code (0=no changes, 1=changes found)
  wgo diff --quiet && echo "All good!" || echo "Changes detected!"

  # Use in scripts and pipelines
  if ! wgo diff --quiet; then
    echo "⚠️  Infrastructure drift detected!"
    wgo diff --stat
  fi`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runDiffCommand(cmd, args)
		},
	}

	// Unix-style flags with clear descriptions
	cmd.Flags().Bool("name-only", false, "show only names of changed resources (like 'git diff --name-only')")
	cmd.Flags().Bool("stat", false, "show change statistics (like 'git diff --stat')")
	cmd.Flags().BoolP("quiet", "q", false, "silent mode - no output, just exit code (0=no changes, 1=changes)")
	cmd.Flags().String("format", "", "output format: unix (default), simple, name-only, stat")

	return cmd
}

func (a *App) createSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Quick setup and auto-configuration",
		Long:  `Automatically detect and configure infrastructure providers`,
		Run: func(cmd *cobra.Command, args []string) {
			a.runSetupCommand(cmd, args)
		},
	}
}