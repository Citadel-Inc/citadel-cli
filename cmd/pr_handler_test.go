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

const (
	testNSPath = "myorg/myrepo"
	testPRNum  = int64(7)
	testPRUUID = "11111111-2222-3333-4444-555555555555"
)

func openPR() map[string]any {
	return map[string]any{
		"id":           testPRUUID,
		"namespace_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"number":       testPRNum,
		"title":        "My feature",
		"body_markdown": "## Description\n\nAdds a new thing.",
		"state":        "open",
		"source_ref":   "feature/my-thing",
		"target_ref":   "main",
		"head_sha":     "abcdef1234567890",
		"base_sha":     "fedcba9876543210",
		"author_id":    "ffffffff-ffff-ffff-ffff-ffffffffffff",
		"created_at":   time.Now().UTC().Format(time.RFC3339),
		"updated_at":   time.Now().UTC().Format(time.RFC3339),
	}
}

func makePRsResp(prs []map[string]any) map[string]any {
	return map[string]any{"pull_requests": prs, "next_cursor": ""}
}

func prPath(suffix ...string) string {
	base := "/namespaces/myorg/myrepo/pulls"
	if len(suffix) == 0 {
		return base
	}
	return base + "/" + strings.Join(suffix, "/")
}

// ── pr list ───────────────────────────────────────────────────────────────────

func TestPRList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath(): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, 200, makePRsResp([]map[string]any{openPR()}))
		},
	}))
	if err := rootFor(cmd.PrCmd, "list", "-R", testNSPath).Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestPRList_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath(): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makePRsResp([]map[string]any{openPR()}))
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "list", "-R", testNSPath, "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("output not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 PR, got %d", len(rows))
	}
}

func TestPRList_Empty(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath(): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makePRsResp(nil))
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "list", "-R", testNSPath).Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No pull requests") {
		t.Fatalf("expected empty message, got: %s", buf.String())
	}
}

func TestPRList_StateFilter(t *testing.T) {
	var gotState string
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath(): func(w http.ResponseWriter, r *http.Request) {
			gotState = r.URL.Query().Get("state")
			writeJSON(t, w, 200, makePRsResp(nil))
		},
	}))
	_ = rootFor(cmd.PrCmd, "list", "-R", testNSPath, "--state", "closed").Execute()
	if gotState != "closed" {
		t.Fatalf("want state=closed, got %q", gotState)
	}
}

func TestPRList_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath(): func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"not_found"}`, 404)
		},
	}))
	err := rootFor(cmd.PrCmd, "list", "-R", testNSPath).Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

// ── pr view ───────────────────────────────────────────────────────────────────

func TestPRView_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"pull_request": openPR(),
				"reviewers":    []any{},
			})
		},
	}))
	if err := rootFor(cmd.PrCmd, "view", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestPRView_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"pull_request": openPR(),
				"reviewers":    []any{},
			})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "view", "-R", testNSPath, "7", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output not valid JSON: %v\nbody: %s", err, buf.String())
	}
	pr, _ := out["pull_request"].(map[string]any)
	if pr["number"].(float64) != float64(testPRNum) {
		t.Fatalf("want number=%d, got %v", testPRNum, pr["number"])
	}
}

func TestPRView_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7"): func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"not_found"}`, 404)
		},
	}))
	err := rootFor(cmd.PrCmd, "view", "-R", testNSPath, "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

// ── pr create ─────────────────────────────────────────────────────────────────

func TestPRCreate_Happy(t *testing.T) {
	var gotBody map[string]any
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath(): func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				http.Error(w, "bad body", 400)
				return
			}
			writeJSON(t, w, 201, openPR())
		},
	}))
	if err := rootFor(cmd.PrCmd, "create", "-R", testNSPath,
		"--title", "My feature",
		"--source", "feature/my-thing",
		"--target", "main",
		"--body", "desc").Execute(); err != nil {
		t.Fatal(err)
	}
	if gotBody["title"] != "My feature" {
		t.Fatalf("want title='My feature', got %v", gotBody["title"])
	}
	if gotBody["source_ref"] != "feature/my-thing" {
		t.Fatalf("want source_ref='feature/my-thing', got %v", gotBody["source_ref"])
	}
}

func TestPRCreate_InvalidRefs(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath(): func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_refs"}`))
		},
	}))
	err := rootFor(cmd.PrCmd, "create", "-R", testNSPath,
		"--title", "X", "--source", "nosuchbranch", "--target", "main",
		"--body", "b").Execute()
	if err == nil || !strings.Contains(err.Error(), "refs could not be resolved") {
		t.Fatalf("want invalid_refs message, got %v", err)
	}
}

// ── pr close ──────────────────────────────────────────────────────────────────

