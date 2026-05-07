package cmd_test

// Tests for output format variants (yaml, csv, ndjson) and --all/--output json
// conflict validation across the major list commands.  Complements the basic
// happy-path coverage in handler_test.go and more_handler_test.go.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── agent list ───────────────────────────────────────────────────────────────

func TestAgentList_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": "00000000-0000-0000-0000-00000000000a", "name": "bot", "owner_user_id": "u1"},
			}))
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": "00000000-0000-0000-0000-00000000000a", "name": "bot", "owner_user_id": "u1"},
			}))
		},
	}))
	var sb strings.Builder
	if err := rootForOut(cmd.AgentCmd, &sb, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "name") {
		t.Fatalf("expected csv header, got %q", sb.String())
	}
}

func TestAgentList_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": "00000000-0000-0000-0000-00000000000a", "name": "bot", "owner_user_id": "u1"},
			}))
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_AllOutputJsonConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{}))
		},
	}))
	err := rootFor(cmd.AgentCmd, "list", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict error, got %v", err)
	}
}

func TestAgentList_EmptyYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_EmptyCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_EmptyNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentList_EmptyJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.AgentCmd, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── namespace list ────────────────────────────────────────────────────────────

func TestNsList_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
				{"slug": "myorg", "display_name": "My Org", "created_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_AllOutputJsonConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{}})
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "list", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict, got %v", err)
	}
}

func TestNsList_EmptyYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_EmptyCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_EmptyNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_EmptyJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── namespace members ─────────────────────────────────────────────────────────

func TestNsMembers_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
				{"user_id": "u1", "slug": "alice", "is_owner": true, "permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
				{"user_id": "u1", "slug": "alice", "is_owner": false, "permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
				{"user_id": "u1", "slug": "alice", "is_owner": false, "permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_AllOutputJsonConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{}})
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict, got %v", err)
	}
}

func TestNsMembers_EmptyYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_EmptyCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_EmptyNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_EmptyJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── namespace transfer list-pending ──────────────────────────────────────────

func TestNsTransferListPending_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{
				{
					"id":               "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
					"org_namespace_id": "x", "org_slug": "o",
					"from_user_id": "a", "from_user_slug": "alice",
					"to_user_id": "b", "to_user_slug": "bob",
					"expires_at": "2026-02-01T00:00:00Z",
					"created_at": "2026-01-01T00:00:00Z",
				},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{
				{
					"id":               "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
					"org_namespace_id": "x", "org_slug": "o",
					"from_user_id": "a", "from_user_slug": "alice",
					"to_user_id": "b", "to_user_slug": "bob",
					"expires_at": "2026-02-01T00:00:00Z",
					"created_at": "2026-01-01T00:00:00Z",
				},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{
				{
					"id":               "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
					"org_namespace_id": "x", "org_slug": "o",
					"from_user_id": "a", "from_user_slug": "alice",
					"to_user_id": "b", "to_user_slug": "bob",
					"expires_at": "2026-02-01T00:00:00Z",
					"created_at": "2026-01-01T00:00:00Z",
				},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_AllOutputJsonConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict, got %v", err)
	}
}

func TestNsTransferListPending_EmptyYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_EmptyCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_EmptyNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_EmptyJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── ssh-key list ──────────────────────────────────────────────────────────────

func sshKeysJSON(rows []map[string]any) map[string]any {
	return map[string]any{"keys": rows}
}

func TestSSHKeyList_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{
				{"id": "kk1", "fingerprint": "SHA256:abc", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.SSHKeyCmd, "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyList_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{
				{"id": "kk1", "fingerprint": "SHA256:abc", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	var sb strings.Builder
	if err := rootForOut(cmd.SSHKeyCmd, &sb, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "fingerprint") {
		t.Fatalf("expected csv header, got %q", sb.String())
	}
}

func TestSSHKeyList_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{
				{"id": "kk1", "fingerprint": "SHA256:abc", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.SSHKeyCmd, "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyList_EmptyYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.SSHKeyCmd, "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyList_EmptyCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.SSHKeyCmd, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyList_EmptyNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.SSHKeyCmd, "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyList_EmptyJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/ssh-keys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, sshKeysJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.SSHKeyCmd, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestSSHKeyList_InvalidOutput(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	err := rootFor(cmd.SSHKeyCmd, "list", "--output", "binary").Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown format") {
		t.Fatalf("want unknown format error, got %v", err)
	}
}

// ── token list ────────────────────────────────────────────────────────────────

const testAgentID = "00000000-0000-0000-0000-00000000000a"

func tokenTestServer(t *testing.T, tokenRows []map[string]any) http.HandlerFunc {
	t.Helper()
	return route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": testAgentID, "name": "mybot", "owner_user_id": "u1"},
			}))
		},
		"GET /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, tokensJSON(tokenRows))
		},
	})
}

func TestTokenList_OutputYAML(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{
		{"id": "00000000-0000-0000-0000-000000000011", "agent_id": testAgentID, "created_at": "2026-01-01T00:00:00Z"},
	}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_OutputCSV(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{
		{"id": "00000000-0000-0000-0000-000000000011", "agent_id": testAgentID, "created_at": "2026-01-01T00:00:00Z"},
	}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_OutputNdjson(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{
		{"id": "00000000-0000-0000-0000-000000000011", "agent_id": testAgentID, "created_at": "2026-01-01T00:00:00Z"},
	}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_AllOutputJsonConflict(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{}))
	err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict, got %v", err)
	}
}

func TestTokenList_EmptyYAML(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_EmptyCSV(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_EmptyNdjson(t *testing.T) {
	withServer(t, tokenTestServer(t, []map[string]any{}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── oauth clients list ────────────────────────────────────────────────────────

func oauthClientRow() map[string]any {
	return map[string]any{
		"id":             "11111111-1111-1111-1111-111111111111",
		"client_id":      "cid1",
		"name":           "App",
		"allowed_scopes": []string{"openid"},
		"redirect_uris":  []string{"https://x"},
		"is_public":      false,
		"created_at":     "2026-01-01T00:00:00Z",
		"updated_at":     "2026-01-01T00:00:00Z",
	}
}

func TestOAuthClientsList_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{oauthClientRow()}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthClientsList_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{oauthClientRow()}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthClientsList_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{oauthClientRow()}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthClientsList_AllOutputJsonConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{}))
		},
	}))
	err := rootFor(cmd.OauthCmd, "clients", "list", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict, got %v", err)
	}
}

func TestOAuthClientsList_EmptyYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthClientsList_EmptyCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthClientsList_EmptyNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthClientsList_EmptyJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, clientsJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.OauthCmd, "clients", "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── audit sessions list ───────────────────────────────────────────────────────

func TestAuditSessionsList_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{
				{"session_id": "s1", "actor_slug": "alice", "actor_type": "user", "event_count": 1, "namespace_slug": "myorg"},
			}})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "list", "--ns", "myorg", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsList_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{
				{"session_id": "s1", "actor_slug": "alice", "actor_type": "user", "event_count": 1},
			}})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "list", "--ns", "myorg", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsList_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{
				{"session_id": "s1", "actor_slug": "alice", "actor_type": "user", "event_count": 1},
			}})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "list", "--ns", "myorg", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsList_AllOutputJsonConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{}})
		},
	}))
	err := rootFor(cmd.AuditCmd, "sessions", "list", "--ns", "myorg", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("want --all conflict, got %v", err)
	}
}
