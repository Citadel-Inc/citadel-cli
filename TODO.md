# TODO — follow-on and carry-forward

Planning backlog only. Parked/rejected specs (`specs/parked/`) and live-smoke-only residuals stay out unless reopened by product decision.

Do **not** restore `account passkey` / `account device` — those verbs are out of scope for this CLI (settings-panel workflows, not developer-loop). Phase B MFA recovery under that tree is not planned.

---

## Deferred

### 14b. Homebrew tap release-time bump

**Feature.** `Citadel-Inc/homebrew-tap` exists with `Formula/citadel-cli.rb` at v0.6.0 (`brew tap citadel-inc/tap && brew install citadel-cli`). Remaining: bump the formula on each release.

| | |
| --- | --- |
| **Packages / files** | `.github/workflows/release.yml` (new bump job after Create Release), `Citadel-Inc/homebrew-tap` `Formula/citadel-cli.rb`, `docs/cli.md` § Distribution |
| **Work** | After the release is created, rewrite `version`, both `url` lines and both `sha256` values (darwin arm64 and amd64, from `dist/SHA256SUMS`) in the formula, then commit and push to the tap `main` over SSH using the `HOMEBREW_TAP_DEPLOY_KEY` Actions secret |
| **Traps** | Deploy key "citadel-cli release" (write) is on the tap and `HOMEBREW_TAP_DEPLOY_KEY` is set in this repo. Publishing waits on NOMAD's go |
| **Acceptance** | Tagging `v*` leaves the tap formula at the new version with matching checksums; docs link the tap |

---

## Explicitly not tracked here

| Item | Why |
| ------ | ----- |
| `cli-webhook-test`, billing, avatar, privacy, account-export, mcp-stdio/stream | Rejected or superseded: export/billing/webhook-test in the Parked table of `specs/README.md`, mcp-stdio/stream in the HTTPS MCP decision in `specs/parked/README.md`, avatar/privacy as the settings-panel row below |
| Restoring `account *` security verbs | Out of scope — browser settings-panel concern, not CLI |
| Live-smoke / C1-only residuals on done specs | Env-gated operator work, not product features |
| shadcn / web component work | This repo is a Go Cobra CLI; browser UX lives in Citadel web |
