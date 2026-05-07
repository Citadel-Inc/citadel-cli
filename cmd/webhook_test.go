package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestRepoWebhookList_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/api/namespaces/acme%2Fdemo/webhooks", "/api/namespaces/acme/demo/webhooks") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"webhooks": []map[string]any{{
				"id":                  "11111111-1111-1111-1111-111111111111",
				"namespace_id":        "ns1",
				"namespace_path":      "acme/demo",
				"name":                "issues",
				"target_url":          "https://example.test/webhook",
				"event_kinds":         []string{"issue.opened"},
				"include_descendants": false,
				"active":              true,
				"created_at":          "2026-05-07T00:00:00Z",
				"updated_at":          "2026-05-07T00:00:00Z",
				"secret_hint":         "abcd1234",
			}},
			"next_cursor": "",
		})
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "webhook", "list", "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"namespace_path": "acme/demo"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRepoWebhookCreate_HumanShowsReturnedSecret(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/api/namespaces/acme%2Fdemo/webhooks", "/api/namespaces/acme/demo/webhooks") {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Name               string   `json:"name"`
			TargetURL          string   `json:"target_url"`
			EventKinds         []string `json:"event_kinds"`
			IncludeDescendants bool     `json:"include_descendants"`
			Active             bool     `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name != "issues" || body.TargetURL != "https://example.test/inbox" || len(body.EventKinds) != 2 || body.IncludeDescendants || !body.Active {
			t.Fatalf("unexpected create body: %+v", body)
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"id":                  "11111111-1111-1111-1111-111111111111",
			"namespace_id":        "ns1",
			"namespace_path":      "acme/demo",
			"name":                "issues",
			"target_url":          "https://example.test/inbox",
			"event_kinds":         []string{"comment.created", "issue.opened"},
			"include_descendants": false,
			"active":              true,
			"created_at":          "2026-05-07T00:00:00Z",
			"updated_at":          "2026-05-07T00:00:00Z",
			"secret_hint":         "abcd1234",
			"cleartext_secret":    "super-secret-value",
		})
	})

	var out strings.Builder
	root := rootForOut(cmd.RepoCmd, &out,
		"webhook", "create", "-R", "acme/demo",
		"--name", "issues",
		"--url", "https://example.test/inbox",
		"--events", "issue.opened,comment.created",
	)
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Created webhook 11111111-1111-1111-1111-111111111111 for acme/demo.") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "Secret (save now; shown once): super-secret-value") {
		t.Fatalf("missing secret in output: %s", got)
	}
}

func TestRepoWebhookGet_JSONFiltersFromList(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/api/namespaces/acme%2Fdemo/webhooks", "/api/namespaces/acme/demo/webhooks") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"webhooks": []map[string]any{
				{
					"id":                  "11111111-1111-1111-1111-111111111111",
					"namespace_id":        "ns1",
					"namespace_path":      "acme/demo",
					"name":                "issues",
					"target_url":          "https://example.test/one",
					"event_kinds":         []string{"issue.opened"},
					"include_descendants": false,
					"active":              true,
					"created_at":          "2026-05-07T00:00:00Z",
					"updated_at":          "2026-05-07T00:00:00Z",
				},
				{
					"id":                  "22222222-2222-2222-2222-222222222222",
					"namespace_id":        "ns1",
					"namespace_path":      "acme/demo",
					"name":                "comments",
					"target_url":          "https://example.test/two",
					"event_kinds":         []string{"comment.created"},
					"include_descendants": false,
					"active":              false,
					"created_at":          "2026-05-07T00:00:00Z",
					"updated_at":          "2026-05-07T00:00:00Z",
				},
			},
		})
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out,
		"webhook", "get", "-R", "acme/demo", "22222222-2222-2222-2222-222222222222", "--output", "json",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "11111111-1111-1111-1111-111111111111") || !strings.Contains(out.String(), `"id": "22222222-2222-2222-2222-222222222222"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestNamespaceWebhookDelete_DryRun(t *testing.T) {
	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out,
		"webhook", "delete", "acme", "33333333-3333-3333-3333-333333333333", "--dry-run",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Would DELETE /api/namespaces/acme/webhooks/33333333-3333-3333-3333-333333333333 (skipped; --dry-run)"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("unexpected dry-run output: %s", out.String())
	}
}

func TestNamespaceWebhookCreate_IncludeDescendants(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/namespaces/acme/webhooks" {
			http.NotFound(w, r)
			return
		}
		var body struct {
			IncludeDescendants bool `json:"include_descendants"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !body.IncludeDescendants {
			t.Fatalf("expected include_descendants=true, got false")
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"id":                  "44444444-4444-4444-4444-444444444444",
			"namespace_id":        "ns1",
			"namespace_path":      "acme",
			"target_url":          "https://example.test/ns",
			"event_kinds":         []string{"issue.opened"},
			"include_descendants": true,
			"active":              true,
			"created_at":          "2026-05-07T00:00:00Z",
			"updated_at":          "2026-05-07T00:00:00Z",
			"secret_hint":         "abcd1234",
			"cleartext_secret":    "namespace-secret",
		})
	})

	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out,
		"webhook", "create", "acme",
		"--url", "https://example.test/ns",
		"--events", "issue.opened",
		"--include-descendants",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "namespace-secret") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
