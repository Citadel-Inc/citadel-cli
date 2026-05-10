# Tasks — cli-self-host-setup

Status: DRAFT 101905ZMAY26

## P0

- [ ] RECON: command registration pattern in cmd/; Supabase client instantiation; schema_migrations table structure.
- [ ] Config struct + YAML read/write (internal/selfhost/config.go).
- [ ] Health check probes (API, DB, migrations) — internal/selfhost/health.go.
- [ ] Migration apply logic (idempotent) — internal/selfhost/migrate.go.
- [ ] Bootstrap token generation (admin scope, duration) — internal/selfhost/token.go.
- [ ] Command subgroups wired in cmd/self_host.go (init, health, migrate, bootstrap-token, telemetry).

## P1

- [ ] Unit tests: config read/write, migration idempotency, health scenarios, token generation + validation.
- [ ] All verbs support --batch flag; validate params; fail gracefully if missing.
- [ ] Error handling: opaque to stdout, detailed to logs (--debug).
- [ ] No secrets in logs; Supabase keys + tokens redacted from debug output.
- [ ] docs/cli.md section "Self-host setup" with examples.

## P2

- [ ] Integration tests gated on CITADEL_TEST_SELF_HOST_LIVE (deploy test instance, run verbs end-to-end).
- [ ] Q-table ratification (incl. Q3 remote-fetch Phase 2 enhance, signal definitions, telemetry defaults).
- [ ] Spec close + push.
