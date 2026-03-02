package util

import (
	"fmt"
	"os"
)

// Standard exit codes following Spectre Tools conventions
const (
	ExitSuccess          = 0 // Command succeeded, no problems
	ExitProblemsWarning  = 1 // Only WARNING-level problems found
	ExitProblemsCritical = 2 // CRITICAL or FATAL problems found
	ExitInvalidInput     = 3 // Invalid user input or configuration
	ExitRuntimeError     = 4 // Runtime error (connection failure, etc.)
)

// Exit terminates the program with the given exit code
func Exit(code int) {
	os.Exit(code)
}

// ExitWithError prints an error message and exits with the given code
func ExitWithError(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(code)
}
