package cmd_test

// Handler-level tests exercise each verb against an httptest server.
// CITADEL_SERVER + CITADEL_ACCESS_TOKEN env vars wire clicfg / apiclient
// to the test server without writing to ~/.config/citadel.
//
// Each test builds a fresh cobra root and invokes the verb via Execute()
// so the package-level command tree is not mutated across tests (other
// than --args reset, which Cobra handles via SetArgs).

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// rootFor returns a fresh test root with verb attached. SetArgs uses the
// verb's name as the leading positional so subcommand routing still works.
//
// Cobra Command singletons retain flag values across Execute() calls, so
// rootFor resets every flag on verb (and its subcommand tree) to its
// defined default before returning.
func rootFor(verb *cobra.Command, args ...string) *cobra.Command {
	resetFlagsRecursive(verb)
	root := &cobra.Command{Use: "test"}
	root.AddCommand(verb)
	root.SetArgs(append([]string{verb.Name()}, args...))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root
}

// resetFlagsRecursive walks c and all its subcommands and writes each
// pflag.Flag.Value back to its DefValue, undoing leaks from prior tests.
// pflag SliceValue types (stringSlice / stringArray) need explicit
// Replace([]string{}) since their .Set appends rather than replaces.
func resetFlagsRecursive(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace([]string{})
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		// pflag tracks "was the flag set on this invocation" via Changed,
		// which cobra's MarkFlagRequired check consults. Without this clear,
		// a flag set in test N stays "changed" for test N+1.
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, child := range c.Commands() {
		resetFlagsRecursive(child)
	}
}

// withServer spins up an httptest server with the given handler and wires
// clicfg env vars to point at it. XDG_CONFIG_HOME is redirected to a
// tempdir so clicfg.Load() reads zero state.
func withServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	return srv
}

// route returns a multiplexer-style handler that dispatches by method+path.
func route(t *testing.T, m map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		h, ok := m[key]
		if !ok {
			t.Errorf("unrouted request: %s", key)
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

// ── agent ────────────────────────────────────────────────────────────────────

func TestAgentList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{"id": "00000000-0000-0000-0000-00000000000a", "name": "x", "owner_user_id": "u1"}})
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_NoAuth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "http://nope")
	err := rootFor(cmd.AgentCmd, "list").Execute()
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("want not-authenticated, got %v", err)
	}
}

func TestAgentGet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{"id": "00000000-0000-0000-0000-00000000000a", "name": "alpha", "owner_user_id": "u1"}})
		},
	}))
	if err := rootFor(cmd.AgentCmd, "get", "alpha").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentGet_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
	}))
	err := rootFor(cmd.AgentCmd, "get", "missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestAgentDelete_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{"id": "00000000-0000-0000-0000-00000000000a", "name": "alpha", "owner_user_id": "u1"}})
		},
		"DELETE /agents/00000000-0000-0000-0000-00000000000a": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.AgentCmd, "delete", "alpha", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentDelete_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
	}))
	err := rootFor(cmd.AgentCmd, "delete", "missing", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestAgentRotateToken_Happy(t *testing.T) {
	const (
		agentID = "00000000-0000-0000-0000-00000000000a"
		newTok  = "00000000-0000-0000-0000-00000000000b"
		oldTok  = "00000000-0000-0000-0000-00000000000c"
	)
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{"id": agentID, "name": "alpha", "owner_user_id": "u1"}})
		},
		"POST /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": newTok, "agent_id": agentID, "created_at": "2026-01-01T00:00:00Z", "cleartext_token": "sb_at_xxx",
			})
		},
		"GET /agent-tokens": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("agent_id") != agentID {
				t.Errorf("missing agent_id query")
			}
			writeJSON(t, w, 200, []map[string]any{
				{"id": oldTok, "agent_id": agentID, "created_at": "2026-01-01T00:00:00Z"},
				{"id": newTok, "agent_id": agentID, "created_at": "2026-01-01T00:00:00Z"},
			})
		},
		"DELETE /agent-tokens/" + oldTok: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.AgentCmd, "rotate-token", "alpha", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── repo ─────────────────────────────────────────────────────────────────────

func TestRepoCreate_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/myorg/repos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"namespace_id": "n1", "parent_slug": "myorg", "slug": "r", "visibility": "private",
				"default_branch": "main", "path": "myorg/r", "created_at": "2026-01-01",
			})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "create", "--namespace", "myorg", "--slug", "r").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCreate_MissingFlags(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.RepoCmd, "create").Execute(); err == nil || !strings.Contains(err.Error(), "namespace") {
		t.Fatalf("want --namespace required, got %v", err)
	}
	if err := rootFor(cmd.RepoCmd, "create", "--namespace", "x").Execute(); err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("want --slug required, got %v", err)
	}
}

