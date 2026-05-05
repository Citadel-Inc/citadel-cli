# Tasks — cli-kg-extended

Status: DRAFT 050506ZMAY26

## P0

- [ ] [HUMAN] NOMAD ratifies Q-table (Q1–Q2).
- [ ] A1. Verify `apiclient` path prefix for existing `kg impact`; align new routes with `/api/namespaces/.../kg/...`.
- [ ] A2. Implement `kg search`, `kg symbols`, `kg files`, `kg walk`, `kg fulltext`, `kg diff` with httptest fixtures.
- [ ] A3. Wire `root.go` if new registrations needed.

## P1

- [ ] B1. Human tables for search/symbols where JSON is unwieldy.
- [ ] B2. Pagination (`--cursor`, `--limit`, `--all`) where server returns `next_cursor`.

## P2

- [ ] C1. Live integration test (`CITADEL_TEST_KG_EXTENDED_LIVE=1`).
- [ ] C2. Spec close.
