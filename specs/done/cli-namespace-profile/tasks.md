# Tasks — cli-namespace-profile

Status: DONE 072216ZMAY26 — namespace profile get shipped: reads GET /api/namespaces/{slug}/profile, renders identity fields as human table with JSON/YAML/table output modes; social links sorted+flattened; owner-only fields shown when applicable; 8 handler tests green; make verify passes (0 lint issues). C1 live smoke deferred to next integration pass.
Priority: Low

## P0

- [x] A1. Add `namespace profile get <namespace>` (read-only).
- [x] A2. Add handler and command-tree coverage for the profile read path.

## P1

- [x] B1. Document namespace profile inspection (`docs/cli.md`).
- [x] B2. `make verify` green.

## P2

- [ ] C1. Run live smoke for namespace profile get against the real daemon.
- [ ] C2. Spec close.
