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

const testMemberUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
const testOwnerUUID = "00000000-1111-2222-3333-444444444444"

func makeMembersResp(members []map[string]any) map[string]any {
	return map[string]any{"members": members}
}

func aliceMember() map[string]any {
	return map[string]any{
		"user_id":      testMemberUUID,
		"slug":         "alice",
		"display_name": "Alice Example",
		"email":        "alice@example.com",
		"is_owner":     false,
		"permissions":  []string{"code:read"},
		"joined_at":    time.Now().UTC().Format(time.RFC3339),
	}
}

func ownerMember() map[string]any {
	return map[string]any{
		"user_id":      testOwnerUUID,
		"slug":         "orgowner",
		"display_name": "Org Owner",
		"email":        "owner@example.com",
		"is_owner":     true,
		"permissions":  []string{"members:write"},
		"joined_at":    time.Now().UTC().Format(time.RFC3339),
	}
}

// ── org member list ───────────────────────────────────────────────────────────

func TestOrgMemberList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeMembersResp([]map[string]any{ownerMember(), aliceMember()}))
		},
	}))
	if err := rootFor(cmd.OrgCmd, "member", "list", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOrgMemberList_JSON(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeMembersResp([]map[string]any{ownerMember(), aliceMember()}))
		},
	}))
	if err := rootForOut(cmd.OrgCmd, &buf, "member", "list", "myorg", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("output not valid JSON array: %v\nbody: %s", err, buf.String())
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 members, got %d", len(rows))
	}
}

func TestOrgMemberList_Empty(t *testing.T) {
	var buf bytes.Buffer
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeMembersResp(nil))
		},
	}))
	if err := rootForOut(cmd.OrgCmd, &buf, "member", "list", "myorg").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No members") {
		t.Fatalf("expected empty message, got: %s", buf.String())
	}
}

func TestOrgMemberList_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/nosuchorg/members": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"not_found"}`, 404)
		},
	}))
	err := rootFor(cmd.OrgCmd, "member", "list", "nosuchorg").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error, got %v", err)
	}
}

// ── org member set-permissions ────────────────────────────────────────────────

func TestOrgMemberSetPermissions_ByUUID(t *testing.T) {
	var patchBody map[string]any
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
				http.Error(w, "bad body", 400)
				return
			}
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.OrgCmd, "member", "set-permissions", "myorg", testMemberUUID,
		"--permission", "code:read,code:write").Execute(); err != nil {
		t.Fatal(err)
	}
	perms, _ := patchBody["permissions"].([]any)
	if len(perms) != 2 {
		t.Fatalf("want 2 permissions, got %v", perms)
	}
}

func TestOrgMemberSetPermissions_BySlug(t *testing.T) {
	patched := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeMembersResp([]map[string]any{aliceMember()}))
		},
		"PATCH /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, _ *http.Request) {
			patched = true
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.OrgCmd, "member", "set-permissions", "myorg", "alice",
		"--permission", "members:read").Execute(); err != nil {
		t.Fatal(err)
	}
	if !patched {
		t.Fatal("expected PATCH to be called after slug resolution")
	}
}

func TestOrgMemberSetPermissions_ClearAll(t *testing.T) {
	var patchBody map[string]any
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&patchBody); err != nil {
				http.Error(w, "bad body", 400)
				return
			}
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.OrgCmd, "member", "set-permissions", "myorg", testMemberUUID).Execute(); err != nil {
		t.Fatal(err)
	}
	perms, _ := patchBody["permissions"].([]any)
	if len(perms) != 0 {
		t.Fatalf("want empty permissions (clear all), got %v", perms)
	}
}

func TestOrgMemberSetPermissions_CannotModifyOwner(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /orgs/myorg/members/" + testOwnerUUID: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":"cannot_modify_owner"}`))
		},
	}))
	err := rootFor(cmd.OrgCmd, "member", "set-permissions", "myorg", testOwnerUUID,
		"--permission", "code:read").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot change permissions for the org owner") {
		t.Fatalf("want cannot_modify_owner message, got %v", err)
	}
}

func TestOrgMemberSetPermissions_InvalidPermission(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_permission"}`))
		},
	}))
	err := rootFor(cmd.OrgCmd, "member", "set-permissions", "myorg", testMemberUUID,
		"--permission", "bogus:atom").Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown permission atom") {
		t.Fatalf("want invalid_permission message, got %v", err)
	}
}

// ── org member remove ─────────────────────────────────────────────────────────

func TestOrgMemberRemove_ByUUID(t *testing.T) {
	deleted := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, _ *http.Request) {
			deleted = true
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.OrgCmd, "member", "remove", "myorg", testMemberUUID, "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected DELETE to be called")
	}
}

func TestOrgMemberRemove_BySlug(t *testing.T) {
	deleted := false
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeMembersResp([]map[string]any{aliceMember()}))
		},
		"DELETE /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, _ *http.Request) {
			deleted = true
			w.WriteHeader(204)
		},
	}))
	if err := rootFor(cmd.OrgCmd, "member", "remove", "myorg", "alice", "--yes").Execute(); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected DELETE to be called after slug resolution")
	}
}

func TestOrgMemberRemove_CannotRemoveOwner(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /orgs/myorg/members/" + testOwnerUUID: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":"cannot_remove_owner"}`))
		},
	}))
	err := rootFor(cmd.OrgCmd, "member", "remove", "myorg", testOwnerUUID, "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot remove the org owner") {
		t.Fatalf("want cannot_remove_owner message, got %v", err)
	}
}

func TestOrgMemberRemove_SelfRemovalLockout(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /orgs/myorg/members/" + testMemberUUID: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			_, _ = w.Write([]byte(`{"error":"self_removal_lockout"}`))
		},
	}))
	err := rootFor(cmd.OrgCmd, "member", "remove", "myorg", testMemberUUID, "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "no other members:write holder") {
		t.Fatalf("want self_removal_lockout message, got %v", err)
	}
}

func TestOrgMemberRemove_SlugNotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /orgs/myorg/members": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, makeMembersResp([]map[string]any{ownerMember()}))
		},
	}))
	err := rootFor(cmd.OrgCmd, "member", "remove", "myorg", "nosuchslug", "--yes").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not-found error for unresolvable slug, got %v", err)
	}
}
