package cmd_test

// Tests targeting remaining coverage gaps: multi-page --all mode, auth status
// agent-token branches, account passkey/device format variants, and audit-session
// branches not hit by the baseline handler tests.

import (
"net/http"
"os"
"path/filepath"
"strings"
"testing"

"github.com/Rethunk-Tech/citadel-cli/cmd"
"github.com/google/uuid"
)

// ── agent list: multi-page --all + misc branches ──────────────────────────────

func TestAgentList_AllYAMLTwoPages(t *testing.T) {
id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
cur := "Y3Vyc29yMQ=="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id1.String(), "name": "botA", "owner_user_id": "u1"},
}, "next_cursor": cur})
case 2:
if r.URL.Query().Get("cursor") != cur {
t.Errorf("page 2 expected cursor=%q got %q", cur, r.URL.Query().Get("cursor"))
}
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id2.String(), "name": "botB", "owner_user_id": "u1"},
}})
default:
t.Fatalf("unexpected page %d", pages)
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AgentCmd, &sb, "list", "--all", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
if pages != 2 {
t.Fatalf("want 2 pages, got %d", pages)
}
if !strings.Contains(sb.String(), "botA") || !strings.Contains(sb.String(), "botB") {
t.Fatalf("expected both agents in yaml, got %q", sb.String())
}
}

func TestAgentList_AllNdjsonTwoPages(t *testing.T) {
id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
cur := "Y3Vyc29yMg=="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id1.String(), "name": "botA", "owner_user_id": "u1"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id2.String(), "name": "botB", "owner_user_id": "u1"},
}})
default:
t.Fatalf("unexpected page %d", pages)
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AgentCmd, &sb, "list", "--all", "--output", "ndjson").Execute(); err != nil {
t.Fatal(err)
}
lines := strings.Split(strings.TrimSpace(sb.String()), "\n")
if len(lines) != 2 {
t.Fatalf("want 2 ndjson lines, got %d: %q", len(lines), sb.String())
}
}

func TestAgentList_AllCSVTwoPages(t *testing.T) {
id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
cur := "Y3Vyc29yMw=="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id1.String(), "name": "botA", "owner_user_id": "u1"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id2.String(), "name": "botB", "owner_user_id": "u1"},
}})
default:
t.Fatalf("unexpected page %d", pages)
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AgentCmd, &sb, "list", "--all", "--output", "csv").Execute(); err != nil {
t.Fatal(err)
}
csvLines := strings.Split(strings.TrimSpace(sb.String()), "\n")
// header + 2 data rows = 3
if len(csvLines) < 3 {
t.Fatalf("want ≥3 csv lines, got %d: %q", len(csvLines), sb.String())
}
}

func TestAgentList_PaginationHint(t *testing.T) {
id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, r *http.Request) {
writeJSON(t, w, 200, map[string]any{"agents": []map[string]any{
{"id": id.String(), "name": "bot", "owner_user_id": "u1"},
}, "next_cursor": "Y3Vyc29yNA=="})
},
}))
// table output with next_cursor → should print hint
var sb strings.Builder
if err := rootForOut(cmd.AgentCmd, &sb, "list").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAgentList_TableWithModelHint(t *testing.T) {
id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, r *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": id.String(), "name": "bot", "owner_user_id": "u1", "model_hint": "gpt-4o"},
}))
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AgentCmd, &sb, "list").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "gpt-4o") {
t.Fatalf("expected model hint in table, got %q", sb.String())
}
}

// ── namespace list: multi-page --all + table row + pagination hint ────────────

func TestNsList_AllYAMLTwoPages(t *testing.T) {
cur := "bmFtZXNwYWNlY3Vy"
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "orgA", "display_name": "Org A", "created_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "orgB", "display_name": "Org B", "created_at": "2026-02-01T00:00:00Z"},
}})
default:
t.Fatalf("unexpected page %d", pages)
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "list", "--all", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "orgA") || !strings.Contains(sb.String(), "orgB") {
t.Fatalf("expected both orgs in yaml, got %q", sb.String())
}
}

