package cmd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Citadel-Inc/citadel-cli/cmd"
)

// sseServe emits one init event then cancels after the client has had time to read it.
func sseServe(t *testing.T, path, data string, cancel context.CancelFunc, gotQuery *string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		*gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "event: init\nid: 1\ndata: %s\n\n", data)
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Error("responseWriter does not support Flush")
			return
		}
		fl.Flush()
		time.AfterFunc(300*time.Millisecond, cancel)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAuditListWatch_ndjsonCarriesEventAndFilters(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/audit/events", `{"id":"42","ts":"2026-01-01T00:00:00Z","kind":"repo.create","actor_type":"user","payload":{}}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.AuditCmd, "list", "--watch", "--output", "ndjson", "--kind", "repo.*", "--namespace", "acme", "--limit", "5")

	var line struct {
		Type    string `json:"type"`
		Payload struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &line); err != nil {
		t.Fatalf("stdout %q: %v", out, err)
	}
	if line.Type != "init" || line.Payload.ID != "42" || line.Payload.Kind != "repo.create" {
		t.Fatalf("unexpected line %+v", line)
	}
	for _, want := range []string{"kind=repo.%2A", "namespace=acme", "limit=5"} {
		if !strings.Contains(q, want) {
			t.Errorf("query %q missing %q", q, want)
		}
	}
}

func TestAuditListWatch_tableRow(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/audit/events", `{"id":"42","ts":"2026-01-01T00:00:00Z","kind":"repo.create","actor_slug":"alice","actor_type":"user","payload":{}}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.AuditCmd, "list", "--watch")
	if !strings.Contains(out, "repo.create") || !strings.Contains(out, "alice") {
		t.Fatalf("table output %q", out)
	}
}

func TestNotificationListWatch_unreadTable(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/api/me/notifications", `{"id":"n1","kind":"issue.comment","summary":"New comment","created_at":"2026-01-01T00:00:00Z"}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.NotificationCmd, "list", "--watch", "--unread")
	if !strings.Contains(out, "New comment") {
		t.Fatalf("table output %q", out)
	}
	if !strings.Contains(q, "unread=1") {
		t.Errorf("query %q missing unread=1", q)
	}
}

func TestNotificationListWatch_rejectsJSONOutput(t *testing.T) {
	withServer(t, route(t, nil))
	err := rootFor(cmd.NotificationCmd, "list", "--watch", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used with --watch") {
		t.Fatalf("want watch/output error, got %v", err)
	}
}

func TestAuditListWatch_serverWithoutSSEFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"events":[]}`)
	}))
	defer srv.Close()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "tok")

	resetFlagsRecursive(cmd.AuditCmd)
	resetCtxRecursive(cmd.AuditCmd, t.Context())
	root := &cobra.Command{Use: "test"}
	addTestRootGroups(root)
	root.AddCommand(cmd.AuditCmd)
	root.SetArgs([]string{"audit", "list", "--watch"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true

	errCh := make(chan error, 1)
	go func() { errCh <- root.Execute() }()
	select {
	case err := <-errCh:
		if err == nil || !strings.Contains(err.Error(), "text/event-stream") {
			t.Fatalf("want not-event-stream error, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("watch hung against a server without SSE")
	}
}

// Row shapes below are what citadel's issue and PR list SSE handlers emit:
// the same element the JSON list returns, keyed by entity id.
func TestIssueListWatch_ndjsonCarriesRowAndFilters(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/namespaces/acme/issues", `{"id":"11111111-1111-1111-1111-111111111111","number":7,"title":"Broken build","state":"open","updated_at":"2026-01-01T00:00:00Z"}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.IssueCmd, "list", "-R", "acme", "--watch", "--output", "ndjson", "--state", "closed", "--label", "bug", "--assignee", "alice", "--limit", "5")

	var line struct {
		Type    string `json:"type"`
		Payload struct {
			Number int64  `json:"number"`
			Title  string `json:"title"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &line); err != nil {
		t.Fatalf("stdout %q: %v", out, err)
	}
	if line.Type != "init" || line.Payload.Number != 7 || line.Payload.Title != "Broken build" {
		t.Fatalf("unexpected line %+v", line)
	}
	for _, want := range []string{"state=closed", "label=bug", "assignee=alice", "limit=5"} {
		if !strings.Contains(q, want) {
			t.Errorf("query %q missing %q", q, want)
		}
	}
}

func TestIssueListWatch_tableRow(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/namespaces/acme/issues", `{"id":"11111111-1111-1111-1111-111111111111","number":7,"title":"Broken build","state":"open","updated_at":"2026-01-01T00:00:00Z"}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.IssueCmd, "list", "-R", "acme", "--watch")
	if !strings.Contains(out, "Broken build") || !strings.Contains(out, "STATE") {
		t.Fatalf("table output %q", out)
	}
}

func TestPRListWatch_tableRowAndState(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/namespaces/acme/pulls", `{"id":"22222222-2222-2222-2222-222222222222","number":3,"title":"Add feature","state":"open","source_ref":"feat","target_ref":"main","author_id":"33333333-3333-3333-3333-333333333333","created_at":"2026-01-01T00:00:00Z","labels":[],"assignees":[]}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.PrCmd, "list", "-R", "acme", "--watch", "--state", "merged")
	if !strings.Contains(out, "Add feature") || !strings.Contains(out, "feat") {
		t.Fatalf("table output %q", out)
	}
	if !strings.Contains(q, "state=merged") {
		t.Errorf("query %q missing state=merged", q)
	}
}

func TestPRListWatch_ndjson(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var q string
	srv := sseServe(t, "/namespaces/acme/pulls", `{"id":"22222222-2222-2222-2222-222222222222","number":3,"title":"Add feature","state":"open"}`, cancel, &q)
	out := runWatchCmdOut(t, ctx, srv.URL, cmd.PrCmd, "list", "-R", "acme", "--watch", "--output", "ndjson")
	var line struct {
		Type    string `json:"type"`
		Payload struct {
			Number int64 `json:"number"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &line); err != nil {
		t.Fatalf("stdout %q: %v", out, err)
	}
	if line.Type != "init" || line.Payload.Number != 3 {
		t.Fatalf("unexpected line %+v", line)
	}
}

func TestIssueAndPRListWatch_rejectJSONOutput(t *testing.T) {
	withServer(t, route(t, nil))
	for _, c := range []*cobra.Command{cmd.IssueCmd, cmd.PrCmd} {
		err := rootFor(c, "list", "-R", "acme", "--watch", "--output", "json").Execute()
		if err == nil || !strings.Contains(err.Error(), "cannot be used with --watch") {
			t.Fatalf("%s: want watch/output error, got %v", c.Name(), err)
		}
	}
}
