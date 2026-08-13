package cmd_test

import (
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestSearch_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.SearchCmd, "ab", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestSearch_QueryTooShort_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.SearchCmd, "x").Execute()
	if err == nil || err.Error() != "query must be at least 2 characters" {
		t.Fatalf("want query validation error, got %v", err)
	}
}

func TestSearch_InvalidScope_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.SearchCmd, "ab", "--scope", "bogus").Execute()
	if err == nil || err.Error() != `invalid scope "bogus" (use namespaces, repos, or all)` {
		t.Fatalf("want scope validation error, got %v", err)
	}
}

func TestSearch_InvalidLimit_Hermetic(t *testing.T) {
	for _, limit := range []string{"-1", "26"} {
		t.Run(limit, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv("CITADEL_ACCESS_TOKEN", "")
			t.Setenv("CITADEL_SERVER", "")
			t.Setenv("CITADEL_REPO", "")

			err := rootFor(cmd.SearchCmd, "ab", "--limit", limit).Execute()
			if err == nil || err.Error() != "--limit must be between 1 and 25" {
				t.Fatalf("want limit validation error, got %v", err)
			}
		})
	}
}
