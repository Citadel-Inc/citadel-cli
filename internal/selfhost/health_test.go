package selfhost_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/selfhost"
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
	// API server — returns 200 on /api/health.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
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
	report := selfhost.CheckHealth(context.Background(), cfg)
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
	report := selfhost.CheckHealth(context.Background(), cfg)
	if report.Overall() != selfhost.HealthRed {
		t.Errorf("Overall = %s; want RED when API is down", report.Overall())
	}
}

func TestHealthMigrationsMissing(t *testing.T) {
	// API server healthy.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
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
	report := selfhost.CheckHealth(context.Background(), cfg)
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
	report := selfhost.CheckHealth(context.Background(), cfg)
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
