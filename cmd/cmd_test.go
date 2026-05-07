package cmd_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
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

func TestIssueCmdExists(t *testing.T) {
	if cmd.IssueCmd == nil {
		t.Fatal("IssueCmd is nil")
	}
}

func TestIssueCmdHelp(t *testing.T) {
	helpRuns(t, cmd.IssueCmd)
}

func TestIssueSubcommands(t *testing.T) {
	for _, name := range []string{"list", "view", "create", "comment", "close", "reopen", "label", "close-refs"} {
		if !hasSubcmd(cmd.IssueCmd, name) {
			t.Errorf("citadel issue: missing subcommand %q", name)
		}
	}
}

func TestIssueListFlags(t *testing.T) {
	c := findSubcmd(t, cmd.IssueCmd, "list")
	for _, flag := range []string{"repo", "state", "label", "assignee", "limit", "cursor", "all", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel issue list: missing flag --%s", flag)
		}
	}
}

func TestIssueViewFlags(t *testing.T) {
	c := findSubcmd(t, cmd.IssueCmd, "view")
	for _, flag := range []string{"repo", "web", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel issue view: missing flag --%s", flag)
		}
	}
}

func TestIssueCreateFlags(t *testing.T) {
	c := findSubcmd(t, cmd.IssueCmd, "create")
	for _, flag := range []string{"repo", "title", "body", "label", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel issue create: missing flag --%s", flag)
		}
	}
}

func TestIssueCommentFlags(t *testing.T) {
	c := findSubcmd(t, cmd.IssueCmd, "comment")
	for _, flag := range []string{"repo", "body", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel issue comment: missing flag --%s", flag)
		}
	}
}

func TestIssueLabelFlags(t *testing.T) {
	c := findSubcmd(t, cmd.IssueCmd, "label")
	for _, flag := range []string{"repo", "add", "remove", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel issue label: missing flag --%s", flag)
		}
	}
}

func TestRepoSubcommands(t *testing.T) {
	for _, name := range []string{"create", "list", "get", "delete", "branch", "tag"} {
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
	for _, flag := range []string{"output", "repo", "no-cwd-repo"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo get: missing --%s flag", flag)
		}
	}
}

func TestRepoDeleteDestructiveFlags(t *testing.T) {
	c := findSubcmd(t, cmd.RepoCmd, "delete")
	for _, flag := range []string{"yes", "output", "repo", "no-cwd-repo"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo delete: missing flag --%s", flag)
		}
	}
}

func TestRepoBranchSubcommands(t *testing.T) {
	branch := findSubcmd(t, cmd.RepoCmd, "branch")
	for _, name := range []string{"list", "delete", "set-default"} {
		if !hasSubcmd(branch, name) {
			t.Errorf("citadel repo branch: missing subcommand %q", name)
		}
	}
}

func TestRepoBranchListFlags(t *testing.T) {
	branch := findSubcmd(t, cmd.RepoCmd, "branch")
	c := findSubcmd(t, branch, "list")
	for _, flag := range []string{"repo", "no-cwd-repo", "limit", "cursor", "all", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo branch list: missing flag --%s", flag)
		}
	}
}

func TestRepoBranchDeleteFlags(t *testing.T) {
	branch := findSubcmd(t, cmd.RepoCmd, "branch")
	c := findSubcmd(t, branch, "delete")
	for _, flag := range []string{"repo", "no-cwd-repo", "dry-run", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo branch delete: missing flag --%s", flag)
		}
	}
}

func TestRepoBranchSetDefaultFlags(t *testing.T) {
	branch := findSubcmd(t, cmd.RepoCmd, "branch")
	c := findSubcmd(t, branch, "set-default")
	for _, flag := range []string{"repo", "no-cwd-repo", "dry-run", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo branch set-default: missing flag --%s", flag)
		}
	}
}

func TestRepoTagSubcommands(t *testing.T) {
	tag := findSubcmd(t, cmd.RepoCmd, "tag")
	for _, name := range []string{"list", "create", "delete"} {
		if !hasSubcmd(tag, name) {
			t.Errorf("citadel repo tag: missing subcommand %q", name)
		}
	}
}

func TestRepoTagListFlags(t *testing.T) {
	tag := findSubcmd(t, cmd.RepoCmd, "tag")
	c := findSubcmd(t, tag, "list")
	for _, flag := range []string{"repo", "no-cwd-repo", "limit", "cursor", "all", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo tag list: missing flag --%s", flag)
		}
	}
}

func TestRepoTagCreateFlags(t *testing.T) {
	tag := findSubcmd(t, cmd.RepoCmd, "tag")
	c := findSubcmd(t, tag, "create")
	for _, flag := range []string{"repo", "no-cwd-repo", "ref", "message", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo tag create: missing flag --%s", flag)
		}
	}
}

func TestRepoTagDeleteFlags(t *testing.T) {
	tag := findSubcmd(t, cmd.RepoCmd, "tag")
	c := findSubcmd(t, tag, "delete")
	for _, flag := range []string{"repo", "no-cwd-repo", "dry-run", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo tag delete: missing flag --%s", flag)
		}
	}
}

func TestRepoDeployTokenSubcommands(t *testing.T) {
	deploy := findSubcmd(t, cmd.RepoCmd, "deploy-token")
	for _, name := range []string{"list", "create", "revoke"} {
		if !hasSubcmd(deploy, name) {
			t.Errorf("citadel repo deploy-token: missing subcommand %q", name)
		}
	}
}

