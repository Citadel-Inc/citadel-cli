package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
	"github.com/Rethunk-Tech/citadel-cli/internal/pager"
	"github.com/spf13/cobra"
)

// newRootCmd builds the citadel-cli cobra root with every subcommand wired in.
// Exposed so tests can drive the same tree main() ships.
func newRootCmd() *cobra.Command {
	return cmd.NewRootCmd()
}

// run executes the CLI with the given args and returns the process exit code.
// Stays in package main for testability — main() is then a tiny shim.
func run(args []string, stderr io.Writer) int {
	root := newRootCmd()
	root.SetArgs(args)

	// Pre-parse flags to honor --no-pager / CITADEL_NO_PAGER before any
	// subcommand writes to stdout. Cheap, idempotent: cobra parses again
	// during Execute. Skipped when args clearly target a non-paging path
	// (--help / --version / completion script generation).
	disablePager := slices.ContainsFunc(args, func(a string) bool {
		return a == "--no-pager" || a == "completion" || a == "__complete" || a == "--help" || a == "-h" || a == "--version"
	}) || os.Getenv("CITADEL_NO_PAGER") != ""
	cleanup, _ := pager.Start(disablePager)
	defer cleanup()

	if err := root.Execute(); err != nil {
		// ErrToolCallFailed is the sentinel signaling tools/call returned
		// isError=true; the result has already been printed, so suppress
		// the duplicate "Error:" line and exit with code 2.
		if errors.Is(err, cmd.ErrToolCallFailed) {
			return 2
		}
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", cmd.FriendlyError(err))
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}
