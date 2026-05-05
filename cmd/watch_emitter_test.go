package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/sseclient"
)

func TestSSEWatchQuery(t *testing.T) {
	q := sseWatchQuery(25, "cur123", true, url.Values{"namespace": []string{"myorg"}})
	vals, err := url.ParseQuery(q)
	if err != nil {
		t.Fatal(err)
	}
	if vals.Get("limit") != "25" || vals.Get("cursor") != "cur123" || vals.Get("all") != "true" || vals.Get("namespace") != "myorg" {
		t.Fatalf("query = %v", vals)
	}
	if sseWatchQuery(50, "", false, nil) == "" {
		t.Fatal("expected non-empty query")
	}
}

func TestStdoutIsTTY_nonFileWriter(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	if stdoutIsTTY(cmd) {
		t.Fatal("bytes.Buffer must not look like a TTY")
	}
}

func TestFlushOut_nonFile(t *testing.T) {
	flushOut(&bytes.Buffer{}) // must not panic
}

func TestNotifyWatchContext_smoke(t *testing.T) {
	ctx, stop := notifyWatchContext(context.Background())
	defer stop()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
}

func TestNewWatchSSEHandler_ndjsonVsTable(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("output", "", "")
	cmd.Flags().String("color", "never", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Flags().Set("output", "ndjson"); err != nil {
		t.Fatal(err)
	}
	h, err := newWatchSSEHandler(cmd, watchRepos, watchTableCtx{repoParentNS: "ns"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := h.(*ndjsonWatchEmitter); !ok {
		t.Fatalf("got %T", h)
	}

	if err := cmd.Flags().Set("output", ""); err != nil {
		t.Fatal(err)
	}
	h2, err := newWatchSSEHandler(cmd, watchRepos, watchTableCtx{repoParentNS: "ns"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := h2.(*tableWatchEmitter); !ok {
		t.Fatalf("got %T", h2)
	}
}

func TestNdjsonWatchEmitter_Handle(t *testing.T) {
	buf := &bytes.Buffer{}
	e := newNdjsonWatchEmitter(buf)
	if err := e.Handle(sseclient.Event{Type: "init", Data: []byte(`{"k":1}`)}); err != nil {
		t.Fatal(err)
	}
	if err := e.Handle(sseclient.Event{Type: "ping", Data: nil}); err != nil {
		t.Fatal(err)
	}
	var lines [][]byte
	for _, line := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines: %q", len(lines), buf.String())
	}
	var row map[string]json.RawMessage
	if err := json.Unmarshal(lines[0], &row); err != nil {
		t.Fatal(err)
	}
	if string(row["type"]) != `"init"` {
		t.Fatalf("first line: %s", lines[0])
	}
}

func TestValidateWatchOutput_unknownFormat(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().Bool("watch", false, "")
	c.Flags().String("output", "", "")
	if err := c.Flags().Set("watch", "true"); err != nil {
		t.Fatal(err)
	}
	if err := c.Flags().Set("output", "binary"); err != nil {
		t.Fatal(err)
	}
	if err := validateWatchOutput(c); err == nil {
		t.Fatal("expected error")
	}
}

func TestAddWatchFlag_andWatchFlag(t *testing.T) {
	c := &cobra.Command{}
	addWatchFlag(c)
	if err := c.Flags().Set("watch", "true"); err != nil {
		t.Fatal(err)
	}
	if !watchFlag(c) {
		t.Fatal("watchFlag")
	}
	if err := c.Flags().Set("watch", "false"); err != nil {
		t.Fatal(err)
	}
	if watchFlag(c) {
		t.Fatal("expected false")
	}
}

func TestTableWatchEmitter_repo_appendMode(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("color", "never", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	em := newTableWatchEmitter(cmd, watchRepos, watchTableCtx{repoParentNS: "ns"})
	if err := em.Handle(sseclient.Event{Type: "init", Data: []byte(`{"path":"ns/a","visibility":"private","default_branch":"main","created_at":"2026-01-01"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := em.Handle(sseclient.Event{Type: "add", Data: []byte(`{"path":"ns/b","visibility":"public","default_branch":"main","created_at":"2026-01-02"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := em.Handle(sseclient.Event{Type: "update", Data: []byte(`{"path":"ns/a","visibility":"public","default_branch":"main","created_at":"2026-01-01"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := em.Handle(sseclient.Event{Type: "remove", Data: []byte(`{"slug":"b","namespace":"ns"}`)}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "PATH") || !strings.Contains(out, "ns/a") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestTableWatchEmitter_allKinds_smoke(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("color", "never", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	cases := []struct {
		kind watchListKind
		ctx  watchTableCtx
		typ  string
		data string
	}{
		{watchAgents, watchTableCtx{}, "init", `{"id":"00000000-0000-0000-0000-000000000001","name":"bot","owner_user_id":"u1"}`},
		{watchOAuthClients, watchTableCtx{}, "init", `{"id":"11111111-1111-1111-1111-111111111111","client_id":"cid","name":"App","allowed_scopes":["openid"],"redirect_uris":["https://x"],"is_public":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`},
		{watchOrgs, watchTableCtx{}, "init", `{"slug":"o","display_name":"O","created_at":"2026-01-01T00:00:00Z"}`},
		{watchOrgMembers, watchTableCtx{orgSlug: "o"}, "init", `{"user_id":"u1","slug":"alice","display_name":"Alice","is_owner":true,"permissions":[],"joined_at":"2026-01-01T00:00:00Z"}`},
		{watchTransfersPending, watchTableCtx{}, "init", `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","org_namespace_id":"x","org_slug":"o","from_user_id":"a","from_user_slug":"fs","to_user_id":"b","to_user_slug":"ts","expires_at":"2026-02-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z"}`},
		{watchAgentTokens, watchTableCtx{}, "init", `{"id":"00000000-0000-0000-0000-000000000022","agent_id":"00000000-0000-0000-0000-000000000001","created_at":"2026-01-01T00:00:00Z"}`},
	}
	for i, tc := range cases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			buf.Reset()
			cmd.SetOut(buf)
			em := newTableWatchEmitter(cmd, tc.kind, tc.ctx)
			if err := em.Handle(sseclient.Event{Type: tc.typ, Data: []byte(tc.data)}); err != nil {
				t.Fatal(err)
			}
			if buf.Len() == 0 {
				t.Fatal("no output")
			}
		})
	}
}

func TestTableWatchEmitter_redrawPath(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("color", "always", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	em := newTableWatchEmitter(cmd, watchRepos, watchTableCtx{repoParentNS: "ns"})
	em.redraw = true // force redraw branch without needing a real TTY fd
	if err := em.Handle(sseclient.Event{Type: "init", Data: []byte(`{"path":"ns/a","visibility":"private","default_branch":"main","created_at":"2026-01-01"}`)}); err != nil {
		t.Fatal(err)
	}
	if err := em.Handle(sseclient.Event{Type: "init", Data: []byte(`{"path":"ns/b","visibility":"private","default_branch":"main","created_at":"2026-01-02"}`)}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if em.paintedLines <= 0 {
		t.Fatalf("expected painted lines")
	}
	if !strings.Contains(out, "\033[") {
		t.Fatalf("expected ANSI cursor sequence in redraw mode, out=%q lines=%d", out, em.paintedLines)
	}
}

func TestTableWatchEmitter_ignoreUnknownEventType(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("color", "never", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	em := newTableWatchEmitter(cmd, watchRepos, watchTableCtx{repoParentNS: "ns"})
	if err := em.Handle(sseclient.Event{Type: "ping", Data: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("unexpected output %q", buf.String())
	}
}