func TestRepoDeployTokenListFlags(t *testing.T) {
	deploy := findSubcmd(t, cmd.RepoCmd, "deploy-token")
	c := findSubcmd(t, deploy, "list")
	for _, flag := range []string{"repo", "no-cwd-repo", "limit", "cursor", "all", "watch", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo deploy-token list: missing flag --%s", flag)
		}
	}
}

func TestRepoDeployTokenCreateFlags(t *testing.T) {
	deploy := findSubcmd(t, cmd.RepoCmd, "deploy-token")
	c := findSubcmd(t, deploy, "create")
	for _, flag := range []string{"repo", "no-cwd-repo", "name", "expires", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo deploy-token create: missing flag --%s", flag)
		}
	}
}

func TestRepoDeployTokenRevokeFlags(t *testing.T) {
	deploy := findSubcmd(t, cmd.RepoCmd, "deploy-token")
	c := findSubcmd(t, deploy, "revoke")
	for _, flag := range []string{"repo", "no-cwd-repo", "dry-run", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel repo deploy-token revoke: missing flag --%s", flag)
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
	for _, name := range []string{"list", "get", "members", "transfer", "deploy-token"} {
		if !hasSubcmd(cmd.NamespaceCmd, name) {
			t.Errorf("citadel namespace: missing subcommand %q", name)
		}
	}
}

func TestNamespaceDeployTokenSubcommands(t *testing.T) {
	deploy := findSubcmd(t, cmd.NamespaceCmd, "deploy-token")
	for _, name := range []string{"list", "create", "revoke"} {
		if !hasSubcmd(deploy, name) {
			t.Errorf("citadel namespace deploy-token: missing subcommand %q", name)
		}
	}
}

func TestNamespaceDeployTokenListFlags(t *testing.T) {
	deploy := findSubcmd(t, cmd.NamespaceCmd, "deploy-token")
	c := findSubcmd(t, deploy, "list")
	for _, flag := range []string{"limit", "cursor", "all", "watch", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel namespace deploy-token list: missing flag --%s", flag)
		}
	}
}

func TestNamespaceDeployTokenCreateFlags(t *testing.T) {
	deploy := findSubcmd(t, cmd.NamespaceCmd, "deploy-token")
	c := findSubcmd(t, deploy, "create")
	for _, flag := range []string{"name", "expires", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel namespace deploy-token create: missing flag --%s", flag)
		}
	}
}

func TestNamespaceDeployTokenRevokeFlags(t *testing.T) {
	deploy := findSubcmd(t, cmd.NamespaceCmd, "deploy-token")
	c := findSubcmd(t, deploy, "revoke")
	for _, flag := range []string{"dry-run", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel namespace deploy-token revoke: missing flag --%s", flag)
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

// ── kg ────────────────────────────────────────────────────────────────────────

func TestKgImpactRepoFlags(t *testing.T) {
	impact := findSubcmd(t, cmd.KgCmd, "impact")
	for _, flag := range []string{"json", "depth", "repo", "no-cwd-repo"} {
		if !hasFlag(impact, flag) {
			t.Errorf("citadel kg impact: missing --%s flag", flag)
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

// ── oauth clients ───────────────────────────────────────────────────────────

func TestOauthCmdExists(t *testing.T) {
	if cmd.OauthCmd == nil {
		t.Fatal("OauthCmd is nil")
	}
}

func TestOauthCmdHelp(t *testing.T) {
	helpRuns(t, cmd.OauthCmd)
}

func TestOauthClientsSubcommands(t *testing.T) {
	clients := findSubcmd(t, cmd.OauthCmd, "clients")
	for _, name := range []string{"list", "create", "show", "rotate-secret", "revoke"} {
		if !hasSubcmd(clients, name) {
			t.Errorf("citadel oauth clients: missing subcommand %q", name)
		}
	}
}

func TestOauthClientsListFlags(t *testing.T) {
	clients := findSubcmd(t, cmd.OauthCmd, "clients")
	c := findSubcmd(t, clients, "list")
	for _, flag := range []string{"org", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel oauth clients list: missing flag --%s", flag)
		}
	}
}

func TestOauthClientsCreateFlags(t *testing.T) {
	clients := findSubcmd(t, cmd.OauthCmd, "clients")
	c := findSubcmd(t, clients, "create")
	for _, flag := range []string{"name", "redirect-uri", "org", "public", "description", "scope", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel oauth clients create: missing flag --%s", flag)
		}
	}
}

func TestOauthClientsShowOutputFlag(t *testing.T) {
	clients := findSubcmd(t, cmd.OauthCmd, "clients")
	c := findSubcmd(t, clients, "show")
	if !hasFlag(c, "output") {
		t.Error("citadel oauth clients show: missing --output flag")
	}
}

func TestOauthClientsRotateSecretFlags(t *testing.T) {
	clients := findSubcmd(t, cmd.OauthCmd, "clients")
	c := findSubcmd(t, clients, "rotate-secret")
	for _, flag := range []string{"output", "copy-to-clipboard"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel oauth clients rotate-secret: missing flag --%s", flag)
		}
	}
}

func TestOauthClientsRevokeDestructiveFlags(t *testing.T) {
	clients := findSubcmd(t, cmd.OauthCmd, "clients")
	c := findSubcmd(t, clients, "revoke")
	for _, flag := range []string{"yes", "output"} {
		if !hasFlag(c, flag) {
			t.Errorf("citadel oauth clients revoke: missing flag --%s", flag)
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
