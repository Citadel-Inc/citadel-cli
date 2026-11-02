package cmd_test

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // test-owned temp path
}

func runHook(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	cmd.NewRootCmd() // wires HookCmd's subcommands once
	var out strings.Builder
	root := rootForOut(cmd.HookCmd, &out, args...)
	root.SetIn(strings.NewReader(stdin))
	if err := root.Execute(); err != nil {
		t.Fatalf("hook %v: %v", args, err)
	}
	return out.String()
}

func gitRepoWithOrigin(t *testing.T, origin string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", origin}} {
		c := exec.CommandContext(t.Context(), "git", args...) //nolint:gosec // fixed git verb, test-owned args
		c.Dir = dir
		if b, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, b)
		}
	}
	return dir
}

func TestHookSessionLifecycle_Hermetic(t *testing.T) {
	runtimeDir := t.TempDir()
	envFile := filepath.Join(t.TempDir(), "env")
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Setenv("CLAUDE_ENV_FILE", envFile)
	t.Setenv("CITADEL_GIT_HOSTS", "")

	var created, renewed, deleted atomic.Int32
	var reason atomic.Value
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	srv := withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/sessions"):
			var body struct {
				Client string `json:"client"`
				Key    string `json:"ssh_public_key"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Client != "claude-code" || !strings.HasPrefix(body.Key, "ssh-ed25519 ") {
				t.Errorf("bad session body: %+v", body)
			}
			created.Add(1)
			writeJSON(t, w, 201, map[string]any{"session_id": "sess-1", "agent_id": "a1", "agent_name": "claude-code-box", "token": "tok'en", "expires_at": expires})
		case r.Method == http.MethodPost && r.URL.Path == "/agent-sessions/sess-1/renew":
			renewed.Add(1)
			writeJSON(t, w, 200, map[string]any{"expires_at": time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)})
		case r.Method == http.MethodDelete && r.URL.Path == "/agent-sessions/sess-1":
			deleted.Add(1)
			reason.Store(r.URL.Query().Get("reason"))
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unrouted request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	})
	_ = srv

	// A GitHub repo is untouched: no output, no request, no env file.
	gh := gitRepoWithOrigin(t, "git@github.com:acme/app.git")
	if out := runHook(t, `{"session_id":"sess-1","cwd":"`+gh+`"}`, "session-start"); out != "" || created.Load() != 0 {
		t.Fatalf("github repo must be a no-op, out=%q created=%d", out, created.Load())
	}
	if _, err := os.Stat(envFile); err == nil {
		t.Fatal("env file written for a non-src.land repo")
	}

	repo := gitRepoWithOrigin(t, "git@src.land:acme/app.git")
	in := `{"session_id":"sess-1","cwd":"` + repo + `"}`
	out := runHook(t, in, "session-start")
	var got struct {
		H struct {
			Ctx string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil || !strings.Contains(got.H.Ctx, "claude-code-box") {
		t.Fatalf("session-start output = %q (%v)", out, err)
	}
	env, _ := readFile(envFile)
	keyPath := filepath.Join(runtimeDir, "citadel", "sessions", "sess-1", "id_ed25519")
	for _, want := range []string{"export GIT_SSH_COMMAND='ssh -i '\\''" + keyPath, "IdentitiesOnly=yes", "export CITADEL_AGENT_SESSION='sess-1'", `export CITADEL_ACCESS_TOKEN='tok'\''en'`} {
		if !strings.Contains(string(env), want) {
			t.Errorf("env file missing %q:\n%s", want, env)
		}
	}
	if st, err := os.Stat(keyPath); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("key file: %v %v", st, err)
	}

	// More than 5 minutes left: renew makes no request.
	runHook(t, in, "renew")
	if renewed.Load() != 0 {
		t.Fatal("renew called the server with time remaining")
	}
	stateFile := filepath.Join(runtimeDir, "citadel", "sessions", "sess-1", "state.json")
	near, _ := json.Marshal(map[string]any{"session_id": "sess-1", "agent_name": "claude-code-box", "expires_at": time.Now().Add(time.Minute)})
	if err := os.WriteFile(stateFile, near, 0o600); err != nil {
		t.Fatal(err)
	}
	runHook(t, in, "renew")
	if renewed.Load() != 1 {
		t.Fatalf("renew calls = %d, want 1", renewed.Load())
	}
	runHook(t, in, "renew")
	if renewed.Load() != 1 {
		t.Fatal("renew did not update the stored expiry")
	}

	runHook(t, in, "session-end")
	runHook(t, in, "session-end")
	if deleted.Load() != 1 || reason.Load() != "session_end" {
		t.Fatalf("deleted=%d reason=%v", deleted.Load(), reason.Load())
	}
	if _, err := os.Stat(filepath.Dir(keyPath)); err == nil {
		t.Fatal("session dir not removed")
	}
}

func TestHookSessionStart_ServerDown_Hermetic(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	envFile := filepath.Join(t.TempDir(), "env")
	t.Setenv("CLAUDE_ENV_FILE", envFile)
	srv := withServer(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
	_ = srv
	repo := gitRepoWithOrigin(t, "git@src.land:acme/app.git")
	out := runHook(t, `{"session_id":"s2","cwd":"`+repo+`"}`, "session-start")
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "own SSH key") {
		t.Fatalf("failure context missing: %q", out)
	}
	if _, err := os.Stat(envFile); err == nil {
		t.Fatal("env file written on failure")
	}
}

func TestHookInstallMerge_Hermetic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	orig := `{"model":"opus","hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"mine.sh"}]}]}}`
	if err := os.WriteFile(path, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}

	runHook(t, "", "install")
	first, _ := readFile(path)
	runHook(t, "", "install")
	second, _ := readFile(path)
	if string(first) != string(second) {
		t.Fatal("install is not idempotent")
	}
	s := string(first)
	for _, want := range []string{`"model": "opus"`, "mine.sh", "citadel-cli hook session-start", "citadel-cli hook renew", "citadel-cli hook session-end"} {
		if !strings.Contains(s, want) {
			t.Errorf("settings missing %q:\n%s", want, s)
		}
	}

	runHook(t, "", "uninstall")
	left, _ := readFile(path)
	if strings.Contains(string(left), "citadel-cli") || !strings.Contains(string(left), "mine.sh") || !strings.Contains(string(left), "opus") {
		t.Fatalf("uninstall must remove only citadel entries:\n%s", left)
	}

	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd.NewRootCmd()
	var sink strings.Builder
	root := rootForOut(cmd.HookCmd, &sink, "install")
	if err := root.Execute(); err == nil {
		t.Fatal("install must refuse an unparseable file")
	}
	if b, _ := readFile(path); string(b) != "{not json" {
		t.Fatal("unparseable file was rewritten")
	}
}