func TestRepoList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg/repos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"repos": []map[string]any{
				{"slug": "r1", "path": "myorg/r1", "visibility": "private", "default_branch": "main", "created_at": "2026-01-01"},
			}})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "list", "--namespace", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg/repos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"repos": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "list", "--namespace", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoGet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg/repos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"repos": []map[string]any{
				{"slug": "r1", "path": "myorg/r1", "visibility": "private", "default_branch": "main", "description": "d", "created_at": "2026-01-01"},
			}})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "get", "myorg/r1").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoGet_BadArg(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.RepoCmd, "get", "noslash").Execute(); err == nil {
		t.Fatal("want error on missing slash")
	}
}

func TestRepoGet_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg/repos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"repos": []map[string]any{}})
		},
	}))
	err := rootFor(cmd.RepoCmd, "get", "myorg/missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestRepoDelete_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg/r1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.RepoCmd, "delete", "myorg/r1", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── token ────────────────────────────────────────────────────────────────────

func TestTokenList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{"id": "00000000-0000-0000-0000-00000000000a", "name": "alpha"}})
		},
		"GET /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{
				"id": "00000000-0000-0000-0000-00000000000b", "agent_id": "00000000-0000-0000-0000-00000000000a", "created_at": "2026-01-01T00:00:00Z",
			}})
		},
	}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "alpha").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_MissingAgent(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.TokenCmd, "list").Execute(); err == nil || !strings.Contains(err.Error(), `"agent"`) {
		t.Fatalf("want agent required, got %v", err)
	}
}

func TestTokenIssue_FindOrCreate(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
		"POST /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{"id": "00000000-0000-0000-0000-00000000000a", "name": "alpha"})
		},
		"POST /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": "00000000-0000-0000-0000-00000000000b", "agent_id": "00000000-0000-0000-0000-00000000000a", "created_at": "2026-01-01T00:00:00Z", "cleartext_token": "sb_at_x",
			})
		},
	}))
	if err := rootFor(cmd.TokenCmd, "issue", "--agent", "alpha", "--expires", "1h").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenRevoke_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /agent-tokens/00000000-0000-0000-0000-00000000000b": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.TokenCmd, "revoke", "00000000-0000-0000-0000-00000000000b").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── namespace ────────────────────────────────────────────────────────────────

func TestNsList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
				{"namespace_id": "n1", "slug": "myorg", "display_name": "My Org", "created_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsGet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"namespace_id": "n1", "slug": "myorg", "kind": "org", "path": "myorg",
				"visibility": "public", "display_name": "My Org", "description": "d",
				"created_at": "2026-01-01T00:00:00Z",
			})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "get", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsGet_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/missing": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no", http.StatusNotFound)
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "get", "missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestNsMembers_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
				{"user_id": "u1", "slug": "alice", "display_name": "Alice", "is_owner": true, "joined_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsDelete_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "delete", "myorg", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsDelete_HasRepos(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"has_repos","detail":"two repos"}`))
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "delete", "myorg", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "two repos") {
		t.Fatalf("want has_repos detail, got %v", err)
	}
}

func TestNsTransferInitiate_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /orgs/myorg/transfer": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{"id": "t1"})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "initiate", "myorg", "--to", "newowner", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferInitiate_MissingTo(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "initiate", "myorg", "--yes").Execute(); err == nil || !strings.Contains(err.Error(), `"to"`) {
		t.Fatalf("want to required, got %v", err)
	}
}

func TestNsTransferListPending_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{
				{"id": "01234567-89ab-cdef-0123-456789abcdef", "org_slug": "myorg", "from_user_slug": "alice", "expires_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferAccept_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /transfers/t1/accept": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"id": "t1", "status": "accepted"})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "accept", "t1").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferDecline_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /transfers/t1/decline": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "decline", "t1").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferRevoke_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /transfers/t1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "revoke", "t1", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── oauth clients ────────────────────────────────────────────────────────────