func TestPRClose_Happy(t *testing.T) {
	deleted := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE " + prPath("7"): func(w http.ResponseWriter, _ *http.Request) {
			deleted = true
			writeJSON(t, w, 200, openPR())
		},
	}))
	if err := rootFor(cmd.PrCmd, "close", "-R", testNSPath, "7", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected DELETE to be called")
	}
}

func TestPRClose_InvalidState(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE " + prPath("7"): func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"error":"invalid_state"}`))
		},
	}))
	err := rootFor(cmd.PrCmd, "close", "-R", testNSPath, "7", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not in a state") {
		t.Fatalf("want invalid_state message, got %v", err)
	}
}

// ── pr merge ──────────────────────────────────────────────────────────────────

func TestPRMerge_Happy(t *testing.T) {
	merged := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "merge"): func(w http.ResponseWriter, _ *http.Request) {
			merged = true
			writeJSON(t, w, 200, openPR())
		},
	}))
	if err := rootFor(cmd.PrCmd, "merge", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
	if !merged {
		t.Fatal("expected POST /merge to be called")
	}
}

func TestPRMerge_AlreadyMerged(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "merge"): func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"error":"already_merged"}`))
		},
	}))
	err := rootFor(cmd.PrCmd, "merge", "-R", testNSPath, "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "already merged") {
		t.Fatalf("want already_merged message, got %v", err)
	}
}

func TestPRMerge_Conflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "merge"): func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(409)
			_, _ = w.Write([]byte(`{"error":"merge_conflict"}`))
		},
	}))
	err := rootFor(cmd.PrCmd, "merge", "-R", testNSPath, "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "merge conflict") {
		t.Fatalf("want merge_conflict message, got %v", err)
	}
}

func TestPRMerge_ApprovalRequired(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "merge"): func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(422)
			_, _ = w.Write([]byte(`{"error":"approval_required"}`))
		},
	}))
	err := rootFor(cmd.PrCmd, "merge", "-R", testNSPath, "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "approval") {
		t.Fatalf("want approval_required message, got %v", err)
	}
}

// ── pr diff ───────────────────────────────────────────────────────────────────

func TestPRDiff_StatTable(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "diff"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"files": []map[string]any{
					{"path": "foo.go", "status": "modified", "additions": 4, "deletions": 1},
					{"path": "bar.go", "status": "added", "additions": 22, "deletions": 0},
				},
				"base_ref": "main",
				"head_ref": "feature/my-thing",
				"base_sha": "fedcba9876543210",
				"head_sha": "abcdef1234567890",
			})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "diff", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "foo.go") {
		t.Fatalf("expected file stat in output, got: %s", buf.String())
	}
}

func TestPRDiff_FileUnified(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "diff", "file"): func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("path") != "foo.go" {
				http.NotFound(w, r)
				return
			}
			writeJSON(t, w, 200, map[string]any{
				"path":      "foo.go",
				"status":    "modified",
				"additions": 1,
				"deletions": 0,
				"unified":   "--- a/foo.go\n+++ b/foo.go\n@@ -1 +1,2 @@\n+added line\n",
			})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "diff", "-R", testNSPath, "7", "--file", "foo.go").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "--- a/foo.go") {
		t.Fatalf("expected unified diff in output, got: %s", buf.String())
	}
}

// ── pr check ──────────────────────────────────────────────────────────────────

func TestPRCheck_Mergeable(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "mergeability"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"mergeable": true, "reason": "clean"})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "check", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "MERGEABLE") {
		t.Fatalf("expected MERGEABLE in output, got: %s", buf.String())
	}
}

func TestPRCheck_NotMergeable(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "mergeability"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"mergeable": false, "reason": "conflict"})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "check", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "NOT MERGEABLE") {
		t.Fatalf("expected NOT MERGEABLE in output, got: %s", buf.String())
	}
}

func TestPRCheck_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "mergeability"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"mergeable": true, "reason": "fast_forward"})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "check", "-R", testNSPath, "7", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("output not valid JSON: %v\nbody: %s", err, buf.String())
	}
	if out["mergeable"] != true {
		t.Fatalf("want mergeable=true, got %v", out["mergeable"])
	}
}

// ── pr comment ────────────────────────────────────────────────────────────────

func TestPRCommentList_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "comments"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"comments": []map[string]any{
					{
						"id":            "cc111111-0000-0000-0000-000000000000",
						"pr_id":         testPRUUID,
						"author_id":     "ffffffff-ffff-ffff-ffff-ffffffffffff",
						"body_markdown": "Looks good!",
						"created_at":    time.Now().UTC().Format(time.RFC3339),
						"updated_at":    time.Now().UTC().Format(time.RFC3339),
					},
				},
			})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "comment", "list", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Looks good!") {
		t.Fatalf("expected comment in output, got: %s", buf.String())
	}
}

func TestPRCommentAdd_Happy(t *testing.T) {
	added := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "comments"): func(w http.ResponseWriter, r *http.Request) {
			added = true
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			writeJSON(t, w, 201, map[string]any{
				"id":            "cc111111-0000-0000-0000-000000000000",
				"pr_id":         testPRUUID,
				"author_id":     "ffffffff-ffff-ffff-ffff-ffffffffffff",
				"body_markdown": b["body_markdown"],
				"created_at":    time.Now().UTC().Format(time.RFC3339),
				"updated_at":    time.Now().UTC().Format(time.RFC3339),
			})
		},
	}))
	if err := rootFor(cmd.PrCmd, "comment", "add", "-R", testNSPath, "7", "--body", "LGTM").Execute(); err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected POST /comments to be called")
	}
}

// ── pr reviewer ───────────────────────────────────────────────────────────────

func TestPRReviewerList_Happy(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET " + prPath("7", "reviewers"): func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{
				"reviewers": []map[string]any{
					{
						"user_id":    "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
						"status":     "approved",
						"updated_at": time.Now().UTC().Format(time.RFC3339),
					},
				},
			})
		},
	}))
	if err := rootForOut(cmd.PrCmd, &buf, "reviewer", "list", "-R", testNSPath, "7").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "approved") {
		t.Fatalf("expected reviewer status in output, got: %s", buf.String())
	}
}

func TestPRReviewerAdd_Happy(t *testing.T) {
	added := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "reviewers"): func(w http.ResponseWriter, r *http.Request) {
			added = true
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["user_id"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
				http.Error(w, "wrong user_id", 400)
				return
			}
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.PrCmd, "reviewer", "add", "-R", testNSPath, "7",
		"--reviewer", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee").Execute(); err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected POST /reviewers to be called")
	}
}

func TestPRReviewerAdd_InvalidUUID(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "reviewers"): func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_user_id"}`))
		},
	}))
	err := rootFor(cmd.PrCmd, "reviewer", "add", "-R", testNSPath, "7",
		"--reviewer", "not-a-uuid").Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid reviewer UUID") {
		t.Fatalf("want invalid_user_id message, got %v", err)
	}
}

