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

func TestFetchRepoBranchNames_DecodesRefsPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/namespaces/acme/repos/demo/refs" || r.URL.Query().Get("kind") != "branch" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"refs": []any{
				map[string]any{"name": "release"},
				map[string]any{"name": "main"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchRepoBranchNames(context.Background(), c, "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "main" || got[1] != "release" {
		t.Fatalf("got %q want sorted [main release]", got)
	}
}

func TestFetchRepoTagNames_DecodesRefsPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/namespaces/acme/repos/demo/refs" || r.URL.Query().Get("kind") != "tag" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"refs": []any{
				map[string]any{"name": "v2.0.0"},
				map[string]any{"name": "v1.0.0"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c, err := apiclient.New(clicfg.Config{AccessToken: "t"}, apiclient.Options{Server: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FetchRepoTagNames(context.Background(), c, "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "v1.0.0" || got[1] != "v2.0.0" {
		t.Fatalf("got %q want sorted tag names", got)
	}
}
