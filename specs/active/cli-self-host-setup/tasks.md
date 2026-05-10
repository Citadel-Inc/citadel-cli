# Tasks — cli-self-host-setup

Status: DONE 102020ZMAY26 — P0 + P1 (minus --debug split) shipped on feat/cli-self-host-setup: internal/selfhost package (config YAML r/w, GREEN/AMBER/RED health probes, supabase-CLI migrate, HS256 bootstrap-token); cmd/self_host.go group (init, health, migrate, bootstrap-token, telemetry); --batch persistent flag; 15 race-clean unit tests; docs/cli.md Self-host section with migrate prerequisite note. P1 row 3 (--debug output split) and P2 live integration tests deferred.

## P0

- [x] RECON: command registration pattern in cmd/; Supabase client instantiation; schema_migrations table structure.
- [x] Config struct + YAML read/write (internal/selfhost/config.go).
- [x] Health check probes (API, DB, migrations) — internal/selfhost/health.go.
- [x] Migration apply logic (idempotent) — internal/selfhost/migrate.go.
- [x] Bootstrap token generation (admin scope, duration) — internal/selfhost/token.go.
- [x] Command subgroups wired in cmd/self_host.go (init, health, migrate, bootstrap-token, telemetry).

## P1

- [x] Unit tests: config read/write, migration idempotency, health scenarios, token generation + validation.
- [x] All verbs support --batch flag; validate params; fail gracefully if missing.
- [ ] Error handling: opaque to stdout, detailed to logs (--debug).
- [x] No secrets in logs; Supabase keys + tokens redacted from debug output.
- [x] docs/cli.md section "Self-host setup" with examples.

## P2

- [ ] Integration tests gated on CITADEL_TEST_SELF_HOST_LIVE (deploy test instance, run verbs end-to-end).
- [ ] Q-table ratification (incl. Q3 remote-fetch Phase 2 enhance, signal definitions, telemetry defaults).
- [ ] Spec close + push.
