package cmd_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestIssueMilestoneList_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/milestones", "/namespaces/acme/demo/milestones") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"milestones": []map[string]any{
				{
					"id":          "11111111-1111-1111-1111-111111111111",
					"namespace_id": "ns1",
					"title":       "v1.0",
					"description": "first release",
					"state":       "open",
					"due_on":      "2026-06-01T00:00:00Z",
					"created_at":  "2026-05-07T00:00:00Z",
					"progress": map[string]any{
						"open_count":   2,
						"closed_count": 1,
						"total":        3,
						"percent":      33,
					},
				},
			},
		})
	})

	var out strings.Builder
	if err := rootForOut(cmd.IssueCmd, &out, "milestone", "list", "-R", "acme/demo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"title": "v1.0"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestIssueMilestoneView_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !issuePathMatches(r, "/namespaces/acme%2Fdemo/milestones/11111111-1111-1111-1111-111111111111", "/namespaces/acme/demo/milestones/11111111-1111-1111-1111-111111111111") {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"id":          "11111111-1111-1111-1111-111111111111",
			"namespace_id": "ns1",
			"title":       "v1.0",
			"description": "first release",
			"state":       "open",
			"created_at":  "2026-05-07T00:00:00Z",
			"progress": map[string]any{
				"open_count":   2,
				"closed_count": 1,
				"total":        3,
				"percent":      33,
			},
		})
	})
	if err := rootFor(cmd.IssueCmd, "milestone", "view", "-R", "acme/demo", "11111111-1111-1111-1111-111111111111").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueMilestoneView_NotFound(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
	})
	err := rootFor(cmd.IssueCmd, "milestone", "view", "-R", "acme/demo", "11111111-1111-1111-1111-111111111111").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

func TestIssueMilestoneCreate_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/namespaces/acme%2Fdemo/milestones", "/namespaces/acme/demo/milestones") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["title"] != "v1.0" || body["description"] != "first release" || body["due_on"] != "2026-06-01" {
			t.Fatalf("unexpected body: %#v", body)
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"id":          "11111111-1111-1111-1111-111111111111",
			"namespace_id": "ns1",
			"title":       "v1.0",
			"description": "first release",
			"state":       "open",
			"due_on":      "2026-06-01T00:00:00Z",
			"created_at":  "2026-05-07T00:00:00Z",
			"progress": map[string]any{
				"open_count":   0,
				"closed_count": 0,
				"total":        0,
				"percent":      0,
			},
		})
	})
	if err := rootFor(cmd.IssueCmd, "milestone", "create", "-R", "acme/demo", "--title", "v1.0", "--description", "first release", "--due-on", "2026-06-01").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueMilestoneEdit_Happy(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !issuePathMatches(r, "/namespaces/acme%2Fdemo/milestones/11111111-1111-1111-1111-111111111111", "/namespaces/acme/demo/milestones/11111111-1111-1111-1111-111111111111") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["state"] != "closed" || body["due_on"] != "" {
			t.Fatalf("unexpected body: %#v", body)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"id":          "11111111-1111-1111-1111-111111111111",
			"namespace_id": "ns1",
			"title":       "v1.0",
			"description": "first release",
			"state":       "closed",
			"created_at":  "2026-05-07T00:00:00Z",
			"closed_at":   "2026-05-08T00:00:00Z",
			"progress": map[string]any{
				"open_count":   0,
				"closed_count": 2,
				"total":        2,
				"percent":      100,
			},
		})
	})
	if err := rootFor(cmd.IssueCmd, "milestone", "edit", "-R", "acme/demo", "11111111-1111-1111-1111-111111111111", "--state", "closed", "--due-on=").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestIssueMilestoneDelete_JSON(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !issuePathMatches(r, "/namespaces/acme%2Fdemo/milestones/11111111-1111-1111-1111-111111111111", "/namespaces/acme/demo/milestones/11111111-1111-1111-1111-111111111111") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	var out strings.Builder
	if err := rootForOut(cmd.IssueCmd, &out, "milestone", "delete", "-R", "acme/demo", "11111111-1111-1111-1111-111111111111", "--yes", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"status": "ok"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestIssueCreate_WithMilestonePayload(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !issuePathMatches(r, "/namespaces/acme%2Fdemo/issues", "/namespaces/acme/demo/issues") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["milestone_id"]; got != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("milestone_id=%v", got)
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
			"milestone_id":   "11111111-1111-1111-1111-111111111111",
			"created_at":     "2026-05-06T00:00:00Z",
			"updated_at":     "2026-05-06T00:00:00Z",
		})
	})
	if err := rootFor(cmd.IssueCmd, "create", "-R", "acme/demo", "--title", "Ship it", "--body", "hello", "--milestone", "11111111-1111-1111-1111-111111111111").Execute(); err != nil {
		t.Fatal(err)
	}
}
