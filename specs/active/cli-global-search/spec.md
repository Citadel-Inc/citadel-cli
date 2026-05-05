# Spec — cli-global-search

| | |
|---|---|
| Status | DRAFT 050506ZMAY26 |
| Authored | 050506ZMAY26 |
| Owner | Bastion (J-3) |
| Carry-forward from | Discovery gap: `GET /api/search` and `GET /api/search/namespaces/public` exist (`internal/api/searchapi`) but CLI has no unified search verb for namespaces/repos. |

## Why

Operators navigating large tenants need quick **fuzzy discovery** from the terminal for onboarding, scripting, and Bastion-scale repo counts.

## In scope

**Parent command:** `citadel-cli search` (top-level; mirrors common forge UX).

| Verb / flags | HTTP |
|--------------|------|
| `search <query>` | `GET /api/search?q=…&scope=namespaces|repos|all&limit=…` |
| `search public <query>` | `GET /api/search/namespaces/public?q=…` (unauthenticated behaviour — verify whether JWT middleware allows anonymous; if auth required, document). |

**Cross-cutting**

- Minimum query length and scope validation mirror server (see `searchapi` tests — short `q` → 400).
- **cli-output-formats** for machine consumption.
- Table output for humans (namespace path, repo path, kind).

## Out of scope

- **Issue/search or KG fulltext** — use `issue` spec / `cli-kg-extended` respectively.
- **GraphQL** — none.

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Single `search` with `--scope` vs subcommands? | **Open** — flags on `search`. |
| Q2 | Name `search public` vs `--public` flag? | **Open** — `--public` might be clearer than a second subcommand. |

## Acceptance

- A1. Authenticated search works end-to-end in tests.
- A2. Public namespaces path documented + tested per actual auth middleware behaviour.
- A3. Q-table ratified.
- A4. Optional `CITADEL_TEST_SEARCH_LIVE=1`.
