package cmd_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func setRepoHermeticEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_ACCESS_TOKEN", "")
	t.Setenv("CITADEL_SERVER", "")
	t.Setenv("CITADEL_REPO", "")
}

func TestRepoCreate_BadOutput_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "acme", "--slug", "repo", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), `--output for create supports json or default human summary only; got "toml"`) {
		t.Fatalf("want create output validation error, got %v", err)
	}
}

func TestRepoCreate_HumanOutput(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"POST /namespaces/acme/repos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"parent_slug": "acme",
				"slug":        "repo",
				"visibility":  "private",
			})
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.RepoCmd, &stdout,
		"create", "--namespace", "acme", "--slug", "repo",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Created acme/repo (private)\n" {
		t.Fatalf("stdout = %q, want %q", got, "Created acme/repo (private)\n")
	}
}

func TestRepoCreate_EmptyNamespace_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "", "--slug", "repo").Execute()
	if err == nil || !strings.Contains(err.Error(), "namespace cannot be empty") {
		t.Fatalf("want namespace validation error, got %v", err)
	}
}

func TestRepoCreate_EmptySlug_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "acme", "--slug", "").Execute()
	if err == nil || !strings.Contains(err.Error(), "slug cannot be empty") {
		t.Fatalf("want slug validation error, got %v", err)
	}
}

func TestRepoCreate_BadVisibility_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "create", "--namespace", "acme", "--slug", "repo", "--visibility", "internal").Execute()
	if err == nil || !strings.Contains(err.Error(), "visibility must be public or private") {
		t.Fatalf("want visibility validation error, got %v", err)
	}
}

func TestRepoList_BadOutput_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "list", "--namespace", "acme", "--output", "toml").Execute()
	if err == nil || !strings.Contains(err.Error(), "--output: unknown format") {
		t.Fatalf("want output validation error, got %v", err)
	}
}

func TestRepoList_WatchJSON_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "list", "--namespace", "acme", "--watch", "--output", "json").Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot be used with --watch") {
		t.Fatalf("want watch/output validation error, got %v", err)
	}
}

func TestRepoList_BadCursor_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "list", "--namespace", "acme", "--cursor", "not-base64!!!").Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid --cursor") {
		t.Fatalf("want cursor validation error, got %v", err)
	}
}

func TestRepoDelete_HumanOutput(t *testing.T) {
	withServer(t, route(t, map[string]http.HandlerFunc{
		"DELETE /namespaces/acme/repo": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))

	var stdout strings.Builder
	if err := rootForOut(cmd.RepoCmd, &stdout,
		"delete", "acme/repo", "--yes",
	).Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "Deleted acme/repo\n" {
		t.Fatalf("stdout = %q, want %q", got, "Deleted acme/repo\n")
	}
}

func TestRepoDelete_DryRunHermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	var out strings.Builder
	if err := rootForOut(cmd.RepoCmd, &out, "delete", "myorg/myrepo", "--dry-run").Execute(); err != nil {
		t.Fatalf("repo delete dry-run: %v", err)
	}
	if got := out.String(); got != "Would DELETE /namespaces/myorg/myrepo (skipped; --dry-run)\n" {
		t.Fatalf("dry-run output = %q, want %q", got, "Would DELETE /namespaces/myorg/myrepo (skipped; --dry-run)\n")
	}
}

func TestRepoDelete_BadOutput_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "delete", "acme/repo", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for delete supports json or default human summary only; got "toml"` {
		t.Fatalf("want delete output validation error, got %v", err)
	}
}

func TestRepoDelete_BadOutput_NoRepo_Hermetic(t *testing.T) {
	setRepoHermeticEnv(t)

	err := rootFor(cmd.RepoCmd, "delete", "--no-cwd-repo", "--output", "toml").Execute()
	if err == nil || err.Error() != `--output for delete supports json or default human summary only; got "toml"` {
		t.Fatalf("want delete output validation error without repo, got %v", err)
	}
}
