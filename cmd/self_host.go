package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/internal/selfhost"
)

// selfHostDebugFlag returns true when --debug is set on the self-host command group.
// When enabled, detailed diagnostic information is written to stderr via slog;
// user-facing output continues to go to stdout.
func selfHostDebugFlag(cmd *cobra.Command) bool {
	// InheritedFlags includes persistent flags from all ancestors.
	v, _ := cmd.InheritedFlags().GetBool("debug")
	if v {
		return true
	}
	// Also check local flags for the case where cmd IS the self-host group.
	lv, _ := cmd.Flags().GetBool("debug")
	return lv
}

// selfHostLogger returns a slog.Logger that writes to stderr when --debug is
// set, or to io.Discard otherwise.  Structured JSON format for machine
// parseability; no secrets are ever passed to this logger.
func selfHostLogger(cmd *cobra.Command) *slog.Logger {
	if selfHostDebugFlag(cmd) {
		return slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError + 10, // effectively discard
	}))
}

// selfHostDebugf writes a formatted debug line to stderr when --debug is set.
// Use for one-liner diagnostics that do not need structured key-value pairs.
func selfHostDebugf(cmd *cobra.Command, format string, args ...any) {
	if selfHostDebugFlag(cmd) {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[debug] "+format+"\n", args...)
	}
}

// SelfHostCmd is the top-level `citadel self-host` command group.
var SelfHostCmd = &cobra.Command{
	Use:   "self-host",
	Short: "Manage a self-hosted Citadel deployment",
	Long: `Commands for initializing, operating, and health-checking a
self-hosted Citadel installation.

Config is stored at ~/.citadel/self-host.yaml (override with
CITADEL_SELF_HOST_CONFIG env var).`,
}

// selfHostInitCmd — interactive wizard that writes the self-host config.
var selfHostInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize self-host configuration (interactive wizard)",
	Long: `Collect API endpoint, Supabase URL, admin key, and JWT secret, then
write them to ~/.citadel/self-host.yaml (or CITADEL_SELF_HOST_CONFIG).

Pass --batch to read all required values from flags without prompting.`,
	RunE: runSelfHostInit,
}

// selfHostHealthCmd — probe deployment health.
var selfHostHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of the self-hosted deployment",
	Long: `Probe the Citadel API, Supabase REST endpoint, and migration state.

Exit codes:
  0 — GREEN (all healthy)
  1 — AMBER (functional but migrations pending) or RED (component unreachable)`,
	RunE: runSelfHostHealth,
}

// selfHostMigrateCmd — apply pending migrations.
var selfHostMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending database migrations",
	Long: `Apply pending Supabase migrations idempotently via the supabase CLI.
The supabase binary must be on PATH.

Migrations are tracked in the schema_migrations table; already-applied
migrations are skipped automatically.`,
	RunE: runSelfHostMigrate,
}

// selfHostBootstrapTokenCmd — generate admin bootstrap JWT.
var selfHostBootstrapTokenCmd = &cobra.Command{
	Use:   "bootstrap-token",
	Short: "Generate a bootstrap admin JWT and print to stdout",
	Long: `Mint a Supabase-compatible service_role JWT using the jwt_secret from
~/.citadel/self-host.yaml.  The token is printed to stdout once; it is
never written to disk (per Q6).

Use --duration to override the default 7-day validity window.`,
	RunE: runSelfHostBootstrapToken,
}

// selfHostTelemetryCmd — telemetry enable/disable.
var selfHostTelemetryCmd = &cobra.Command{
	Use:   "telemetry <enable|disable>",
	Short: "Enable or disable anonymous usage telemetry",
	Long: `Flip the global telemetry flag in ~/.citadel/self-host.yaml.

Telemetry is disabled by default.  When enabled, anonymous usage data is
sent to Rethunk-Tech endpoints.  No secrets or personal data are included.`,
	Args:      cobra.ExactArgs(1),
	RunE:      runSelfHostTelemetry,
	ValidArgs: []string{"enable", "disable"},
}

// ─── init ────────────────────────────────────────────────────────────────────

