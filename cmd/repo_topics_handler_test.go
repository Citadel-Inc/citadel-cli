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
	if !strings.Contains(buf.String(), "(no topics)") {
		t.Fatalf("expected '(no topics)', got: %s", buf.String())
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
	if !strings.Contains(buf.String(), "cleared") {
		t.Fatalf("expected 'cleared', got: %s", buf.String())
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
	if !strings.Contains(buf.String(), "(no topics found)") {
		t.Fatalf("expected '(no topics found)', got: %s", buf.String())
	}
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
