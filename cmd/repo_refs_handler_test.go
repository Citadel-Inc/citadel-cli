package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestRepoBranchList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/acme/repos/demo/refs": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("kind"); got != "branch" {
				t.Fatalf("kind = %q, want branch", got)
			}
			writeJSON(t, w, 200, map[string]any{
				"refs": []map[string]any{
					{"name": "main", "sha": "abc123", "kind": "branch", "date": "2026-05-06T00:00:00Z"},
				},
			})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "branch", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoBranchList_BadOutput_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`, true, "branch", "list", "acme/demo")
}

func TestRepoBranchList_BadOutput_NoRepo_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`, true, "branch", "list")
}

func TestRepoBranchList_MissingRepo_Hermetic(t *testing.T) {
	assertRepoRefListMissingRepo(t, "branch", "list")
}

func TestRepoBranchList_AllJSON_Hermetic(t *testing.T) {
	setRepoRefHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "branch", "list", "acme/demo", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used with --output json") {
		t.Fatalf("want all/json validation error, got %v", err)
	}
}

func TestRepoBranchDelete_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/acme/repos/demo/refs/branches": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("name"); got != "feature/test" {
				t.Fatalf("name = %q, want feature/test", got)
			}
			writeJSON(t, w, 200, map[string]any{"status": "deleted", "kind": "branch", "name": "feature/test"})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "branch", "delete", "acme/demo", "feature/test").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoBranchDelete_BadOutput_NoRepo_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output for delete supports json or default human summary only; got "toml"`, true, "branch", "delete", "main")
}

func TestRepoBranchDelete_DefaultConflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/acme/repos/demo/refs/branches": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusConflict, map[string]any{"error": "default_branch"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "branch", "delete", "acme/demo", "main").Execute()
	if err == nil || !strings.Contains(err.Error(), "default branch") {
		t.Fatalf("want default-branch conflict, got %v", err)
	}
}

func TestRepoBranchDelete_DryRun(t *testing.T) {
	var sb strings.Builder
	if err := rootForOut(cmd.RepoCmd, &sb, "branch", "delete", "acme/demo", "main", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "Would DELETE") {
		t.Fatalf("want dry-run preview, got %q", sb.String())
	}
}

func TestRepoBranchSetDefault_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"PATCH /namespaces/acme/repos/demo/default-branch": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, 200, map[string]any{"default_branch": "release"})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "branch", "set-default", "acme/demo", "release").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoBranchSetDefault_BadOutput_NoRepo_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output for set-default supports json or default human summary only; got "toml"`, true, "branch", "set-default", "main")
}

func TestRepoBranchSetDefault_DryRun(t *testing.T) {
	var sb strings.Builder
	if err := rootForOut(cmd.RepoCmd, &sb, "branch", "set-default", "acme/demo", "release", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "Would PATCH") {
		t.Fatalf("want dry-run preview, got %q", sb.String())
	}
}

func TestRepoTagList_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"GET /namespaces/acme/repos/demo/refs": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("kind"); got != "tag" {
				t.Fatalf("kind = %q, want tag", got)
			}
			writeJSON(t, w, 200, map[string]any{
				"refs": []map[string]any{
					{"name": "v1.0.0", "sha": "abc123", "kind": "tag", "date": "2026-05-06T00:00:00Z"},
				},
			})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "tag", "list", "acme/demo").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoTagList_BadOutput_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`, true, "tag", "list", "acme/demo")
}

func TestRepoTagList_BadOutput_NoRepo_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output: unknown format "toml" (use json|yaml|ndjson|csv|table)`, true, "tag", "list", "--no-cwd-repo")
}

func TestRepoTagList_MissingRepo_Hermetic(t *testing.T) {
	assertRepoRefListMissingRepo(t, "tag", "list")
}

func TestRepoTagList_AllJSON_Hermetic(t *testing.T) {
	setRepoRefHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "tag", "list", "acme/demo", "--all", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "--all cannot be used with --output json") {
		t.Fatalf("want all/json validation error, got %v", err)
	}
}

func TestRepoTagCreate_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/acme/repos/demo/refs/tags": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{"name": "v1.0.0", "sha": "abc123", "annotated": false})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "tag", "create", "acme/demo", "v1.0.0", "--ref", "main").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoTagCreate_BadOutput_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output for create supports json or default human summary only; got "toml"`, true, "tag", "create", "acme/demo", "v1.0.0", "--ref", "main")
}

func TestRepoTagCreate_Conflict(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/acme/repos/demo/refs/tags": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusConflict, map[string]any{"error": "tag_exists"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "tag", "create", "acme/demo", "v1.0.0", "--ref", "main").Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("want tag-exists error, got %v", err)
	}
}

func assertRepoRefBadOutput(t *testing.T, want string, exact bool, args ...string) {
	t.Helper()
	setRepoRefHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, append(args, "--output", "toml")...).Execute()
	if err == nil {
		t.Fatalf("want output validation error, got %v", err)
	}
	if exact {
		if err.Error() != want {
			t.Fatalf("want output validation error %q, got %v", want, err)
		}
		return
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("want output validation error containing %q, got %v", want, err)
	}
}

func assertRepoRefListMissingRepo(t *testing.T, args ...string) {
	t.Helper()
	setRepoRefHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, append(args, "--no-cwd-repo")...).Execute()
	if err == nil || err.Error() != "repository required: pass -R <namespace>/<slug>, set CITADEL_REPO, or omit --no-cwd-repo to infer from git" {
		t.Fatalf("want repository path error, got %v", err)
	}
}

func setRepoRefHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func TestRepoTagDelete_Happy(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/acme/repos/demo/refs/tags": func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("name"); got != "v1.0.0" {
				t.Fatalf("name = %q, want v1.0.0", got)
			}
			writeJSON(t, w, 200, map[string]any{"status": "deleted", "kind": "tag", "name": "v1.0.0"})
		},
	}))
	if err := rootFor(cmd.RepoCmd, "tag", "delete", "acme/demo", "v1.0.0").Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestRepoTagDelete_BadOutput_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output for delete supports json or default human summary only; got "toml"`, true, "tag", "delete", "acme/demo", "v1.0.0")
}

func TestRepoTagDelete_BadOutput_NoRepo_Hermetic(t *testing.T) {
	assertRepoRefBadOutput(t, `--output for delete supports json or default human summary only; got "toml"`, true, "tag", "delete", "v1.0.0")
}

func TestRepoTagDelete_NotFound(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/acme/repos/demo/refs/tags": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{"error": "tag_not_found"})
		},
	}))
	err := rootFor(cmd.RepoCmd, "tag", "delete", "acme/demo", "missing").Execute()
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want tag-not-found error, got %v", err)
	}
}

func TestRepoTagDelete_DryRun(t *testing.T) {
	var sb strings.Builder
	if err := rootForOut(cmd.RepoCmd, &sb, "tag", "delete", "acme/demo", "v1.0.0", "--dry-run").Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "Would DELETE") {
		t.Fatalf("want dry-run preview, got %q", sb.String())
	}
}