func runSelfHostInit(cmd *cobra.Command, _ []string) error {
	batch, _ := cmd.Flags().GetBool("batch")
	log := selfHostLogger(cmd)

	// Load existing config as defaults so re-runs are non-destructive.
	cfg, loadErr := selfhost.Load()
	if loadErr != nil {
		log.Debug("failed to load existing config; starting fresh", "error", loadErr)
	} else {
		log.Debug("loaded existing config", "summary", cfg.DebugSummary())
	}

	apiEndpoint, _ := cmd.Flags().GetString("api-endpoint")
	supabaseURL, _ := cmd.Flags().GetString("supabase-url")
	adminKey, _ := cmd.Flags().GetString("admin-key")
	jwtSecret, _ := cmd.Flags().GetString("jwt-secret")

	var err error
	cfg.APIEndpoint, err = resolveField("API endpoint (e.g. https://citadel.example.com)",
		apiEndpoint, cfg.APIEndpoint, batch, false)
	if err != nil {
		return err
	}
	cfg.SupabaseURL, err = resolveField("Supabase URL (e.g. https://abc.supabase.co)",
		supabaseURL, cfg.SupabaseURL, batch, false)
	if err != nil {
		return err
	}
	cfg.AdminKey, err = resolveField("Supabase service-role admin key",
		adminKey, cfg.AdminKey, batch, true)
	if err != nil {
		return err
	}
	cfg.JWTSecret, err = resolveField("Supabase JWT secret (used to mint bootstrap tokens)",
		jwtSecret, cfg.JWTSecret, batch, true)
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		log.Debug("config validation failed", "error", err)
		return fmt.Errorf("config validation: %w", err)
	}
	log.Debug("config validated; saving", "summary", cfg.DebugSummary())
	if err := cfg.Save(); err != nil {
		log.Debug("save failed", "error", err)
		return fmt.Errorf("save self-host config: %w", err)
	}

	path, _ := selfhost.ConfigPath()
	log.Debug("config saved", "path", path)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Self-host config written to %s\n", path)
	return nil
}

// resolveField returns the effective value for a config field, prompting
// interactively if no flag/existing value is available and batch is false.
// If secret is true, the prompt hides input (no echo); for simplicity in
// this implementation we do not suppress terminal echo (Phase 2 hardening).
func resolveField(label, flagVal, existing string, batch, _ bool) (string, error) {
	if flagVal != "" {
		return strings.TrimSpace(flagVal), nil
	}
	if existing != "" {
		if !batch {
			_, _ = fmt.Fprintf(os.Stderr, "%s [%s]: ", label, maskSecret(existing))
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if v := strings.TrimSpace(scanner.Text()); v != "" {
				return v, nil
			}
		}
		return existing, nil
	}
	if batch {
		return "", fmt.Errorf("--batch set but %q not supplied via flag and no existing value", label)
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s: ", label)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	v := strings.TrimSpace(scanner.Text())
	if v == "" {
		return "", fmt.Errorf("%q is required", label)
	}
	return v, nil
}

// maskSecret returns a display-safe mask for an existing secret value.
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

// ─── health ──────────────────────────────────────────────────────────────────

func runSelfHostHealth(cmd *cobra.Command, _ []string) error {
	log := selfHostLogger(cmd)
	cfg, err := selfhost.Load()
	if err != nil {
		log.Debug("load self-host config failed", "error", err)
		return fmt.Errorf("load self-host config: %w", err)
	}
	log.Debug("loaded self-host config", "summary", cfg.DebugSummary())
	if err := cfg.Validate(); err != nil {
		log.Debug("config validation failed", "error", err)
		return fmt.Errorf("self-host config incomplete: %w\nRun `citadel self-host init` to configure", err)
	}

	selfHostDebugf(cmd, "probing api=%s supabase=%s", cfg.APIEndpoint, cfg.SupabaseURL)
	report := selfhost.CheckHealth(cmd.Context(), cfg)
	w := cmd.OutOrStdout()
	for _, p := range report.Probes {
		log.Debug("probe result", "name", p.Name, "status", p.Status.String(), "detail", p.Detail)
		_, _ = fmt.Fprintln(w, p.String())
	}
	overall := report.Overall()
	_, _ = fmt.Fprintf(w, "\nOverall: %s\n", overall)

	if overall != selfhost.HealthGreen {
		// Opaque message to stdout already written; detailed probe info was logged via --debug.
		return errors.New("health check did not pass (see above)")
	}
	return nil
}

// ─── migrate ─────────────────────────────────────────────────────────────────

func runSelfHostMigrate(cmd *cobra.Command, _ []string) error {
	log := selfHostLogger(cmd)
	cfg, err := selfhost.Load()
	if err != nil {
		log.Debug("load self-host config failed", "error", err)
		return fmt.Errorf("load self-host config: %w", err)
	}
	log.Debug("loaded self-host config", "summary", cfg.DebugSummary())
	if err := cfg.Validate(); err != nil {
		log.Debug("config validation failed", "error", err)
		return fmt.Errorf("self-host config incomplete: %w\nRun `citadel self-host init` to configure", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Applying migrations…")
	selfHostDebugf(cmd, "invoking supabase CLI for db push")
	result, err := selfhost.ApplyMigrations(cmd.Context(), cfg)
	if err != nil {
		log.Debug("migration apply failed", "error", err)
		return fmt.Errorf("migrate: %w", err)
	}
	log.Debug("migration apply complete", "applied", result.Applied)
	if result.Message != "" {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Message)
	}
	if result.Applied > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Applied %d migration(s).\n", result.Applied)
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No new migrations to apply.")
	}
	return nil
}

// ─── bootstrap-token ─────────────────────────────────────────────────────────

func runSelfHostBootstrapToken(cmd *cobra.Command, _ []string) error {
	log := selfHostLogger(cmd)
	cfg, err := selfhost.Load()
	if err != nil {
		log.Debug("load self-host config failed", "error", err)
		return fmt.Errorf("load self-host config: %w", err)
	}
	// JWT secret is required; other fields optional for this verb.
	if cfg.JWTSecret == "" {
		return errors.New("jwt_secret not configured; run `citadel self-host init` to set it")
	}
	// Log that we have a secret but NEVER log its value.
	log.Debug("jwt_secret present", "jwt_secret", "[REDACTED]")

	durationStr, _ := cmd.Flags().GetString("duration")
	var duration time.Duration
	if durationStr != "" {
		d, err := time.ParseDuration(durationStr)
		if err != nil {
			log.Debug("invalid --duration flag", "value", durationStr, "error", err)
			return fmt.Errorf("invalid --duration %q: %w", durationStr, err)
		}
		if d <= 0 {
			return errors.New("--duration must be positive")
		}
		duration = d
	}
	log.Debug("generating bootstrap token", "duration", duration)

	token, err := selfhost.GenerateBootstrapToken(cfg, duration)
	if err != nil {
		log.Debug("token generation failed", "error", err)
		return err
	}
	log.Debug("bootstrap token generated", "token", "[REDACTED]")

	// Q6: token to stdout only. Never log the token value.
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), token)
	return nil
}

