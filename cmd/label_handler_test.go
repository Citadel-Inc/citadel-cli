package cmd_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

// labelJSON is a complete label fixture.
func labelJSON(slug, name, color string, isDefault bool) map[string]any {
	return map[string]any{
		"id":           "00000000-0000-0000-0000-000000000001",
		"namespace_id": "00000000-0000-0000-0000-000000000002",
		"slug":         slug,
		"display_name": name,
		"color":        color,
		"description":  "a test label",
		"is_default":   isDefault,
	}
}

func labelsPayload(labels ...map[string]any) map[string]any {
	return map[string]any{"labels": labels}
}

// ── list ─────────────────────────────────────────────────────────────────────

func TestLabelList_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/labels", "/namespaces/acme/demo/labels") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, labelsPayload(labelJSON("bug", "Bug", "#d73a4a", true)))
	})
	if err := rootFor(cmd.LabelCmd, "list", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLabelList_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/labels", "/namespaces/acme/demo/labels") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, labelsPayload(labelJSON("bug", "Bug", "#d73a4a", true)))
	})
	var out strings.Builder
	if err := rootForOut(cmd.LabelCmd, &out, "list", "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"slug": "bug"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestLabelList_Empty(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, labelsPayload())
	})
	if err := rootFor(cmd.LabelCmd, "list", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

// ── create ───────────────────────────────────────────────────────────────────

func TestLabelCreate_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/namespaces/acme%2Fdemo/labels", "/namespaces/acme/demo/labels") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["slug"] != "good-first-issue" {
			t.Fatalf("unexpected slug: %v", body["slug"])
		}
		if body["display_name"] != "Good First Issue" {
			t.Fatalf("unexpected display_name: %v", body["display_name"])
		}
		if body["color"] != "#a2eeef" {
			t.Fatalf("unexpected color: %v", body["color"])
		}
		writeJSON(t, w, http.StatusCreated, labelJSON("good-first-issue", "Good First Issue", "#a2eeef", false))
	})
	if err := rootFor(cmd.LabelCmd, "create", "-R", "acme/demo",
		"--name", "Good First Issue", "--color", "a2eeef").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLabelCreate_ExplicitSlug(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["slug"] != "gfi" {
			t.Fatalf("unexpected slug: %v", body["slug"])
		}
		writeJSON(t, w, http.StatusCreated, labelJSON("gfi", "Good First Issue", "#a2eeef", false))
	})
	if err := rootFor(cmd.LabelCmd, "create", "-R", "acme/demo",
		"--name", "Good First Issue", "--color", "a2eeef", "--slug", "gfi").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLabelCreate_Conflict(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"label_already_exists"}`, http.StatusConflict)
	})
	err := rootFor(cmd.LabelCmd, "create", "-R", "acme/demo",
		"--name", "Bug", "--color", "d73a4a").Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want already-exists error, got %v", err)
	}
}

func TestLabelCreate_BadColor(t *testing.T) {
	err := rootFor(cmd.LabelCmd, "create", "-R", "acme/demo",
		"--name", "Bug", "--color", "not-hex").Execute()
	if err == nil || !strings.Contains(err.Error(), "--color") {
		t.Fatalf("want color-format error, got %v", err)
	}
}

// ── edit ─────────────────────────────────────────────────────────────────────

func TestLabelEdit_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && issuePathMatches(r, "/namespaces/acme%2Fdemo/labels", "/namespaces/acme/demo/labels"):
			writeJSON(t, w, http.StatusOK, labelsPayload(labelJSON("bug", "Bug", "#d73a4a", true)))
		case r.Method == http.MethodPatch && issuePathMatches(r, "/namespaces/acme%2Fdemo/labels/bug", "/namespaces/acme/demo/labels/bug"):
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["color"] != "#ff0000" {
				t.Fatalf("unexpected color in patch: %v", body["color"])
			}
			// display_name and description must be preserved from existing label
			if body["display_name"] != "Bug" {
				t.Fatalf("display_name not preserved: %v", body["display_name"])
			}
			writeJSON(t, w, http.StatusOK, labelJSON("bug", "Bug", "#ff0000", true))
		default:
			http.NotFound(w, r)
		}
	})
	if err := rootFor(cmd.LabelCmd, "edit", "-R", "acme/demo", "bug", "--color", "ff0000").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLabelEdit_NoFlags(t *testing.T) {
	err := rootFor(cmd.LabelCmd, "edit", "-R", "acme/demo", "bug").Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("want at-least-one-flag error, got %v", err)
	}
}

func TestLabelEdit_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(t, w, http.StatusOK, labelsPayload())
			return
		}
		http.NotFound(w, r)
	})
	err := rootFor(cmd.LabelCmd, "edit", "-R", "acme/demo", "missing", "--name", "X").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

// ── delete ───────────────────────────────────────────────────────────────────

func TestLabelDelete_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !issuePathMatches(r, "/namespaces/acme%2Fdemo/labels/bug", "/namespaces/acme/demo/labels/bug") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := rootFor(cmd.LabelCmd, "delete", "-R", "acme/demo", "bug", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestLabelDelete_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	var out strings.Builder
	if err := rootForOut(cmd.LabelCmd, &out, "delete", "-R", "acme/demo", "bug", "--yes", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"status": "ok"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestLabelDelete_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"label_not_found"}`, http.StatusNotFound)
	})
	err := rootFor(cmd.LabelCmd, "delete", "-R", "acme/demo", "missing", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestLabelDelete_Blocked(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"label_delete_blocked"}`, http.StatusConflict)
	})
	err := rootFor(cmd.LabelCmd, "delete", "-R", "acme/demo", "bug", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "last default label") {
		t.Fatalf("want blocked error, got %v", err)
	}
}

func TestLabelDelete_DryRun(t *testing.T) {
	var out strings.Builder
	if err := rootForOut(cmd.LabelCmd, &out, "delete", "-R", "acme/demo", "bug", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Would DELETE") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}
