# Spec — cli-namespace-profile

| | |
|---|---|
| Status | DRAFT |
| Authored | 072310ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Dossier-backed settings/profile and first-class identity follow-up: the daemon already ships namespace profile read/edit routes, but `citadel-cli` has no profile surface for user/org namespace metadata. |

## Why

The dossier positions identities and namespace presence as first-class, and the daemon already exposes public-or-owner reads plus owner-only edits for `namespace_profiles`.  
Today, terminal users cannot inspect or patch namespace profile metadata without raw REST calls even though adjacent namespace operations already exist in the CLI.

## In scope

- `citadel-cli namespace profile get <namespace>`
- `citadel-cli namespace profile edit <namespace>`
- Human/table plus structured output for reads; human summary plus `--json` for edits
- Tests and docs for namespace profile inspection/mutation

### API mapping

| Verb | Method + Path |
|------|---------------|
| `get` | `GET /api/namespaces/{slug}/profile` |
| `edit` | `PATCH /api/namespaces/{slug}/profile` |

### Fields

- `display_name`, `legal_entity_name`, `public_email`, `location`, `website_url`
- `bio`, `pronouns`, `company`, `social_links`
- `timezone`, `preferred_locale`, `pgp_fingerprint`, `sponsor_url`, `billing_email`

## Out of scope

- Avatar upload/delete (separate avatar surfaces)
- Full profile README authoring
- Domain verification or org-member management

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `namespace profile` instead of adding a new top-level `profile` noun. | **Ratified 072310ZMAY26** — keeps the namespace path explicit and aligns with existing namespace commands. |
| Q2 | `edit` uses flags for scalar fields and a structured JSON/YAML input path for `social_links` if needed. | **Ratified 072310ZMAY26** — keeps common edits simple without forcing giant JSON blobs for every change. |
| Q3 | Read output defaults to a human summary/table but preserves `--output json|yaml|table`. | **Ratified 072310ZMAY26** — matches other single-resource read verbs. |

## Acceptance criteria

- `namespace profile get` reads the daemon profile route and renders key metadata
- `namespace profile edit` PATCHes any selected subset of mutable fields
- Private-profile visibility and owner-only mutation errors surface cleanly
- Docs explain namespace profile inspection and editing
- `make verify` passes