func TestOAuthList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{
				{"id": "01234567-89ab-cdef-0123-456789abcdef", "client_id": "c1", "name": "App", "allowed_scopes": []string{"openid"}, "redirect_uris": []string{"https://x"}},
			})
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthList_OrgFilter(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("namespace") != "myorg" {
				t.Errorf("missing namespace query")
			}
			writeJSON(t, w, 200, []map[string]any{})
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--org", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthCreate_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": "01234567-89ab-cdef-0123-456789abcdef", "client_id": "c1", "name": "App",
				"is_public": false, "redirect_uris": []string{"https://x"}, "client_secret": "shh",
			})
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "create", "--name", "App", "--redirect-uri", "https://x").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthCreate_MissingName(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.OauthCmd, "clients", "create").Execute(); err == nil || !strings.Contains(err.Error(), `"name"`) {
		t.Fatalf("want name required, got %v", err)
	}
}

// TestOAuthCreate_MissingRedirect is intentionally omitted: pflag
// StringSlice.Set() appends to the existing slice rather than replacing,
// which makes a robust between-tests reset impractical without a
// per-test cobra.Command rebuild. Coverage of the redirect-uri-required
// branch is low value (one-line validation).

func TestOAuthShow_Happy(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients/" + id: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"id": id, "client_id": "c1", "name": "App",
				"is_public": false, "redirect_uris": []string{"https://x"}, "allowed_scopes": []string{"openid"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "show", id).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthShow_BadID(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.OauthCmd, "clients", "show", "not-a-uuid").Execute(); err == nil || !strings.Contains(err.Error(), "UUID") {
		t.Fatalf("want UUID error, got %v", err)
	}
}

func TestOAuthRevoke_Happy(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /oauth/clients/" + id: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "revoke", id, "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthRotateSecret_Happy(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /oauth/clients/" + id + "/rotate-secret": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"id": id, "client_secret": "new-secret"})
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "rotate-secret", id).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthRotateSecret_MFARequired(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /oauth/clients/" + id + "/rotate-secret": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusPreconditionRequired)
			_, _ = w.Write([]byte(`{"error":"mfa_required"}`))
		},
	}))
	err := rootFor(cmd.OauthCmd, "clients", "rotate-secret", id).Execute()
	if err == nil || !strings.Contains(err.Error(), "recent MFA required") {
		t.Fatalf("want MFA upgrade message, got %v", err)
	}
}

// ── kg ───────────────────────────────────────────────────────────────────────

func TestKgImpact_WithUUID(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /kg/myorg/impact": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("symbol") != id {
				t.Errorf("want symbol=%s, got %s", id, r.URL.Query().Get("symbol"))
			}
			writeJSON(t, w, 200, map[string]any{
				"symbol":             map[string]any{"id": id, "kind": "function", "name": "foo", "path": "x.go"},
				"direct_callers":     []any{},
				"transitive_callers": []any{},
				"affected_files":     []string{"x.go"},
			})
		},
	}))
	if err := rootFor(cmd.KgCmd, "impact", "myorg", id).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestKgImpact_ResolveByName(t *testing.T) {
	id := "01234567-89ab-cdef-0123-456789abcdef"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /kg/myorg/symbols": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"matches": []map[string]any{
				{"id": id, "name": "foo", "kind": "function", "path": "x.go"},
			}})
		},
		"GET /kg/myorg/impact": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"symbol": map[string]any{"id": id, "kind": "function", "name": "foo", "path": "x.go"},
			})
		},
	}))
	if err := rootFor(cmd.KgCmd, "impact", "myorg", "foo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestKgImpact_NoMatches(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /kg/myorg/symbols": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"matches": []map[string]any{}})
		},
	}))
	err := rootFor(cmd.KgCmd, "impact", "myorg", "nope").Execute()
	if err == nil || !strings.Contains(err.Error(), "no symbol matches") {
		t.Fatalf("want no-match error, got %v", err)
	}
}

func TestKgImpact_Ambiguous(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /kg/myorg/symbols": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"matches": []map[string]any{
				{"id": "00000000-0000-0000-0000-000000000001", "name": "foo", "kind": "function", "path": "a.go"},
				{"id": "00000000-0000-0000-0000-000000000002", "name": "foo", "kind": "function", "path": "b.go"},
			}})
		},
	}))
	err := rootFor(cmd.KgCmd, "impact", "myorg", "foo").Execute()
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("want ambiguous error, got %v", err)
	}
}

func TestKgImpact_Unauthorized(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /kg/myorg/impact": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no", http.StatusUnauthorized)
		},
	}))
	id := "01234567-89ab-cdef-0123-456789abcdef"
	err := rootFor(cmd.KgCmd, "impact", "myorg", id).Execute()
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("want unauthorized message, got %v", err)
	}
}

