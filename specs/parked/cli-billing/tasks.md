# Tasks — cli-billing

Status: PARKED 091233ZMAY26 — REJECTED — out of scope per NCA order 091232ZMAY26. Billing management is a web/dashboard surface; CLI billing commands not in current phase mandate.

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q4): command root, browser-open behavior, route confirmation, seat-list output shape.
- [ ] A1. Survey `billingapi` handler in `citadel` to confirm routes, request/response shapes, and auth requirements.
- [ ] A2. Scaffold `cmd/billing.go` with `billing` top-level command (or under `org`/`namespace` per Q1 ruling).
- [ ] A3. Implement `billing status [-R <slug>]` — GET status, table output showing plan, state, seat counts.
- [ ] A4. Implement `billing checkout [-R <slug>] [--plan solo|team]` — POST checkout, print URL; auto-open on TTY.
- [ ] A5. Implement `billing portal [-R <slug>]` — POST portal, print URL; auto-open on TTY.
- [ ] A6. Implement `billing seat list [-R <slug>]` — GET seats, `--output` parity.
- [ ] A7. Implement `billing seat add [-R <slug>] <user-slug>` — PUT seat; 403 guard surfaced clearly.
- [ ] A8. Implement `billing seat remove [-R <slug>] <user-slug>` — DELETE seat; confirm prompt or `--yes`.

## P1

- [ ] B1. Handler tests: status, checkout/portal URL print, seat list/add/remove happy paths + auth rejection.
- [ ] B2. Shell completion for `--plan` values and user-slug on seat subcommands.
- [ ] B3. Update `docs/cli.md` with billing command documentation.

## P2

- [ ] C1. [HUMAN] Live smoke: status, checkout URL, seat assign/revoke against a real org subscription.
- [ ] C2. Spec close.
