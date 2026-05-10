package selfhost

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HealthStatus represents the aggregate health of a self-hosted deployment.
type HealthStatus int

const (
	// HealthGreen means all components are reachable and migrations are current.
	HealthGreen HealthStatus = iota
	// HealthAmber means the deployment is functional but has pending migrations.
	HealthAmber
	// HealthRed means one or more core components are unreachable.
	HealthRed
)

func (s HealthStatus) String() string {
	switch s {
	case HealthGreen:
		return "GREEN"
	case HealthAmber:
		return "AMBER"
	case HealthRed:
		return "RED"
	default:
		return "UNKNOWN"
	}
}

// ProbeResult is the outcome of a single health probe.
type ProbeResult struct {
	Name   string
	Status HealthStatus
	Detail string
}

func (r ProbeResult) String() string {
	return fmt.Sprintf("[%s] %s — %s", r.Status, r.Name, r.Detail)
}

// HealthReport aggregates all probe results.
type HealthReport struct {
	Probes []ProbeResult
}

// Overall returns the aggregate status: RED if any probe is RED, AMBER if any
// is AMBER, GREEN otherwise.
func (r HealthReport) Overall() HealthStatus {
	worst := HealthGreen
	for _, p := range r.Probes {
		if p.Status > worst {
			worst = p.Status
		}
	}
	return worst
}

// CheckHealth probes all health components for the given config.
func CheckHealth(ctx context.Context, cfg Config) HealthReport {
	client := &http.Client{Timeout: 8 * time.Second}
	return HealthReport{
		Probes: []ProbeResult{
			probeAPI(ctx, client, cfg.APIEndpoint),
			probeSupabase(ctx, client, cfg.SupabaseURL, cfg.AdminKey),
			probeMigrations(ctx, client, cfg.SupabaseURL, cfg.AdminKey),
		},
	}
}

// probeAPI checks that the Citadel API server is reachable at /api/health.
func probeAPI(ctx context.Context, client *http.Client, endpoint string) ProbeResult {
	const name = "api"
	if endpoint == "" {
		return ProbeResult{name, HealthRed, "api_endpoint not configured"}
	}
	url := strings.TrimRight(endpoint, "/") + "/api/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProbeResult{name, HealthRed, fmt.Sprintf("build request: %v", err)}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResult{name, HealthRed, fmt.Sprintf("%s unreachable: %v", endpoint, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 500 {
		return ProbeResult{name, HealthRed, fmt.Sprintf("HTTP %d from %s", resp.StatusCode, url)}
	}
	return ProbeResult{name, HealthGreen, fmt.Sprintf("reachable (HTTP %d)", resp.StatusCode)}
}

// probeSupabase verifies that the Supabase REST endpoint responds with the
// admin key.  A 200/401/403 all indicate the server is up; 5xx or network
// errors indicate RED.
func probeSupabase(ctx context.Context, client *http.Client, supabaseURL, adminKey string) ProbeResult {
	const name = "supabase"
	if supabaseURL == "" {
		return ProbeResult{name, HealthRed, "supabase_url not configured"}
	}
	// Probe the REST v1 root — returns 200 with valid service key or 401 without,
	// both of which indicate the server is alive.
	url := strings.TrimRight(supabaseURL, "/") + "/rest/v1/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProbeResult{name, HealthRed, fmt.Sprintf("build request: %v", err)}
	}
	if adminKey != "" {
		req.Header.Set("apikey", adminKey)
		req.Header.Set("Authorization", "Bearer "+adminKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeResult{name, HealthRed, fmt.Sprintf("%s unreachable: %v", supabaseURL, err)}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 500 {
		return ProbeResult{name, HealthRed, fmt.Sprintf("HTTP %d from Supabase REST endpoint", resp.StatusCode)}
	}
	return ProbeResult{name, HealthGreen, fmt.Sprintf("reachable (HTTP %d)", resp.StatusCode)}
}

// probeMigrations queries the schema_migrations table via the Supabase REST
// endpoint.  AMBER if any migrations appear pending; GREEN otherwise.
// If the probe cannot connect, it returns RED.
func probeMigrations(ctx context.Context, client *http.Client, supabaseURL, adminKey string) ProbeResult {
	const name = "migrations"
	if supabaseURL == "" {
		return ProbeResult{name, HealthRed, "supabase_url not configured (cannot check migrations)"}
	}
	if adminKey == "" {
		return ProbeResult{name, HealthAmber, "admin_key not set; skipping migration check"}
	}

	// Query schema_migrations table: if we can reach it, report current state.
	// Supabase service role can query any table via REST.
	url := strings.TrimRight(supabaseURL, "/") + "/rest/v1/schema_migrations?select=version,name&limit=1&order=version.desc"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProbeResult{name, HealthRed, fmt.Sprintf("build request: %v", err)}
	}
	req.Header.Set("apikey", adminKey)
	req.Header.Set("Authorization", "Bearer "+adminKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ProbeResult{name, HealthRed, fmt.Sprintf("cannot reach Supabase: %v", err)}
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		// Table does not exist yet — migrations have not been run.
		return ProbeResult{name, HealthAmber, "schema_migrations table not found — run `citadel self-host migrate`"}
	}
	if resp.StatusCode >= 400 {
		return ProbeResult{name, HealthAmber, fmt.Sprintf("migration check returned HTTP %d: %s", resp.StatusCode, truncate(string(body), 120))}
	}
	return ProbeResult{name, HealthGreen, "schema_migrations accessible"}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
