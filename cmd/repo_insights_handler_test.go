package cmd_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// ── insights ──────────────────────────────────────────────────────────────────

func makeInsightsFull() map[string]any {
	return map[string]any{
		"topics":    []string{"go", "cli"},
		"star_count": 5,
		"pin_count":  1,
		"counts": map[string]any{
			"open_issues":      3,
			"open_milestones":  1,
			"branches":         4,
			"tags":             2,
			"contributors_30d": 2,
		},
		"releases": []map[string]any{
			{
				"name": "v1.0.0", "sha": "abc1234567890def",
				"tagged_at": "2024-01-15T12:00:00Z", "is_annotated": true, "annotation": "First release",
			},
		},
		"activity": []int{0, 1, 2, 3, 4, 5, 6, 7},
		"recent_contributors": []map[string]any{
			{"email": "alice@example.com", "author": "alice", "count": 12, "slug": "alice", "display_name": "Alice"},
		},
		"languages": map[string]any{"Go": 98304, "YAML": 2048},
		"license":   map[string]any{"spdx": "MIT", "name": "MIT License", "path": "LICENSE"},
	}
}

func makeInsightsEmpty() map[string]any {
	return map[string]any{
		"topics":    []string{},
		"star_count": 0,
		"pin_count":  0,
		"counts": map[string]any{
			"open_issues":      0,
			"open_milestones":  0,
			"branches":         0,
			"tags":             0,
			"contributors_30d": 0,
		},
	}
}

func TestRepoInsights_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/insights": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeInsightsFull())
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "insights", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Stars", "Counts", "License", "Languages", "Activity"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestRepoInsights_EmptyRepo(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/insights": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeInsightsEmpty())
		},
	}))
	// Empty repo has no git-backed fields — should not panic
	if err := rootForOut(cmd.RepoCmd, &buf, "insights", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Stars") {
		t.Fatalf("expected 'Stars' in output, got: %s", buf.String())
	}
}

func TestRepoInsights_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/insights": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeInsightsFull())
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "insights", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if out["star_count"] == nil {
		t.Fatalf("expected star_count field in JSON output")
	}
}

func TestRepoInsights_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/missing/insights": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 404, map[string]any{"error": "not_found"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "insights", "acme/missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestRepoInsights_NoLicense(t *testing.T) {
	var buf bytes.Buffer
	resp := makeInsightsFull()
	resp["license"] = nil
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /api/namespaces/acme/repos/demo/insights": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, resp)
		},
	}))
	if err := rootForOut(cmd.RepoCmd, &buf, "insights", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "(none)") {
		t.Fatalf("expected '(none)' for missing license, got: %s", buf.String())
	}
}
