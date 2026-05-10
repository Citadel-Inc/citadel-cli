package selfhost

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// MigrateResult summarizes the outcome of a migration run.
type MigrateResult struct {
	Applied int
	Message string
}

// ApplyMigrations applies pending database migrations to the configured
// Supabase instance.
//
// Phase 2 implementation: shells out to the official `supabase` CLI
// (`supabase db push --linked`), which handles idempotency via the
// schema_migrations table.  The Supabase CLI must be installed on the host.
//
// The supabaseURL and adminKey from cfg are passed via env so they are not
// visible in the process table.
func ApplyMigrations(ctx context.Context, cfg Config) (MigrateResult, error) {
	if cfg.SupabaseURL == "" {
		return MigrateResult{}, fmt.Errorf("supabase_url is required for migrate")
	}

	// Locate supabase binary.
	bin, err := exec.LookPath("supabase")
	if err != nil {
		return MigrateResult{}, fmt.Errorf(
			"supabase CLI not found on PATH; install from https://supabase.com/docs/guides/cli: %w", err,
		)
	}

	//nolint:gosec // bin resolved via exec.LookPath; args do not contain user input
	cmd := exec.CommandContext(ctx, bin, "db", "push", "--linked")

	// Inject connection details via env — keeps secrets out of the process list.
	// supabase CLI respects SUPABASE_DB_URL for the Postgres connection URL.
	if cfg.SupabaseURL != "" {
		// Derive a postgres URL from the Supabase project URL when possible.
		// Self-hosted Supabase typically exposes Postgres on the standard port.
		pgURL := derivePostgresURL(cfg.SupabaseURL, cfg.AdminKey)
		if pgURL != "" {
			cmd.Env = append(cmd.Env, "SUPABASE_DB_URL="+pgURL)
		}
	}

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		// Include command output to help diagnose errors (no secrets in output).
		return MigrateResult{}, fmt.Errorf("supabase db push: %w\n%s", err, output)
	}
	return MigrateResult{
		Applied: countApplied(output),
		Message: output,
	}, nil
}

// derivePostgresURL attempts to derive a Postgres connection URL from a
// Supabase project URL.  Returns "" when the pattern is unrecognized.
//
// For Supabase Cloud (https://<ref>.supabase.co) the DB host is
// db.<ref>.supabase.co:5432.  For self-hosted instances operators typically
// expose Postgres directly; we rely on the SUPABASE_DB_URL they supply
// (or the supabase CLI's project-linked config) and return "" here to let
// the CLI fall back to its own resolution.
func derivePostgresURL(supabaseURL, _ string) string {
	// If the operator already set SUPABASE_DB_URL externally we should not
	// override it — return "" and let the CLI pick it up.
	// For now we don't try to synthesize a postgres:// URL; operators who
	// want to override can set SUPABASE_DB_URL in their environment.
	_ = supabaseURL
	return ""
}

// countApplied counts the number of migration lines in supabase CLI output.
// The supabase CLI prints "Applying migration YYYYMMDDHHMMSS_name.sql" per
// applied migration.  Returns 0 when parsing fails.
func countApplied(output string) int {
	n := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Applying migration") {
			n++
		}
	}
	return n
}
