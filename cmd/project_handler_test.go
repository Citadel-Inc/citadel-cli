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

func TestProjectStatusRollup_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "status", "rollup", " \t").Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectStatusDrilldown_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "status", "drilldown", " \t").Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectEdgeAdd_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "add", " \t",
		"--from-namespace-id", "not-a-uuid",
		"--from-kind", "namespace",
		"--to-kind", "repo",
		"--edge-type", "contains",
	).Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectEdgeDelete_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "delete", " \t", "00000000-0000-0000-0000-000000000001",
	).Execute()
	if err == nil || err.Error() != "namespace path cannot be empty" {
		t.Fatalf("error = %v, want namespace path cannot be empty", err)
	}
}

func TestProjectEdgeRestore_EmptyPathHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "restore", " \t", "00000000-0000-0000-0000-000000000001",
	).Execute()
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

func TestProjectEdgeAdd_InvalidFromNamespaceIDHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "add", "ns",
		"--from-namespace-id", "not-a-uuid",
		"--from-kind", "namespace",
		"--to-kind", "repo",
		"--edge-type", "contains",
	).Execute()
	if err == nil || err.Error() != "--from-namespace-id must be a UUID" {
		t.Fatalf("error = %v, want --from-namespace-id must be a UUID", err)
	}
}

func TestProjectEdgeAdd_FromKindRequiredHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "add", "ns",
		"--from-namespace-id", "00000000-0000-0000-0000-000000000001",
		"--from-kind", "",
		"--to-kind", "repo",
		"--edge-type", "contains",
	).Execute()
	if err == nil || err.Error() != "--from-kind is required" {
		t.Fatalf("error = %v, want --from-kind is required", err)
	}
}

func TestProjectEdgeAdd_InvalidAttrsJSONHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "add", "ns",
		"--from-namespace-id", "00000000-0000-0000-0000-000000000001",
		"--from-kind", "namespace",
		"--to-kind", "repo",
		"--edge-type", "contains",
		"--attrs-json", "{",
	).Execute()
	if err == nil || err.Error() != "--attrs-json must be valid JSON" {
		t.Fatalf("error = %v, want --attrs-json must be valid JSON", err)
	}
}

func TestProjectEdgeDelete_InvalidEdgeIDHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "edge", "delete", "ns", "not-a-uuid").Execute()
	if err == nil || err.Error() != "edge-id must be a UUID" {
		t.Fatalf("error = %v, want edge-id must be a UUID", err)
	}
}

func TestProjectEdgeRestore_InvalidEdgeIDHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "edge", "restore", "ns", "not-a-uuid").Execute()
	if err == nil || err.Error() != "edge-id must be a UUID" {
		t.Fatalf("error = %v, want edge-id must be a UUID", err)
	}
}

func TestProjectEdgeRestore_BadOutputHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(
		cmd.ProjectCmd,
		"edge", "restore", "ns", "00000000-0000-0000-0000-000000000001",
		"--output", "toml",
	).Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf(`error = %v, want --output: unknown format "toml" (use json|yaml|table)`, err)
	}
}

func TestProjectReindex_BadOutputHermetic(t *testing.T) {
	setHermeticProjectEnv(t)

	err := rootFor(cmd.ProjectCmd, "reindex", "ns", "--output", "toml").Execute()
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
