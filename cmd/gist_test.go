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
	"testing"

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
