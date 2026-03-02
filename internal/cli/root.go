package cli

import (
	"runtime"

	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool
	version    string // stored for use in subcommands (baseline metadata)
)

// BuildInfo holds version and build metadata.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"goVersion"`
}

// NewRootCommand creates the root command for infranow
func NewRootCommand(ver, commit, date string) *cobra.Command {
	version = ver
	info := BuildInfo{
		Version:   ver,
		Commit:    commit,
		Date:      date,
		GoVersion: runtime.Version(),
	}

	rootCmd := &cobra.Command{
		Use:   "infranow",
		Short: "Attention-first infrastructure monitoring",
		Long: `infranow is a CLI/TUI tool that consumes Prometheus metrics and
deterministically identifies the most important infrastructure problems right now.

It prioritizes silence when systems are healthy and surfaces only ranked,
actionable problems when intervention is required.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default: $HOME/.infranow.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// Add subcommands
	rootCmd.AddCommand(NewMonitorCommand())
	rootCmd.AddCommand(newVersionCommand(info))

	return rootCmd
}
