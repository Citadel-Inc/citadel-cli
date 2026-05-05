# Human blockers

Items that need **human / live-environment** follow-up outside what CI and httptest can enforce.

## [`cli-output-formats`](active/cli-output-formats/)

| Task | Owner | Notes |
|------|--------|--------|
| P2 C2 — operator CSV smoke | NOMAD / operator | Paste a real export (e.g. `citadel-cli repo list -n … --output csv`) into Excel / Sheets and confirm columns align with the frozen header contract in README. |

## [`cli-error-format`](active/cli-error-format/)

| Task | Owner | Notes |
|------|--------|--------|
| P2 C1 — live 429 envelope | NOMAD / operator | Hit a real or intentionally staged endpoint returning **429** with **Retry-After**; confirm `--output json` stdout envelope includes `retry_after_seconds`. Automated httptest already covers parser/unit paths in `internal/apiclient`. |
| P2 C2 — exit-code map review | NOMAD / operator | Confirm per-kind exit codes **1–7** do not break in-house wrapper scripts; document any intentional breakage. |

When an item is cleared, remove its row here and reflect closure in the spec / tasks via **citadel-sdd** (`spec_task_check`, `spec_close`, etc.) — do not edit checkbox state by hand.
