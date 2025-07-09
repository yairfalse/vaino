package main

import "github.com/yairfalse/wgo/cmd/wgo/commands"

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	builtBy   = "unknown"
)

func main() {
	commands.SetVersionInfo(version, commit, buildTime, builtBy)
	commands.Execute()
}
