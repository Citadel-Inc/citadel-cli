# Human blockers

Items that need **human / live-environment** follow-up outside what CI and httptest can enforce.

## [`cli-watch`](done/cli-watch/)

| Task | Owner | Notes |
|------|--------|-------|
| P2 C2 — operator live watch smoke | NOMAD / operator | Run `repo list --watch` against a live namespace, mutate from another shell, confirm stdout events. Automated coverage lives in `cmd/watch_sse_integration_test.go` (scripted SSE sequence). |

## [`cli-oauth-login`](active/cli-oauth-login/)

| Task | Owner | Notes |
|------|--------|-------|
| P2 C1 — PKCE browser flow smoke | NOMAD / operator | Run `citadel-cli auth login` against a live Citadel instance with a registered confidential client; confirm the browser callback completes, token is stored, and `auth status` shows ACTIVE. |
| P2 C3 — production smoke (token refresh) | NOMAD / operator | Perform a full login, wait for / force near-expiry, confirm the refresh-token exchange fires automatically and the session stays ACTIVE. Requires a live Citadel instance with OAuth issuer configured. |

## [`cli-mcp-resources`](done/cli-mcp-resources/)

| Task | Owner | Notes |
|------|--------|-------|
| C1 — Claude Desktop end-to-end smoke | NOMAD / operator | Register citadel-cli as an MCP server in Claude Desktop, invoke `mcp resources list`, confirm the resource URI list renders correctly in the UI. |

When an item is cleared, remove its row here and reflect closure in the spec / tasks via **citadel-sdd** (`spec_task_check`, `spec_close`, etc.) — do not edit checkbox state by hand.
