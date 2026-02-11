package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool
	version    string // Stored for use in subcommands
)

// NewRootCommand creates the root command for infranow
func NewRootCommand(ver, commit, date string) *cobra.Command {
	version = ver // Store for subcommands
	rootCmd := &cobra.Command{
		Use:   "infranow",
		Short: "Attention-first infrastructure monitoring",
		Long: `infranow is a CLI/TUI tool that consumes Prometheus metrics and
deterministically identifies the most important infrastructure problems right now.

It prioritizes silence when systems are healthy and surfaces only ranked,
actionable problems when intervention is required.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: $HOME/.infranow.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// Add subcommands
	rootCmd.AddCommand(NewMonitorCommand())
	rootCmd.AddCommand(newVersionCommand())

	return rootCmd
}
