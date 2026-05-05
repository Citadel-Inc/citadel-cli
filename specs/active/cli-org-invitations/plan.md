# Plan — cli-org-invitations

## ORIENT

- Server: `internal/api/orgsmembersapi/handler.go` — routes and handler names for list/create/revoke/pending/accept.
- Related existing CLI: `namespace members` (read-only members list) — invitations are separate product path.

## RECON

- Read `HandleCreateInvitation` request body struct tags.
- Confirm whether `list` returns cursor — align flags with `internal/pagination` usage in server.

## Implementation sketch

- New `cmd/org_invitation.go` or nested under `cmd/namespace.go` — prefer dedicated file to limit merge conflicts.
- `accept` uses token as positional arg; POST path `/api/invitations/{token}/accept`.

## Risks

- **Token in shell history** — document `--token-file` or stdin pattern for accept verb if sensitive.
