# Spec — cli-self-host-setup

| | |
|---|---|
| Status | DONE 102042ZMAY26 — P1+P2 residuals complete: --debug flag wired to stderr with secrets redacted, httptest-backed integration tests gated on CITADEL_TEST_SELF_HOST_LIVE, docs/cli.md enriched with troubleshooting section and live test recipe, Q-table ratified. |
| Priority | P1 |
| Authored | 101905ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Phase 2 deliverable XV.8: "Self-host enterprise tier (customers run Citadel + Supabase on their own infra)." Companion spec `ops-enterprise-self-host-packaging` (citadel repo) handles binary/container build; this spec defines CLI verbs for customer self-hosted initialization + operational health checks. |

## Why

Phase 2 strategy (XVI.1, R3 off-ramp) transitions Citadel from Supabase Cloud to self-hosted Supabase, enabling customers to run Citadel + Supabase on their own infrastructure (sovereign-AI, regulated industries, FedRAMP/HIPAA). `citadel-cli` today is a public-SaaS client. Phase 2 requires CLI verbs to bootstrap a customer's self-hosted install: initialize Supabase, apply migrations, generate service tokens, health-check the deployment. This spec adds those verbs.

## In scope

- `citadel self-host init` — interactive wizard: collects API endpoint, Supabase project URL, admin API key (required), and outputs a local config file (`~/.citadel/self-host.yaml`).
- `citadel self-host health` — probe the deployment: HTTP GET to `/api/health`, Supabase connectivity check (query a test table), database migration status (read `schema_migrations` table), and report summary (GREEN/AMBER/RED).
- `citadel self-host migrate` — apply pending Supabase migrations: read migrations from packaged bundle or remote repo, connect to target Supabase instance, apply cleanly (idempotent via existing `schema_migrations` table), abort on error.
- `citadel self-host bootstrap-token` — generate a bootstrap JWT (admin scope) for initial operator setup; accepts `--duration` flag (default 7 days); outputs token to stdout for piping.
- `citadel self-host telemetry {enable|disable}` — opt in/out of anonymous usage telemetry; stored in local config.
- Common error handling: opaque errors to stdout, detailed errors to logs (--debug flag).
- No interactive prompts when `--batch` flag set; validate all required params, fail if missing.

## Out of scope

- Interactive WebAuthn / OAuth setup (Phase 2+ admin dashboard feature).
- Automatic container orchestration (Helm charts / k8s manifests — separate ops spec).
- Cost tracking / billing per customer (Phase 3 multi-tenant meter spec).
- Disaster recovery (backups, point-in-time restore) — separate ops runbook.

## Decision log

| Q | Decision | Rationale |
|---|----------|-----------|
| Q1 | Ratified 101905ZMAY26 — `~/.citadel/self-host.yaml` default; `CITADEL_SELF_HOST_CONFIG` env-var override. | Env-var-first (`CITADEL_SELF_HOST_CONFIG` path override); default to `~/.citadel/self-host.yaml`. Operators can CI-inject via env. |
| Q2 | Ratified 101905ZMAY26 — Bootstrap token always admin scope Phase 2; Phase 3 may add role scoping. | Always `admin` for Phase 2 simplicity; role scoping deferred until Phase 3 multi-tenant requirements clarified. |
| Q3 | Ratified 101905ZMAY26 — Packaged migration bundle Phase 2 primary; remote fetch Phase 2 optional enhance. | Packaged bundle shipped with binary from citadel repo; remote fetch (Git over HTTPS) as Phase 2 optional enhance. Reduces external deps at deploy time. |
| Q4 | Ratified 101905ZMAY26 — Health check RED if API or DB unreachable; AMBER if migrations pending. | Separate signals per health component: RED = deployment non-functional; AMBER = deployable but not production-ready (pending migrations). |
| Q5 | Ratified 101905ZMAY26 — Telemetry global opt-out in config; commands respect `telemetry: false` flag. | Global opt-out in config file; no per-command override needed; telemetry disabled by default unless explicitly enabled. |
| Q6 | Ratified 101905ZMAY26 — Bootstrap token to stdout only; no auto file write. | Stdout only; operators handle piping / logging. Clear security boundary: no automatic file write avoids accidental token persistence. |

## Acceptance criteria

- A1. `citadel self-host init` collects endpoint, Supabase URL, admin key; writes config to `~/.citadel/self-host.yaml` or `CITADEL_SELF_HOST_CONFIG` path.
- A2. `citadel self-host health` probes API, Supabase, migrations; returns GREEN/AMBER/RED status + detail per component.
- A3. `citadel self-host migrate` applies pending migrations idempotently; aborts on error with clear message.
- A4. `citadel self-host bootstrap-token` generates JWT (admin scope, configurable duration); outputs token to stdout.
- A5. `citadel self-host telemetry {enable|disable}` updates config; global flag respected by all commands.
- A6. All verbs accept `--batch` flag; no interactive prompts when set; fail if required params missing.
- A7. Unit tests: config reading/writing, migration tracking (no double-apply), health-check scenarios (all GREEN/AMBER/RED cases), token generation + validation.
- A8. Integration tests (gated on `CITADEL_TEST_SELF_HOST_LIVE`): deploy test instance; run init/health/migrate/bootstrap verbs end-to-end.
- A9. `docs/cli.md` — new section "Self-host setup" with examples (init, health check, migration, token bootstrap, telemetry).
- A10. No secrets in logs; Supabase keys + generated tokens redacted from debug output.
- A11. Q-table ratified (incl. Q3 remote-fetch Phase 2 enhance, Q4 health-check signal definitions, Q5 telemetry default).
