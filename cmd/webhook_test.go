package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
	"github.com/Rethunk-Tech/citadel-cli/internal/pagination"
)

const testWebhookID = "11111111-1111-1111-1111-111111111111"
const testDeliveryID = "55555555-5555-5555-5555-555555555555"

func webhookPayload() map[string]any {
	return map[string]any{
		"id":                  testWebhookID,
		"namespace_id":        "ns1",
		"namespace_path":      "acme/demo",
		"name":                "ci-hook",
		"target_url":          "https://example.test/hook",
		"event_kinds":         []string{"issue.opened"},
		"include_descendants": false,
		"active":              true,
		"created_at":          "2026-05-07T00:00:00Z",
		"updated_at":          "2026-05-07T00:00:00Z",
		"secret_hint":         "abcd1234",
	}
}

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

func TestRepoWebhookList_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "list", "acme/demo", "--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookList_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "list", "--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookList_MissingRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd, "webhook", "list", "--no-cwd-repo").Execute()
	const want = "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git"
	if err == nil || err.Error() != want {
		t.Fatalf("want exact namespace path error %q, got %v", want, err)
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

func TestRepoWebhookCreate_MissingURL_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd,
		"webhook", "create", "acme/demo",
		"--events", "issue.opened",
	).Execute()
	if err == nil || !strings.Contains(err.Error(), "--url is required") {
		t.Fatalf("want missing URL validation error, got %v", err)
	}
}

func TestRepoWebhookCreate_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "create", "acme/demo",
		"--url", "https://example.test/inbox",
		"--events", "issue.opened",
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookCreate_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "create",
		"--url", "https://example.test/inbox",
		"--events", "issue.opened",
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookEdit_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "edit", testWebhookID,
		"--name", "renamed",
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
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

func TestRepoWebhookGet_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "get", testWebhookID, "--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
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

func TestRepoWebhookEdit_URLAndRotateSecret(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks/"+testWebhookID,
			"/api/namespaces/acme/demo/webhooks/"+testWebhookID) {
			http.NotFound(w, r)
			return
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		var targetURL string
		if err := json.Unmarshal(body["target_url"], &targetURL); err != nil {
			t.Fatal(err)
		}
		var rotateSecret bool
		if err := json.Unmarshal(body["rotate_secret"], &rotateSecret); err != nil {
			t.Fatal(err)
		}
		if targetURL != "https://example.test/updated" || !rotateSecret {
			t.Fatalf("unexpected edit body: %+v", body)
		}
		for _, field := range []string{"name", "event_kinds", "active", "include_descendants"} {
			if _, ok := body[field]; ok {
				t.Fatalf("unset field %q was sent: %v", field, body)
			}
		}
		payload := webhookPayload()
		payload["target_url"] = targetURL
		payload["cleartext_secret"] = "rotated-secret"
		writeJSON(t, w, http.StatusOK, payload)
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out,
		"webhook", "edit", "-R", "acme/demo", testWebhookID,
		"--url", "https://example.test/updated",
		"--rotate-secret",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Updated webhook "+testWebhookID+" for acme/demo.") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "Secret (save now; shown once): rotated-secret") {
		t.Fatalf("missing rotated secret: %s", out.String())
	}
}

