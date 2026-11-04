package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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
				Client string   `json:"client"`
				Key    string   `json:"ssh_public_key"`
				Scopes []string `json:"scopes"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Client != "claude-code" || !strings.HasPrefix(body.Key, "ssh-ed25519 ") {
				t.Errorf("bad session body: %+v", body)
			}
			if !slices.Contains(body.Scopes, "pulls:write") || !slices.Contains(body.Scopes, "issues:write") {
				t.Errorf("scopes = %v", body.Scopes)
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
	for _, want := range []string{" hook git-ssh '\\''" + keyPath, "export CITADEL_AGENT_SESSION='sess-1'", `export CITADEL_ACCESS_TOKEN='tok'\''en'`} {
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

// The harness payloads are as Claude Code and Cursor send them. A session names
// its repo only for an origin on the server it opens on, for either harness.
func TestHookSessionStartSendsRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("CLAUDE_ENV_FILE", filepath.Join(t.TempDir(), "env"))
	t.Setenv("CITADEL_GIT_HOSTS", "")
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	repos := []any{}
	var gone atomic.Bool
	srv := withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/sessions"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			repos = append(repos, body["repo"])
			writeJSON(t, w, 201, map[string]any{"session_id": "sess", "agent_name": "a", "token": "tok", "expires_at": expires})
		case gone.Load():
			writeJSON(t, w, 409, map[string]any{"error": "session_ended"})
		default:
			http.NotFound(w, r)
		}
	})
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	host := u.Hostname()

	claude := func(id, cwd string) (string, []string) {
		return `{"session_id":"` + id + `","transcript_path":"/home/dev/.claude/projects/x/` + id + `.jsonl","cwd":"` + cwd + `","hook_event_name":"SessionStart","source":"startup"}`, []string{"session-start"}
	}
	cursor := func(id, cwd string) (string, []string) {
		return `{"session_id":"` + id + `","is_background_agent":false,"composer_mode":"agent","model":"default","hook_event_name":"sessionStart","cursor_version":"2026.10.28","workspace_roots":["` + cwd + `"],"user_email":null}`, []string{"session-start", "--client", "cursor"}
	}
	type harness = func(id, cwd string) (string, []string)
	for _, tc := range []struct {
		name   string
		h      harness
		origin string
		gitEnv string
		want   any
	}{
		{"claude ssh", claude, "ssh://git@" + host + ":2222/acme/app.git", "", "acme/app"},
		{"claude https", claude, srv.URL + "/acme/proj/app.git", "", "acme/proj/app"},
		{"cursor ssh", cursor, "ssh://git@" + host + ":2222/acme/app.git", "", "acme/app"},
		{"cursor https", cursor, srv.URL + "/acme/app.git", "", "acme/app"},
		{"claude, hosted origin on a self-hosted server", claude, "git@src.land:acme/app.git", "", nil},
		{"cursor, hosted origin on a self-hosted server", cursor, "git@src.land:acme/app.git", "", nil},
		{"named git host", claude, "git@git.corp.example:acme/app.git", "git.corp.example", "acme/app"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CITADEL_GIT_HOSTS", tc.gitEnv)
			repos = repos[:0]
			in, args := tc.h(strings.ReplaceAll(tc.name, " ", "-"), gitRepoWithOrigin(t, tc.origin))
			runHook(t, in, args...)
			if len(repos) != 1 || repos[0] != tc.want {
				t.Fatalf("repo sent = %v, want %v", repos, tc.want)
			}
		})
	}

	// A session reissued after expiry keeps saying what it is for.
	repos = repos[:0]
	repo := gitRepoWithOrigin(t, "ssh://git@"+host+":2222/acme/app.git")
	in, args := claude("reissue", repo)
	runHook(t, in, args...)
	state := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "citadel", "sessions", "reissue", "state.json")
	expired, _ := json.Marshal(map[string]any{"session_id": "sess", "agent_name": "a", "expires_at": time.Now().Add(-time.Minute), "token": "tok", "repo": "acme/app"})
	if err := os.WriteFile(state, expired, 0o600); err != nil { //nolint:gosec // test-owned temp path
		t.Fatal(err)
	}
	gone.Store(true)
	runHook(t, in, "renew")
	if len(repos) != 2 || repos[1] != "acme/app" {
		t.Fatalf("reissue repos = %v", repos)
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

// An https origin on a Citadel host gets a git credential helper scoped to that
// host; a real `git credential fill` then answers with the session token, which
// never appears in the env file.
func TestHookSessionStart_HTTPSCredential_Hermetic(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	envFile := filepath.Join(t.TempDir(), "env")
	t.Setenv("CLAUDE_ENV_FILE", envFile)
	t.Setenv("CITADEL_GIT_HOSTS", "")
	t.Setenv("GIT_CONFIG_COUNT", "")
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, 201, map[string]any{"session_id": "s3", "agent_id": "a1", "agent_name": "claude-code-box", "token": "sess-token-123", "expires_at": expires})
	})

	repo := gitRepoWithOrigin(t, "https://git.src.land/acme/app.git")
	runHook(t, `{"session_id":"s3","cwd":"`+repo+`"}`, "session-start")
	env, _ := readFile(envFile)
	for _, want := range []string{"export GIT_CONFIG_COUNT='2'", "export GIT_CONFIG_KEY_0='credential.https://git.src.land.helper'", "hook git-credential"} {
		if !strings.Contains(string(env), want) {
			t.Errorf("env file missing %q:\n%s", want, env)
		}
	}

	// The helper is the test binary here; point it at a real build of the CLI.
	bin := filepath.Join(t.TempDir(), "citadel-cli")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "..") //nolint:gosec // fixed go verb, test-owned path
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v %s", err, b)
	}
	self, _ := os.Executable()
	script := strings.ReplaceAll(string(env), "'"+self+"'", "'"+bin+"'")
	fill := exec.CommandContext(t.Context(), "bash", "-c", `set -e; eval "$ENVSCRIPT"; printf 'protocol=https\nhost=git.src.land\n\n' | git credential fill`)
	fill.Env = append(cliEnv(), "ENVSCRIPT="+script, "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	got, err := fill.CombinedOutput()
	if err != nil {
		t.Fatalf("git credential fill: %v %s", err, got)
	}
	if !strings.Contains(string(got), "username=x\n") || !strings.Contains(string(got), "password=sess-token-123\n") {
		t.Fatalf("credential fill = %q", got)
	}
	if n := strings.Count(string(env), "sess-token-123"); n != 1 {
		t.Fatalf("token appears %d times in the env file, want only CITADEL_ACCESS_TOKEN", n)
	}
}

