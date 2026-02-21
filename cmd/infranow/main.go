package main

import (
	"fmt"
	"os"

	"github.com/ppiankov/infranow/internal/cli"
	"github.com/ppiankov/infranow/internal/util"
)

var (
	// Version information (set via ldflags during build)
	version = "0.1.2"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	rootCmd := cli.NewRootCommand(version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		util.Exit(util.ExitRuntimeError)
	}
}