func TestNsList_AllNdjsonTwoPages(t *testing.T) {
cur := "bmFtZXNwYWNlY3VyMg=="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "orgA", "display_name": "Org A", "created_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "orgB", "display_name": "Org B", "created_at": "2026-02-01T00:00:00Z"},
}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "list", "--all", "--output", "ndjson").Execute(); err != nil {
t.Fatal(err)
}
lines := strings.Split(strings.TrimSpace(sb.String()), "\n")
if len(lines) != 2 {
t.Fatalf("want 2 ndjson lines, got %d", len(lines))
}
}

func TestNsList_AllCSVTwoPages(t *testing.T) {
cur := "bmFtZXNwYWNlY3VyMw=="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "orgA", "display_name": "Org A", "created_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "orgB", "display_name": "Org B", "created_at": "2026-02-01T00:00:00Z"},
}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "list", "--all", "--output", "csv").Execute(); err != nil {
t.Fatal(err)
}
if csvLines := strings.Split(strings.TrimSpace(sb.String()), "\n"); len(csvLines) < 3 {
t.Fatalf("want header + 2 rows, got %d: %q", len(csvLines), sb.String())
}
}

func TestNsList_TableWithNextCursor(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs": func(w http.ResponseWriter, r *http.Request) {
writeJSON(t, w, 200, map[string]any{"orgs": []map[string]any{
{"slug": "myorg", "display_name": "My Org", "created_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": "aGludA=="})
},
}))
// table output with next_cursor → prints hint line
if err := rootFor(cmd.NamespaceCmd, "list").Execute(); err != nil {
t.Fatal(err)
}
}

// ── namespace members: multi-page + table branches ────────────────────────────

func TestNsMembers_AllYAMLTwoPages(t *testing.T) {
cur := "bWVtYmVyY3Vy"
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs/myorg/members": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
{"user_id": "u1", "slug": "alice", "is_owner": true, "permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
{"user_id": "u2", "slug": "bob", "is_owner": false, "permissions": []string{}, "joined_at": "2026-02-01T00:00:00Z"},
}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "members", "myorg", "--all", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "alice") || !strings.Contains(sb.String(), "bob") {
t.Fatalf("expected both members, got %q", sb.String())
}
}

func TestNsMembers_TableWithDisplayName(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs/myorg/members": func(w http.ResponseWriter, r *http.Request) {
writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
{"user_id": "u1", "slug": "alice", "display_name": "Alice Smith", "is_owner": false,
"permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
}})
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "members", "myorg").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "Alice Smith") {
t.Fatalf("expected display name in table, got %q", sb.String())
}
}

func TestNsMembers_TableNoDisplayName(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /orgs/myorg/members": func(w http.ResponseWriter, r *http.Request) {
writeJSON(t, w, 200, map[string]any{"members": []map[string]any{
{"user_id": "u1", "slug": "alice", "is_owner": true,
"permissions": []string{}, "joined_at": "2026-01-01T00:00:00Z"},
}})
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "members", "myorg").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "alice") {
t.Fatalf("expected slug fallback in table, got %q", sb.String())
}
}

// ── token list: multi-page --all ──────────────────────────────────────────────

func TestTokenList_AllYAMLTwoPages(t *testing.T) {
cur := "dG9rZW5jdXI="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": testAgentID, "name": "mybot", "owner_user_id": "u1"},
}))
},
"GET /agent-tokens": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"tokens": []map[string]any{
{"id": "00000000-0000-0000-0000-000000000011", "agent_id": testAgentID, "created_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"tokens": []map[string]any{
{"id": "00000000-0000-0000-0000-000000000022", "agent_id": testAgentID, "created_at": "2026-02-01T00:00:00Z"},
}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.TokenCmd, &sb, "list", "--agent", "mybot", "--all", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
}

func TestTokenList_AllNdjsonTwoPages(t *testing.T) {
cur := "dG9rZW5jdXIy"
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": testAgentID, "name": "mybot", "owner_user_id": "u1"},
}))
},
"GET /agent-tokens": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"tokens": []map[string]any{
{"id": "00000000-0000-0000-0000-000000000011", "agent_id": testAgentID, "created_at": "2026-01-01T00:00:00Z"},
}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"tokens": []map[string]any{
{"id": "00000000-0000-0000-0000-000000000022", "agent_id": testAgentID, "created_at": "2026-02-01T00:00:00Z"},
}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.TokenCmd, &sb, "list", "--agent", "mybot", "--all", "--output", "ndjson").Execute(); err != nil {
t.Fatal(err)
}
lines := strings.Split(strings.TrimSpace(sb.String()), "\n")
if len(lines) != 2 {
t.Fatalf("want 2 ndjson lines, got %d", len(lines))
}
}

