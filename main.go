package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "citadel-cli",
		Short: "Citadel CLI — authentication, token, and MCP agent interface",
		Long: `citadel-cli is the command-line client for managing authentication, agent tokens,
and MCP tool interactions against the Citadel server (the server binary is "citadel", cmd/citadel).

Server URL defaults to https://api.src.land; override with CITADEL_SERVER or --server.`,
		Version: "0.0.1-alpha",
	}

	// --server flag + CITADEL_SERVER env var. Persistent so every subcommand
	// inherits it. Resolved against cfg.ServerURL by clicfg.ResolveServerURL().
	root.PersistentFlags().String("server", "", "Server URL (overrides CITADEL_SERVER env and stored config)")

	root.AddCommand(cmd.AuthCmd)
	root.AddCommand(cmd.TokenCmd)
	root.AddCommand(cmd.McpCmd)
	root.AddCommand(cmd.KgCmd)
	root.AddCommand(cmd.RepoCmd)
	root.AddCommand(cmd.NamespaceCmd)
	root.AddCommand(cmd.AgentCmd)
	root.AddCommand(cmd.OauthCmd)

	if err := root.Execute(); err != nil {
		// errToolCallFailed is the sentinel signaling tools/call returned
		// isError=true; the result has already been printed, so suppress
		// the duplicate "Error:" line and exit with code 2.
		if errors.Is(err, cmd.ErrToolCallFailed) {
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
