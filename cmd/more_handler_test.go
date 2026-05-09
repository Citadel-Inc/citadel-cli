package cmd_test

// HTTP handler tests for account, namespace, audit-sessions, and token commands.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── account passkey ───────────────────────────────────────────────────────────

func passkeyJSON(rows []map[string]any) map[string]any {
	return map[string]any{"passkeys": rows}
}

func deviceJSON(rows []map[string]any) map[string]any {
	return map[string]any{"devices": rows}
}

func TestAccountPasskeyList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, passkeyJSON([]map[string]any{
				{"id": "pk1", "name": "YubiKey", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.AccountCmd, "passkey", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountPasskeyList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, passkeyJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.AccountCmd, "passkey", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountPasskeyList_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, passkeyJSON([]map[string]any{
				{"id": "pk1", "name": "YubiKey", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.AccountCmd, "passkey", "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountPasskeyList_DirectArray(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
			// server returns bare array (no wrapper key)
			writeJSON(t, w, 200, []map[string]any{
				{"id": "pk1", "name": "Key1", "created_at": "2026-01-01T00:00:00Z"},
			})
		},
	}))
	if err := rootFor(cmd.AccountCmd, "passkey", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountPasskeyList_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, passkeyJSON([]map[string]any{
				{"id": "pk1", "name": "Key1", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	var stdout strings.Builder
	if err := rootForOut(cmd.AccountCmd, &stdout, "passkey", "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "id") {
		t.Fatalf("expected csv header, got %q", stdout.String())
	}
}

func TestAccountPasskeyDelete_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /account/passkeys/missing": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	}))
	err := rootFor(cmd.AccountCmd, "passkey", "delete", "missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestAccountPasskeyRename_MissingName(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	err := rootFor(cmd.AccountCmd, "passkey", "rename", "pk1").Execute()
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("want --name required error, got %v", err)
	}
}

func TestAccountDeviceList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, deviceJSON([]map[string]any{
				{"id": "dev1", "name": "Laptop", "user_agent": "Mozilla/5.0", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.AccountCmd, "device", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeviceList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, deviceJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.AccountCmd, "device", "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeviceList_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, deviceJSON([]map[string]any{
				{"id": "dev1", "name": "Laptop", "user_agent": "Go-http", "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.AccountCmd, "device", "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeviceRevoke_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /auth/devices/dev1": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))
	if err := rootFor(cmd.AccountCmd, "device", "revoke", "dev1").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAccountDeviceRevoke_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /auth/devices/missing": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	}))
	err := rootFor(cmd.AccountCmd, "device", "revoke", "missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

// ── namespace ─────────────────────────────────────────────────────────────────

func TestNsList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
				{"slug": "myorg", "display_name": "My Org", "created_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_OutputCSV(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
				{"slug": "myorg", "display_name": "My Org", "created_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	var stdout strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &stdout, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "slug") {
		t.Fatalf("want csv header, got %q", stdout.String())
	}
}

func TestNsList_OutputNdjson(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
				{"slug": "myorg", "display_name": "My Org", "created_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "list", "--output", "ndjson").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsList_InvalidOutput(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	err := rootFor(cmd.NamespaceCmd, "list", "--output", "binary").Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown format") {
		t.Fatalf("want unknown format error, got %v", err)
	}
}

func TestNsMembers_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsMembers_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
				{"user_id": "u1", "slug": "alice", "is_owner": false, "permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
			}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "members", "myorg", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsTransferListPending_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /transfers/pending": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{}})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "transfer", "list-pending", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── audit sessions ────────────────────────────────────────────────────────────

func TestAuditSessionsList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{
				{
					"session_id": "sess-abc123", "actor_slug": "alice",
					"actor_type": "user", "namespace_slug": "myorg",
					"event_count": 5, "last_event_at": "2026-01-01T01:00:00Z",
				},
			}})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "list", "--ns", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsList_NoNS(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	err := rootFor(cmd.AuditCmd, "sessions", "list").Execute()
	if err == nil || !strings.Contains(err.Error(), "namespace required") {
		t.Fatalf("want namespace required error, got %v", err)
	}
}

func TestAuditSessionsList_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{
				{"session_id": "s1", "actor_slug": "alice", "actor_type": "user", "event_count": 1},
			}})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "list", "--ns", "myorg", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsList_WithNamespaceAlias(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"sessions": []map[string]any{}})
		},
	}))
	// --namespace / -n is an alias for --ns
	if err := rootFor(cmd.AuditCmd, "sessions", "list", "--namespace", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsShow_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions/sess-abc": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"session_id": "sess-abc", "events": []map[string]any{
					{"action": "repo.create", "resource_id": "r1"},
				},
			})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "show", "sess-abc").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsShow_OutputJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions/sess-xyz": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"session_id": "sess-xyz"})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "show", "sess-xyz", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditSessionsShow_OutputYAML(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/sessions/sess-xyz": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"session_id": "sess-xyz"})
		},
	}))
	if err := rootFor(cmd.AuditCmd, "sessions", "show", "sess-xyz", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── token list ────────────────────────────────────────────────────────────────

func TestTokenList_Empty(t *testing.T) {
	const agentID = "00000000-0000-0000-0000-00000000000a"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": agentID, "name": "mybot", "owner_user_id": "u1"},
			}))
		},
		"GET /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, tokensJSON([]map[string]any{}))
		},
	}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_OutputJSON(t *testing.T) {
	const agentID = "00000000-0000-0000-0000-00000000000a"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": agentID, "name": "mybot", "owner_user_id": "u1"},
			}))
		},
		"GET /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, tokensJSON([]map[string]any{
				{"id": "00000000-0000-0000-0000-000000000011", "agent_id": agentID, "created_at": "2026-01-01T00:00:00Z"},
			}))
		},
	}))
	if err := rootFor(cmd.TokenCmd, "list", "--agent", "mybot", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestTokenList_AgentNotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, agentsJSON([]map[string]any{}))
		},
	}))
	err := rootFor(cmd.TokenCmd, "list", "--agent", "missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want agent not-found, got %v", err)
	}
}