// ── auth status: agent-token branches ────────────────────────────────────────

// writeAgentConfig writes a config.toml with agent_id to the temp XDG dir.
func writeAgentConfig(t *testing.T, dir string, agentID, agentName string, expiresAt string) {
t.Helper()
cfgDir := filepath.Join(dir, "citadel")
if err := os.MkdirAll(cfgDir, 0700); err != nil {
t.Fatal(err)
}
content := "access_token = \"tok\"\nagent_id = \"" + agentID + "\"\nagent_name = \"" + agentName + "\"\n"
if expiresAt != "" {
content += "expires_at = " + expiresAt + "\n"
}
if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0600); err != nil {
t.Fatal(err)
}
}

func TestAuthStatus_AgentToken_Active(t *testing.T) {
dir := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", dir)
t.Setenv("CITADEL_ACCESS_TOKEN", "")
// expires_at in the future
writeAgentConfig(t, dir, "agent-uuid-1", "mybot", "2099-01-01T00:00:00Z")
if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAuthStatus_AgentToken_Expired(t *testing.T) {
dir := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", dir)
t.Setenv("CITADEL_ACCESS_TOKEN", "")
// expires_at in the past
writeAgentConfig(t, dir, "agent-uuid-2", "oldbot", "2000-01-01T00:00:00Z")
if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAuthStatus_AgentToken_NoExpiry(t *testing.T) {
dir := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", dir)
t.Setenv("CITADEL_ACCESS_TOKEN", "")
// No expires_at line — IsZero() is true, skips expiry display
writeAgentConfig(t, dir, "agent-uuid-3", "noexpbot", "")
if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAuthStatus_AgentToken_WithUserUUID(t *testing.T) {
dir := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", dir)
t.Setenv("CITADEL_ACCESS_TOKEN", "")
cfgDir := filepath.Join(dir, "citadel")
if err := os.MkdirAll(cfgDir, 0700); err != nil {
t.Fatal(err)
}
content := "access_token = \"tok\"\nagent_id = \"agent-uuid-4\"\nagent_name = \"mybot\"\nuser_uuid = \"user-uuid-1\"\nexpires_at = 2099-01-01T00:00:00Z\n"
if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0600); err != nil {
t.Fatal(err)
}
// runStatus uses fmt.Printf directly (not cmd.OutOrStdout), so we only verify no error.
if err := rootFor(cmd.AuthCmd, "status").Execute(); err != nil {
t.Fatal(err)
}
}

// ── account passkey list: yaml + ndjson + direct-array response ───────────────

func TestAccountPasskeyList_YAML(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{"passkeys": []map[string]any{
{"id": "pk1", "name": "MyKey", "created_at": "2026-01-01T00:00:00Z"},
}})
},
}))
if err := rootFor(cmd.AccountCmd, "passkey", "list", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAccountPasskeyList_Ndjson(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /account/passkeys": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{"passkeys": []map[string]any{
{"id": "pk1", "name": "MyKey", "created_at": "2026-01-01T00:00:00Z"},
}})
},
}))
if err := rootFor(cmd.AccountCmd, "passkey", "list", "--output", "ndjson").Execute(); err != nil {
t.Fatal(err)
}
}

// ── account passkey rename: json + yaml + invalid output ─────────────────────

func TestAccountPasskeyRename_JSON(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"PATCH /account/passkeys/pk1": func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusOK)
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AccountCmd, &sb, "passkey", "rename", "pk1", "--name", "newname", "--output", "json").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "newname") {
t.Fatalf("expected name in JSON output, got %q", sb.String())
}
}

func TestAccountPasskeyRename_YAML(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"PATCH /account/passkeys/pk1": func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusOK)
},
}))
if err := rootFor(cmd.AccountCmd, "passkey", "rename", "pk1", "--name", "newname", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAccountPasskeyRename_InvalidOutput(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{}))
err := rootFor(cmd.AccountCmd, "passkey", "rename", "pk1", "--name", "n", "--output", "ndjson").Execute()
if err == nil || !strings.Contains(err.Error(), "ndjson") {
t.Fatalf("want invalid output error, got %v", err)
}
}

