package cmd_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestRepoDeployTokenListJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/myorg/myrepo/deploy-tokens": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("limit"); got == "" {
				t.Fatalf("expected limit query")
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"deploy_tokens": []map[string]any{
					{
						"id":             "tok-1",
						"namespace_path": "myorg/myrepo",
						"name":           "ci",
						"created_at":     "2026-05-07T01:00:00Z",
						"scopes":         []string{},
					},
				},
			})
		},
	}))

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "deploy-token", "list", "-R", "myorg/myrepo", "--output", "json").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"id": "tok-1"`) {
		t.Fatalf("expected token json, got %s", out.String())
	}
}

func TestRepoDeployTokenCreateJSON(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/myorg/myrepo/deploy-tokens": func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Name             string `json:"name"`
				ExpiresInSeconds *int64 `json:"expires_in_seconds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body.Name != "ci" {
				t.Fatalf("name = %q", body.Name)
			}
			if body.ExpiresInSeconds == nil || *body.ExpiresInSeconds != 3600 {
				t.Fatalf("expires_in_seconds = %v", body.ExpiresInSeconds)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"id":              "tok-2",
				"namespace_path":  "myorg/myrepo",
				"name":            body.Name,
				"created_at":      "2026-05-07T01:00:00Z",
				"scopes":          []string{},
				"cleartext_token": "secret-token",
			})
		},
	}))

	var out strings.Builder
	root := rootForOut(cmd.RepoCmd, &out, "deploy-token", "create", "-R", "myorg/myrepo", "--name", "ci", "--expires", "1h", "--output", "json")
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"cleartext_token": "secret-token"`) {
		t.Fatalf("expected cleartext token json, got %s", out.String())
	}
}

func TestRepoDeployTokenRevokeDryRun(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "deploy-token", "revoke", "-R", "myorg/myrepo", "tok-3", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Would DELETE /namespaces/myorg%2Fmyrepo/deploy-tokens/tok-3") {
		t.Fatalf("unexpected dry-run output: %s", out.String())
	}
}

func TestNamespaceDeployTokenRevokeNotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/myorg/deploy-tokens/tok-404": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{"error": "not_found"})
		},
	}))

	err := rootFor(cmd.NamespaceCmd, "deploy-token", "revoke", "myorg", "tok-404").Execute()
	if err == nil || !strings.Contains(err.Error(), "deploy token tok-404 not found in myorg") {
		t.Fatalf("unexpected error: %v", err)
	}
}
