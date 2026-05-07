# Spec — cli-namespace-profile

| | |
|---|---|
| Status | DRAFT |
| Priority | Low — read-only terminal utility; no `gh` analogue for mutation |
| Authored | 072310ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Dossier-backed first-class identity follow-up: `citadel-cli` already owns the namespace CRUD surface; a read path for profile metadata is a natural `namespace get` complement. |

## Why

`citadel-cli namespace get` currently shows repo/member counts and visibility but not the richer namespace identity metadata (display name, bio, location, social links, website).  
A `namespace profile get` surface gives terminal users enough profile context for tooling scripts and automation without requiring a browser session.  
Profile **editing** is a settings-panel concern (no `gh profile edit` exists) and is explicitly out of scope.

## In scope

- `citadel-cli namespace profile get <namespace>` — read-only
- Human/table plus structured output (`--output json|yaml|table`)
- Tests and docs

### API mapping

| Verb | Method + Path |
|------|---------------|
| `get` | `GET /api/namespaces/{slug}/profile` |

### Key fields rendered

- `display_name`, `bio`, `location`, `website_url`, `public_email`
- `pronouns`, `company`, `timezone`, `social_links`
- `stats` (repo count, member count for orgs)

## Out of scope

- `namespace profile edit` — settings-panel concern; no `gh` analogue; not a dev-loop action
- Avatar upload/delete (separate avenue if ever needed)
- Full profile README authoring
- Domain verification or org-member management

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `namespace profile` instead of adding a top-level `profile` noun. | **Ratified 072310ZMAY26** — keeps namespace path explicit. |
| Q2 | Read-only; editing parked after product review. | **Ratified 070001ZMAY26** — profile mutation is a settings-panel concern; not a dev-loop action; no GitHub CLI analogue. |
| Q3 | Output defaults to human summary/table; `--output json|yaml|table` supported. | **Ratified 072310ZMAY26** — matches other single-resource read verbs. |

## Acceptance criteria

- `namespace profile get` reads the daemon profile route and renders key identity fields
- Private-namespace visibility gate surfaces cleanly (non-owner receives an appropriate error)
- Docs explain the profile inspection workflow
- `make verify` passes
