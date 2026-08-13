package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func notifJSON(items []map[string]any, nextCursor string) map[string]any {
	return map[string]any{"items": items, "next_cursor": nextCursor}
}

func makeNotif(id, kind, summary string) map[string]any {
	return map[string]any{
		"id":         id,
		"kind":       kind,
		"summary":    summary,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	}
}

// ── notification list ─────────────────────────────────────────────────────────

func TestNotificationList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, notifJSON([]map[string]any{
				makeNotif("notif-1", "issue.comment", "Someone commented on your issue"),
				makeNotif("notif-2", "repo.push", "A push was made to main"),
			}, ""))
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, notifJSON([]map[string]any{}, ""))
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "list").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationList_EmptyHuman(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, notifJSON([]map[string]any{}, ""))
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.NotificationCmd, &stdout, "list").Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "No notifications.\n" {
		t.Fatalf("empty notification list output = %q", stdout.String())
	}
}

func TestNotificationList_PaginationHint(t *testing.T) {
	const nextCursor = "dG9tZWN1cnNvcg"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, notifJSON([]map[string]any{
				makeNotif("notif-1", "issue.comment", "Hello"),
			}, nextCursor))
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.NotificationCmd, &stdout, "list").Execute(); err != nil {
		t.Fatal(err)
	}
	want := "ID       KIND           STATUS  NAMESPACE  SUMMARY\n" +
		"notif-1  issue.comment  unread  -          Hello\n" +
		"(use --cursor " + nextCursor + " for more, or --all to fetch everything)\n"
	if stdout.String() != want {
		t.Fatalf("notification pagination output = %q, want %q", stdout.String(), want)
	}
}

func TestNotificationList_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, notifJSON([]map[string]any{
				makeNotif("notif-1", "issue.comment", "Hello"),
			}, ""))
		},
	}))
	if err := rootForOut(cmd.NotificationCmd, &buf, "list", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out))
	}
}

func TestNotificationList_UnreadFilter(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("unread") != "1" {
				t.Errorf("expected unread=1 query param; got %q", r.URL.RawQuery)
			}
			writeJSON(t, w, 200, notifJSON([]map[string]any{}, ""))
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "list", "--unread").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationList_CSV(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, notifJSON([]map[string]any{
				makeNotif("notif-1", "issue.comment", "Hello"),
			}, ""))
		},
	}))
	if err := rootForOut(cmd.NotificationCmd, &buf, "list", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "id,kind,summary") {
		t.Fatalf("CSV output missing header, got: %s", buf.String())
	}
}

func TestNotificationList_NoAuth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "http://nope")
	err := rootFor(cmd.NotificationCmd, "list").Execute()
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("want not-authenticated, got %v", err)
	}
}

func setNotificationHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func TestNotificationList_BadOutput_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "list", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)` {
		t.Fatalf("want exact output validation error, got %v", err)
	}
}

func TestNotificationList_BadCursor_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "list", "--cursor", "not-base64!!!").Execute()
	if err == nil || err.Error() != `invalid --cursor: invalid_cursor: desc` {
		t.Fatalf("want invalid cursor error, got %v", err)
	}
}

func TestNotificationList_AllJSON_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "list", "--all", "--output", "json").Execute()
	if err == nil || err.Error() != "--all with --output json is not supported; use --output ndjson for streaming JSON" {
		t.Fatalf("want exact all/json validation error, got %v", err)
	}
}

// ── notification read ─────────────────────────────────────────────────────────

func TestNotificationRead_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /api/me/notifications/notif-1/read": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"ok": true})
		},
	}))
	var stdout strings.Builder
	if err := rootForOut(cmd.NotificationCmd, &stdout, "read", "notif-1").Execute(); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Notification notif-1 marked as read.\n" {
		t.Fatalf("notification read output = %q", stdout.String())
	}
}

func TestNotificationRead_EmptyID_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "read", " \t").Execute()
	if err == nil || err.Error() != "notification id required" {
		t.Fatalf("want exact empty-id validation error, got %v", err)
	}
}

func TestNotificationRead_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /api/me/notifications/missing-id/read": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"code":"not_found"}`, 404)
		},
	}))
	err := rootFor(cmd.NotificationCmd, "read", "missing-id").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

