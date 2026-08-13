package cmd_test

import (
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func prepareAgentMutationEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func TestAgentDelete_BadOutput_Hermetic(t *testing.T) {
	prepareAgentMutationEnv(t)
	err := rootFor(cmd.AgentCmd, "delete", "alpha", "--yes", "--output", "toml").Execute()
	if err == nil {
		t.Fatal("expected bad output error, got nil")
	}
	want := `--output for delete supports json or default human summary only; got "toml"`
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

func TestAgentRotateToken_BadOutput_Hermetic(t *testing.T) {
	prepareAgentMutationEnv(t)
	err := rootFor(cmd.AgentCmd, "rotate-token", "alpha", "--yes", "--output", "toml").Execute()
	if err == nil {
		t.Fatal("expected bad output error, got nil")
	}
	want := `--output for rotate-token supports json or default human summary only; got "toml"`
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
}
