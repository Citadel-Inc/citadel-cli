package selfhost_test

// Integration tests for the self-host init→health→migrate→bootstrap-token chain.
//
// These tests are gated on the CITADEL_TEST_SELF_HOST_LIVE environment variable.
// When the variable is unset they skip immediately and are safe to run in CI.
//
// Two execution modes:
//
//  1. httptest mode (default when CITADEL_TEST_SELF_HOST_LIVE=1 but no real
//     instance is configured): stands up httptest servers for the Citadel API
//     and Supabase REST stubs, then exercises the full verb chain against them.
//
//  2. Live mode (when CITADEL_SELF_HOST_API and CITADEL_SELF_HOST_SUPABASE_URL
//     are also set): exercises the same chain against a real deployment.
//
// Environment variables:
//
//	CITADEL_TEST_SELF_HOST_LIVE=1   — enable the tests (required)
//	CITADEL_SELF_HOST_API           — real API endpoint (enables live mode)
//	CITADEL_SELF_HOST_SUPABASE_URL  — real Supabase URL (enables live mode)
//	CITADEL_SELF_HOST_ADMIN_KEY     — service-role key for live mode
//	CITADEL_SELF_HOST_JWT_SECRET    — JWT signing secret for live mode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/internal/selfhost"
)

// TestLiveSelfHost_chain_optIn exercises the init→health→bootstrap-token chain.
// When no real instance is configured it uses httptest stubs (always GREEN).
func TestLiveSelfHost_chain_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_SELF_HOST_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_SELF_HOST_LIVE=1 to run self-host integration tests")
	}

	apiURL, supabaseURL, adminKey, jwtSecret := resolveEndpoints(t)

	// ── Step 1: validate config round-trip ───────────────────────────────────
	t.Run("config_round_trip", func(t *testing.T) {
		cfgPath := writeTempConfig(t, apiURL, supabaseURL, adminKey, jwtSecret)
		t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)

		loaded, err := selfhost.Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if loaded.APIEndpoint != apiURL {
			t.Errorf("APIEndpoint: got %q, want %q", loaded.APIEndpoint, apiURL)
		}
		if loaded.SupabaseURL != supabaseURL {
			t.Errorf("SupabaseURL: got %q, want %q", loaded.SupabaseURL, supabaseURL)
		}
		if err := loaded.Validate(); err != nil {
			t.Errorf("Validate: %v", err)
		}
	})

	// ── Step 2: health check ─────────────────────────────────────────────────
	t.Run("health", func(t *testing.T) {
		cfg := selfhost.Config{
			APIEndpoint: apiURL,
			SupabaseURL: supabaseURL,
			AdminKey:    adminKey,
			JWTSecret:   jwtSecret,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		report := selfhost.CheckHealth(ctx, cfg)
		t.Logf("health report overall: %s", report.Overall())
		for _, p := range report.Probes {
			t.Logf("  probe: %s", p)
		}
		// In httptest mode, stubs always return GREEN.
		// In live mode, we accept GREEN or AMBER (pending migrations).
		if report.Overall() == selfhost.HealthRed {
			t.Errorf("health overall = RED; deployment unreachable")
		}
	})

	// ── Step 3: bootstrap token ───────────────────────────────────────────────
	t.Run("bootstrap_token", func(t *testing.T) {
		cfg := selfhost.Config{
			JWTSecret: jwtSecret,
		}
		token, err := selfhost.GenerateBootstrapToken(cfg, 24*time.Hour)
		if err != nil {
			t.Fatalf("GenerateBootstrapToken: %v", err)
		}
		if strings.TrimSpace(token) == "" {
			t.Fatal("GenerateBootstrapToken returned empty token")
		}
		// Validate the token round-trips.
		claims, err := selfhost.ValidateBootstrapToken(token, jwtSecret)
		if err != nil {
			t.Fatalf("ValidateBootstrapToken: %v", err)
		}
		if role, _ := claims["role"].(string); role != "service_role" {
			t.Errorf("claim role = %q; want service_role", role)
		}
		if sub, _ := claims["sub"].(string); sub != "citadel-bootstrap" {
			t.Errorf("claim sub = %q; want citadel-bootstrap", sub)
		}
		t.Logf("bootstrap token valid (role=%s, sub=%s)", claims["role"], claims["sub"])
	})

	// ── Step 4: token expiry boundary ────────────────────────────────────────
	t.Run("bootstrap_token_custom_duration", func(t *testing.T) {
		cfg := selfhost.Config{JWTSecret: jwtSecret}
		token, err := selfhost.GenerateBootstrapToken(cfg, 1*time.Hour)
		if err != nil {
			t.Fatalf("GenerateBootstrapToken (1h): %v", err)
		}
		claims, err := selfhost.ValidateBootstrapToken(token, jwtSecret)
		if err != nil {
			t.Fatalf("ValidateBootstrapToken: %v", err)
		}
		expRaw, ok := claims["exp"]
		if !ok {
			t.Fatal("missing exp claim")
		}
		var expUnix float64
		switch v := expRaw.(type) {
		case float64:
			expUnix = v
		case json.Number:
			expUnix, _ = v.Float64()
		default:
			t.Fatalf("unexpected exp type %T", expRaw)
		}
		expTime := time.Unix(int64(expUnix), 0)
		// Should expire in ~1h; allow 5-min slack.
		want := time.Now().Add(1*time.Hour - 5*time.Minute)
		if expTime.Before(want) {
			t.Errorf("exp = %v; want at least %v (1h from now minus 5min slack)", expTime, want)
		}
	})
}

