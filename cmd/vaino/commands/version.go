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
		Short: "Show version of the Finnish creator god",
		Long:  `Display version information for VAINO including divine build details.`,
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

	fmt.Printf("VAINO - The Finnish Creator God version %s\n", Version)
	fmt.Printf("  forged: %s\n", Commit)
	fmt.Printf("  blessed: %s\n", BuildTime)
	fmt.Printf("  crafted by: %s\n", BuiltBy)
	fmt.Println()
	fmt.Println("🌲 Ancient Finnish wisdom for modern infrastructure")
	fmt.Println("🔥 The creator god who actually BUILDS things!")
	fmt.Println("https://github.com/yairfalse/vaino")
}
