package monitor

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// copyToClipboard writes text to the system clipboard.
// Returns a user-facing status message.
func copyToClipboard(text string) string {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy") //nolint:gosec // no user input in command name
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard") //nolint:gosec // no user input
	default:
		return "clipboard not supported on " + runtime.GOOS
	}

	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("clipboard error: %v", err)
	}
	return "Copied to clipboard"
}