// TestLiveSelfHost_health_amber_optIn exercises an AMBER state when the
// schema_migrations table is absent.
func TestLiveSelfHost_health_amber_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_SELF_HOST_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_SELF_HOST_LIVE=1 to run self-host integration tests")
	}
	// Only meaningful with httptest stubs; skip in live mode.
	if os.Getenv("CITADEL_SELF_HOST_API") != "" {
		t.Skip("skipping amber stub test in live mode (real instance has its own migration state)")
	}

	// API: healthy.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer apiSrv.Close()

	// Supabase: REST root OK, but schema_migrations 404 → AMBER.
	supSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer supSrv.Close()

	cfg := selfhost.Config{
		APIEndpoint: apiSrv.URL,
		SupabaseURL: supSrv.URL,
		AdminKey:    "test-key",
		JWTSecret:   "test-secret-32-bytes-padded-xxxx",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report := selfhost.CheckHealth(ctx, cfg)
	t.Logf("amber test overall: %s", report.Overall())
	for _, p := range report.Probes {
		t.Logf("  probe: %s", p)
	}
	if report.Overall() != selfhost.HealthAmber && report.Overall() != selfhost.HealthRed {
		t.Errorf("expected AMBER or RED with no schema_migrations table, got %s", report.Overall())
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// resolveEndpoints returns (apiURL, supabaseURL, adminKey, jwtSecret).
// If real env vars are set, returns those. Otherwise, starts httptest stubs.
func resolveEndpoints(t *testing.T) (apiURL, supabaseURL, adminKey, jwtSecret string) {
	t.Helper()

	apiURL = strings.TrimSpace(os.Getenv("CITADEL_SELF_HOST_API"))
	supabaseURL = strings.TrimSpace(os.Getenv("CITADEL_SELF_HOST_SUPABASE_URL"))
	adminKey = strings.TrimSpace(os.Getenv("CITADEL_SELF_HOST_ADMIN_KEY"))
	jwtSecret = strings.TrimSpace(os.Getenv("CITADEL_SELF_HOST_JWT_SECRET"))

	if apiURL != "" && supabaseURL != "" {
		// Live mode: use real credentials.
		if adminKey == "" {
			t.Log("CITADEL_SELF_HOST_ADMIN_KEY unset; some health probes may return AMBER")
		}
		if jwtSecret == "" {
			t.Fatal("CITADEL_SELF_HOST_JWT_SECRET required for bootstrap-token tests in live mode")
		}
		return
	}

	// httptest stub mode: stand up minimal servers.
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(apiSrv.Close)

	supSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/v1/":
			w.WriteHeader(http.StatusOK)
		default:
			// schema_migrations: return a row so health is GREEN.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"version":"20260101","name":"init"}]`))
		}
	}))
	t.Cleanup(supSrv.Close)

	return apiSrv.URL, supSrv.URL, "stub-admin-key", "stub-jwt-secret-32-bytes-padded-x"
}

// writeTempConfig writes a Config to a temp file and returns the path.
func writeTempConfig(t *testing.T, apiURL, supabaseURL, adminKey, jwtSecret string) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := dir + "/self-host.yaml"
	t.Setenv("CITADEL_SELF_HOST_CONFIG", cfgPath)
	cfg := selfhost.Config{
		APIEndpoint: apiURL,
		SupabaseURL: supabaseURL,
		AdminKey:    adminKey,
		JWTSecret:   jwtSecret,
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("writeTempConfig Save: %v", err)
	}
	return cfgPath
}
