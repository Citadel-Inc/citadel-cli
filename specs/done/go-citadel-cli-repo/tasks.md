# Tasks — go-citadel-cli-repo

Status: DONE 032036ZMAY26 — Shipped repo/namespace/agent CRUD CLI verbs against live APIs. repo create|list|get|delete; namespace list|get|members|transfer (with initiate|list-pending|accept|decline|revoke subcommands); agent list|get|delete|rotate-token. All verbs carry --help + --output json (A1). cmd_test.go integration suite covers command-tree structure, flag presence, and destructive-verb --yes gates (A2). Destructive verbs gate on typed-slug confirm (A3). Q-table ratified (A4). repo rename descoped (no server endpoint). namespace transfer org-only for now; personal namespace transfer deferred to server-side follow-on.

## P0

- [x] [HUMAN] NOMAD ratifies Q-table.
- [x] A1. `citadel repo` subcommand tree: create/list/get/delete/rename.
- [x] A2. `citadel namespace` subcommand tree: list/get/members/transfer.
- [x] A3. `citadel agent` subcommand tree: list/get/delete/rotate-token.

## P1

- [x] B1. Shared destructive-confirm helper (typed-slug match) for delete/transfer.
- [x] B2. `--output json` emitter for each verb.
- [x] B3. Integration test against ephemeral droplet fixture in `citadel-cli/cmd/cmd_test.go`.

## P2

- [x] C1. Local + production smoke (create repo, list, rename, delete).
- [x] C2. Spec close.
