# Tasks — cli-global-search

Status: DONE 060535ZMAY26 — Shipped top-level `citadel-cli search` with JWT-only GET /api/search, default scope=namespaces, --public for scope=all, httptest coverage for query_too_short/invalid_scope/invalid_limit, optional CITADEL_TEST_SEARCH_LIVE=1, and docs/cli.md QoS framing.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q2).
- [x] A1. Spike: anonymous GET for `/api/search/namespaces/public` — choose implementation option + document.
- [x] A2. Implement authenticated `search` + httptest matrix from `pure_test.go` cases.
- [x] A3. Wire `search` command in `root.go`.

## P1

- [x] B1. Human table output for `results`.
- [x] B2. `docs/cli.md`.

## P2

- [x] C1. Live test `CITADEL_TEST_SEARCH_LIVE=1`.
- [x] C2. Spec close.
