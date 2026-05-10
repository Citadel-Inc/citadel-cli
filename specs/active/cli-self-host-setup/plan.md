# Plan — cli-self-host-setup

## ORIENT

- **Existing CLI shape:** `cmd/` root — see how current verbs (e.g., `citadel account`, `citadel repos`) are registered.
- **Existing config:** look for current `~/.citadel/` usage (auth tokens, etc.); integrate self-host config there.
- **Existing Supabase integration:** `internal/db/` — how is client instantiated? (likely env var for URL + key). Self-host will need custom endpoint.
- **JWT generation:** existing token handling in auth code (if any); may be able to reuse.

## RECON

1. Grep `cmd/` for command registration pattern; understand subcommand nesting (e.g., `citadel account passkey` → `cmd/account.go` with subcommands).
2. Survey `internal/` for Supabase client instantiation; identify where URL + key are read.
3. Check if JWT lib already in deps (likely from daemon codebase, but verify).
4. Review migration schema (`supabase/migrations/` in citadel repo) — schema_migrations table structure.

## Implementation sketch

1. **`cmd/self_host.go`** — command group with subcommands: `init`, `health`, `migrate`, `bootstrap-token`, `telemetry`.
2. **`internal/selfhost/config.go`** — config struct (endpoint, supabase_url, admin_key, telemetry), read/write YAML.
3. **`internal/selfhost/health.go`** — probes (HTTP, DB connectivity, migration status).
4. **`internal/selfhost/migrate.go`** — reads bundled migrations, applies via Supabase client.
5. **`internal/selfhost/token.go`** — JWT generation (admin scope, duration param).
6. **`cmd/self_host.go` subcommands** — thin wrappers calling internal funcs; error handling + output formatting.
7. **Unit tests** — config read/write, migration idempotency, health scenarios, token generation.
8. **Integration tests** — live instance (if CITADEL_TEST_SELF_HOST_LIVE).

## Risks

- **Migration ordering** — if citadel migrations aren't idempotent (some aren't by design), double-apply fails. Verify schema_migrations table prevents re-runs.
- **Supabase client auth** — admin key is sensitive; ensure no log leaks.
- **Network timeouts** — health check may hang if endpoint unreachable. Set short timeout (5–10s).

## Appendix: Schema contract (survey during RECON)

Supabase's built-in `schema_migrations` table (or `public.schema_migrations` if custom):

| Column | Type | Notes |
|--------|------|-------|
| version | bigint | Migration version (e.g., 20260101120000) |
| name | text | Migration name (e.g., "create_releases_table") |
| statements | text[] | SQL statements |
| execution_time_ms | int | Execution time |
| executed_at | timestamp | When applied |
