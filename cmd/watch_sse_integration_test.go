package cmd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

type safeStdout struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *safeStdout) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *safeStdout) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// TestRepoListWatch_scriptedSSESequence_ndjson is cli-watch B6: an httptest
// server emits init×3 → add → update → remove, drops the connection, then on
// reconnect (with Last-Event-ID) emits add again. The CLI watch path must
// surface events to stdout in order without duplicates after resume.
func TestRepoListWatch_scriptedSSESequence_ndjson(t *testing.T) {
	var gen atomic.Int32
	var sawResume atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/namespaces/myorg/repos" {
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			t.Errorf("want Accept text/event-stream, got %q", r.Header.Get("Accept"))
		}
		fl := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")

		switch gen.Add(1) {
		case 1:
			emit := func(id, typ, data string) {
				_, _ = fmt.Fprintf(w, "event: %s\nid: %s\ndata: %s\n\n", typ, id, data)
				fl.Flush()
			}
			emit("1", "init", `{"seq":1}`)
			emit("2", "init", `{"seq":2}`)
			emit("3", "init", `{"seq":3}`)
			emit("4", "add", `{"seq":4}`)
			emit("5", "update", `{"seq":5}`)
			emit("6", "remove", `{"seq":6}`)
			return // EOF → client reconnects with Last-Event-ID: 6
		case 2:
			if got := r.Header.Get("Last-Event-ID"); got != "6" {
				t.Errorf("reconnect Last-Event-ID = %q want 6", got)
			} else {
				sawResume.Store(true)
			}
			_, _ = fmt.Fprintf(w, "event: add\nid: 7\ndata: {\"seq\":7}\n\n")
			fl.Flush()
			<-r.Context().Done()
		default:
			http.Error(w, "unexpected third SSE connection", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "tok")

	ctx, cancel := context.WithCancel(context.Background())
	out := &safeStdout{}
	resetFlagsRecursive(cmd.RepoCmd)
	resetCtxRecursive(cmd.RepoCmd)
	setOutRecursive(cmd.RepoCmd, out, io.Discard)

	root := &cobra.Command{Use: "test"}
	root.AddCommand(cmd.RepoCmd)
	root.SetArgs([]string{"repo", "list", "--namespace", "myorg", "--watch", "--output", "ndjson", "--limit", "10"})
	root.SetOut(out)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true

	errCh := make(chan error, 1)
	go func() { errCh <- root.ExecuteContext(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-errCh:
			if err != nil && err != context.Canceled {
				t.Errorf("ExecuteContext: %v", err)
			}
		case <-time.After(8 * time.Second):
			t.Error("timed out waiting for watch goroutine to exit")
		}
	}()

	wantTypes := []string{"init", "init", "init", "add", "update", "remove", "add"}
	deadline := time.After(45 * time.Second)
	tick := time.NewTicker(15 * time.Millisecond)
	defer tick.Stop()

waitLoop:
	for {
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("ExecuteContext ended early: %v stdout=%q resume=%v", err, out.String(), sawResume.Load())
			}
			t.Fatalf("ExecuteContext returned nil before cancel stdout=%q resume=%v srvGen=%d", out.String(), sawResume.Load(), gen.Load())
		case <-deadline:
			types := ndjsonWatchEventTypes(out.String())
			t.Fatalf("timeout: got types=%v (want %d); stdout=%q resume=%v", types, len(wantTypes), out.String(), sawResume.Load())
		case <-tick.C:
			types := ndjsonWatchEventTypes(out.String())
			if len(types) < len(wantTypes) {
				continue waitLoop
			}
			for i := range wantTypes {
				if types[i] != wantTypes[i] {
					t.Fatalf("event %d: got type %q want %q; stdout=%q", i, types[i], wantTypes[i], out.String())
				}
			}
			break waitLoop
		}
	}

	if !sawResume.Load() {
		t.Fatal("expected second SSE connection with Last-Event-ID resume")
	}
}

func ndjsonWatchEventTypes(out string) []string {
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	var types []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if row.Type != "" {
			types = append(types, row.Type)
		}
	}
	return types
}
