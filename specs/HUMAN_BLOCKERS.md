# Human blockers

Items that need **human / live-environment** follow-up outside what CI and httptest can enforce.

## [`cli-output-formats`](done/cli-output-formats/)

| Task | Owner | Notes |
|------|--------|--------|
| P2 C2 — operator CSV smoke | NOMAD / operator | Paste a real export (e.g. `citadel-cli repo list -n … --output csv`) into Excel / Sheets and confirm columns align with the frozen header contract in README. |

## [`cli-error-format`](done/cli-error-format/)

| Task | Owner | Notes |
|------|--------|--------|
| P2 C1 — live 429 envelope | NOMAD / operator | Hit a real or intentionally staged endpoint returning **429** with **Retry-After**; confirm `--output json` stdout envelope includes `retry_after_seconds`. Automated httptest already covers parser/unit paths in `internal/apiclient`. |
| P2 C2 — exit-code map review | NOMAD / operator | Confirm per-kind exit codes **1–7** do not break in-house wrapper scripts; document any intentional breakage. |

When an item is cleared, remove its row here and reflect closure in the spec / tasks via **citadel-sdd** (`spec_task_check`, `spec_close`, etc.) — do not edit checkbox state by hand.

## [`cli-watch`](done/cli-watch/)

| Task | Owner | Notes |
|------|--------|-------|
| P0 A1 — SSE on every list path | Citadel core / companion server spec | Server emits init/add/update/remove, `:keepalive`, `Last-Event-ID`; CLI client and B6 httptest already assume this contract. |
| P2 C2 — operator live watch smoke | NOMAD / operator | Run `repo list --watch` against a live namespace, mutate from another shell, confirm stdout events. |
