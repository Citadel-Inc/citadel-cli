# Plan — cli-milestones

## Approach

Add `cmd/issue_milestone.go` with a `milestoneCmd` cobra command tree that nests
under the existing `issueCmd`. Use the same `repocontext` resolution as other
issue verbs (via `-R`). All HTTP calls go through `newAPIClient`.

## Command tree

```
citadel-cli issue milestone
├── list    GET  /api/namespaces/{slug}/milestones
├── view    GET  /api/namespaces/{slug}/milestones/{id}
├── create  POST /api/namespaces/{slug}/milestones
├── edit    PUT  /api/namespaces/{slug}/milestones/{id}
└── delete  DELETE /api/namespaces/{slug}/milestones/{id}
```

Also add `--milestone` to existing `issue create`.

## Data shapes

`milestoneRow` (list item):
- `id` UUID
- `title` string
- `state` string
- `due_on` *time.Time
- `created_at` time.Time

`milestoneDetail` (view):
- embeds `milestoneRow`
- `description` string
- `closed_at` *time.Time
- `progress.open_count`, `progress.closed_count`, `progress.total`, `progress.percent`

## Table output

`list`: `ID  TITLE  STATE  DUE_ON  CREATED_AT`
`view`: TITLE / STATE / DESCRIPTION / DUE_ON / PROGRESS / CREATED_AT
