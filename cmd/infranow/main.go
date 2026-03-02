package main

import (
	"fmt"
	"os"

	"github.com/ppiankov/infranow/internal/cli"
	"github.com/ppiankov/infranow/internal/util"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := cli.NewRootCommand(version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		util.Exit(util.ExitRuntimeError)
	}
}
