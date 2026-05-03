package cmd_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel/cmd/citadel-cli/cmd"
)

// hasSubcmd reports whether parent has a direct subcommand named name.
func hasSubcmd(parent *cobra.Command, name string) bool {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return true
		}
	}
	return false
}

// hasFlag reports whether c has a flag named name (local or persistent).
func hasFlag(c *cobra.Command, name string) bool {
	return c.Flags().Lookup(name) != nil || c.PersistentFlags().Lookup(name) != nil
}

// helpRuns verifies that --help executes without error for cmd.
func helpRuns(t *testing.T, c *cobra.Command) {
	t.Helper()
	c.SetOut(new(bytes.Buffer))
	c.SetErr(new(bytes.Buffer))
	c.SetArgs([]string{"--help"})
	// ResetFlags is not needed; we just check Execute doesn't error.
	// Cobra treats --help as a special case that always returns nil.
	if err := c.Execute(); err != nil {
		t.Fatalf("%s --help returned error: %v", c.Name(), err)
	}
}

// ── repo ─────────────────────────────────────────────────────────────────────

func TestRepoCmdExists(t *testing.T) {
	if cmd.RepoCmd == nil {
		t.Fatal("RepoCmd is nil")
	}
}

func TestRepoCmdHelp(t *testing.T) {
	helpRuns(t, cmd.RepoCmd)
}

func TestRepoSubcommands(t *testing.T) {
	for _, name := range []string{"create", "list", "get", "delete"} {
		if !hasSubcmd(cmd.RepoCmd, name) {
			t.Errorf("citadel repo: missing subcommand %q", name)
		}
	}
}

func TestRepoCreateFlags(t *testing.T) {
	c := findSubcmd(t, cmd.RepoCmd, "create")
	for _, flag := range []string{"namespace", "slug", "description", "visibility", "default-branch", "init-with-readme", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo create: missing flag --%s", flag)
		}
	}
}

func TestRepoListFlags(t *testing.T) {
	c := findSubcmd(t, cmd.RepoCmd, "list")
	for _, flag := range []string{"namespace", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo list: missing flag --%s", flag)
		}
	}
}

func TestRepoGetOutputFlag(t *testing.T) {
	c := findSubcmd(t, cmd.RepoCmd, "get")
	if !hasFlag(c, "output") {
		t.Error("citadel repo get: missing --output flag")
	}
}

func TestRepoDeleteDestructiveFlags(t *testing.T) {
	c := findSubcmd(t, cmd.RepoCmd, "delete")
	for _, flag := range []string{"yes", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo delete: missing flag --%s", flag)
		}
	}
}

// ── namespace ────────────────────────────────────────────────────────────────

func TestNamespaceCmdExists(t *testing.T) {
	if cmd.NamespaceCmd == nil {
		t.Fatal("NamespaceCmd is nil")
	}
}

func TestNamespaceCmdHelp(t *testing.T) {
	helpRuns(t, cmd.NamespaceCmd)
}

func TestNamespaceSubcommands(t *testing.T) {
	for _, name := range []string{"list", "get", "members", "transfer"} {
		if !hasSubcmd(cmd.NamespaceCmd, name) {
			t.Errorf("citadel namespace: missing subcommand %q", name)
		}
	}
}

func TestNamespaceOutputFlags(t *testing.T) {
	for _, name := range []string{"list", "get", "members"} {
		c := findSubcmd(t, cmd.NamespaceCmd, name)
		if !hasFlag(c, "output") {
			t.Errorf("citadel namespace %s: missing --output flag", name)
		}
	}
}

func TestNamespaceTransferSubcommands(t *testing.T) {
	transfer := findSubcmd(t, cmd.NamespaceCmd, "transfer")
	for _, name := range []string{"initiate", "list-pending", "accept", "decline", "revoke"} {
		if !hasSubcmd(transfer, name) {
			t.Errorf("citadel namespace transfer: missing subcommand %q", name)
		}
	}
}

func TestNamespaceTransferInitiateFlags(t *testing.T) {
	transfer := findSubcmd(t, cmd.NamespaceCmd, "transfer")
	initiate := findSubcmd(t, transfer, "initiate")
	for _, flag := range []string{"to", "yes", "output"} {
		if !hasFlag(initiate, flag) {
			t.Errorf("citadel namespace transfer initiate: missing flag --%s", flag)
		}
	}
}

func TestNamespaceTransferRevokeFlags(t *testing.T) {
	transfer := findSubcmd(t, cmd.NamespaceCmd, "transfer")
	revoke := findSubcmd(t, transfer, "revoke")
	for _, flag := range []string{"yes", "output"} {
		if !hasFlag(revoke, flag) {
			t.Errorf("citadel namespace transfer revoke: missing flag --%s", flag)
		}
	}
}

// ── agent ────────────────────────────────────────────────────────────────────

func TestAgentCmdExists(t *testing.T) {
	if cmd.AgentCmd == nil {
		t.Fatal("AgentCmd is nil")
	}
}

func TestAgentCmdHelp(t *testing.T) {
	helpRuns(t, cmd.AgentCmd)
}

func TestAgentSubcommands(t *testing.T) {
	for _, name := range []string{"list", "get", "delete", "rotate-token"} {
		if !hasSubcmd(cmd.AgentCmd, name) {
			t.Errorf("citadel agent: missing subcommand %q", name)
		}
	}
}

func TestAgentOutputFlags(t *testing.T) {
	for _, name := range []string{"list", "get", "rotate-token"} {
		c := findSubcmd(t, cmd.AgentCmd, name)
		if !hasFlag(c, "output") {
			t.Errorf("citadel agent %s: missing --output flag", name)
		}
	}
}

func TestAgentDeleteDestructiveFlags(t *testing.T) {
	c := findSubcmd(t, cmd.AgentCmd, "delete")
	for _, flag := range []string{"yes", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel agent delete: missing flag --%s", flag)
		}
	}
}

func TestAgentRotateTokenDestructiveFlags(t *testing.T) {
	c := findSubcmd(t, cmd.AgentCmd, "rotate-token")
	for _, flag := range []string{"yes", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel agent rotate-token: missing flag --%s", flag)
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// findSubcmd returns the named subcommand or fails the test.
func findSubcmd(t *testing.T, parent *cobra.Command, name string) *cobra.Command {
	t.Helper()
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("%s: subcommand %q not found", parent.Name(), name)
	return nil
}
