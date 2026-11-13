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
// The config holds no database password, so the Postgres connection comes
// from the CLI's linked project or the operator's SUPABASE_DB_URL.
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
	// cmd.Env stays nil so the CLI inherits PATH, HOME and any SUPABASE_DB_URL.
	cmd := exec.CommandContext(ctx, bin, "db", "push", "--linked")

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

// countApplied counts the number of migration lines in supabase CLI output.
// The supabase CLI prints "Applying migration YYYYMMDDHHMMSS_name.sql" per
// applied migration.  Returns 0 when parsing fails.
func countApplied(output string) int {
	n := 0
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "Applying migration") {
			n++
		}
	}
	return n
}