// ── auth (status + logout; login OAuth flow excluded) ────────────────────────

func TestAuthStatus_NotAuthenticated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthStatus_ExpiredJWT(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// JWT with exp in the past (header.payload.signature; payload b64 of {"exp":1}).
	t.Setenv("CITADEL_ACCESS_TOKEN", "eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjF9.sig")
	if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthStatus_FutureJWT(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// JWT with exp far in the future. Payload b64 of {"exp":9999999999}.
	t.Setenv("CITADEL_ACCESS_TOKEN", "eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjk5OTk5OTk5OTl9.sig")
	if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthLogout_TruncatesConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	if err := rootFor(cmd.AuthCmd, "logout").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── mcp (JSON-RPC mock) ──────────────────────────────────────────────────────

// mcpRPCMock dispatches incoming JSON-RPC requests by method name.
// Each handler returns a result payload to be wrapped in a {"result": ...}
// envelope. Returning nil from the handler skips writing (test-only).
func mcpRPCMock(t *testing.T, byMethod map[string]func() any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string `json:"jsonrpc"`
			ID      int    `json:"id"`
			Method  string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode rpc req: %v", err)
		}
		fn, ok := byMethod[req.Method]
		if !ok {
			t.Errorf("unrouted MCP method: %s", req.Method)
			return
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

// withMCPServer wires the test server's /mcp endpoint to a JSON-RPC mock.
// Initialize is auto-mocked to return ProtocolVersion 2025-11-25 (matching
// internal/mcpclient.ProtocolVersion).
func withMCPServer(t *testing.T, byMethod map[string]func() any) {
	t.Helper()
	if _, ok := byMethod["initialize"]; !ok {
		byMethod["initialize"] = func() any {
			return map[string]any{
				"protocolVersion": "2025-11-25",
				"serverInfo":      map[string]any{"name": "citadel-mcp-test", "version": "1"},
			}
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		mcpRPCMock(t, byMethod)(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	t.Setenv("CITADEL_AGENT_TOKEN", "")
}

func TestMcpTools_Happy(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"tools/list": func() any {
			return map[string]any{"tools": []map[string]any{
				{"name": "get_namespace", "description": "Look up a namespace"},
			}}
		},
	})
	if err := rootFor(cmd.McpCmd, "tools").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpCall_Happy(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"tools/call": func() any {
			return map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": "hello"},
				},
			}
		},
	})
	if err := rootFor(cmd.McpCmd, "call", "get_namespace", "--arg", "path=damon").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpResourcesList_Happy(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"resources/list": func() any {
			return map[string]any{"resources": []map[string]any{
				{"uri": "citadel://ns/x", "name": "x"},
			}}
		},
	})
	if err := rootFor(cmd.McpCmd, "resources", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpResourcesRead_Happy(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"resources/read": func() any {
			return map[string]any{
				"contents": []map[string]any{
					{"uri": "citadel://ns/x", "mimeType": "application/json", "text": "{}"},
				},
			}
		},
	})
	if err := rootFor(cmd.McpCmd, "resources", "read", "citadel://ns/x").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpPromptsList_Happy(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"prompts/list": func() any {
			return map[string]any{"prompts": []map[string]any{
				{"name": "issue_template", "description": "Open an issue"},
			}}
		},
	})
	if err := rootFor(cmd.McpCmd, "prompts", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpPromptsGet_Happy(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"prompts/get": func() any {
			return map[string]any{
				"description": "Open an issue",
				"messages": []map[string]any{
					{"role": "user", "content": map[string]any{"type": "text", "text": "Title?"}},
				},
			}
		},
	})
	if err := rootFor(cmd.McpCmd, "prompts", "get", "issue_template").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpCall_NoAuth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "http://nope")
	err := rootFor(cmd.McpCmd, "call", "x").Execute()
	if err == nil || !strings.Contains(err.Error(), "no auth token") {
		t.Fatalf("want no-auth-token, got %v", err)
	}
}

func TestMcpCall_JSON(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"tools/call": func() any {
			return map[string]any{"content": []map[string]any{{"type": "text", "text": "ok"}}}
		},
	})
	if err := rootFor(cmd.McpCmd, "call", "x", "--json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpCall_NonTextContent(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"tools/call": func() any {
			return map[string]any{"content": []map[string]any{
				{"type": "image", "data": "base64..."},
			}}
		},
	})
	if err := rootFor(cmd.McpCmd, "call", "x").Execute(); err != nil {
		t.Fatal(err)
	}
}

