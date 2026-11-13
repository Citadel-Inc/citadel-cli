package selfhost_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Citadel-Inc/citadel-cli/internal/selfhost"
)

// buildCfg constructs a Config pointing at the given test servers.
func buildCfg(apiURL, supabaseURL string) selfhost.Config {
	return selfhost.Config{
		APIEndpoint: apiURL,
		SupabaseURL: supabaseURL,
		AdminKey:    "test-admin-key",
		JWTSecret:   "test-jwt-secret",
	}
}

func TestHealthAllGreen(t *testing.T) {
	// API server — returns 200 on /api/healthz.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiSrv.Close()

	// Supabase server — returns 200 on /rest/v1/ and /rest/v1/schema_migrations.
	supSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/v1/":
			w.WriteHeader(http.StatusOK)
		default:
			// schema_migrations query
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"version":"20260101","name":"init"}]`))
		}
	}))
	defer supSrv.Close()

	cfg := buildCfg(apiSrv.URL, supSrv.URL)
	report := selfhost.CheckHealth(t.Context(), cfg)
	if report.Overall() != selfhost.HealthGreen {
		for _, p := range report.Probes {
			t.Logf("probe: %s", p)
		}
		t.Errorf("Overall = %s; want GREEN", report.Overall())
	}
}

func TestHealthAPIDown(t *testing.T) {
	// Non-routable address to simulate unreachable API.
	cfg := buildCfg("http://127.0.0.1:1", "http://127.0.0.1:1")
	report := selfhost.CheckHealth(t.Context(), cfg)
	if report.Overall() != selfhost.HealthRed {
		t.Errorf("Overall = %s; want RED when API is down", report.Overall())
	}
}

func TestHealthMigrationsMissing(t *testing.T) {
	// API server healthy.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiSrv.Close()

	// Supabase: REST root OK, but schema_migrations returns 404.
	supSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// schema_migrations not found
		w.WriteHeader(http.StatusNotFound)
	}))
	defer supSrv.Close()

	cfg := buildCfg(apiSrv.URL, supSrv.URL)
	report := selfhost.CheckHealth(t.Context(), cfg)
	overall := report.Overall()
	if overall != selfhost.HealthAmber && overall != selfhost.HealthRed {
		t.Errorf("Overall = %s; want AMBER or RED when migrations missing", overall)
	}
}

func TestHealthAPIServerError(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiSrv.Close()

	supSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer supSrv.Close()

	cfg := buildCfg(apiSrv.URL, supSrv.URL)
	report := selfhost.CheckHealth(t.Context(), cfg)
	if report.Overall() != selfhost.HealthRed {
		t.Errorf("Overall = %s; want RED on API 500", report.Overall())
	}
}

func TestHealthStatusStrings(t *testing.T) {
	cases := []struct {
		s    selfhost.HealthStatus
		want string
	}{
		{selfhost.HealthGreen, "GREEN"},
		{selfhost.HealthAmber, "AMBER"},
		{selfhost.HealthRed, "RED"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("HealthStatus(%d).String() = %q; want %q", c.s, got, c.want)
		}
	}
}

// The Compose air-gap stack runs no PostgREST; its web origin answers every
// unknown path with the app, which must not read as a healthy Supabase.
func TestHealthWebPageInPlaceOfPostgREST(t *testing.T) {
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/healthz" {
			_, _ = w.Write([]byte("ok"))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html>"))
	}))
	defer web.Close()
	report := selfhost.CheckHealth(t.Context(), buildCfg(web.URL, web.URL))
	if report.Overall() != selfhost.HealthAmber {
		t.Fatalf("Overall = %v, want AMBER: %+v", report.Overall(), report.Probes)
	}
}

func TestHealthKeepsAdminKeyOffRedirectsToAnotherOrigin(t *testing.T) {
	var leaked []string
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("Apikey") + r.Header.Get("Authorization"); v != "" {
			leaked = append(leaked, r.URL.Path+": "+v)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer other.Close()
	sup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+r.URL.RequestURI(), http.StatusFound)
	}))
	defer sup.Close()

	selfhost.CheckHealth(t.Context(), buildCfg(sup.URL, sup.URL))
	if len(leaked) != 0 {
		t.Fatalf("admin key sent to another origin: %v", leaked)
	}
}
