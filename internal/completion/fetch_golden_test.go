package completion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/clicfg"
)

// Golden API shapes from Citadel list endpoints (httptest, not hand-rolled mocks).
func TestFetchRepoSlugs_DecodesReposPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/namespaces/acme/repos" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"repos": []any{
				map[string]any{"slug": "z", "path": "acme/z"},
				map[string]any{"slug": "a", "path": "acme/a"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchRepoSlugs(context.Background(), c, "acme")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "z"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %q want sorted %q", got, want)
	}
}

func TestFetchRepoSlugs_RejectsEmptyNamespace(t *testing.T) {
	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: "http://unused"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = FetchRepoSlugs(context.Background(), c, "  ")
	if err == nil {
		t.Fatal("expected error for empty namespace")
	}
}

func TestFetchAgentNames_DecodesAgentArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/agents" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"agents": []any{
				map[string]any{"name": "beta", "id": "00000000-0000-0000-0000-000000000002"},
				map[string]any{"name": "alpha", "id": "00000000-0000-0000-0000-000000000001"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchAgentNames(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("got %q want sorted [alpha beta]", got)
	}
}

func TestFetchOAuthClientIDs_DecodesClientList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/oauth/clients" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"clients": []any{
				map[string]any{"id": "22222222-2222-2222-2222-222222222222", "name": "B"},
				map[string]any{"id": "11111111-1111-1111-1111-111111111111", "name": "A"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchOAuthClientIDs(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("got %q want sorted UUID strings", got)
	}
}

func TestFetchAgentTokenIDs_DecodesTokenList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/agent-tokens" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
			{"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchAgentTokenIDs(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("got %q want sorted ids", got)
	}
}

func TestFetchSSHKeyIDs_DecodesKeyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/account/ssh-keys" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []any{
				map[string]any{"id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"},
				map[string]any{"id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchSSHKeyIDs(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("got %q want sorted ids", got)
	}
}
