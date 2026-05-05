package completion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLookupSecondCallUsesCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("CITADEL_NO_COMPLETION_CACHE", "")

	var apiCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/orgs" {
			apiCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"orgs":[{"slug":"z","namespace_id":"1","created_at":"2026-01-01T00:00:00Z"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", srv.URL)
	t.Setenv("CITADEL_ACCESS_TOKEN", "tok")

	start := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	origNow := now
	now = func() time.Time { return start }
	t.Cleanup(func() { now = origNow })

	ctx := context.Background()
	if _, err := Lookup(ctx, "", KeyOrgs, FetchOrgNamespaceSlugs); err != nil {
		t.Fatal(err)
	}
	if apiCalls != 1 {
		t.Fatalf("first lookup: want 1 API call, got %d", apiCalls)
	}

	if _, err := Lookup(ctx, "", KeyOrgs, FetchOrgNamespaceSlugs); err != nil {
		t.Fatal(err)
	}
	if apiCalls != 1 {
		t.Fatalf("second lookup: want cache hit (still 1 API call), got %d", apiCalls)
	}

	now = func() time.Time { return start.Add(61 * time.Second) }
	if _, err := Lookup(ctx, "", KeyOrgs, FetchOrgNamespaceSlugs); err != nil {
		t.Fatal(err)
	}
	if apiCalls != 2 {
		t.Fatalf("after TTL: want 2 API calls, got %d", apiCalls)
	}
}
