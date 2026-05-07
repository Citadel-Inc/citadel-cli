# Plan — cli-repo-insights

## Implementation approach

Single `repoInsightsCmd` subcommand nested directly under `RepoCmd`:

1. `repo insights [<ns/repo>]` — calls `GET /api/namespaces/{ns}/repos/{repo}/insights`
   - Returns rich aggregate object (see spec for full shape)
   - Human mode: sections
     - **Topics:** comma-separated (or "none")
     - **Counts:** table of open issues, milestones, branches, tags, contributors_30d, stars, pins
     - **License:** SPDX identifier + name (or "None detected")
     - **Languages:** top-5 horizontal bar with percentages
     - **Recent contributors:** table (name/slug, commits in 30d)
     - **Activity (52w):** single sparkline line using ▁▂▃▄▅▆▇█ chars
   - JSON mode: emit full API response via `emitJSON`

## Notes

- Activity sparkline: scale max bucket to █; empty weeks → space or ▁.
- Language bar: compute sum, show top 5 as `Go 73%  YAML 20%  Other 7%`.
- If daemon returns empty repo (no `activity` key), print "Empty repository" notice for git-backed sections.
- License nil → "None detected".
