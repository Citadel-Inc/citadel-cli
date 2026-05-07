# Spec — cli-repo-insights

| | |
|---|---|
| Status | IN_PROGRESS |
| Authored | 270640ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Daemon exposes `GET .../insights` aggregate endpoint; CLI has no summary/overview surface for repositories. Discovered during K.5 gap analysis. |

## Why

When evaluating a repository — for onboarding, due-diligence, or scripting — developers need a
single-command overview: topics, issue counts, language distribution, license, recent activity,
and top contributors.  Without a CLI surface, this requires multiple API calls or web UI
navigation.  The `repo insights` command mirrors GitHub's repository home-page summary block.

## In scope

- `citadel-cli repo insights [<namespace>/<repo>]` — print aggregate repository metadata
- Human output: structured sections (counts, languages, license, recent contributors, activity
  sparkline — last 52 weeks as a Unicode bar chart)
- JSON output: forward the full daemon response as-is
- `--output json` respected via standard output flag
- Tests and docs for the new command

### API mapping

| Verb | Method + Path |
|------|---------------|
| `insights` | `GET /api/namespaces/{ns}/repos/{repo}/insights` |

### Response shape (daemon)

```json
{
  "topics": ["go","cli"],
  "counts": {"open_issues":3,"open_milestones":1,"branches":4,"tags":2,"contributors_30d":2},
  "star_count": 5,
  "pin_count": 1,
  "releases": [{"name":"v1.0.0","sha":"...","tagged_at":"2025-01-01T00:00:00Z","is_annotated":false}],
  "activity": [0,1,2,0,...],
  "recent_contributors": [{"email":"dev@example.com","author":"Dev","count":12,"slug":"dev","display_name":"Dev User"}],
  "languages": {"Go":98304,"YAML":2048},
  "license": {"spdx":"MIT","name":"MIT","path":"LICENSE"}
}
```

## Out of scope

- Writing any insights fields (topics are managed via `repo topic set`)
- Repository star/pin actions (web-only Phase 0; Phase 1 social surface)
- Activity aggregation beyond what the daemon returns (52-week bucketed int slice)

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | `insights` is a direct child of `repo` (not nested under `repo browse`) — it is a summary action, not a browse action. | **Ratified 270640ZMAY26** — `repo insights` reads naturally; nesting under `browse` would imply file-system context. |
| Q2 | Activity sparkline uses Unicode block characters (▁▂▃▄▅▆▇█) based on 52-week buckets. | **Ratified 270640ZMAY26** — compact single-line representation; matches modern terminal tools like `gh`'s contribution graph. |
| Q3 | Language bar in human output shows top 5 languages with percentage; remainder collapsed to "Other". | **Ratified 270640ZMAY26** — mirrors GitHub's language bar aesthetics and avoids long tails. |