// ── notification read-all ─────────────────────────────────────────────────────

func TestNotificationReadAll_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /api/me/notifications/read-all": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"ok": true})
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "read-all").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── notification unread-count ─────────────────────────────────────────────────

func TestNotificationUnreadCount_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications/unread-count": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"count": 7})
		},
	}))
	if err := rootForOut(cmd.NotificationCmd, &buf, "unread-count").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "7") {
		t.Fatalf("expected count 7 in output, got: %s", buf.String())
	}
}

func TestNotificationUnreadCount_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notifications/unread-count": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"count": 3})
		},
	}))
	if err := rootForOut(cmd.NotificationCmd, &buf, "unread-count", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if out["count"] != float64(3) {
		t.Fatalf("expected count=3, got %v", out["count"])
	}
}

func TestNotificationUnreadCount_BadOutput_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "unread-count", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want exact output validation error, got %v", err)
	}
}

// ── notification prefs get ────────────────────────────────────────────────────

func prefsBody() map[string]any {
	return map[string]any{
		"email_digest_cadence": "daily",
		"kinds": []map[string]any{
			{"kind": "issue.comment", "label": "Issue comments", "enabled": true},
			{"kind": "repo.push", "label": "Repository pushes", "enabled": false},
		},
	}
}

func TestNotificationPrefsGet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notification-prefs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, prefsBody())
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "prefs", "get").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationPrefsGet_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/me/notification-prefs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, prefsBody())
		},
	}))
	if err := rootForOut(cmd.NotificationCmd, &buf, "prefs", "get", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if out["email_digest_cadence"] != "daily" {
		t.Fatalf("expected daily cadence, got %v", out["email_digest_cadence"])
	}
}

func TestNotificationPrefsGet_BadOutput_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "prefs", "get", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want exact output validation error, got %v", err)
	}
}

// ── notification prefs set ────────────────────────────────────────────────────

func TestNotificationPrefsSet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /api/me/notification-prefs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, prefsBody())
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "prefs", "set", "--email-digest", "weekly").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationPrefsSet_YAML(t *testing.T) {
	var stdout strings.Builder
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /api/me/notification-prefs": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, prefsBody())
		},
	}))
	if err := rootForOut(cmd.NotificationCmd, &stdout,
		"prefs", "set", "--email-digest", "weekly", "--output", "yaml",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "email_digest_cadence: daily") {
		t.Fatalf("YAML output missing cadence key, got %q", stdout.String())
	}
}

func TestNotificationPrefsSet_KindOverrides(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /api/me/notification-prefs": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("bad request body: %v", err)
			}
			overrides, ok := body["kind_overrides"].(map[string]any)
			if !ok {
				t.Fatal("missing kind_overrides in request")
			}
			if overrides["issue.comment"] != true {
				t.Errorf("expected issue.comment=true, got %v", overrides["issue.comment"])
			}
			if overrides["repo.push"] != false {
				t.Errorf("expected repo.push=false, got %v", overrides["repo.push"])
			}
			writeJSON(t, w, 200, prefsBody())
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "prefs", "set",
		"--enable", "issue.comment",
		"--disable", "repo.push",
	).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationPrefsSet_NoFlags(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "prefs", "set").Execute()
	if err == nil || err.Error() != "at least one of --email-digest, --enable, or --disable is required" {
		t.Fatalf("want exact at-least-one error, got %v", err)
	}
}

func TestNotificationPrefsSet_InvalidCadence(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "prefs", "set", "--email-digest", "monthly").Execute()
	if err == nil || err.Error() != `--email-digest must be never, daily, or weekly; got "monthly"` {
		t.Fatalf("want exact cadence error, got %v", err)
	}
}

func TestNotificationPrefsSet_BadOutput_Hermetic(t *testing.T) {
	setNotificationHermeticEnv(t)

	err := rootFor(cmd.NotificationCmd, "prefs", "set", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want exact output validation error, got %v", err)
	}
}
