package cmd_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func issuePathMatches(r *http.Request, encoded, decoded string) bool {
	return r.URL.EscapedPath() == encoded || r.URL.Path == decoded
}

func TestIssueList_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues", "/namespaces/acme/demo/issues") {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("state"); got != "open" {
			t.Fatalf("state=%q want open", got)
		}
		writeJSON(t, w, 200, map[string]any{
			"issues": []map[string]any{
				{
					"id":             "00000000-0000-0000-0000-000000000001",
					"namespace_id":   "00000000-0000-0000-0000-000000000002",
					"namespace_path": "acme/demo",
					"number":         7,
					"title":          "Ship it",
					"body_markdown":  "body",
					"state":          "open",
					"author_id":      "00000000-0000-0000-0000-000000000003",
					"created_at":     "2026-05-06T00:00:00Z",
					"updated_at":     "2026-05-06T00:00:00Z",
				},
			},
			"next_cursor": "",
		})
	})
	if err := rootFor(cmd.IssueCmd, "list", "-R", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueList_InvalidState(t *testing.T) {
	err := rootFor(cmd.IssueCmd, "list", "-R", "acme/demo", "--state", "weird").Execute()
	if err == nil || !strings.Contains(err.Error(), "--state") {
		t.Fatalf("want invalid-state error, got %v", err)
	}
}

func TestIssueList_NoAuth(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "http://nope")
	err := rootFor(cmd.IssueCmd, "list", "-R", "acme/demo").Execute()
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("want not-authenticated, got %v", err)
	}
}

func TestIssueView_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7", "/namespaces/acme/demo/issues/7") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, 200, map[string]any{
			"issue": map[string]any{
				"id":             "00000000-0000-0000-0000-000000000001",
				"namespace_id":   "00000000-0000-0000-0000-000000000002",
				"namespace_path": "acme/demo",
				"number":         7,
				"title":          "Ship it",
				"body_markdown":  "body",
				"state":          "open",
				"author_id":      "00000000-0000-0000-0000-000000000003",
				"created_at":     "2026-05-06T00:00:00Z",
				"updated_at":     "2026-05-06T00:00:00Z",
			},
			"comments": []map[string]any{},
			"labels":   []map[string]any{},
		})
	})
	if err := rootFor(cmd.IssueCmd, "view", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueView_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	})
	err := rootFor(cmd.IssueCmd, "view", "-R", "acme/demo", "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestIssueCreate_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues", "/namespaces/acme/demo/issues") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["title"]; got != "Ship it" {
			t.Fatalf("title=%v", got)
		}
		if got := body["body_markdown"]; got != "hello" {
			t.Fatalf("body_markdown=%v", got)
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"id":             "00000000-0000-0000-0000-000000000001",
			"namespace_id":   "00000000-0000-0000-0000-000000000002",
			"namespace_path": "acme/demo",
			"number":         7,
			"title":          "Ship it",
			"body_markdown":  "hello",
			"state":          "open",
			"author_id":      "00000000-0000-0000-0000-000000000003",
			"created_at":     "2026-05-06T00:00:00Z",
			"updated_at":     "2026-05-06T00:00:00Z",
		})
	})
	if err := rootFor(cmd.IssueCmd, "create", "-R", "acme/demo", "--title", "Ship it", "--body", "hello").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCommentAdd_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/comments", "/namespaces/acme/demo/issues/7/comments") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["body_markdown"]; got != "looks good" {
			t.Fatalf("body_markdown=%v", got)
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"id":            "00000000-0000-0000-0000-000000000099",
			"issue_id":      "00000000-0000-0000-0000-000000000001",
			"author_id":     "00000000-0000-0000-0000-000000000003",
			"body_markdown": "looks good",
			"created_at":    "2026-05-06T00:00:00Z",
			"updated_at":    "2026-05-06T00:00:00Z",
			"edit_count":    0,
		})
	})
	if err := rootFor(cmd.IssueCmd, "comment", "add", "-R", "acme/demo", "7", "--body", "looks good").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCommentAdd_EmptyBody(t *testing.T) {
	err := rootFor(cmd.IssueCmd, "comment", "add", "-R", "acme/demo", "7", "--body", "").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("want empty-body error, got %v", err)
	}
}

func TestIssueEdit_TitleOnly(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7", "/namespaces/acme/demo/issues/7") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["title"]; got != "New title" {
			t.Fatalf("title=%v", got)
		}
		writeJSON(t, w, 200, map[string]any{
			"id":             "00000000-0000-0000-0000-000000000001",
			"namespace_id":   "00000000-0000-0000-0000-000000000002",
			"namespace_path": "acme/demo",
			"number":         7,
			"title":          "New title",
			"body_markdown":  "body",
			"state":          "open",
			"author_id":      "00000000-0000-0000-0000-000000000003",
			"created_at":     "2026-05-06T00:00:00Z",
			"updated_at":     "2026-05-06T00:00:00Z",
		})
	})
	if err := rootFor(cmd.IssueCmd, "edit", "-R", "acme/demo", "7", "--title", "New title").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueEdit_InvalidState(t *testing.T) {
	err := rootFor(cmd.IssueCmd, "edit", "-R", "acme/demo", "7", "--state", "weird").Execute()
	if err == nil || !strings.Contains(err.Error(), "--state") {
		t.Fatalf("want invalid-state error, got %v", err)
	}
}