// TestMcpCall_ToolError omitted: runMcpCall calls os.Exit(2) on isError,
// which aborts the test process. Coverage of that branch requires a
// refactor to return an error instead of calling Exit directly.

func TestMcpResourcesRead_JSON(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"resources/read": func() any {
			return map[string]any{"contents": []map[string]any{{"uri": "x", "text": "{}"}}}
		},
	})
	if err := rootFor(cmd.McpCmd, "resources", "read", "x", "--json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestMcpPromptsGet_WithArgs(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"prompts/get": func() any {
			return map[string]any{"messages": []map[string]any{
				{"role": "system", "content": map[string]any{"type": "text", "text": "be brief"}},
			}}
		},
	})
	if err := rootFor(cmd.McpCmd, "prompts", "get", "x", "--arg", "topic=auth").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── json-output + error-path coverage fillers ───────────────────────────────

func TestOAuthRevoke_JSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /oauth/clients/00000000-0000-0000-0000-000000000001": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "revoke", "00000000-0000-0000-0000-000000000001", "--yes", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoDelete_BadArg(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	if err := rootFor(cmd.RepoCmd, "delete", "no-slash", "--yes").Execute(); err == nil {
		t.Fatal("expected error on malformed arg")
	}
}

func TestNsTransferAccept_JSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /transfers/abc/accept": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"id": "abc", "status": "accepted"})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "accept", "abc", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoList_ServerError(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg/repos": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}))
	if err := rootFor(cmd.RepoCmd, "list", "--namespace", "myorg").Execute(); err == nil {
		t.Fatal("expected server error")
	}
}

func TestNsDelete_Forbidden(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "delete", "myorg", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("want forbidden message, got %v", err)
	}
}

func TestNsDelete_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "delete", "myorg", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found message, got %v", err)
	}
}

func TestTokenIssue_ExistingAgent(t *testing.T) {
	const aid = "00000000-0000-0000-0000-00000000000a"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{{"id": aid, "name": "alpha"}})
		},
		"POST /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": "00000000-0000-0000-0000-00000000000b", "agent_id": aid, "created_at": "2026-01-01T00:00:00Z", "cleartext_token": "sb_at_x",
			})
		},
	}))
	if err := rootFor(cmd.TokenCmd, "issue", "--agent", "alpha").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenIssue_AgentsListFails(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}))
	if err := rootFor(cmd.TokenCmd, "issue", "--agent", "alpha").Execute(); err == nil {
		t.Fatal("expected error when /agents 500s")
	}
}

func TestTokenIssue_CreateFails(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
		"POST /agents": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		},
	}))
	err := rootFor(cmd.TokenCmd, "issue", "--agent", "alpha").Execute()
	if err == nil || !strings.Contains(err.Error(), "create agent") {
		t.Fatalf("want create-agent error, got %v", err)
	}
}

func TestOAuthRotateSecret_WithClipboard(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only path")
	}
	dir := t.TempDir()
	stub := dir + "/wl-copy"
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /oauth/clients/00000000-0000-0000-0000-000000000001/rotate-secret": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"id": "00000000-0000-0000-0000-000000000001", "client_id": "ci", "name": "x",
				"redirect_uris": []string{}, "allowed_scopes": []string{},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
				"client_secret": "sek",
			})
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "rotate-secret", "00000000-0000-0000-0000-000000000001", "--copy-to-clipboard").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoDelete_DeleteFails(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg/r": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	}))
	if err := rootFor(cmd.RepoCmd, "delete", "myorg/r", "--yes").Execute(); err == nil {
		t.Fatal("expected error when delete returns 500")
	}
}

func TestMcpCall_ToolError(t *testing.T) {
	withMCPServer(t, map[string]func() any{
		"tools/call": func() any {
			return map[string]any{"isError": true, "content": []map[string]any{{"type": "text", "text": "boom"}}}
		},
	})
	err := rootFor(cmd.McpCmd, "call", "x").Execute()
	if err == nil || !errors.Is(err, cmd.ErrToolCallFailed) {
		t.Fatalf("want ErrToolCallFailed, got %v", err)
	}
}

func TestMcpUnauthorizedSurface(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
	err := rootFor(cmd.McpCmd, "tools").Execute()
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("want surfaceErr-mapped unauthorized, got %v", err)
	}
}
