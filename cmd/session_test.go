package cmd_test

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

func TestSessionListAndRevoke(t *testing.T) {
	var revokedPath, revokedQuery string
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agent-sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{
				{"id": "s-live", "agent_name": "claude-code-box", "client": "claude-code", "created_at": time.Now(), "expires_at": time.Now().Add(time.Hour)},
				{"id": "s-gone", "agent_name": "claude-code-box", "client": "cursor", "created_at": time.Now(), "expires_at": time.Now().Add(time.Hour), "ended_at": time.Now(), "end_reason": "revoked"},
			}})
		},
		"DELETE /agent-sessions/s-live": func(w http.ResponseWriter, r *http.Request) {
			revokedPath, revokedQuery = r.URL.Path, r.URL.RawQuery
			w.WriteHeader(http.StatusNoContent)
		},
	}))

	var out bytes.Buffer
	if err := rootForOut(cmd.SessionCmd, &out, "list", "--live").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "s-live") || strings.Contains(out.String(), "s-gone") {
		t.Fatalf("--live should list only s-live:\n%s", out.String())
	}

	out.Reset()
	if err := rootForOut(cmd.SessionCmd, &out, "revoke", "s-live").Execute(); err != nil {
		t.Fatal(err)
	}
	if revokedPath != "/agent-sessions/s-live" || revokedQuery != "reason=revoked" || !strings.Contains(out.String(), "revoked session s-live") {
		t.Fatalf("revoke: path=%q query=%q out=%q", revokedPath, revokedQuery, out.String())
	}
}

func TestHookStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	claude := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claude, 0o700); err != nil {
		t.Fatal(err)
	}
	settings := `{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"citadel-cli hook session-start"}]}]}}`
	if err := os.WriteFile(filepath.Join(claude, "settings.json"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	exp := time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)
	if err := os.WriteFile(filepath.Join(state, "state.json"), []byte(`{"session_id":"s-1","expires_at":"`+exp+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CITADEL_HOOK_STATE", state)
	t.Setenv("CITADEL_AGENT_SESSION", "s-1")

	var out bytes.Buffer
	if err := rootForOut(cmd.HookCmd, &out, "status").Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"claude-code (user scope)", "session: s-1 expires", "set=true live=true"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("status missing %q:\n%s", want, out.String())
		}
	}
}
