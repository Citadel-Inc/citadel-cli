package main

import (
	"bytes"
	"testing"

	"github.com/Rethunk-Tech/citadel/cmd/citadel-cli/cmd"
	"github.com/spf13/cobra"
)

func TestRootCommandHelp(t *testing.T) {
	root := &cobra.Command{
		Use:   "citadel-cli",
		Short: "Citadel CLI — authentication, token, and MCP agent interface",
		Long: `Citadel is a command-line client for managing authentication, agent tokens,
and MCP tool interactions with the Citadel server.

Server URL defaults to https://api.src.land; override with CITADEL_SERVER or --server.`,
		Version: "0.0.1-alpha",
	}

	root.AddCommand(cmd.AuthCmd)
	root.AddCommand(cmd.TokenCmd)
	root.AddCommand(cmd.McpCmd)

	// Test that help runs without error
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	help := buf.String()
	if help == "" {
		t.Error("Help output is empty")
	}

	// Verify expected subcommands are in help
	expectedCommands := []string{"auth", "token", "mcp"}
	for _, cmd := range expectedCommands {
		if !bytes.Contains(buf.Bytes(), []byte(cmd)) {
			t.Errorf("Expected subcommand %q not found in help output", cmd)
		}
	}
}

func TestAuthSubcommands(t *testing.T) {
	// Verify auth subcommands exist
	if cmd.AuthCmd == nil {
		t.Fatal("AuthCmd is nil")
	}
	if len(cmd.AuthCmd.Commands()) == 0 {
		t.Error("AuthCmd has no subcommands")
	}
	expectedAuth := []string{"login", "status", "logout"}
	for _, expected := range expectedAuth {
		found := false
		for _, subcmd := range cmd.AuthCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected auth subcommand %q not found", expected)
		}
	}
}

func TestTokenSubcommands(t *testing.T) {
	// Verify token subcommands exist
	if cmd.TokenCmd == nil {
		t.Fatal("TokenCmd is nil")
	}
	if len(cmd.TokenCmd.Commands()) == 0 {
		t.Error("TokenCmd has no subcommands")
	}
	expectedToken := []string{"list", "issue", "revoke"}
	for _, expected := range expectedToken {
		found := false
		for _, subcmd := range cmd.TokenCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected token subcommand %q not found", expected)
		}
	}
}

func TestMcpSubcommands(t *testing.T) {
	// Verify mcp subcommands exist
	if cmd.McpCmd == nil {
		t.Fatal("McpCmd is nil")
	}
	if len(cmd.McpCmd.Commands()) == 0 {
		t.Error("McpCmd has no subcommands")
	}
	expectedMcp := []string{"tools", "call"}
	for _, expected := range expectedMcp {
		found := false
		for _, subcmd := range cmd.McpCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected mcp subcommand %q not found", expected)
		}
	}
}
