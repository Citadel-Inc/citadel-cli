package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func setAuditHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
}

func TestAuditRBAC_ListHappy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"events": []map[string]any{{
					"id": "event-1", "ts": "2026-08-09T12:00:00Z", "kind": "repo.created",
					"actor_type": "user", "namespace_slug": "myorg", "payload": map[string]any{},
				}},
				"next_cursor": "",
			})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.AuditCmd, &stdout, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "event-1") {
		t.Fatalf("want event in output, got %q", stdout.String())
	}
}

func TestAuditList_EmptyHuman(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"events":      []any{},
				"next_cursor": "",
			})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.AuditCmd, &stdout, "list").Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "No audit events found.\n" {
		t.Fatalf("stdout = %q, want %q", got, "No audit events found.\n")
	}
}

func TestAuditRBAC_ListNamespaceFilter(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("namespace"); got != "myorg" {
				t.Fatalf("namespace query = %q, want myorg", got)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"events":      []any{},
				"next_cursor": "",
			})
		},
	}))

	if err := rootFor(cmd.AuditCmd, "list", "--namespace", "myorg", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditRBAC_ListBadOutput_Hermetic(t *testing.T) {
	setAuditHermeticEnv(t)

	err := rootFor(cmd.AuditCmd,
		"list",
		"--output", "toml",
	).Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestAuditRBAC_ListNegativeLimit_Hermetic(t *testing.T) {
	setAuditHermeticEnv(t)

	err := rootFor(cmd.AuditCmd,
		"list",
		"--limit", "-1",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--limit must be between") {
		t.Fatalf("want limit validation error, got %v", err)
	}
}

func TestAuditRBAC_ShowHappy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events/event-1": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"id": "event-1", "ts": "2026-08-09T12:00:00Z", "kind": "repo.created",
				"actor_type": "user", "payload": map[string]any{"repo": "myorg/repo"},
				"cascade_children": []any{},
			})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.AuditCmd, &stdout, "show", "event-1", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "event-1") {
		t.Fatalf("want event in output, got %q", stdout.String())
	}
}

func TestAuditRBAC_ShowBadOutput_Hermetic(t *testing.T) {
	setAuditHermeticEnv(t)

	err := rootFor(cmd.AuditCmd,
		"show", "event-1",
		"--output", "toml",
	).Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestAuditRBAC_ShowNotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events/missing": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "event not found", http.StatusNotFound)
		},
	}))

	err := rootFor(cmd.AuditCmd, "show", "missing").Execute()
	if err == nil {
		t.Fatal("expected not-found error")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "404") && !strings.Contains(msg, "not found") {
		t.Fatalf("want 404/not-found error, got %v", err)
	}
}

func TestAuditRBAC_ListForbidden(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "audit read forbidden", http.StatusForbidden)
		},
	}))

	err := rootFor(cmd.AuditCmd, "list").Execute()
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "403") && !strings.Contains(msg, "forbidden") && !strings.Contains(msg, "permission") {
		t.Fatalf("want 403/forbidden/permission error, got %v", err)
	}
}

func TestAuditRBAC_ShowForbidden(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /audit/events/event-1": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "audit read forbidden", http.StatusForbidden)
		},
	}))

	err := rootFor(cmd.AuditCmd, "show", "event-1").Execute()
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "403") && !strings.Contains(msg, "forbidden") && !strings.Contains(msg, "permission") {
		t.Fatalf("want 403/forbidden/permission error, got %v", err)
	}
}
