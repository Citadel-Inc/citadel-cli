package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetGistTestFlags() {
	commands := []*cobra.Command{
		gistListCmd,
		gistViewCmd,
		gistCreateCmd,
		gistEditCmd,
		gistDeleteCmd,
		gistRawCmd,
	}
	for _, command := range commands {
		command.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Name == "file" {
				flag.Changed = false
				return
			}
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		})
	}
}

func executeGistTestCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	resetGistTestFlags()
	root := NewRootCmd()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(args)
	err := root.Execute()
	return output.String(), err
}

func TestGistRegistered(t *testing.T) {
	root := NewRootCmd()
	for _, command := range root.Commands() {
		if command == GistCmd {
			return
		}
	}
	t.Fatal("gist command is not registered on root")
}

func TestGistCRUDAndRaw(t *testing.T) {
	const rawContent = "package main\n\nfunc main() {}\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api")
		switch {
		case r.Method == http.MethodGet && path == "/gists":
			if got := r.URL.Query().Get("limit"); got != "2" {
				t.Errorf("limit = %q, want 2", got)
			}
			if got := r.URL.Query().Get("offset"); got != "1" {
				t.Errorf("offset = %q, want 1", got)
			}
			_, _ = io.WriteString(w, `{"gists":[{"id":"g1","title":"One","visibility":"public","updated_at":"2026-08-09T00:00:00Z"}]}`)
		case r.Method == http.MethodGet && path == "/gists/g1":
			_, _ = io.WriteString(w, `{"gist":{"id":"g1","title":"One","visibility":"public"},"files":[{"path":"src/main.go","raw":"package main"}]}`)
		case r.Method == http.MethodPost && path == "/gists":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			if got := body["visibility"]; got != "public" {
				t.Errorf("create visibility = %v, want public", got)
			}
			files, ok := body["files"].(map[string]any)
			if !ok || files["main.go"] != "package main" {
				t.Errorf("create files = %v", body["files"])
			}
			_, _ = io.WriteString(w, `{"id":"g1","commit_sha":"c1"}`)
		case r.Method == http.MethodPut && path == "/gists/g1":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode edit body: %v", err)
			}
			if got := body["visibility"]; got != "private" {
				t.Errorf("edit visibility = %v, want private", got)
			}
			_, _ = io.WriteString(w, `{"id":"g1","commit_sha":"c2"}`)
		case r.Method == http.MethodDelete && path == "/gists/g1":
			_, _ = io.WriteString(w, `{"deleted":true}`)
		case r.Method == http.MethodGet && path == "/gists/g1/raw/src/main.go":
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, rawContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("CITADEL_SERVER", server.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	output, err := executeGistTestCommand(t, "gist", "list", "--limit", "2", "--offset", "1", "--output", "json")
	if err != nil {
		t.Fatalf("gist list: %v", err)
	}
	if !strings.Contains(output, `"id": "g1"`) {
		t.Fatalf("gist list output = %q", output)
	}

	output, err = executeGistTestCommand(t, "gist", "view", "g1", "--output", "json")
	if err != nil {
		t.Fatalf("gist view: %v", err)
	}
	if !strings.Contains(output, `"path": "src/main.go"`) {
		t.Fatalf("gist view output = %q", output)
	}

	output, err = executeGistTestCommand(t, "gist", "create", "--title", "One", "--file", "main.go=package main", "--output", "json")
	if err != nil {
		t.Fatalf("gist create: %v", err)
	}
	if !strings.Contains(output, `"commit_sha": "c1"`) {
		t.Fatalf("gist create output = %q", output)
	}

	output, err = executeGistTestCommand(t, "gist", "edit", "g1", "--visibility", "secret", "--output", "json")
	if err != nil {
		t.Fatalf("gist edit: %v", err)
	}
	if !strings.Contains(output, `"commit_sha": "c2"`) {
		t.Fatalf("gist edit output = %q", output)
	}

	output, err = executeGistTestCommand(t, "gist", "delete", "g1", "--yes", "--output", "json")
	if err != nil {
		t.Fatalf("gist delete: %v", err)
	}
	if !strings.Contains(output, `"deleted": true`) {
		t.Fatalf("gist delete output = %q", output)
	}

	output, err = executeGistTestCommand(t, "gist", "raw", "g1", "src/main.go")
	if err != nil {
		t.Fatalf("gist raw: %v", err)
	}
	if output != rawContent {
		t.Fatalf("gist raw output = %q, want %q", output, rawContent)
	}

	target := filepath.Join(t.TempDir(), "main.go")
	if _, err := executeGistTestCommand(t, "gist", "raw", "g1", "src/main.go", "--output-file", target); err != nil {
		t.Fatalf("gist raw --output-file: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read downloaded gist: %v", err)
	}
	if string(data) != rawContent {
		t.Fatalf("downloaded gist = %q, want %q", data, rawContent)
	}
}

func TestGistList_NegativeOffset(t *testing.T) {
	_, err := executeGistTestCommand(t, "gist", "list", "--offset", "-1", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "--offset cannot be negative") {
		t.Fatalf("gist list negative offset error = %v", err)
	}
}

func TestGistList_NegativeLimit(t *testing.T) {
	_, err := executeGistTestCommand(t, "gist", "list", "--limit", "-1", "--output", "json")
	if err == nil || !strings.Contains(err.Error(), "--limit cannot be negative") {
		t.Fatalf("gist list negative limit error = %v", err)
	}
}

