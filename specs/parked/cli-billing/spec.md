# Spec — cli-billing

| | |
|---|---|
| Status | PARKED 091233ZMAY26 — REJECTED — out of scope per NCA order 091232ZMAY26. Billing management is a web/dashboard surface; CLI billing commands not in current phase mandate. |
| Authored | 091227ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | `go-billing-polar-pricing` + `billing-seat-management`: Polar billing substrate DONE; no CLI surface exists |

## Why

Phase 1 freemium launch (PoL XV.3.1) requires self-service plan management. The billing backend is complete (Polar checkout, portal, status, seat assignment via `billing-seat-management`). The web UI exposes plan management under Account / Org settings. The CLI has no billing surface at all. Developers scripting CI pipelines need to inspect plan status, and org admins benefit from seat management without leaving the terminal.

## In scope

### Plan status
- `citadel-cli billing status [-R <org-slug>]` — show current plan (Free/Solo/Team), subscription state, and seat counts for a namespace

### Checkout and portal
- `citadel-cli billing checkout [-R <org-slug>] [--plan solo|team]` — print a Polar checkout URL; open in browser automatically when a TTY is detected
- `citadel-cli billing portal [-R <org-slug>]` — print the Polar customer portal URL; open in browser automatically on TTY

### Seat management (Team plan)
- `citadel-cli billing seat list [-R <org-slug>]` — list assigned seats (user, assigned-by, assigned-at)
- `citadel-cli billing seat add [-R <org-slug>] <user-slug>` — assign a seat to a namespace member
- `citadel-cli billing seat remove [-R <org-slug>] <user-slug>` — revoke a seat from a member

### General
- `--output json|yaml|csv|ndjson|table` parity on list commands
- Auth: JWT-gated; caller must hold `members:write` on the org for seat management

## Out of scope

- Enterprise contract / offline invoicing
- Billing event history / invoice list — future spec if Polar exposes it
- Plan downgrade confirmation flow — handled by the Polar portal
- Raw Polar webhook inspection

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Command root: top-level `billing` vs `org billing` vs `namespace billing`? | Open |
| Q2 | Browser auto-open for checkout/portal URLs: use `xdg-open` / `open` when TTY, else print? | Open |
| Q3 | Server routes: confirm `/api/namespaces/{slug}/billing/status|checkout|portal` and `/api/namespaces/{slug}/billing/seats/{user_slug}` | Open |
| Q4 | Seat list output: include plan-tier seat limit if exposed by API? | Open |