// ── agent create ─────────────────────────────────────────────────────────────

func TestAgentCreate_Happy(t *testing.T) {
	agentID := "11111111-1111-1111-1111-111111111111"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": agentID, "name": "mybot", "owner_user_id": "u1",
			})
		},
		"POST /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id":              "33333333-3333-3333-3333-333333333333",
				"agent_id":        agentID,
				"cleartext_token": "secret-tok",
				"created_at":      "2026-01-01T00:00:00Z",
			})
		},
	}))
	if err := rootFor(cmd.AgentCmd, "create", "mybot").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentCreate_JSON(t *testing.T) {
	agentID := "22222222-2222-2222-2222-222222222222"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id": agentID, "name": "ci-runner", "owner_user_id": "u1",
			})
		},
		"POST /agent-tokens": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 201, map[string]any{
				"id":              "44444444-4444-4444-4444-444444444444",
				"agent_id":        agentID,
				"cleartext_token": "ci-secret",
				"created_at":      "2026-01-01T00:00:00Z",
			})
		},
	}))
	if err := rootFor(cmd.AgentCmd, "create", "ci-runner", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAgentCreate_DuplicateName(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 409, map[string]any{"error": "agent with this name already exists"})
		},
	}))
	err := rootFor(cmd.AgentCmd, "create", "existing").Execute()
	if err == nil || !strings.Contains(err.Error(), "already taken") {
		t.Fatalf("want 'already taken' error, got %v", err)
	}
}

func TestAgentCreate_Forbidden(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /agents": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 403, map[string]any{"error": "forbidden"})
		},
	}))
	err := rootFor(cmd.AgentCmd, "create", "blocked").Execute()
	if err == nil || !strings.Contains(err.Error(), "insufficient permission") {
		t.Fatalf("want 'insufficient permission' error, got %v", err)
	}
}

// ── namespace rename ──────────────────────────────────────────────────────────

func TestNsRename_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/myorg/rename": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, map[string]any{"slug": "new-org"})
		},
	}))
	if err := rootFor(cmd.NamespaceCmd, "rename", "myorg", "--new-slug", "new-org", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNsRename_Conflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/myorg/rename": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"conflict"}`, http.StatusConflict)
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "rename", "myorg", "--new-slug", "taken", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "already taken") {
		t.Fatalf("want 'already taken' error, got %v", err)
	}
}

func TestNsRename_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/gone/rename": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		},
	}))
	err := rootFor(cmd.NamespaceCmd, "rename", "gone", "--new-slug", "other", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestNsRename_MissingNewSlug(t *testing.T) {
	err := rootFor(cmd.NamespaceCmd, "rename", "myorg", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "--new-slug") {
		t.Fatalf("want --new-slug error, got %v", err)
	}
}

func TestNsRename_DryRun(t *testing.T) {
	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out, "rename", "myorg", "--new-slug", "new-org", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Would rename") {
		t.Fatalf("expected 'Would rename' in output, got: %s", out.String())
	}
}
