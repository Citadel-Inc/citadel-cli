package cmd_test

// SSE integration tests for the watch-list variants not covered by
// watch_sse_integration_test.go (agent, oauth-clients, namespace, token).
// Each test spins up an httptest server that emits one init event, then
// immediately cancels the test context so consumeSSEWatch returns nil.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// sseOnce is a helper that starts a test server emitting exactly one SSE
// event (kind=init, data=jsonData) on path, then blocking until the request
// context is canceled. cancel is called after the flush so the watch loop
// exits cleanly.
func sseOnce(t *testing.T, path, jsonData string, cancel context.CancelFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Error("responseWriter does not support Flush")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = fmt.Fprintf(w, "event: init\ndata: %s\n\n", jsonData)
		fl.Flush()
		cancel()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv
}

// runWatchCmd executes verb via ExecuteContext(ctx) and returns any error that
// is not context.Canceled / context.DeadlineExceeded.
func runWatchCmd(t *testing.T, ctx context.Context, srvURL string, verb *cobra.Command, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srvURL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	resetFlagsRecursive(verb)
	resetCtxRecursive(verb)
	setOutRecursive(verb, io.Discard, io.Discard)

	root := &cobra.Command{Use: "test"}
	root.AddCommand(verb)
	root.SetArgs(append([]string{verb.Name()}, args...))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true

	errCh := make(chan error, 1)
	go func() { errCh <- root.ExecuteContext(ctx) }()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ExecuteContext: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for watch command to exit")
	}
}

// ── agent list --watch ────────────────────────────────────────────────────────

func TestAgentListWatch_smokeNdjson(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"id":"00000000-0000-0000-0000-000000000001","name":"bot","owner_user_id":"u1"}`
	srv := sseOnce(t, "/agents", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.AgentCmd, "list", "--watch", "--output", "ndjson")
}

func TestAgentListWatch_smokeTable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"id":"00000000-0000-0000-0000-000000000001","name":"bot","owner_user_id":"u1"}`
	srv := sseOnce(t, "/agents", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.AgentCmd, "list", "--watch")
}

// ── oauth clients list --watch ────────────────────────────────────────────────

func TestOAuthClientsListWatch_smokeNdjson(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"id":"11111111-1111-1111-1111-111111111111","client_id":"cid","name":"App","allowed_scopes":["openid"],"redirect_uris":["https://x"],"is_public":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`
	srv := sseOnce(t, "/oauth/clients", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.OauthCmd, "clients", "list", "--watch", "--output", "ndjson")
}

func TestOAuthClientsListWatch_withOrg(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"id":"11111111-1111-1111-1111-111111111111","client_id":"cid","name":"App","allowed_scopes":[],"redirect_uris":["https://x"],"is_public":false,"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`
	srv := sseOnce(t, "/oauth/clients", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.OauthCmd, "clients", "list", "--watch", "--output", "ndjson", "--org", "myorg")
}

// ── namespace list --watch ────────────────────────────────────────────────────

func TestNsListWatch_smokeNdjson(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"slug":"myorg","display_name":"My Org","created_at":"2026-01-01T00:00:00Z"}`
	srv := sseOnce(t, "/orgs", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.NamespaceCmd, "list", "--watch", "--output", "ndjson")
}

func TestNsListWatch_smokeTable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"slug":"myorg","display_name":"My Org","created_at":"2026-01-01T00:00:00Z"}`
	srv := sseOnce(t, "/orgs", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.NamespaceCmd, "list", "--watch")
}

// ── namespace members --watch ─────────────────────────────────────────────────

func TestNsMembersWatch_smokeNdjson(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"user_id":"u1","slug":"alice","display_name":"Alice","is_owner":true,"permissions":[],"joined_at":"2026-01-01T00:00:00Z"}`
	srv := sseOnce(t, "/orgs/myorg/members", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.NamespaceCmd, "members", "myorg", "--watch", "--output", "ndjson")
}

// ── namespace transfer list-pending --watch ───────────────────────────────────

func TestNsTransferListPendingWatch_smokeNdjson(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const data = `{"id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","org_namespace_id":"x","org_slug":"o","from_user_id":"a","from_user_slug":"fs","to_user_id":"b","to_user_slug":"ts","expires_at":"2026-02-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z"}`
	srv := sseOnce(t, "/transfers/pending", data, cancel)
	runWatchCmd(t, ctx, srv.URL, cmd.NamespaceCmd, "transfer", "list-pending", "--watch", "--output", "ndjson")
}

// ── token list --watch ────────────────────────────────────────────────────────

func TestTokenListWatch_smokeNdjson(t *testing.T) {
	const agentID = "00000000-0000-0000-0000-000000000001"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/agents":
			// token list calls findAgentByName which fetches /agents
			writeJSON(t, w, 200, agentsJSON([]map[string]any{
				{"id": agentID, "name": "mybot", "owner_user_id": "u1"},
			}))
		case "/agent-tokens":
			fl := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprintf(w, "event: init\ndata: {\"id\":\"%s\",\"agent_id\":\"%s\",\"created_at\":\"2026-01-01T00:00:00Z\"}\n\n", agentID, agentID)
			fl.Flush()
			cancel()
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	runWatchCmd(t, ctx, srv.URL, cmd.TokenCmd, "list", "--agent", "mybot", "--watch", "--output", "ndjson")
}