// cliEnv is the environment for a test that launches the CLI binary. Outside
// this test process the binary uses the real backend, so a dead bus keeps it
// off the developer's keyring.
func cliEnv() []string {
	return append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS=unix:path=/nonexistent")
}

func TestEveryBinaryHarnessUsesCLIEnv(t *testing.T) {
	src, err := os.ReadFile("hook_handler_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), ".Env = "+"append(os."+"Environ()") {
		t.Fatal("a harness sets its environment without cliEnv")
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "citadel-cli")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "..") //nolint:gosec // fixed go verb, test-owned path
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v %s", err, b)
	}
	return bin
}

// The ssh wrapper offers the session key to a Citadel host only; any other
// host gets the arguments untouched.
func TestHookGitSSH_HostScoped_Hermetic(t *testing.T) {
	bin := buildCLI(t)
	fakeDir := t.TempDir()
	argsFile := filepath.Join(fakeDir, "args")
	fake := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + shellQ(argsFile) + "\n"
	if err := os.WriteFile(filepath.Join(fakeDir, "ssh"), []byte(fake), 0o700); err != nil { //nolint:gosec // test fake must be executable
		t.Fatal(err)
	}
	run := func(host string) string {
		c := exec.CommandContext(t.Context(), bin, "hook", "git-ssh", "/k/id", "-o", "SendEnv=GIT_PROTOCOL", "-p", "2222", "git@"+host, "git-upload-pack 'a/b.git'") //nolint:gosec // test-owned binary
		c.Env = append(cliEnv(), "PATH="+fakeDir+":"+os.Getenv("PATH"), "CITADEL_GIT_HOSTS=", "XDG_CONFIG_HOME="+t.TempDir(), "HOME="+t.TempDir())
		if b, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git-ssh %s: %v %s", host, err, b)
		}
		b, _ := readFile(argsFile)
		return string(b)
	}
	tail := "-o\nSendEnv=GIT_PROTOCOL\n-p\n2222\ngit@%s\ngit-upload-pack 'a/b.git'\n"
	if got, want := run("src.land"), "-i\n/k/id\n-o\nIdentitiesOnly=yes\n"+strings.Replace(tail, "%s", "src.land", 1); got != want {
		t.Errorf("citadel host args = %q, want %q", got, want)
	}
	if got, want := run("github.com"), strings.Replace(tail, "%s", "github.com", 1); got != want {
		t.Errorf("other host args = %q, want %q", got, want)
	}
}

