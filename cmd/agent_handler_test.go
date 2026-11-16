package cmd_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

func prepareAgentMutationEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func TestAgentList_BadOutput_Hermetic(t *testing.T) {
	prepareAgentMutationEnv(t)
	err := rootFor(cmd.AgentCmd, "list", "--output", "toml").Execute()
	if err == nil {
		t.Fatal("expected bad output error, got nil")
	}
	want := `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
}

func TestAgentCreate_BadOutput_Hermetic(t *testing.T) {
	prepareAgentMutationEnv(t)
	err := rootFor(cmd.AgentCmd, "create", "alpha", "--output", "toml").Execute()
	if err == nil {
		t.Fatal("expected bad output error, got nil")
	}
	want := `--output for create supports json or default human summary only; got "toml"`
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err)
	}
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

// A server that keeps returning the same next_cursor must end the paginated
// lookup with an error rather than spin forever.
func TestAgentGet_RepeatedCursorStops(t *testing.T) {
	prepareAgentMutationEnv(t)
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"agents":      []map[string]any{{"id": "00000000-0000-0000-0000-00000000000a", "name": "other", "owner_user_id": "u1"}},
				"next_cursor": "same",
			})
		},
	}))

	done := make(chan error, 1)
	go func() { done <- rootFor(cmd.AgentCmd, "get", "other").Execute() }()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), `cursor "same" again`) {
			t.Fatalf("expected repeated-cursor error, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("agent get looped forever on a repeated next_cursor")
	}
}
