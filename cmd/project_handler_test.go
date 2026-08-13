package cmd_test

import (
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestProjectPinChain_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "pin-chain", " \t").Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectPinChain_BadOutputHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "pin-chain", "ns", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf(`error = %v, want --output: unknown format "toml" (use json|yaml|table)`, err)
	}
}

func TestProjectWalk_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "walk", " \t").Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectNeighbors_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "neighbors", " \t", "--kind", "foo").Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectWalk_KindRequiredHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "walk", "ns").Execute()
	if err == nil || err.Error() != "--kind is required" {
		t.Fatalf("error = %v, want --kind is required", err)
	}
}

func TestProjectNeighbors_KindRequiredHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "neighbors", "ns").Execute()
	if err == nil || err.Error() != "--kind is required" {
		t.Fatalf("error = %v, want --kind is required", err)
	}
}

func TestProjectReindex_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "reindex", " \t").Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectWalk_BadOutputHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "walk", "ns", "--output", "toml", "--kind", "foo").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf(`error = %v, want --output: unknown format "toml" (use json|yaml|table)`, err)
	}
}

func setHermeticProjectEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}