func TestIssueEdit_NoFlags(t *testing.T) {
	err := rootFor(cmd.IssueCmd, "edit", "-R", "acme/demo", "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("want no-flags error, got %v", err)
	}
}

func TestIssueAssign_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/assignees", "/namespaces/acme/demo/issues/7/assignees") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		assignees, ok := body["assignees"].([]any)
		if !ok || len(assignees) != 1 || assignees[0] != "alice" {
			t.Fatalf("unexpected assignees: %#v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := rootFor(cmd.IssueCmd, "assign", "-R", "acme/demo", "7", "--set", "alice").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueAssign_Clear(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/assignees", "/namespaces/acme/demo/issues/7/assignees") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if assignees, ok := body["assignees"].([]any); !ok || len(assignees) != 0 {
			t.Fatalf("expected empty assignees, got: %#v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := rootFor(cmd.IssueCmd, "assign", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCommentList_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/comments", "/namespaces/acme/demo/issues/7/comments") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, 200, map[string]any{
			"comments": []map[string]any{
				{
					"id":            "00000000-0000-0000-0000-000000000099",
					"issue_id":      "00000000-0000-0000-0000-000000000001",
					"author_id":     "00000000-0000-0000-0000-000000000003",
					"body_markdown": "looks good",
					"created_at":    "2026-05-06T00:00:00Z",
					"updated_at":    "2026-05-06T00:00:00Z",
					"edit_count":    0,
				},
			},
		})
	})
	if err := rootFor(cmd.IssueCmd, "comment", "list", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCommentList_Empty(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/comments", "/namespaces/acme/demo/issues/7/comments") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, 200, map[string]any{"comments": []any{}})
	})
	if err := rootFor(cmd.IssueCmd, "comment", "list", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCommentEdit_Happy(t *testing.T) {
	const commentID = "00000000-0000-0000-0000-000000000099"
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/namespaces/acme%2Fdemo/issues/comments/" + commentID
		wantPathDec := "/namespaces/acme/demo/issues/comments/" + commentID
		if r.Method != http.MethodPatch || !issuePathMatches(r, wantPath, wantPathDec) {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["body_markdown"]; got != "updated text" {
			t.Fatalf("body_markdown=%v", got)
		}
		writeJSON(t, w, 200, map[string]any{
			"id":            commentID,
			"issue_id":      "00000000-0000-0000-0000-000000000001",
			"author_id":     "00000000-0000-0000-0000-000000000003",
			"body_markdown": "updated text",
			"created_at":    "2026-05-06T00:00:00Z",
			"updated_at":    "2026-05-06T01:00:00Z",
			"edit_count":    1,
		})
	})
	if err := rootFor(cmd.IssueCmd, "comment", "edit", "-R", "acme/demo", commentID, "--body", "updated text").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueCommentEdit_EmptyBody(t *testing.T) {
	err := rootFor(cmd.IssueCmd, "comment", "edit", "-R", "acme/demo", "some-id", "--body", "").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
		t.Fatalf("want empty-body error, got %v", err)
	}
}

func TestIssueClose_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7", "/namespaces/acme/demo/issues/7") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["state"]; got != "closed" {
			t.Fatalf("state=%v", got)
		}
		writeJSON(t, w, 200, map[string]any{"status": "ok"})
	})
	if err := rootFor(cmd.IssueCmd, "close", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueClose_Forbidden(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	})
	err := rootFor(cmd.IssueCmd, "close", "-R", "acme/demo", "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("want 403 error, got %v", err)
	}
}

func TestIssueReopen_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7", "/namespaces/acme/demo/issues/7") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["state"]; got != "open" {
			t.Fatalf("state=%v", got)
		}
		writeJSON(t, w, 200, map[string]any{"status": "ok"})
	})
	if err := rootFor(cmd.IssueCmd, "reopen", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueLabel_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/labels", "/namespaces/acme/demo/issues/7/labels") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		add := body["add"].([]any)
		remove := body["remove"].([]any)
		if len(add) != 1 || add[0] != "bug" || len(remove) != 1 || remove[0] != "triage" {
			t.Fatalf("unexpected label payload: %#v", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := rootFor(cmd.IssueCmd, "label", "-R", "acme/demo", "7", "--add", "bug", "--remove", "triage").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueLabel_Conflict(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"conflict"}`, http.StatusConflict)
	})
	err := rootFor(cmd.IssueCmd, "label", "-R", "acme/demo", "7", "--add", "bug").Execute()
	if err == nil || !strings.Contains(err.Error(), "409") {
		t.Fatalf("want 409 error, got %v", err)
	}
}

func TestIssueCloseRefs_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues/7/close-refs", "/namespaces/acme/demo/issues/7/close-refs") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, 200, map[string]any{
			"close_refs": []map[string]any{
				{
					"referenced_namespace_path": "acme/dep",
					"closing_commit_sha":        "abcdef1234567890",
					"resolved_at":               "2026-05-06T00:00:00Z",
				},
			},
		})
	})
	if err := rootFor(cmd.IssueCmd, "close-refs", "-R", "acme/demo", "7").Execute(); err != nil {
		t.Fatal(err)
	}
}
