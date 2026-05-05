package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

// Exercise completion helpers the same way the shell does: real httptest
// server returning Citadel-shaped JSON, plus cobra flag state on commands
// attached to a root (so PersistentFlags resolve).

func testServerTokenEnv(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "test-token")
}

func repoGetFromRoot(t *testing.T, root *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range root.Commands() {
		if c.Name() != "repo" {
			continue
		}
		for _, sc := range c.Commands() {
			if sc.Name() == "get" {
				return sc
			}
		}
	}
	t.Fatal("repo get not found")
	return nil
}

func TestCompleteOutputFormats_GoldenList(t *testing.T) {
	got, d := completeOutputFormats(nil, nil, "")
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
	if len(got) != 5 {
		t.Fatalf("want 5 formats, got %d: %v", len(got), got)
	}
}

func TestCompleteAgentNames_HappyPath(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agents" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"agents": []any{
			map[string]any{"name": "zeus", "id": "1"},
			map[string]any{"name": "apollo", "id": "2"},
		}})
	})
	root := NewRootCmd()
	agentGet := findAgentGet(t, root)
	agentGet.SetContext(context.Background())
	vals, d := completeAgentNames(agentGet, []string{}, "")
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
	if len(vals) != 2 || vals[0] != "apollo" || vals[1] != "zeus" {
		t.Fatalf("got %q", vals)
	}
}

func findAgentGet(t *testing.T, root *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range root.Commands() {
		if c.Name() != "agent" {
			continue
		}
		for _, sc := range c.Commands() {
			if sc.Name() == "get" {
				return sc
			}
		}
	}
	t.Fatal("agent get not found")
	return nil
}

func TestCompleteOrgNamespaceSlugs_HappyPath(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"orgs": []any{map[string]any{"slug": "zeta", "namespace_id": "1", "created_at": "2026-01-01T00:00:00Z"}}})
	})
	root := NewRootCmd()
	nsGet := findNsGet(t, root)
	nsGet.SetContext(context.Background())
	vals, d := completeOrgNamespaceSlugs(nsGet, []string{}, "")
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
	if len(vals) != 1 || vals[0] != "zeta" {
		t.Fatalf("got %q", vals)
	}
}

func findNsGet(t *testing.T, root *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range root.Commands() {
		if c.Name() != "namespace" && c.Name() != "ns" {
			continue
		}
		for _, sc := range c.Commands() {
			if sc.Name() == "get" {
				return sc
			}
		}
	}
	t.Fatal("namespace get not found")
	return nil
}

func TestCompleteOAuthClientIDs_HappyPath(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/clients" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"clients": []any{
			map[string]any{"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "name": "A"},
		}})
	})
	root := NewRootCmd()
	show := findOauthShow(t, root)
	show.SetContext(context.Background())
	vals, d := completeOAuthClientIDs(show, []string{}, "")
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
	if len(vals) != 1 {
		t.Fatalf("got %q", vals)
	}
}

func findOauthShow(t *testing.T, root *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range root.Commands() {
		if c.Name() != "oauth" {
			continue
		}
		for _, oc := range c.Commands() {
			if oc.Name() != "clients" {
				continue
			}
			for _, sc := range oc.Commands() {
				if sc.Name() == "show" {
					return sc
				}
			}
		}
	}
	t.Fatal("oauth clients show not found")
	return nil
}

func TestCompleteTokenIDs_HappyPath(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent-tokens" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}})
	})
	root := NewRootCmd()
	revoke := findTokenRevoke(t, root)
	revoke.SetContext(context.Background())
	vals, d := completeTokenIDs(revoke, []string{}, "")
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
	if len(vals) != 1 {
		t.Fatalf("got %q", vals)
	}
}

func findTokenRevoke(t *testing.T, root *cobra.Command) *cobra.Command {
	t.Helper()
	for _, c := range root.Commands() {
		if c.Name() != "token" {
			continue
		}
		for _, sc := range c.Commands() {
			if sc.Name() == "revoke" {
				return sc
			}
		}
	}
	t.Fatal("token revoke not found")
	return nil
}

func TestResolveRepoNamespaceForCompletion_FromRFlag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := NewRootCmd()
	get := repoGetFromRoot(t, root)
	if err := get.Flags().Set("repo", "acme/myrepo"); err != nil {
		t.Fatal(err)
	}
	ns, err := ResolveRepoNamespaceForCompletion(get)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "acme" {
		t.Fatalf("got %q", ns)
	}
}

func TestCompleteRepoSlugs_HappyPath(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/namespaces/myorg/repos" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"repos": []any{
				map[string]any{"slug": "zebra"},
				map[string]any{"slug": "alpha"},
			},
		})
	})
	root := NewRootCmd()
	get := repoGetFromRoot(t, root)
	if err := get.Flags().Set("repo", "myorg/existing"); err != nil {
		t.Fatal(err)
	}
	get.SetContext(context.Background())
	vals, d := completeRepoSlugs(get, []string{}, "")
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
	if len(vals) != 2 || vals[0] != "alpha" || vals[1] != "zebra" {
		t.Fatalf("got %q", vals)
	}
}

func TestCompleteRepoSlugs_PositionalAlreadyPresent_NoCompletion(t *testing.T) {
	root := NewRootCmd()
	get := repoGetFromRoot(t, root)
	get.SetContext(context.Background())
	vals, d := completeRepoSlugs(get, []string{"already"}, "")
	if len(vals) != 0 {
		t.Fatalf("got %v", vals)
	}
	if d != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive %v", d)
	}
}
