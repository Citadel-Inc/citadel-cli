# Spec — cli-repo-browse

| | |
|---|---|
| Status | IN_PROGRESS |
| Authored | 270640ZMAY26 |
| Owner | Copilot |
| Carry-forward from | Daemon exposes `GET .../tree` and `GET .../blob` browse endpoints; CLI has no file-browsing surface. Discovered during K.5/K.8 gap analysis. |

## Why

Citadel repositories support in-browser file browsing but the CLI has no equivalent.
Developers using the terminal need to list directory contents and read files without a local
clone — essential for scripts, agents, and quick review workflows.
GitHub CLI provides `gh api` workarounds; a first-class `repo browse` subtree gives a
workflow-aligned interface consistent with the existing `repo commit` surface.

## In scope

- `citadel-cli repo browse tree [<namespace>/<repo>] [--ref <ref>] [--path <path>]` — list
  directory entries at a given ref and path (defaults to repo root on default branch)
- `citadel-cli repo browse blob [<namespace>/<repo>] <path> [--ref <ref>]` — read file content;
  prints raw text to stdout for human mode; JSON wraps sha/size/binary/encoding/content
- Standard output flags (`--output json|yaml`) on both subcommands
- Tests and docs for all new commands

### API mapping

| Verb | Method + Path |
|------|---------------|
| `browse tree` | `GET /api/namespaces/{ns}/repos/{repo}/tree?ref=&path=` |
| `browse blob` | `GET /api/namespaces/{ns}/repos/{repo}/blob?ref=&path=` |

### Response shapes (daemon)

Tree: `{"ref":"main","path":"cmd","entries":[{"path":"agent.go","mode":"100644","kind":"blob","size":12345,"sha":"..."},...]}`

Blob: `{"sha":"abc","size":1234,"binary":false,"encoding":"utf-8","content":"package main..."}` (binary variant omits content, sets `binary:true`)

## Out of scope

- Recursive tree walk (single directory level only; mirrors `ls` ergonomics)
- Raw file download endpoint (`/raw?ref=&path=`) — deferred; API is authenticated, raw downloads need stream handling
- Write operations (blob creation/update via the KG write API) — separate spec

## Decision log

| Q | Proposal | Status |
|---|----------|--------|
| Q1 | Nest under `repo browse` rather than `repo tree` / `repo blob` — mirrors GitHub CLI `gh browse` / tree subcommand naming. | **Ratified 270640ZMAY26** — `repo browse tree` and `repo browse blob` are self-documenting and discoverable. |
| Q2 | `blob` in human mode prints raw file content directly to stdout; binary files print a single informational line ("Binary file (<N> bytes), SHA <sha>"). | **Ratified 270640ZMAY26** — raw content enables shell composition (`citadel-cli repo browse blob ... | grep ...`). |
| Q3 | `tree` in human table shows: kind icon (📄/📁), name, size (blank for dirs), sha (short 8). | **Ratified 270640ZMAY26** — mirrors `ls -la` aesthetics. |
