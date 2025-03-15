package osext

import (
	"os"

	"golang.org/x/term"
)

// IsInteractive returns true if the program is running in the interactive
// terminal.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd())) && os.Getenv("TERM") != "dumb"
}
