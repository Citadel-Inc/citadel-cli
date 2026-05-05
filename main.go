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
	root := &cobra.Command{
		Use:   "citadel-cli",
		Short: "Citadel CLI — authentication, token, and MCP agent interface",
		Long: `citadel-cli is the command-line client for managing authentication, agent tokens,
and MCP tool interactions against the Citadel server (the server binary is "citadel", cmd/citadel).

Server URL defaults to https://api.src.land; override with CITADEL_SERVER or --server.`,
		Version: "0.0.1-alpha",
		// Did-you-mean: cobra emits "Did you mean ...?" hints on unknown
		// subcommand. Distance 2 catches typos like `rpo` → `repo` while
		// staying tight enough to avoid noise on genuinely-new verbs.
		SuggestionsMinimumDistance: 2,
	}

	// --server flag + CITADEL_SERVER env var. Persistent so every subcommand
	// inherits it. Resolved against cfg.ServerURL by clicfg.ResolveServerURL().
	root.PersistentFlags().String("server", "", "Server URL (overrides CITADEL_SERVER env and stored config)")
	root.PersistentFlags().BoolP("verbose", "v", false, "Print one METHOD URL → STATUS line per HTTP call to stderr")
	root.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential stderr output (overrides --verbose)")
	root.PersistentFlags().Bool("debug-http", false, "Dump full HTTP request/response (Authorization redacted) to stderr")
	root.PersistentFlags().String("color", "auto", "Color output: auto|always|never (honors NO_COLOR)")
	root.PersistentFlags().Bool("no-pager", false, "Do not pipe stdout through $PAGER (CITADEL_PAGER > GIT_PAGER > PAGER > less -FRX)")

	root.AddCommand(cmd.AuthCmd)
	root.AddCommand(cmd.TokenCmd)
	root.AddCommand(cmd.McpCmd)
	root.AddCommand(cmd.KgCmd)
	root.AddCommand(cmd.RepoCmd)
	root.AddCommand(cmd.NamespaceCmd)
	root.AddCommand(cmd.AgentCmd)
	root.AddCommand(cmd.OauthCmd)
	root.AddCommand(cmd.CompletionCmd)
	root.AddCommand(cmd.DoctorCmd)
	root.AddCommand(cmd.ManCmd)

	return root
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
		return a == "--no-pager" || a == "completion" || a == "--help" || a == "-h" || a == "--version"
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
