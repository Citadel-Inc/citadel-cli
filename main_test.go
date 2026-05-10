package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestRun_HelpExitsZero(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"--help"}, &stderr); got != 0 {
		t.Fatalf("run --help exit = %d, stderr=%q", got, stderr.String())
	}
}

func TestRun_VersionExitsZero(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"--version"}, &stderr); got != 0 {
		t.Fatalf("run --version exit = %d", got)
	}
}

func TestRun_UnknownCommand_ExitsOne(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"definitely-not-a-command"}, &stderr); got != 1 {
		t.Fatalf("run unknown exit = %d", got)
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Errorf("stderr missing Error prefix: %q", stderr.String())
	}
}

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	root := newRootCmd()
	want := []string{
		"auth", "token", "mcp", "kg", "repo", "namespace", "org",
		"agent", "oauth", "ssh-key", "completion", "doctor", "man", "audit",
		"issue",
		"search", "project",
	}
	for _, name := range want {
		found := false
		for _, sub := range root.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing subcommand %q", name)
		}
	}
	// The persistent --server flag must be present so subcommands inherit it.
	if root.PersistentFlags().Lookup("server") == nil {
		t.Error("expected persistent --server flag")
	}
}

func TestAuthSubcommands(t *testing.T) {
	// Verify auth subcommands exist
	if cmd.AuthCmd == nil {
		t.Fatal("AuthCmd is nil")
	}
	if len(cmd.AuthCmd.Commands()) == 0 {
		t.Error("AuthCmd has no subcommands")
	}
	expectedAuth := []string{"login", "status", "logout"}
	for _, expected := range expectedAuth {
		found := false
		for _, subcmd := range cmd.AuthCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected auth subcommand %q not found", expected)
		}
	}
}

func TestTokenSubcommands(t *testing.T) {
	// Verify token subcommands exist
	if cmd.TokenCmd == nil {
		t.Fatal("TokenCmd is nil")
	}
	if len(cmd.TokenCmd.Commands()) == 0 {
		t.Error("TokenCmd has no subcommands")
	}
	expectedToken := []string{"list", "issue", "revoke"}
	for _, expected := range expectedToken {
		found := false
		for _, subcmd := range cmd.TokenCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected token subcommand %q not found", expected)
		}
	}
}

func TestMcpSubcommands(t *testing.T) {
	// Verify mcp subcommands exist
	if cmd.McpCmd == nil {
		t.Fatal("McpCmd is nil")
	}
	if len(cmd.McpCmd.Commands()) == 0 {
		t.Error("McpCmd has no subcommands")
	}
	expectedMcp := []string{"tools", "call", "resources", "prompts"}
	for _, expected := range expectedMcp {
		found := false
		for _, subcmd := range cmd.McpCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected mcp subcommand %q not found", expected)
		}
	}
}

func TestRunWriters_NamespaceListJSON_NoToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	var stdout, stderr bytes.Buffer
	code := runWriters([]string{"namespace", "list", "--output", "json"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit %d want 3 (auth_required)", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr=%q want empty", stderr.String())
	}
	var outer struct {
		Error struct {
			Kind string `json:"kind"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &outer); err != nil {
		t.Fatalf("stdout=%q: %v", stdout.String(), err)
	}
	if outer.Error.Kind != "auth_required" {
		t.Fatalf("kind=%q full=%s", outer.Error.Kind, stdout.String())
	}
}

func TestRunWriters_NamespaceListHuman_NoToken(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	var stdout, stderr bytes.Buffer
	code := runWriters([]string{"namespace", "list"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit %d want 3", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout=%q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error:") || !strings.Contains(stderr.String(), "not authenticated") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestRunWriters_JSONThenHuman_NoStaleOutputFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	var s1, e1 bytes.Buffer
	if c := runWriters([]string{"namespace", "list", "--output", "json"}, &s1, &e1); c != 3 {
		t.Fatalf("first exit %d", c)
	}
	var s2, e2 bytes.Buffer
	if c := runWriters([]string{"namespace", "list"}, &s2, &e2); c != 3 {
		t.Fatalf("second exit %d", c)
	}
	if s2.Len() != 0 {
		t.Fatalf("second stdout=%q want empty (human error path)", s2.String())
	}
	if !strings.Contains(e2.String(), "Error:") {
		t.Fatalf("second stderr=%q", e2.String())
	}
}

func mcpJSONRPCHandler(t *testing.T, byMethod map[string]func() any) http.HandlerFunc {
	t.Helper()
	if _, ok := byMethod["initialize"]; !ok {
		byMethod["initialize"] = func() any {
			return map[string]any{
				"protocolVersion": "2025-11-25",
				"serverInfo":      map[string]any{"name": "citadel-mcp-test", "version": "1"},
			}
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode MCP request: %v", err)
		}
		fn, ok := byMethod[req.Method]
		if !ok {
			t.Fatalf("unhandled MCP method %q", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "test-sess")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  fn(),
		})
	}
}

// tools/call with isError=true is a normal server outcome; main maps it to exit 2.
func TestRunWriters_McpToolCallError_ExitCode2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		mcpJSONRPCHandler(t, map[string]func() any{
			"tools/call": func() any {
				return map[string]any{
					"isError": true,
					"content": []map[string]any{{"type": "text", "text": "boom"}},
				}
			},
		})(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	var stdout, stderr bytes.Buffer
	code := runWriters([]string{"mcp", "call", "x"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit=%d want 2 stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
}

func TestRunWriters_RepoNotFound_JSONEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/namespaces/acme/wheat" {
			http.NotFound(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	var stdout, stderr bytes.Buffer
	code := runWriters([]string{"repo", "get", "-R", "acme/wheat", "--output", "json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit=%d want 1 stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var outer struct {
		Error struct {
			Kind    string `json:"kind"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &outer); err != nil {
		t.Fatalf("stdout=%q: %v", stdout.String(), err)
	}
	if outer.Error.Kind != "internal" {
		t.Fatalf("kind=%q want internal for fmt-wrapped repo handler error stdout=%q", outer.Error.Kind, stdout.String())
	}
	if !strings.Contains(outer.Error.Message, "not found") {
		t.Fatalf("message=%q", outer.Error.Message)
	}
}

// GET repo list uses the idempotent retry stack; three consecutive 429s exhaust retries,
// surfacing the last response (including Retry-After) to the JSON error envelope.
func TestRunWriters_RepoList_JSON429_RetryAfterOnFinalResponse(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/namespaces/myorg/repos" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		n := calls.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
		} else {
			w.Header().Set("Retry-After", "77")
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	var stdout, stderr bytes.Buffer
	code := runWriters([]string{"repo", "list", "--namespace", "myorg", "--output", "json"}, &stdout, &stderr)
	if code != 6 {
		t.Fatalf("exit=%d want 6 (rate_limited) stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr=%q want empty for json envelope path", stderr.String())
	}
	var outer struct {
		Error struct {
			Kind              string  `json:"kind"`
			RetryAfterSeconds float64 `json:"retry_after_seconds"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &outer); err != nil {
		t.Fatalf("stdout=%q: %v", stdout.String(), err)
	}
	if outer.Error.Kind != "rate_limited" {
		t.Fatalf("kind=%q stdout=%s", outer.Error.Kind, stdout.String())
	}
	if int(outer.Error.RetryAfterSeconds) != 77 {
		t.Fatalf("retry_after_seconds=%v want 77 stdout=%s", outer.Error.RetryAfterSeconds, stdout.String())
	}
	if calls.Load() != 3 {
		t.Fatalf("server calls=%d want 3", calls.Load())
	}
}
