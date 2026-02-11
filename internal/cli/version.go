package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show infranow version information",
		Long:  `Display version information for infranow including Go version and platform details.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("infranow version %s\n", version)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
}