func TestAccountPasskeyRename_NotFound(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"PATCH /account/passkeys/pk1": func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusNotFound)
},
}))
err := rootFor(cmd.AccountCmd, "passkey", "rename", "pk1", "--name", "n").Execute()
if err == nil || !strings.Contains(err.Error(), "not found") {
t.Fatalf("want not found error, got %v", err)
}
}

// ── account device list: yaml + ndjson + direct-array + zero last-seen ────────

func TestAccountDeviceList_YAML(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{"devices": []map[string]any{
{"id": "dev1", "name": "laptop", "user_agent": "Mozilla/5.0", "created_at": "2026-01-01T00:00:00Z"},
}})
},
}))
if err := rootFor(cmd.AccountCmd, "device", "list", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAccountDeviceList_Ndjson(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{"devices": []map[string]any{
{"id": "dev1", "name": "laptop", "user_agent": "Mozilla/5.0", "created_at": "2026-01-01T00:00:00Z"},
}})
},
}))
if err := rootFor(cmd.AccountCmd, "device", "list", "--output", "ndjson").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAccountDeviceList_DirectArray(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, []map[string]any{
{"id": "dev1", "name": "laptop", "user_agent": "Mozilla/5.0", "created_at": "2026-01-01T00:00:00Z"},
})
},
}))
if err := rootFor(cmd.AccountCmd, "device", "list").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAccountDeviceList_TableWithLastSeen(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{"devices": []map[string]any{
{"id": "dev1", "name": "laptop", "user_agent": "Mozilla/5.0",
"last_seen_at": "2026-06-01T12:00:00Z",
"created_at":   "2026-01-01T00:00:00Z"},
}})
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AccountCmd, &sb, "device", "list").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "2026-06-01") {
t.Fatalf("expected last_seen_at in table, got %q", sb.String())
}
}

func TestAccountDeviceList_TableLongUserAgent(t *testing.T) {
longUA := strings.Repeat("X", 60)
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /auth/devices": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{"devices": []map[string]any{
{"id": "dev1", "name": "laptop", "user_agent": longUA, "created_at": "2026-01-01T00:00:00Z"},
}})
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AccountCmd, &sb, "device", "list").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "...") {
t.Fatalf("expected truncated user agent with '...' in table, got %q", sb.String())
}
}

// ── agent get: yaml + json ────────────────────────────────────────────────────

func TestAgentGet_YAML(t *testing.T) {
id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": id.String(), "name": "bot", "owner_user_id": "u1"},
}))
},
}))
if err := rootFor(cmd.AgentCmd, "get", "bot", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAgentGet_JSON(t *testing.T) {
id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": id.String(), "name": "bot", "owner_user_id": "u1"},
}))
},
}))
if err := rootFor(cmd.AgentCmd, "get", "bot", "--output", "json").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAgentGet_WithModelHint(t *testing.T) {
id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": id.String(), "name": "bot", "owner_user_id": "u1", "model_hint": "claude-3"},
}))
},
}))
var sb strings.Builder
if err := rootForOut(cmd.AgentCmd, &sb, "get", "bot").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "claude-3") {
t.Fatalf("expected model hint in output, got %q", sb.String())
}
}

// ── agent delete: --dry-run branch ───────────────────────────────────────────

// ── agent rotate-token ────────────────────────────────────────────────────────

func TestAgentRotateToken_Success(t *testing.T) {
id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /agents": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, agentsJSON([]map[string]any{
{"id": id.String(), "name": "bot", "owner_user_id": "u1"},
}))
},
"POST /agents/" + id.String() + "/rotate-token": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{
"id":              "00000000-0000-0000-0000-000000000001",
"agent_id":        id.String(),
"created_at":      "2026-01-01T00:00:00Z",
"cleartext_token": "cit_newtoken123",
})
},
}))
// runAgentRotateToken uses fmt.Println for token (not cmd.OutOrStdout); verify no error.
if err := rootFor(cmd.AgentCmd, "rotate-token", "bot", "--yes").Execute(); err != nil {
t.Fatal(err)
}
}

// ── namespace transfer list-pending: --all mode ───────────────────────────────

