# Tasks — cli-projectgraph

Status: DONE 060539ZMAY26 — Delivered top-level `citadel-cli project` with URL-encoded multi-segment namespace paths, read verbs (pin-chain, walk, neighbors, status rollup/drilldown), write verbs (edge add/delete/restore, reindex) with typed confirm/--yes, httptest matrix incl. multi-segment pin-chain + read/write 404 paths, docs/cli.md, optional live test behind CITADEL_TEST_PROJECTGRAPH_LIVE + CITADEL_TEST_PROJECTGRAPH_SLUG. Q4 recovery-scan intentionally unimplemented; P1 B2 remains open.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q4).
- [x] A1. Append **appendix** to `plan.md`: copy query-param tables + JSON structs from `handleStatusRollup`, `handleStatusRollupDrilldown`, `handleReindex`, recovery-scan handler (exact server RECON).
- [x] A2. Implement URL builder + read verbs (`pin-chain`, `walk`, `neighbors`, `status`, `status drilldown`) + httptest coverage per verb.
- [x] A3. Wire `ProjectCmd` in `cmd/root.go`.

## P1

- [x] B1. Write verbs (`edge add/delete/restore`, `reindex`) + `--yes` / confirmation policy per Q3.
- [ ] B2. Admin `recovery-scan` if Q4 ratifies inclusion.
- [x] B3. Human-readable tables + `--output` integration.

## P2

- [x] C1. `docs/cli.md` + HUMANS cross-links.
- [x] C2. Live test `CITADEL_TEST_PROJECTGRAPH_LIVE=1`.
- [x] C3. Spec close.
