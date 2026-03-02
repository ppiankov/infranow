package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(info BuildInfo) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "infranow %s (commit: %s, built: %s, go: %s)\n",
				info.Version, info.Commit, info.Date, info.GoVersion)
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output version as JSON")

	return cmd
}
