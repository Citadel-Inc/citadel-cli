# Plan — cli-git-wrappers

Q-table ratified 071639ZMAY26:

- auth injection = short-lived `GIT_ASKPASS`
- wrapper shape = second-level `repo clone|push|pull`
- behavior = system `git` passthrough
- missing remote on push = prompt to create; `--create` bypasses the prompt

## Proposed file layout

```
cmd/repo_git.go         — repo clone/push/pull + shared auth injection
cmd/repo_git_test.go    — mocked git exec tests
```

## Key constraint

The wrappers must not shell out in a way that leaks credentials through the
command line. Use a short-lived `GIT_ASKPASS` helper per invocation and keep
the rest of the behavior delegated to the on-system `git` binary.