func TestGistDelete_BadOutput_Hermetic(t *testing.T) {
	_, err := executeGistTestCommand(t, "gist", "delete", "g1", "--output", "toml")
	if err == nil || err.Error() != `--output for delete supports json, yaml, or default human summary only; got "toml"` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestGistDelete_YAMLOutput_DryRunHermetic(t *testing.T) {
	output, err := executeGistTestCommand(t, "gist", "delete", "g1", "--output", "yaml", "--dry-run")
	if err != nil {
		t.Fatalf("gist delete YAML dry-run: %v", err)
	}
	if !strings.Contains(output, "Would DELETE /gists/g1 (skipped; --dry-run)") {
		t.Fatalf("gist delete YAML dry-run output = %q", output)
	}
}

func TestGistCreate_DryRun_Hermetic(t *testing.T) {
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	output, err := executeGistTestCommand(t, "gist", "create", "--title", "One", "--file", "main.go=package main", "--dry-run")
	if err != nil {
		t.Fatalf("gist create dry-run: %v", err)
	}
	if output != "Would POST /gists title=One (skipped; --dry-run)\n" {
		t.Fatalf("gist create dry-run output = %q", output)
	}
}

func TestGistEdit_DryRun_Hermetic(t *testing.T) {
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")

	output, err := executeGistTestCommand(t, "gist", "edit", "g1", "--title", "Two", "--dry-run")
	if err != nil {
		t.Fatalf("gist edit dry-run: %v", err)
	}
	if output != "Would PUT /gists/g1 (skipped; --dry-run)\n" {
		t.Fatalf("gist edit dry-run output = %q", output)
	}
}

func TestGistDelete_DryRun_Hermetic(t *testing.T) {
	output, err := executeGistTestCommand(t, "gist", "delete", "g1", "--dry-run")
	if err != nil {
		t.Fatalf("gist delete dry-run: %v", err)
	}
	if !strings.Contains(output, "Would DELETE /gists/g1 (skipped; --dry-run)") {
		t.Fatalf("gist delete dry-run output = %q", output)
	}
}

type gistStreamingBuffer struct {
	mu         sync.Mutex
	buffer     bytes.Buffer
	firstWrite chan struct{}
	once       sync.Once
}

func (b *gistStreamingBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.buffer.Write(p)
	if n > 0 {
		b.once.Do(func() { close(b.firstWrite) })
	}
	return n, err
}

func (b *gistStreamingBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func TestGistRawStreamsBeforeResponseEOF(t *testing.T) {
	const (
		rawPrefix = "package "
		rawSuffix = "main\n"
	)
	firstChunkSent := make(chan struct{})
	sendRemainder := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/gists/g1/raw/main.go" {
			t.Errorf("request path = %q, want /gists/g1/raw/main.go", got)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, rawPrefix)
		w.(http.Flusher).Flush()
		close(firstChunkSent)
		<-sendRemainder
		_, _ = io.WriteString(w, rawSuffix)
	}))
	defer server.Close()
	t.Setenv("CITADEL_SERVER", server.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	resetGistTestFlags()
	root := NewRootCmd()
	output := &gistStreamingBuffer{firstWrite: make(chan struct{})}
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"gist", "raw", "g1", "main.go"})
	done := make(chan error, 1)
	go func() { done <- root.Execute() }()

	select {
	case <-firstChunkSent:
	case <-time.After(2 * time.Second):
		close(sendRemainder)
		<-done
		t.Fatal("server did not send the first response chunk")
	}
	select {
	case <-output.firstWrite:
	case <-time.After(2 * time.Second):
		close(sendRemainder)
		<-done
		t.Fatal("gist raw did not write before response EOF")
	}
	close(sendRemainder)

	if err := <-done; err != nil {
		t.Fatalf("gist raw: %v", err)
	}
	if got, want := output.String(), rawPrefix+rawSuffix; got != want {
		t.Fatalf("gist raw output = %q, want %q", got, want)
	}
}

func TestGistRawRefusesBinaryTTY(t *testing.T) {
	const rawContent = "\x00\x01\x02binary"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/gists/g1/raw/image.bin" {
			t.Errorf("request path = %q, want /gists/g1/raw/image.bin", got)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, rawContent)
	}))
	defer server.Close()
	t.Setenv("CITADEL_SERVER", server.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")

	originalOutputIsTTY := downloadOutputIsTTY
	downloadOutputIsTTY = func(io.Writer) bool { return true }
	defer func() { downloadOutputIsTTY = originalOutputIsTTY }()

	_, err := executeGistTestCommand(t, "gist", "raw", "g1", "image.bin")
	if err == nil || !strings.Contains(err.Error(), "refusing to write binary gist file to a terminal") {
		t.Fatalf("gist raw binary TTY error = %v", err)
	}
	target := filepath.Join(t.TempDir(), "image.bin")
	if _, err := executeGistTestCommand(t, "gist", "raw", "g1", "image.bin", "--output-file", target); err != nil {
		t.Fatalf("gist raw binary --output-file: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read downloaded binary gist: %v", err)
	}
	if string(data) != rawContent {
		t.Fatalf("downloaded binary gist = %q, want %q", data, rawContent)
	}
}
