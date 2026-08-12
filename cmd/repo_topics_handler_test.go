package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── topic list ────────────────────────────────────────────────────────────────

func TestRepoTopicList_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go", "cli", "devtools"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"go", "cli", "devtools"} {
		if !strings.Contains(buf.String(), expected) {
			t.Fatalf("expected %q in output, got: %s", expected, buf.String())
		}
	}
}

func TestRepoTopicList_Empty(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "(no topics)\n" {
		t.Fatalf("empty topic list output = %q", got)
	}
}

func TestRepoTopicList_BadOutput_Hermetic(t *testing.T) {
	assertRepoTopicBadOutput(t, "list", "acme/demo")
}

func TestRepoTopicList_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"topic", "list", "--no-cwd-repo",
		"--output", "xml",
	).Execute()
	if err == nil || err.Error() != `--output: unknown format "xml" (use json|yaml|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestRepoTopicList_MissingRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd, "topic", "list", "--no-cwd-repo").Execute()
	if err == nil || err.Error() != "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git" {
		t.Fatalf("want repository path error, got %v", err)
	}
}

func TestRepoTopicList_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "list", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
}

func TestRepoTopicList_YAML(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "list", "acme/demo", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err == nil {
		t.Fatalf("expected YAML, got JSON: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "topics:") {
		t.Fatalf("expected topics key in YAML, got: %s", buf.String())
	}
}

func TestRepoTopicList_YAML_Padded(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "list", "acme/demo", "--output", " yaml ").Execute(); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	if strings.HasPrefix(out, "{") || !strings.Contains(out, "topics:") {
		t.Fatalf("expected YAML topics output, got: %s", out)
	}
}

// ── topic set ─────────────────────────────────────────────────────────────────

func TestRepoTopicSet_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go", "cli"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "set", "acme/demo", "go", "cli").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Topics set") {
		t.Fatalf("expected 'Topics set', got: %s", buf.String())
	}
}

func TestRepoTopicSet_Clear(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "set", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "Topics cleared.\n" {
		t.Fatalf("clear topics output = %q", got)
	}
}

func TestRepoTopicSet_BadOutput_Hermetic(t *testing.T) {
	assertRepoTopicBadOutput(t, "set", "acme/demo")
}

func TestRepoTopicSet_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"topic", "set", "--no-cwd-repo",
		"--output", "xml",
	).Execute()
	if err == nil || err.Error() != `--output: unknown format "xml" (use json|yaml|table)` {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestRepoTopicSet_MissingRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd, "topic", "set", "--no-cwd-repo").Execute()
	if err == nil || err.Error() != "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git" {
		t.Fatalf("want repository path error, got %v", err)
	}
}

func TestRepoTopicSet_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "set", "acme/demo", "go", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
}

func TestRepoTopicSet_YAML(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT /api/namespaces/acme/repos/demo/topics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"topics": []string{"go"}})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "set", "acme/demo", "go", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err == nil {
		t.Fatalf("expected YAML, got JSON: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "topics:") {
		t.Fatalf("expected topics key in YAML, got: %s", buf.String())
	}
}

// ── topic popular ─────────────────────────────────────────────────────────────

func TestRepoTopicPopular_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/topics/popular": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{
				{"topic": "go", "count": 42},
				{"topic": "cli", "count": 17},
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "popular").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "go") {
		t.Fatalf("expected 'go' in output, got: %s", buf.String())
	}
}

func TestRepoTopicPopular_Empty(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/topics/popular": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "popular").Execute(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "(no topics found)\n" {
		t.Fatalf("empty popular topics output = %q", got)
	}
}

func TestRepoTopicPopular_BadOutput_Hermetic(t *testing.T) {
	assertRepoTopicBadOutput(t, "popular")
}

func TestRepoTopicPopular_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/topics/popular": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{
				{"topic": "go", "count": 42},
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "popular", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(out))
	}
}

func TestRepoTopicPopular_YAML(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/topics/popular": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, []map[string]any{
				{"topic": "go", "count": 42},
			})
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "topic", "popular", "--output", "yaml").Execute(); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(buf.String())
	var object map[string]any
	if err := json.Unmarshal(buf.Bytes(), &object); err == nil {
		t.Fatalf("expected YAML sequence, got JSON object: %s", out)
	}
	if strings.Contains(out, "topics:") {
		t.Fatalf("expected unwrapped YAML sequence, got: %s", out)
	}
	if !strings.HasPrefix(out, "- ") || !strings.Contains(out, "topic: go") {
		t.Fatalf("expected YAML sequence, got: %s", out)
	}
}

func assertRepoTopicBadOutput(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd, append([]string{"topic"}, append(args, "--output", "toml")...)...).Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}
