# Human blockers

Items that need **human / live-environment** follow-up outside what CI and httptest can enforce.

## [`cli-watch`](done/cli-watch/)

| Task | Owner | Notes |
|------|--------|-------|
| P2 C2 — operator live watch smoke | NOMAD / operator | Run `repo list --watch` against a live namespace, mutate from another shell, confirm stdout events. Automated coverage lives in `cmd/watch_sse_integration_test.go` (scripted SSE sequence). |

When an item is cleared, remove its row here and reflect closure in the spec / tasks via **citadel-sdd** (`spec_task_check`, `spec_close`, etc.) — do not edit checkbox state by hand.