// ─── telemetry ───────────────────────────────────────────────────────────────

func runSelfHostTelemetry(cmd *cobra.Command, args []string) error {
	log := selfHostLogger(cmd)
	subcmd := strings.ToLower(strings.TrimSpace(args[0]))
	switch subcmd {
	case "enable", "disable":
		// valid
	default:
		return fmt.Errorf("unknown telemetry action %q; expected 'enable' or 'disable'", subcmd)
	}
	log.Debug("telemetry action", "action", subcmd)

	cfg, err := selfhost.Load()
	if err != nil {
		log.Debug("load self-host config failed", "error", err)
		return fmt.Errorf("load self-host config: %w", err)
	}

	cfg.Telemetry = subcmd == "enable"
	if err := cfg.Save(); err != nil {
		log.Debug("save self-host config failed", "error", err)
		return fmt.Errorf("save self-host config: %w", err)
	}
	log.Debug("telemetry updated", "enabled", cfg.Telemetry)
	if cfg.Telemetry {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Telemetry enabled.")
	} else {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Telemetry disabled.")
	}
	return nil
}

// ─── wiring ──────────────────────────────────────────────────────────────────

func init() {
	SelfHostCmd.AddCommand(selfHostInitCmd)
	SelfHostCmd.AddCommand(selfHostHealthCmd)
	SelfHostCmd.AddCommand(selfHostMigrateCmd)
	SelfHostCmd.AddCommand(selfHostBootstrapTokenCmd)
	SelfHostCmd.AddCommand(selfHostTelemetryCmd)

	// Persistent --batch flag: suppresses all interactive prompts across subcommands.
	SelfHostCmd.PersistentFlags().Bool("batch", false, "Non-interactive mode; fail if required params are missing")

	// Persistent --debug flag: writes detailed diagnostics to stderr.
	// User-facing output remains on stdout; secrets are always redacted.
	SelfHostCmd.PersistentFlags().Bool("debug", false, "Write detailed diagnostic output to stderr (secrets redacted)")

	// init flags
	selfHostInitCmd.Flags().String("api-endpoint", "", "Citadel API endpoint URL")
	selfHostInitCmd.Flags().String("supabase-url", "", "Supabase project URL")
	selfHostInitCmd.Flags().String("admin-key", "", "Supabase service-role key")
	selfHostInitCmd.Flags().String("jwt-secret", "", "Supabase JWT signing secret")

	// bootstrap-token flags
	selfHostBootstrapTokenCmd.Flags().String("duration", "", "Token validity (default 168h = 7 days, e.g. 24h, 720h)")
}
