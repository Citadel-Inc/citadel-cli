package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func setRepoBrowseHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

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

func TestRepoBrowseTree_BadOutput_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "tree", "acme/demo",
		"--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestRepoBrowseTree_MissingRepo_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "tree", "--no-cwd-repo").Execute()
	if err == nil || err.Error() != "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git" {
		t.Fatalf("want repository path error, got %v", err)
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

func TestRepoBrowseBlob_BadOutput_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "blob", "acme/demo",
		"--path", "README.md", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output: unknown format "toml" (use json|yaml|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestRepoBrowseBlob_MissingPath_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "blob", "acme/demo").Execute()
	if err == nil || err.Error() != "--path is required for blob" {
		t.Fatalf("want path required error, got %v", err)
	}
}

func TestRepoBrowseBlob_MissingRepo_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "blob", "--no-cwd-repo",
		"--path", "README.md").Execute()
	if err == nil || err.Error() != "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git" {
		t.Fatalf("want repository path error, got %v", err)
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

// ── browse raw ────────────────────────────────────────────────────────────────

func TestRepoBrowseRaw_TextStream(t *testing.T) {
	const content = "Hello, streamed world!\n"
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/raw": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("path") != "README.md" {
				http.Error(w, "missing path", http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("ref") != "feature/x" {
				http.Error(w, "missing ref", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(content))
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "browse", "raw", "acme/demo", "README.md", "--ref", "feature/x").Execute(); err != nil {
		t.Fatal(err)
	}
	if buf.String() != content {
		t.Fatalf("expected streamed content %q, got %q", content, buf.String())
	}
}

func TestRepoBrowseRaw_FileOutput(t *testing.T) {
	want := append([]byte{0x00, 0xff, 0x10}, bytes.Repeat([]byte("payload"), 2048)...)
	outputPath := t.TempDir() + "/artifact.bin"
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/raw": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(want)
		},
	}))
	if err := rootFor(cmd.RepoCmd, "browse", "raw", "acme/demo", "artifact.bin", "--output-file", outputPath).Execute(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("output file bytes differ: got %d bytes, want %d", len(got), len(want))
	}
}

func TestRepoBrowseRaw_MissingPath_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "raw", "-R", "acme/demo", "").Execute()
	if err == nil || err.Error() != "file path required" {
		t.Fatalf("want file path error, got %v", err)
	}
}

func TestRepoBrowseRaw_MissingRepo_Hermetic(t *testing.T) {
	setRepoBrowseHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "browse", "raw", "README.md",
		"--no-cwd-repo").Execute()
	if err == nil || err.Error() != "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git" {
		t.Fatalf("want repository path error, got %v", err)
	}
}