func shellQ(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func TestHookInstallFlags_Hermetic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".claude", "settings.json")
	runHook(t, "", "install")
	runHook(t, "", "install", "--scopes", "code:read,pulls:write", "--require")
	got, _ := readFile(path)
	s := string(got)
	for _, want := range []string{"hook session-start --scopes code:read,pulls:write", "hook renew --require"} {
		if !strings.Contains(s, want) {
			t.Errorf("settings missing %q:\n%s", want, s)
		}
	}
	if strings.Count(s, "hook renew") != 1 || strings.Count(s, "hook session-start") != 1 {
		t.Errorf("re-install must replace, not duplicate:\n%s", s)
	}
	runHook(t, "", "uninstall")
	if left, _ := readFile(path); strings.Contains(string(left), "citadel-cli") {
		t.Fatalf("uninstall left entries:\n%s", left)
	}
}

// After an idle gap the renew gets 409: a new session on a fresh key (same
// path) replaces it, later CLI calls read the new token from state, and --require denies a
// push only while no session is live.
func TestHookRenewReissueAndRequire_Hermetic(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("CLAUDE_ENV_FILE", filepath.Join(t.TempDir(), "env"))
	t.Setenv("CITADEL_GIT_HOSTS", "")
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	var created atomic.Int32
	var keys []string
	var scopes [][]string
	var gone, revoked atomic.Bool
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/sessions"):
			var body struct {
				Key    string   `json:"ssh_public_key"`
				Scopes []string `json:"scopes"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			keys, scopes = append(keys, body.Key), append(scopes, body.Scopes)
			n := created.Add(1)
			writeJSON(t, w, 201, map[string]any{"session_id": "sess-" + strconv.Itoa(int(n)), "agent_name": "claude-code-box", "token": "tok-" + strconv.Itoa(int(n)), "expires_at": expires})
		case strings.HasSuffix(r.URL.Path, "/renew"):
			if revoked.Load() {
				writeJSON(t, w, 410, map[string]any{"error": "session_revoked"})
				return
			}
			if gone.Load() {
				writeJSON(t, w, 409, map[string]any{"error": "ended"})
				return
			}
			writeJSON(t, w, 200, map[string]any{"expires_at": expires})
		default:
			http.NotFound(w, r)
		}
	})
	repo := gitRepoWithOrigin(t, "git@src.land:acme/app.git")
	in := `{"session_id":"s9","cwd":"` + repo + `","tool_name":"Bash","tool_input":{"command":"git -C . push origin main"}}`
	runHook(t, in, "session-start", "--scopes", "code:read")
	stateFile := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "citadel", "sessions", "s9", "state.json")
	expire := func() {
		near, _ := json.Marshal(map[string]any{"session_id": "sess-1", "agent_name": "claude-code-box", "expires_at": time.Now().Add(-time.Minute), "token": "tok-1", "scopes": []string{"code:read"}})
		if err := os.WriteFile(stateFile, near, 0o600); err != nil { //nolint:gosec // test-owned temp path
			t.Fatal(err)
		}
	}
	if out := runHook(t, in, "renew", "--require"); out != "" {
		t.Fatalf("live session must not be denied: %q", out)
	}

	expire()
	gone.Store(true)
	out := runHook(t, in, "renew", "--require")
	if out != "" {
		t.Fatalf("reissued session must not be denied: %q", out)
	}
	if created.Load() != 2 || keys[0] == keys[1] || !slices.Equal(scopes[1], []string{"code:read"}) {
		t.Fatalf("reissue: created=%d keys=%v scopes=%v", created.Load(), keys, scopes)
	}
	if st, _ := readFile(stateFile); !strings.Contains(string(st), `"token":"tok-2"`) {
		t.Fatalf("state lacks the new token: %s", st)
	}

	cred := exec.CommandContext(t.Context(), buildCLI(t), "hook", "git-credential", "get") //nolint:gosec // test-owned binary
	cred.Env = append(cliEnv(), "CITADEL_ACCESS_TOKEN=stale", "CITADEL_AGENT_SESSION=sess-1", "CITADEL_HOOK_STATE="+filepath.Dir(stateFile))
	if b, err := cred.Output(); err != nil || !strings.Contains(string(b), "password=tok-2\n") {
		t.Fatalf("credential after reissue = %q (%v), want the token from state", b, err)
	}

	// Revoked: stop, drop key and token, and never reissue again.
	expire()
	revoked.Store(true)
	if out := runHook(t, in, "renew", "--require"); !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("revoked session must deny push: %q", out)
	}
	keyFile := filepath.Join(filepath.Dir(stateFile), "id_ed25519")
	if _, err := os.Stat(keyFile); err == nil { //nolint:gosec // test-owned temp path
		t.Fatal("revoked session left its key behind")
	}
	if st, _ := readFile(stateFile); strings.Contains(string(st), "tok-") || !strings.Contains(string(st), `"revoked":true`) {
		t.Fatalf("revoked state = %s", st)
	}
	runHook(t, in, "renew", "--require")
	runHook(t, in, "session-start")
	if created.Load() != 2 {
		t.Fatalf("revoked session was revived: created=%d", created.Load())
	}

	// No state at all: a push to a Citadel host is denied, other commands and hosts are not.
	if err := os.Remove(stateFile); err != nil { //nolint:gosec // test-owned temp path
		t.Fatal(err)
	}
	if out := runHook(t, in, "renew", "--require"); !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("push without a session = %q", out)
	}
	if out := runHook(t, in, "renew"); out != "" {
		t.Fatalf("default stays fail-open: %q", out)
	}
	ls := strings.Replace(in, "git -C . push origin main", "git status", 1)
	if out := runHook(t, ls, "renew", "--require"); out != "" {
		t.Fatalf("non-push denied: %q", out)
	}
	gh := gitRepoWithOrigin(t, "git@github.com:acme/app.git")
	if out := runHook(t, strings.Replace(in, repo, gh, 1), "renew", "--require"); out != "" {
		t.Fatalf("github push denied: %q", out)
	}
}

// `auth git` trades the stored refresh token for a human JWT, registers a fresh
// key under it, and points git's ssh at the host-scoped wrapper.
func TestAuthGit_Hermetic(t *testing.T) {
	bin := buildCLI(t)
	var gotAuth, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth/token":
			writeJSON(t, w, 200, map[string]any{"access_token": "human-jwt", "refresh_token": "rt-2"})
		case "/git-sessions":
			var body struct {
				Key string `json:"ssh_public_key"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotAuth, gotKey = r.Header.Get("Authorization"), body.Key
			writeJSON(t, w, 201, map[string]any{"session_id": "hs-1", "expires_at": time.Now().Add(8 * time.Hour).UTC().Format(time.RFC3339)})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cfgHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cfgHome, "citadel"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgHome, "citadel", "config.toml")
	if err := os.WriteFile(cfgFile, []byte("access_token = \"agent-tok\"\nagent_id = \"a1\"\nrefresh_token = \"rt-1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitCfg := filepath.Join(t.TempDir(), "gitconfig")
	c := exec.CommandContext(t.Context(), bin, "auth", "git") //nolint:gosec // test-owned binary
	c.Env = append(cliEnv(), "CITADEL_SERVER="+srv.URL, "CITADEL_ACCESS_TOKEN=", "XDG_CONFIG_HOME="+cfgHome, "XDG_RUNTIME_DIR="+t.TempDir(), "GIT_CONFIG_GLOBAL="+gitCfg)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("auth git: %v %s", err, out)
	}
	if gotAuth != "Bearer human-jwt" || !strings.HasPrefix(gotKey, "ssh-ed25519 ") {
		t.Fatalf("git-sessions request: auth=%q key=%q", gotAuth, gotKey)
	}
	if g, _ := readFile(gitCfg); !strings.Contains(string(g), "hook git-ssh") {
		t.Fatalf("core.sshCommand not set: %s", g)
	}
	if cfg, _ := readFile(cfgFile); !strings.Contains(string(cfg), "rt-2") {
		t.Fatalf("rotated refresh token not saved: %s", cfg)
	}
}

// Cursor's hooks.json is version 1 with flat entries; install is idempotent,
// replaces changed flags, and uninstall leaves foreign entries alone.
func TestHookInstallCursor_Hermetic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".cursor", "hooks.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version":1,"hooks":{"sessionStart":[{"command":"./mine.sh"}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runHook(t, "", "install", "--client", "cursor")
	runHook(t, "", "install", "--client", "cursor", "--require")
	var cfg struct {
		Version int `json:"version"`
		Hooks   map[string][]struct {
			Command string `json:"command"`
			Type    string `json:"type"`
			Timeout int    `json:"timeout"`
		} `json:"hooks"`
	}
	raw, _ := readFile(path)
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("hooks.json: %v\n%s", err, raw)
	}
	if cfg.Version != 1 || len(cfg.Hooks["sessionStart"]) != 2 || len(cfg.Hooks["sessionEnd"]) != 1 || len(cfg.Hooks["preToolUse"]) != 1 {
		t.Fatalf("unexpected layout:\n%s", raw)
	}
	if got := cfg.Hooks["preToolUse"][0]; got.Command != "citadel-cli hook renew --client cursor --require" || got.Type != "command" || got.Timeout != 10 {
		t.Fatalf("preToolUse entry = %+v", got)
	}
	runHook(t, "", "uninstall", "--client", "cursor")
	left, _ := readFile(path)
	if !strings.Contains(string(left), "./mine.sh") || strings.Contains(string(left), "citadel-cli") {
		t.Fatalf("uninstall must remove only citadel entries:\n%s", left)
	}
}

// Under Cursor the session variables travel in sessionStart's JSON output, the
// session id on later events is conversation_id, and an allowed tool call still
// answers with an explicit permission.
func TestHookCursorSession_Hermetic(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("CITADEL_GIT_HOSTS", "")
	expires := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sessions") {
			var body struct {
				Client string `json:"client"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Client != "cursor" || !strings.HasPrefix(r.URL.Path, "/agents/cursor-") {
				t.Errorf("session opened as client %q on %s (%v)", body.Client, r.URL.Path, err)
			}
			writeJSON(t, w, 201, map[string]any{"session_id": "cs-1", "agent_name": "cursor-box", "token": "tok-c", "expires_at": expires})
			return
		}
		http.NotFound(w, r)
	})
	repo := gitRepoWithOrigin(t, "git@src.land:acme/app.git")
	start := `{"session_id":"c1","workspace_roots":["` + repo + `"]}`
	var out struct {
		Env               map[string]string `json:"env"`
		AdditionalContext string            `json:"additional_context"`
	}
	if err := json.Unmarshal([]byte(runHook(t, start, "session-start", "--client", "cursor")), &out); err != nil {
		t.Fatal(err)
	}
	if out.Env["CITADEL_ACCESS_TOKEN"] != "tok-c" || !strings.Contains(out.Env["GIT_SSH_COMMAND"], "hook git-ssh") || out.AdditionalContext == "" {
		t.Fatalf("sessionStart output = %+v", out)
	}
	local := func(repo, key string) string {
		b, _ := exec.CommandContext(t.Context(), "git", "-C", repo, "config", "--local", "--get-all", key).Output() //nolint:gosec // fixed git verb, test-owned args
		return strings.TrimSpace(string(b))
	}
	if !strings.Contains(local(repo, "core.sshCommand"), "hook git-ssh") {
		t.Fatalf("session-start left no repo-local core.sshCommand: %q", local(repo, "core.sshCommand"))
	}
	pre := `{"conversation_id":"c1","tool_name":"Shell","cwd":"` + repo + `","tool_input":{"command":"git push origin main"}}`
	got := runHook(t, pre, "renew", "--client", "cursor")
	var allow struct {
		Permission   string `json:"permission"`
		UpdatedInput struct {
			Command string `json:"command"`
		} `json:"updated_input"`
	}
	if err := json.Unmarshal([]byte(got), &allow); err != nil || allow.Permission != "allow" ||
		!strings.Contains(allow.UpdatedInput.Command, "hook git-ssh") || !strings.HasSuffix(allow.UpdatedInput.Command, "; git push origin main") {
		t.Fatalf("live session preToolUse = %q (%v)", got, err)
	}
	// citadel-cli in a Shell must act as the session, not the stored login.
	cli := strings.Replace(pre, "git push origin main", "citadel-cli kg symbols --q Greet", 1)
	if err := json.Unmarshal([]byte(runHook(t, cli, "renew", "--client", "cursor")), &allow); err != nil {
		t.Fatal(err)
	}
	if c := allow.UpdatedInput.Command; !strings.Contains(c, "CITADEL_AGENT_SESSION='cs-1'") || !strings.Contains(c, "CITADEL_HOOK_STATE=") ||
		strings.Contains(c, "tok-c") || strings.Contains(c, "git()") || !strings.HasSuffix(c, "; citadel-cli kg symbols --q Greet") {
		t.Fatalf("citadel-cli preToolUse = %q", c)
	}
	plain := strings.Replace(pre, "git push origin main", "ls", 1)
	if got := runHook(t, plain, "renew", "--client", "cursor"); strings.Contains(got, "updated_input") {
		t.Fatalf("non-git command rewritten: %q", got)
	}
	runHook(t, start, "session-end", "--client", "cursor")
	if got := local(repo, "core.sshCommand"); got != "" {
		t.Fatalf("session-end left core.sshCommand behind: %q", got)
	}

	// An https origin also gets the credential helper, reading the state token.
	web := gitRepoWithOrigin(t, "https://src.land/acme/app.git")
	runHook(t, `{"session_id":"c3","workspace_roots":["`+web+`"]}`, "session-start", "--client", "cursor")
	if got := local(web, "credential.https://src.land.helper"); !strings.Contains(got, "hook git-credential --state") {
		t.Fatalf("credential helper = %q", got)
	}
	// A push names its own session's helper, whatever the shared repo config holds.
	webPre := `{"session_id":"c3","tool_name":"Shell","workspace_roots":["` + web + `"],"tool_input":{"command":"git push"}}`
	if got := runHook(t, webPre, "renew", "--client", "cursor"); !strings.Contains(got, "credential.https://src.land.helper=!") || !strings.Contains(got, filepath.Join("sessions", "c3")) {
		t.Fatalf("https preToolUse does not name the session helper: %q", got)
	}
	runHook(t, `{"session_id":"c3","workspace_roots":["`+web+`"]}`, "session-end", "--client", "cursor")
	if got := local(web, "credential.https://src.land.helper"); got != "" {
		t.Fatalf("session-end left the credential helper: %q", got)
	}

	// No state for another conversation: --require denies the push in Cursor's shape.
	other := strings.Replace(pre, `"c1"`, `"c2"`, 1)
	if got := runHook(t, other, "renew", "--client", "cursor", "--require"); !strings.Contains(got, `"permission":"deny"`) {
		t.Fatalf("require without session = %q", got)
	}
}

func TestHookInstallNamesTheServer(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	t.Setenv("CITADEL_SERVER", "")
	out := runHook(t, "", "install")
	if !strings.Contains(out, "server: https://mcp.src.land (built-in default") {
		t.Fatalf("default server not named: %q", out)
	}

	t.Setenv("CITADEL_SERVER", "https://forge.internal.example")
	out = runHook(t, "", "install")
	if !strings.Contains(out, "server: https://forge.internal.example (from CITADEL_SERVER)") {
		t.Fatalf("configured server not named: %q", out)
	}
}

// Cursor reads cli-config.json from CURSOR_CONFIG_DIR, then XDG_CONFIG_HOME/cursor;
// its user hooks.json stays in ~/.cursor.
func TestHookInstallCursorRuleFollowsConfigDir_Hermetic(t *testing.T) {
	home, xdg, dir := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("CURSOR_CONFIG_DIR", "")
	runHook(t, "", "install", "--client", "cursor")
	if raw, err := readFile(filepath.Join(xdg, "cursor", "cli-config.json")); err != nil || !strings.Contains(string(raw), "citadel-cli:kg") {
		t.Fatalf("XDG_CONFIG_HOME rule: %q (%v)", raw, err)
	}
	t.Setenv("CURSOR_CONFIG_DIR", dir)
	runHook(t, "", "install", "--client", "cursor")
	if raw, err := readFile(filepath.Join(dir, "cli-config.json")); err != nil || !strings.Contains(string(raw), "citadel-cli:kg") {
		t.Fatalf("CURSOR_CONFIG_DIR rule: %q (%v)", raw, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cursor", "hooks.json")); err != nil {
		t.Fatalf("user hooks.json: %v", err)
	}
}

// Cursor refuses to start when permissions lacks allow or deny, so the project
// cli.json this writes carries both, an earlier install is repaired, and
// uninstall leaves no half-object behind.
func TestHookInstallCursorProjectPermissionsValid_Hermetic(t *testing.T) {
	t.Chdir(t.TempDir())
	path := filepath.Join(".cursor", "cli.json")
	lists := func() map[string]any {
		raw, err := readFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var cfg struct {
			Permissions map[string]any `json:"permissions"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			t.Fatal(err)
		}
		return cfg.Permissions
	}
	runHook(t, "", "install", "--client", "cursor", "--scope", "project")
	if p := lists(); p["allow"] == nil || p["deny"] == nil {
		t.Fatalf("fresh cli.json permissions = %v", p)
	}
	if err := os.WriteFile(path, []byte(`{"permissions":{"allow":["Shell(citadel-cli:kg*)"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	runHook(t, "", "install", "--client", "cursor", "--scope", "project")
	if p := lists(); p["deny"] == nil {
		t.Fatalf("repaired cli.json permissions = %v", p)
	}
	runHook(t, "", "uninstall", "--client", "cursor", "--scope", "project")
	if p := lists(); p != nil {
		t.Fatalf("after uninstall permissions = %v", p)
	}
}

// Install allows graph queries and keeps other rules; uninstall removes only its own.
func TestHookInstallAllowsGraphQueries_Hermetic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CURSOR_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	for _, tc := range []struct{ client, file, rule string }{
		{"cursor", filepath.Join(home, ".cursor", "cli-config.json"), "Shell(citadel-cli:kg*)"},
		{"claude-code", filepath.Join(home, ".claude", "settings.json"), "Bash(citadel-cli kg *)"},
	} {
		if err := os.MkdirAll(filepath.Dir(tc.file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tc.file, []byte(`{"keep":1,"permissions":{"allow":["Shell(ls)"]}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		runHook(t, "", "install", "--client", tc.client)
		runHook(t, "", "install", "--client", tc.client)
		raw, _ := readFile(tc.file)
		if strings.Count(string(raw), tc.rule) != 1 || !strings.Contains(string(raw), "Shell(ls)") || !strings.Contains(string(raw), `"keep"`) {
			t.Fatalf("%s after install:\n%s", tc.client, raw)
		}
		runHook(t, "", "uninstall", "--client", tc.client)
		raw, _ = readFile(tc.file)
		if strings.Contains(string(raw), tc.rule) || !strings.Contains(string(raw), "Shell(ls)") {
			t.Fatalf("%s after uninstall:\n%s", tc.client, raw)
		}
	}
}