func TestNamespaceWebhookEdit_Flags(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/namespaces/acme/webhooks/"+testWebhookID {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Name               *string  `json:"name"`
			EventKinds         []string `json:"event_kinds"`
			IncludeDescendants *bool    `json:"include_descendants"`
			Active             *bool    `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Name == nil || *body.Name != "renamed" ||
			len(body.EventKinds) != 1 || body.EventKinds[0] != "issue.closed" ||
			body.IncludeDescendants == nil || !*body.IncludeDescendants ||
			body.Active == nil || *body.Active {
			t.Fatalf("unexpected namespace edit body: %+v", body)
		}
		payload := webhookPayload()
		payload["namespace_path"] = "acme"
		payload["name"] = "renamed"
		payload["event_kinds"] = []string{"issue.closed"}
		payload["include_descendants"] = true
		payload["active"] = false
		writeJSON(t, w, http.StatusOK, payload)
	})

	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out,
		"webhook", "edit", "acme", testWebhookID,
		"--name", "renamed",
		"--events", "issue.closed",
		"--active=false",
		"--include-descendants=true",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Updated webhook "+testWebhookID+" for acme.") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRepoWebhookEdit_RequiresChangingFlag(t *testing.T) {
	err := rootFor(cmd.RepoCmd, "webhook", "edit", "acme/demo", testWebhookID).Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one changing flag is required") {
		t.Fatalf("expected missing edit flag error, got %v", err)
	}
}

func TestRepoWebhookEdit_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "edit", "acme/demo", testWebhookID,
		"--name", "renamed",
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

// ── namespace webhook list ────────────────────────────────────────────────────

func TestNamespaceWebhookList_Human(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks",
			"/api/namespaces/acme/demo/webhooks") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"webhooks":    []map[string]any{webhookPayload()},
			"next_cursor": "",
		})
	})
	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out, "webhook", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), testWebhookID) {
		t.Fatalf("expected webhook ID in output, got: %s", out.String())
	}
}

func TestRepoWebhookList_CSV(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks",
			"/api/namespaces/acme/demo/webhooks") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"webhooks":    []map[string]any{webhookPayload()},
			"next_cursor": "",
		})
	})
	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "webhook", "list", "-R", "acme/demo", "--output", "csv").Execute(); err != nil {
		t.Fatal(err)
	}
	// CSV header + data row should be present.
	if !strings.Contains(out.String(), "id,name,namespace_path") {
		t.Fatalf("expected CSV header in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), testWebhookID) {
		t.Fatalf("expected webhook ID in CSV, got: %s", out.String())
	}
}

// ── webhook get (human output / emitWebhookHuman) ────────────────────────────

func TestRepoWebhookGet_Human(t *testing.T) {
	// fetchWebhookByID fetches the list then finds by ID.
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"webhooks":    []map[string]any{webhookPayload()},
			"next_cursor": "",
		})
	})
	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "webhook", "get",
		"acme/demo", testWebhookID).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), testWebhookID) {
		t.Fatalf("expected webhook ID in human output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "Target") {
		t.Fatalf("expected 'Target' label in human output, got: %s", out.String())
	}
}

func TestNamespaceWebhookGet_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"webhooks":    []map[string]any{webhookPayload()},
			"next_cursor": "",
		})
	})
	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out, "webhook", "get",
		"acme/demo", testWebhookID, "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), testWebhookID) {
		t.Fatalf("expected webhook ID in JSON output, got: %s", out.String())
	}
}

// ── webhook delete ────────────────────────────────────────────────────────────

func TestRepoWebhookDelete_DryRun(t *testing.T) {
	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "webhook", "delete",
		"acme/demo", testWebhookID, "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Would DELETE") {
		t.Fatalf("expected 'Would DELETE' in dry-run output, got: %s", out.String())
	}
}

func TestRepoWebhookDelete_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "delete", "acme/demo", testWebhookID,
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookDelete_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "delete", testWebhookID,
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookDelete_Forbidden(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":"forbidden"}`)
	})
	err := rootFor(cmd.RepoCmd, "webhook", "delete",
		"acme/demo", testWebhookID).Execute()
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("want forbidden error, got %v", err)
	}
}

func TestRepoWebhookDelete_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"not found"}`)
	})
	err := rootFor(cmd.RepoCmd, "webhook", "delete",
		"acme/demo", testWebhookID).Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestRepoWebhookDelete_Conflict(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":"limit reached"}`)
	})
	err := rootFor(cmd.RepoCmd, "webhook", "delete",
		"acme/demo", testWebhookID).Execute()
	if err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("want limit error, got %v", err)
	}
}

func TestRepoWebhookDelete_BadRequest(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad request"}`)
	})
	err := rootFor(cmd.RepoCmd, "webhook", "delete",
		"acme/demo", testWebhookID).Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("want invalid error, got %v", err)
	}
}

