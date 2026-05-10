package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func releaseTagBody() map[string]any {
	return map[string]any{
		"id":            "11111111-1111-1111-1111-111111111111",
		"namespace_id":  "ns1",
		"repo_id":       "repo1",
		"tag_name":      "v1.0.0",
		"name":          "v1.0.0",
		"body_markdown": "Initial GA",
		"draft":         false,
		"prerelease":    false,
		"author_id":     "user1",
		"published_at":  "2026-05-10T00:00:00Z",
		"created_at":    "2026-05-09T00:00:00Z",
		"updated_at":    "2026-05-10T00:00:00Z",
	}
}

func TestReleaseList_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/releases", "/namespaces/acme/demo/releases") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"releases": []map[string]any{releaseTagBody()},
			"cursor":   "",
		})
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "list", "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"tag_name": "v1.0.0"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestReleaseList_TableEmpty(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{"releases": []map[string]any{}})
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "list", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "No releases") {
		t.Fatalf("expected 'No releases' message, got: %s", out.String())
	}
}

func TestReleaseList_IncludeDrafts(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("include_drafts") != "true" {
			t.Errorf("missing include_drafts=true query")
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"releases": []map[string]any{}})
	})
	if err := rootFor(cmd.ReleaseCmd, "list", "-R", "acme/demo", "--include-drafts").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseLatest_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/releases/latest", "/namespaces/acme/demo/releases/latest") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, releaseTagBody())
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "latest", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "v1.0.0") {
		t.Fatalf("expected tag in output: %s", out.String())
	}
}

func TestReleaseLatest_None(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := rootFor(cmd.ReleaseCmd, "latest", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "no published releases") {
		t.Fatalf("want no-published error, got %v", err)
	}
}

func TestReleaseView_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/releases/v1.0.0", "/namespaces/acme/demo/releases/v1.0.0") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, releaseTagBody())
	})
	if err := rootFor(cmd.ReleaseCmd, "view", "v1.0.0", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseView_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := rootFor(cmd.ReleaseCmd, "view", "v9.9.9", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}

func TestReleaseCreate_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		if body["tag_name"] != "v1.0.0" {
			t.Errorf("tag_name = %v", body["tag_name"])
		}
		if body["name"] != "release-1" {
			t.Errorf("name = %v", body["name"])
		}
		writeJSON(t, w, http.StatusCreated, releaseTagBody())
	})
	if err := rootFor(cmd.ReleaseCmd, "create", "-R", "acme/demo", "--tag", "v1.0.0", "--name", "release-1").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseCreate_DuplicateTag(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})
	err := rootFor(cmd.ReleaseCmd, "create", "-R", "acme/demo", "--tag", "v1.0.0").Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestReleaseCreate_TagNotPushed(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	})
	err := rootFor(cmd.ReleaseCmd, "create", "-R", "acme/demo", "--tag", "v9.9.9").Execute()
	if err == nil || !strings.Contains(err.Error(), "does not exist on the remote") {
		t.Fatalf("want 422 error, got %v", err)
	}
}

func TestReleaseEdit_NoChange(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should not call server when nothing to update")
		w.WriteHeader(http.StatusOK)
	})
	err := rootFor(cmd.ReleaseCmd, "edit", "v1.0.0", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("want nothing-to-update error, got %v", err)
	}
}

func TestReleaseEdit_DraftFalse(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q", r.Method)
		}
		var body map[string]any
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		if v, ok := body["draft"].(bool); !ok || v != false {
			t.Errorf("draft = %v (%T)", body["draft"], body["draft"])
		}
		writeJSON(t, w, http.StatusOK, releaseTagBody())
	})
	if err := rootFor(cmd.ReleaseCmd, "edit", "v1.0.0", "-R", "acme/demo", "--draft=false").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDelete_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := rootFor(cmd.ReleaseCmd, "delete", "v1.0.0", "-R", "acme/demo", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseDelete_DryRun(t *testing.T) {
	withServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("dry-run must not call the server")
	})
	var out strings.Builder
	if err := rootForOut(cmd.ReleaseCmd, &out, "delete", "v1.0.0", "-R", "acme/demo", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Would DELETE") {
		t.Fatalf("expected dry-run preview, got: %s", out.String())
	}
}

func TestReleaseDelete_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err := rootFor(cmd.ReleaseCmd, "delete", "v9.9.9", "-R", "acme/demo", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found, got %v", err)
	}
}
