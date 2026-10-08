# citadel-cli — agent primer

LLMs: read this first. Humans: [HUMANS.md](HUMANS.md). Commits: [CONTRIBUTING.md](CONTRIBUTING.md).

## Repository shape

```
main.go              Cobra entry
cmd/                 Subcommands
internal/clicfg/     XDG config
internal/completion/ Shell completion cache
internal/mcpclient/  MCP HTTP client
specs/active|parked/ SDD specs
.github/workflows/   ci.yml, cli-release.yml
Makefile             build / verify
```

Specs are edited directly. Releases on `v*` tags.

## Invariants

- Task bullets `- [ ]` / `- [x]`; priority headings only in `tasks.md`.
- Edit spec status, DTG stamps, and `tasks.md` checkboxes directly.

## Test conventions

`go test -race ./...` / `make verify`. Live tests env-gated; safe in CI when unset.