func webhookDeliveryPayload() map[string]any {
	return map[string]any{
		"id":                     testDeliveryID,
		"webhook_id":             testWebhookID,
		"webhook_name":           "ci-hook",
		"webhook_url":            "https://example.test/hook",
		"event_id":               "event-123",
		"event_kind":             "issue.opened",
		"webhook_namespace_path": "acme/demo",
		"source_namespace_path":  "acme/demo",
		"state":                  "failed",
		"attempt_count":          2,
		"last_attempt_at":        "2026-05-07T00:01:00Z",
		"delivered_at":           nil,
		"http_status":            500,
		"response_body":          "internal error",
		"response_headers":       map[string]any{"content-type": "text/plain"},
		"error_message":          "delivery failed",
		"created_at":             "2026-05-07T00:00:00Z",
		"payload":                map[string]any{"action": "opened"},
	}
}

func TestRepoWebhookDeliveriesList_WebhookFilter(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks/deliveries",
			"/api/namespaces/acme/demo/webhooks/deliveries") {
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		if query.Get("webhook_id") != testWebhookID || query.Get("state") != "failed" || query.Get("offset") != "1" {
			t.Fatalf("unexpected delivery filters: %v", query)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"deliveries":  []map[string]any{webhookDeliveryPayload()},
			"next_cursor": "",
		})
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out,
		"webhook", "deliveries", "list", "-R", "acme/demo",
		"--webhook-id", testWebhookID, "--state", "failed", "--offset", "1", "--output", "json",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"event_kind": "issue.opened"`) {
		t.Fatalf("unexpected delivery output: %s", out.String())
	}
}

func TestRepoWebhookDeliveriesList_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "list",
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookDeliveriesList_NegativeOffset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "list", "-R", "acme/demo",
		"--offset", "-1", "--output", "json",
	).Execute()
	if err == nil {
		t.Fatal("expected negative offset error")
	}
	if !strings.Contains(err.Error(), "--offset cannot be negative") {
		t.Fatalf("want negative offset error, got %v", err)
	}
}

func TestRepoWebhookDeliveriesList_NegativeLimit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "list", "-R", "acme/demo",
		"--limit", "-1", "--output", "json",
	).Execute()
	if err == nil {
		t.Fatal("expected negative limit error")
	}
	if !strings.Contains(err.Error(), "--limit must be between") {
		t.Fatalf("want negative limit error, got %v", err)
	}
}

func TestRepoWebhookList_NegativeLimit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := rootFor(cmd.RepoCmd,
		"webhook", "list", "-R", "acme/demo",
		"--limit", "-1", "--output", "json",
	).Execute()
	if err == nil {
		t.Fatal("expected negative limit error")
	}
	if !strings.Contains(err.Error(), "--limit must be between") {
		t.Fatalf("want negative limit error, got %v", err)
	}
}

func TestRepoWebhookDeliveriesList_AllOmitsOffsetOnSecondPage(t *testing.T) {
	next := pagination.EncodeDesc(time.Unix(100, 0).UTC(), uuid.MustParse(testDeliveryID))
	var pages int
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks/deliveries",
			"/api/namespaces/acme/demo/webhooks/deliveries") {
			http.NotFound(w, r)
			return
		}
		pages++
		query := r.URL.Query()
		switch pages {
		case 1:
			if query.Get("offset") != "5" {
				t.Fatalf("first page offset = %q, want 5", query.Get("offset"))
			}
			if query.Get("cursor") != "" {
				t.Fatalf("first page cursor = %q, want empty", query.Get("cursor"))
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"deliveries":  []map[string]any{webhookDeliveryPayload()},
				"next_cursor": next,
			})
		case 2:
			if _, ok := query["offset"]; ok {
				t.Fatalf("second page must omit offset, got %v", query)
			}
			if query.Get("cursor") != next {
				t.Fatalf("second page cursor = %q, want %q", query.Get("cursor"), next)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"deliveries":  []map[string]any{},
				"next_cursor": "",
			})
		default:
			t.Fatalf("unexpected page %d", pages)
		}
	})

	if err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "list", "-R", "acme/demo",
		"--offset", "5", "--all", "--output", "ndjson",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if pages != 2 {
		t.Fatalf("pages = %d, want 2", pages)
	}
}

func TestNamespaceWebhookDeliveriesGet_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/namespaces/acme/webhooks/deliveries/"+testDeliveryID {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"delivery": webhookDeliveryPayload()})
	})

	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out,
		"webhook", "deliveries", "get", "acme", testDeliveryID, "--output", "json",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "55555555-5555-5555-5555-555555555555"`) {
		t.Fatalf("unexpected delivery output: %s", out.String())
	}
	if !strings.Contains(out.String(), `"event_kind": "issue.opened"`) {
		t.Fatalf("missing event kind in delivery output: %s", out.String())
	}
}

