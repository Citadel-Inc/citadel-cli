package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── browse tree ───────────────────────────────────────────────────────────────

func makeBrowseTree(ref, path string, entries []map[string]any) map[string]any {
	return map[string]any{"ref": ref, "path": path, "entries": entries}
}

func makeTreeEntry(p, kind string, size int64, sha string) map[string]any {
	return map[string]any{"path": p, "mode": "100644", "kind": kind, "size": size, "sha": sha}
}

func TestRepoBrowseTree_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/tree": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeBrowseTree("main", "", []map[string]any{
				makeTreeEntry("README.md", "blob", 1024, "abc1234567890def"),
				makeTreeEntry("cmd", "tree", 0, "def234567890abcd"),
			}))
		},
	}))
	if err := rootFor(cmd.RepoCmd, "browse", "tree", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoBrowseTree_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/tree": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeBrowseTree("main", "", []map[string]any{}))
		},
	}))
	if err := rootFor(cmd.RepoCmd, "browse", "tree", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoBrowseTree_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/tree": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeBrowseTree("main", "", []map[string]any{
				makeTreeEntry("main.go", "blob", 512, "abc1234567890def"),
			}))
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "browse", "tree", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
}

func TestRepoBrowseTree_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/tree": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 404, map[string]any{"error": "not_found"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "browse", "tree", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestRepoBrowseTree_WithRefAndPath(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/tree": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("ref") != "feature/x" {
				http.Error(w, "missing ref", http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("path") != "cmd" {
				http.Error(w, "missing path", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, 200, makeBrowseTree("feature/x", "cmd", []map[string]any{
				makeTreeEntry("main.go", "blob", 512, "abc1234567890def"),
			}))
		},
	}))
	if err := rootFor(cmd.RepoCmd, "browse", "tree", "acme/demo", "--ref", "feature/x", "--path", "cmd").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── browse blob ───────────────────────────────────────────────────────────────

func TestRepoBrowseBlob_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/blob": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"sha": "abc1234567890def", "size": 13, "binary": false,
				"encoding": "utf-8", "content": "Hello, world!\n",
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "browse", "blob", "acme/demo", "--path", "README.md").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Hello, world!") {
		t.Fatalf("expected file content in output, got: %s", buf.String())
	}
}

func TestRepoBrowseBlob_Binary(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/blob": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"sha": "abc1234567890def", "size": 9999, "binary": true,
				"encoding": "none", "content": "",
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "browse", "blob", "acme/demo", "--path", "image.png").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Binary file") {
		t.Fatalf("expected binary notice, got: %s", buf.String())
	}
}

func TestRepoBrowseBlob_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/blob": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"sha": "abc1234567890def", "size": 5, "binary": false,
				"encoding": "utf-8", "content": "hello",
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "browse", "blob", "acme/demo", "--path", "f.txt", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
}

func TestRepoBrowseBlob_MissingPath(t *testing.T) {
	// No server needed — error fires before HTTP call
	err := rootFor(cmd.RepoCmd, "browse", "blob", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "--path is required") {
		t.Fatalf("expected --path required error, got: %v", err)
	}
}

func TestRepoBrowseBlob_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/blob": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 404, map[string]any{"error": "not_found"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "browse", "blob", "acme/demo", "--path", "missing.go").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}
