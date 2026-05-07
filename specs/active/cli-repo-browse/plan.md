# Plan — cli-repo-browse

## Implementation approach

Two new subcommands under `repoBrowseCmd`:

1. `repo browse tree` — calls `GET /api/namespaces/{ns}/repos/{repo}/tree?ref=&path=`
   - Returns `{"ref","path","entries":[{"path","mode","kind","size","sha"}]}`
   - Human: tab-writer table with kind icon, name, size (blank for tree), short SHA
   - JSON/YAML: full API response forwarded via `emitJSON`

2. `repo browse blob` — calls `GET /api/namespaces/{ns}/repos/{repo}/blob?ref=&path=`
   - Returns `{"sha","size","binary","encoding","content"}`
   - Human: print `content` raw to stdout; binary: print one line informational
   - JSON: emit the full API response via `emitJSON`

Implementation files:
- `cmd/repo_browse.go` — new file for repoBrowseCmd tree
- Wire in `cmd/repo.go` after existing `repoTagCmd` line

No CSV output needed (tree has variable-width entries; blob is content not tabular).

## Notes

- Both commands support `--output json|yaml` (not list formats since browse doesn't paginate).
- `blob --path` is a required positional arg (or flag). Use `--path` flag like other commands for consistency.
- Error handling: ref_not_found → "ref not found", path_not_found → "path not found", invalid_path → "invalid path".
