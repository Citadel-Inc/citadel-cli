package main

import (
	"fmt"
	"os"

	"github.com/Rethunk-Tech/citadel/cmd/citadel-cli/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "citadel",
		Short: "Citadel CLI — authentication, token, and MCP agent interface",
		Long: `Citadel is a command-line client for managing authentication, agent tokens,
and MCP tool interactions with the Citadel server.

Server URL defaults to https://api.src.land; override with CITADEL_SERVER or --server.`,
		Version: "0.0.1-alpha",
	}

	// --server flag + CITADEL_SERVER env var. Persistent so every subcommand
	// inherits it. Resolved against cfg.ServerURL by clicfg.ResolveServerURL().
	root.PersistentFlags().String("server", "", "Server URL (overrides CITADEL_SERVER env and stored config)")

	root.AddCommand(cmd.AuthCmd)
	root.AddCommand(cmd.TokenCmd)
	root.AddCommand(cmd.McpCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
