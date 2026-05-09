# Plan — cli-billing

Q-table pending NOMAD ratification. Defaults assumed:

- Q1: top-level `billing` command — consistent with `gh billing` idiom; `-R <org-slug>` flag scopes to namespace
- Q2: browser auto-open via `xdg-open` (Linux) / `open` (macOS); print-only fallback when `!isatty(stdout)`
- Q3: server routes confirmed from `billingapi` handler — `/api/namespaces/{slug}/billing/status`, `/checkout`, `/portal`; seats at `/api/namespaces/{slug}/billing/seats/{user_slug}`
- Q4: seat list shows plan-tier limit if returned by status endpoint

## Proposed file layout

```
cmd/billing.go         — billing command tree (status, checkout, portal, seat list/add/remove)
cmd/billing_test.go    — handler tests
docs/cli.md            — billing command section
```
