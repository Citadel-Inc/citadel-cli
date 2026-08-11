package cmd_test

import (
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

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