func TestRepoWebhookDeliveriesGet_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks/deliveries/"+testDeliveryID,
			"/api/namespaces/acme/demo/webhooks/deliveries/"+testDeliveryID) {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"delivery": webhookDeliveryPayload()})
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out,
		"webhook", "deliveries", "get", "-R", "acme/demo", testDeliveryID, "--output", "json",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "55555555-5555-5555-5555-555555555555"`) ||
		!strings.Contains(out.String(), `"event_kind": "issue.opened"`) {
		t.Fatalf("unexpected delivery output: %s", out.String())
	}
}

func TestRepoWebhookDeliveriesGet_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "get", testDeliveryID,
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookDeliveriesRedeliver(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r,
			"/api/namespaces/acme%2Fdemo/webhooks/deliveries/"+testDeliveryID+"/redeliver",
			"/api/namespaces/acme/demo/webhooks/deliveries/"+testDeliveryID+"/redeliver") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"delivery": webhookDeliveryPayload(),
			"sent":     true,
			"state":    "queued",
		})
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out,
		"webhook", "deliveries", "redeliver", "-R", "acme/demo", testDeliveryID,
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Redelivered delivery "+testDeliveryID+" for acme/demo.") {
		t.Fatalf("unexpected redeliver output: %s", out.String())
	}
	if !strings.Contains(out.String(), "Event kind: issue.opened") {
		t.Fatalf("missing event kind in redeliver output: %s", out.String())
	}
}

func TestRepoWebhookDeliveriesRedeliver_BadOutput_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "redeliver", "-R", "acme/demo", testDeliveryID,
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestRepoWebhookDeliveriesRedeliver_BadOutput_NoRepo_Hermetic(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")

	err := rootFor(cmd.RepoCmd,
		"webhook", "deliveries", "redeliver", testDeliveryID,
		"--output", "toml",
	).Execute()
	const want = `--output: unknown format "toml" (use json|yaml|table)`
	if err == nil || err.Error() != want {
		t.Fatalf("want exact output validation error %q, got %v", want, err)
	}
}

func TestNamespaceWebhookDeliveriesRedeliver_DryRun(t *testing.T) {
	var out strings.Builder
	if err := rootForOut(cmd.NamespaceCmd, &out,
		"webhook", "deliveries", "redeliver", "acme", testDeliveryID, "--dry-run",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	want := "Would POST /api/namespaces/acme/webhooks/deliveries/" + testDeliveryID + "/redeliver (skipped; --dry-run)"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("unexpected dry-run output: %s", out.String())
	}
}

func TestNamespaceWebhookDeliveriesGet_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"not found"}`)
	})

	err := rootFor(cmd.NamespaceCmd, "webhook", "deliveries", "get", "acme", testDeliveryID).Execute()
	if err == nil || !strings.Contains(err.Error(), "webhook delivery not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestNamespaceWebhookDeliveriesGet_InvalidID(t *testing.T) {
	err := rootFor(cmd.NamespaceCmd, "webhook", "deliveries", "get", "acme", "not-a-uuid").Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid delivery id") {
		t.Fatalf("want invalid delivery ID error, got %v", err)
	}
}
