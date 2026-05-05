# Plan — cli-org-invitations

## ORIENT

- **Server:** `internal/api/orgsmembersapi/handler.go` — routes registered in `Routes()`; **frontend base** for claim URLs uses `CITADEL_FRONTEND_BASE` (CLI irrelevant except debugging).
- **Permissions:** must match **`orgs.ValidatePermissions`** allow-list — P0 A1 should paste **doc link or atom list** from server help text.

## RECON

- Read **`HandlePendingInvitations`** / **`HandleListInvitations`** response JSON keys (invitation row struct).
- **`HandleRevokeInvitation`** path param name (`{id}` UUID).
- **`HandleAcceptInvitation`** success body shape.

## Implementation sketch

- **`cmd/org_invitation.go`** — parent `orgCmd` **may** already be absent: today `namespace` holds org flows — decide whether to hang under **`namespace invitation`** vs new **`org`** top-level (Q1). If new **`org`** root: aligns with spec title; requires **`root.go`** registration.

## Risks

- **Permission strings** are easy to mistype — consider **`--permissions` from file** as P2 if repeated flags prove brittle.
