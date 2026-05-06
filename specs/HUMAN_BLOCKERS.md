# Human blockers

Items that still need a **human / manual-review** step outside what CI and
httptest can enforce. If a follow-up only needs a live environment or browser
automation, keep it in the owning spec/tasks instead of listing it here.

## [`cli-watch`](done/cli-watch/)

| Task | Owner | Notes |
|------|--------|-------|
| P2 C2 — operator live watch smoke | NOMAD / operator | Run `repo list --watch` against a live namespace, mutate from another shell, confirm stdout events. Automated coverage lives in `cmd/watch_sse_integration_test.go` (scripted SSE sequence). |

When an item is cleared, remove its row here and reflect closure in the spec / tasks via **citadel-sdd** (`spec_task_check`, `spec_close`, etc.) — do not edit checkbox state by hand.
