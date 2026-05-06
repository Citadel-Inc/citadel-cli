# Tasks — cli-kg-extended

Status: DONE 060504ZMAY26 — Shipped extended KG HTTP verbs (search, symbols, files, walk, fulltext, diff); migrated kg impact + symbol resolution to /api/namespaces/{slug}/kg/*; added httptest (401/404/429), docs/cli.md section, plan appendix, and opt-in live test. P1 pagination/table polish remains open.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table (Q1–Q3).
- [x] A1. Path reconciliation audit — document in `plan.md` appendix; add spike PR or inline comment if `kg impact` migrates paths.
- [x] A2. Implement `kg search`, `kg symbols`, `kg files`, `kg walk`, `kg fulltext`, `kg diff` + httptest suite.
- [x] A3. Register subcommands under `KgCmd`.

## P1

- [ ] B1. Pagination UX (`--cursor`, `--limit`, `--all`) mirroring `cli-pagination` where server emits `next_cursor`.
- [ ] B2. Table output for symbols/search when `--output table`.

## P2

- [x] C1. `docs/cli.md` KG section expansion.
- [x] C2. Live test `CITADEL_TEST_KG_EXTENDED_LIVE=1`.
- [x] C3. Spec close.
