package cmd_test

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestRepoCreate_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "acme", "--slug", "repo", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), `--output for create supports json or default human summary only; got "toml"`) {
		t.Fatalf("want create output validation error, got %v", err)
	}
}

func TestRepoCreate_EmptyNamespace_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "", "--slug", "repo").Execute()
	if err == nil || !strings.Contains(err.Error(), "namespace cannot be empty") {
		t.Fatalf("want namespace validation error, got %v", err)
	}
}

func TestRepoCreate_EmptySlug_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "acme", "--slug", "").Execute()
	if err == nil || !strings.Contains(err.Error(), "slug cannot be empty") {
		t.Fatalf("want slug validation error, got %v", err)
	}
}

func TestRepoCreate_BadVisibility_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "acme", "--slug", "repo", "--visibility", "internal").Execute()
	if err == nil || !strings.Contains(err.Error(), "visibility must be public or private") {
		t.Fatalf("want visibility validation error, got %v", err)
	}
}

func TestRepoList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "list", "--namespace", "acme", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestRepoList_WatchJSON_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "list", "--namespace", "acme", "--watch", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used with --watch") {
		t.Fatalf("want watch/output validation error, got %v", err)
	}
}

func TestRepoList_BadCursor_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd, "list", "--namespace", "acme", "--cursor", "not-base64!!!").Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid --cursor") {
		t.Fatalf("want cursor validation error, got %v", err)
	}
}

func TestRepoDelete_DryRunHermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "delete", "myorg/myrepo", "--dry-run").Execute(); err != nil {
		t.Fatalf("repo delete dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "Would DELETE /namespaces/myorg/myrepo (skipped; --dry-run)") {
		t.Fatalf("unexpected dry-run output: %q", out.String())
	}
}
