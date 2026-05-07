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

func makeCommit(sha, author, msg string) map[string]any {
	return map[string]any{
		"sha":             sha,
		"message":         msg,
		"author":          author,
		"author_email":    author + "@example.com",
		"committer":       author,
		"committer_email": author + "@example.com",
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}
}

func commitsPage(items []map[string]any, nextCursor string) map[string]any {
	return map[string]any{"commits": items, "next_cursor": nextCursor, "ref": "main"}
}

// ── list ─────────────────────────────────────────────────────────────────────

func TestRepoCommitList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, commitsPage([]map[string]any{
				makeCommit("abc1234567890def", "alice", "feat: add feature"),
				makeCommit("def234567890abcd", "bob", "fix: fix bug"),
			}, ""))
		},
	}))
	if err := rootFor(cmd.RepoCmd, "commit", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCommitList_Empty(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, commitsPage([]map[string]any{}, ""))
		},
	}))
	if err := rootFor(cmd.RepoCmd, "commit", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCommitList_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, commitsPage([]map[string]any{
				makeCommit("abc1234567890def", "alice", "feat: initial"),
			}, ""))
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "commit", "list", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out))
	}
}

func TestRepoCommitList_RefAndPathFilter(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("ref"); got != "feature/x" {
				t.Errorf("ref = %q, want feature/x", got)
			}
			if got := r.URL.Query().Get("path"); got != "src/main.go" {
				t.Errorf("path = %q, want src/main.go", got)
			}
			writeJSON(t, w, 200, commitsPage([]map[string]any{
				makeCommit("abc1234567890def", "alice", "chore: tweak"),
			}, ""))
		},
	}))
	if err := rootFor(cmd.RepoCmd, "commit", "list", "acme/demo",
		"--ref", "feature/x", "--path", "src/main.go").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCommitList_Pagination(t *testing.T) {
	calls := 0
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits": func(w http.ResponseWriter, r *http.Request) {
			calls++
			cursor := r.URL.Query().Get("after")
			if cursor == "" {
				writeJSON(t, w, 200, commitsPage([]map[string]any{
					makeCommit("page1sha000000000", "alice", "page 1 commit"),
				}, "cursor-page2"))
			} else {
				writeJSON(t, w, 200, commitsPage([]map[string]any{
					makeCommit("page2sha000000000", "bob", "page 2 commit"),
				}, ""))
			}
		},
	}))
	if err := rootFor(cmd.RepoCmd, "commit", "list", "acme/demo", "--all").Execute(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 API calls for --all pagination, got %d", calls)
	}
}

func TestRepoCommitList_404(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/missing/commits": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 404, map[string]any{"error": "not_found"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "commit", "list", "acme/missing").Execute()
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestRepoCommitList_NoAuth(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 401, map[string]any{"error": "unauthenticated"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "commit", "list", "acme/demo").Execute()
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

// ── get ───────────────────────────────────────────────────────────────────────

func TestRepoCommitGet_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/abc1234567890def": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"commit": map[string]any{
					"sha":             "abc1234567890def",
					"message":         "feat: add feature\n\nLonger description here.",
					"author":          "alice",
					"author_email":    "alice@example.com",
					"committer":       "alice",
					"committer_email": "alice@example.com",
					"timestamp":       time.Now().UTC().Format(time.RFC3339),
					"parents":         []string{"def234567890abcd"},
					"files": []map[string]any{
						{"path": "src/main.go", "status": "modified", "additions": 5, "deletions": 2},
					},
					"signature": map[string]any{"present": false, "verified": false},
				},
			})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "commit", "get", "acme/demo", "abc1234567890def").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoCommitGet_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/abc1234567890def": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"commit": map[string]any{
					"sha":     "abc1234567890def",
					"message": "fix: something",
					"parents": []string{},
					"files":   []map[string]any{},
					"signature": map[string]any{
						"present":  true,
						"kind":     "gpg",
						"verified": true,
					},
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "commit", "get", "acme/demo", "abc1234567890def",
		"--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
}

func TestRepoCommitGet_WithPath(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/abc1234567890def/diff": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("path"); got != "src/main.go" {
				t.Errorf("path query = %q, want src/main.go", got)
			}
			writeJSON(t, w, 200, map[string]any{
				"unified":        "--- a/src/main.go\n+++ b/src/main.go\n@@ -1 +1 @@\n-old\n+new\n",
				"truncated":      false,
				"binary":         false,
				"initial_commit": false,
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "commit", "get", "acme/demo", "abc1234567890def",
		"--path", "src/main.go").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "--- a/src/main.go") {
		t.Fatalf("expected unified diff in output, got: %s", buf.String())
	}
}

func TestRepoCommitGet_WithPathJSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/abc1234567890def/diff": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"unified":        "--- a/src/main.go\n+++ b/src/main.go\n@@ -1 +1 @@\n-old\n+new\n",
				"truncated":      false,
				"binary":         false,
				"initial_commit": false,
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "commit", "get", "acme/demo", "abc1234567890def",
		"--path", "src/main.go", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if _, ok := out["unified"]; !ok {
		t.Fatalf("expected 'unified' key in JSON output, got: %v", out)
	}
}

func TestRepoCommitGet_InitialCommit(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/rootsha00000000/diff": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"unified":        "",
				"truncated":      false,
				"binary":         false,
				"initial_commit": true,
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "commit", "get", "acme/demo", "rootsha00000000",
		"--path", "README.md").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "initial commit") {
		t.Fatalf("expected initial commit message, got: %s", buf.String())
	}
}

func TestRepoCommitGet_404(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/deadbeef": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 404, map[string]any{"error": "not_found"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "commit", "get", "acme/demo", "deadbeef").Execute()
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestRepoCommitGet_NoAuth(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/commits/abc1234567890def": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 401, map[string]any{"error": "unauthenticated"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "commit", "get", "acme/demo", "abc1234567890def").Execute()
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
