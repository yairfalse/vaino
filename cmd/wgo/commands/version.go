package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
	BuiltBy   = "unknown"
)

// SetVersionInfo updates the version variables with build-time information
func SetVersionInfo(version, commit, buildTime, builtBy string) {
	if version != "" {
		Version = version
	}
	if commit != "" {
		Commit = commit
	}
	if buildTime != "" {
		BuildTime = buildTime
	}
	if builtBy != "" {
		BuiltBy = builtBy
	}
}

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display version information for WGO including build details.`,
		Run:   runVersion,
	}

	cmd.Flags().Bool("short", false, "show only version number")

	return cmd
}

func runVersion(cmd *cobra.Command, args []string) {
	short, _ := cmd.Flags().GetBool("short")

	if short {
		fmt.Println(Version)
		return
	}

	fmt.Printf("WGO (What's Going On) version %s\n", Version)
	fmt.Printf("  commit: %s\n", Commit)
	fmt.Printf("  built: %s\n", BuildTime)
	fmt.Printf("  built by: %s\n", BuiltBy)
	fmt.Println()
	fmt.Println("Infrastructure drift detection tool")
	fmt.Println("https://github.com/yairfalse/wgo")
}