// ── pr review ────────────────────────────────────────────────────────────────

func TestPRReview_Approve(t *testing.T) {
	reviewed := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT " + prPath("7", "reviews", "me"): func(w http.ResponseWriter, r *http.Request) {
			reviewed = true
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["status"] != "approved" {
				http.Error(w, "want approved", 400)
				return
			}
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.PrCmd, "review", "-R", testNSPath, "7", "--approve").Execute(); err != nil {
		t.Fatal(err)
	}
	if !reviewed {
		t.Fatal("expected PUT /reviews/me to be called")
	}
}

func TestPRReview_RequestChanges(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PUT " + prPath("7", "reviews", "me"): func(w http.ResponseWriter, r *http.Request) {
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["status"] != "changes_requested" {
				http.Error(w, "want changes_requested", 400)
				return
			}
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.PrCmd, "review", "-R", testNSPath, "7", "--request-changes").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestPRReview_CommentOnly(t *testing.T) {
	commented := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "comments"): func(w http.ResponseWriter, _ *http.Request) {
			commented = true
			writeJSON(t, w, 201, map[string]any{
				"id": "00000000-0000-0000-0000-000000000001",
			})
		},
	}))
	if err := rootFor(cmd.PrCmd, "review", "-R", testNSPath, "7", "--comment", "looks good").Execute(); err != nil {
		t.Fatal(err)
	}
	if !commented {
		t.Fatal("expected POST /comments to be called for --comment only")
	}
}

func TestPRReview_CommentAndApprove(t *testing.T) {
	commented, reviewed := false, false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST " + prPath("7", "comments"): func(w http.ResponseWriter, _ *http.Request) {
			commented = true
			writeJSON(t, w, 201, map[string]any{"id": "00000000-0000-0000-0000-000000000002"})
		},
		"PUT " + prPath("7", "reviews", "me"): func(w http.ResponseWriter, _ *http.Request) {
			reviewed = true
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.PrCmd, "review", "-R", testNSPath, "7",
		"--approve", "--comment", "LGTM").Execute(); err != nil {
		t.Fatal(err)
	}
	if !commented || !reviewed {
		t.Fatalf("expected both comment and review: commented=%v reviewed=%v", commented, reviewed)
	}
}

func TestPRReview_NoFlags(t *testing.T) {
	err := rootFor(cmd.PrCmd, "review", "-R", testNSPath, "7").Execute()
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("want no-flags error, got %v", err)
	}
}

func TestPRReview_MutuallyExclusive(t *testing.T) {
	err := rootFor(cmd.PrCmd, "review", "-R", testNSPath, "7", "--approve", "--request-changes").Execute()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("want mutually-exclusive error, got %v", err)
	}
}
