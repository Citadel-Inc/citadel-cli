# Plan — cli-account-export

Q-table partially ratified:

- Q1: `account export` — ratified 091227ZMAY26; fits the existing `account` command group in `cmd/account.go`
- Q2: `--wait` polling interval 5s; max wait 15 min (server bundle build typically 30s–3min); configurable via `--timeout`
- Q3: progress indicator — print "Building export… (elapsed Xs)" on stderr each poll tick when TTY; silent when piped

## Proposed file layout

```
cmd/account.go          — add `export`, `export status`, `export history` under existing account command
cmd/account_export.go   — or separate file if account.go grows large
cmd/account_test.go     — extend or add account_export_test.go
docs/cli.md             — account section: document export subcommands
```

## Server API summary

| Verb | Route | Purpose |
|------|-------|---------|
| POST | `/api/account/export-request` | Trigger new export; returns `{request_id}` |
| GET | `/api/account/export-status/{id}` | Poll status; returns `{status, bundle_url?, bundle_size_bytes?, expires_at?}` |
| GET | `/api/account/export-history` | List last 5 exports |