func TestNsTransferListPending_AllYAMLTwoPages(t *testing.T) {
cur := "dHJhbnNmZXJjdXI="
pages := 0
tr1 := map[string]any{
"id": "aaaaaaaa-bbbb-cccc-dddd-000000000001",
"org_namespace_id": "x", "org_slug": "o",
"from_user_id": "a", "from_user_slug": "alice",
"to_user_id": "b", "to_user_slug": "bob",
"expires_at": "2026-02-01T00:00:00Z",
"created_at": "2026-01-01T00:00:00Z",
}
tr2 := map[string]any{
"id": "aaaaaaaa-bbbb-cccc-dddd-000000000002",
"org_namespace_id": "y", "org_slug": "p",
"from_user_id": "c", "from_user_slug": "carol",
"to_user_id": "d", "to_user_slug": "dave",
"expires_at": "2026-03-01T00:00:00Z",
"created_at": "2026-01-15T00:00:00Z",
}
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /transfers/pending": func(w http.ResponseWriter, r *http.Request) {
pages++
switch pages {
case 1:
writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{tr1}, "next_cursor": cur})
case 2:
writeJSON(t, w, 200, map[string]any{"transfers": []map[string]any{tr2}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.NamespaceCmd, &sb, "transfer", "list-pending", "--all", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "alice") || !strings.Contains(sb.String(), "carol") {
t.Fatalf("expected both transfers in yaml, got %q", sb.String())
}
}

// ── audit sessions: show command ──────────────────────────────────────────────

func TestAuditSessionsShow_Table(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /audit/sessions/sess1": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{
"session_id":    "sess1",
"actor_slug":    "alice",
"actor_type":    "user",
"event_count":   3,
"namespace_slug": "myorg",
"events": []map[string]any{
{"event_type": "login", "created_at": "2026-01-01T00:00:00Z"},
},
})
},
}))
if err := rootFor(cmd.AuditCmd, "sessions", "show", "sess1").Execute(); err != nil {
t.Fatal(err)
}
}

func TestAuditSessionsShow_JSON(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /audit/sessions/sess1": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{
"session_id":  "sess1",
"actor_slug":  "alice",
"actor_type":  "user",
"event_count": 1,
"events":      []map[string]any{},
})
},
}))
if err := rootFor(cmd.AuditCmd, "sessions", "show", "sess1", "--output", "json").Execute(); err != nil {
t.Fatal(err)
}
}

// ── oauth clients list: table output ─────────────────────────────────────────

func TestOAuthClientsList_Table(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /oauth/clients": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, clientsJSON([]map[string]any{oauthClientRow()}))
},
}))
var sb strings.Builder
if err := rootForOut(cmd.OauthCmd, &sb, "clients", "list").Execute(); err != nil {
t.Fatal(err)
}
if !strings.Contains(sb.String(), "App") {
t.Fatalf("expected client name in table, got %q", sb.String())
}
}

func TestOAuthClientsList_AllYAMLTwoPages(t *testing.T) {
cur := "b2F1dGhjdXI="
pages := 0
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /oauth/clients": func(w http.ResponseWriter, r *http.Request) {
pages++
row := oauthClientRow()
switch pages {
case 1:
row["client_id"] = "cid1"
writeJSON(t, w, 200, map[string]any{"clients": []map[string]any{row}, "next_cursor": cur})
case 2:
row["client_id"] = "cid2"
writeJSON(t, w, 200, map[string]any{"clients": []map[string]any{row}})
}
},
}))
var sb strings.Builder
if err := rootForOut(cmd.OauthCmd, &sb, "clients", "list", "--all", "--output", "yaml").Execute(); err != nil {
t.Fatal(err)
}
}

// ── namespace delete ──────────────────────────────────────────────────────────

func TestNsDelete_Success(t *testing.T) {
withServer(t, route(t, map[string]http.HandlerFunc{
"GET /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
writeJSON(t, w, 200, map[string]any{
"slug": "myorg", "kind": "org", "visibility": "private",
"namespace_id": "ns1", "created_at": "2026-01-01T00:00:00Z",
})
},
"DELETE /namespaces/myorg": func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusOK)
},
}))
if err := rootFor(cmd.NamespaceCmd, "delete", "myorg", "--yes").Execute(); err != nil {
t.Fatal(err)
}
}
