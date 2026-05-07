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

// ── notification read ─────────────────────────────────────────────────────────

func TestNotificationRead_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /api/me/notifications/notif-1/read": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"ok": true})
		},
	}))
	if err := rootFor(cmd.NotificationCmd, "read", "notif-1").Execute(); err != nil {
		t.Fatal(err)
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
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	err := rootFor(cmd.NotificationCmd, "prefs", "set").Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("want at-least-one error, got %v", err)
	}
}

func TestNotificationPrefsSet_InvalidCadence(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{}))
	err := rootFor(cmd.NotificationCmd, "prefs", "set", "--email-digest", "monthly").Execute()
	if err == nil || !strings.Contains(err.Error(), "--email-digest") {
		t.Fatalf("want cadence error, got %v", err)
	}
}
