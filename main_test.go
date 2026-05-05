package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestRun_HelpExitsZero(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"--help"}, &stderr); got != 0 {
		t.Fatalf("run --help exit = %d, stderr=%q", got, stderr.String())
	}
}

func TestRun_VersionExitsZero(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"--version"}, &stderr); got != 0 {
		t.Fatalf("run --version exit = %d", got)
	}
}

func TestRun_UnknownCommand_ExitsOne(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"definitely-not-a-command"}, &stderr); got != 1 {
		t.Fatalf("run unknown exit = %d", got)
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Errorf("stderr missing Error prefix: %q", stderr.String())
	}
}

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	root := newRootCmd()
	want := []string{"auth", "token", "mcp", "kg", "repo", "namespace", "agent", "oauth", "completion", "doctor", "man"}
	for _, name := range want {
		found := false
		for _, sub := range root.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subcommand %q", name)
		}
	}
	// The persistent --server flag must be present so subcommands inherit it.
	if root.PersistentFlags().Lookup("server") == nil {
		t.Error("expected persistent --server flag")
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
	expectedMcp := []string{"tools", "call", "resources", "prompts"}
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
