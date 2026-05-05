# Plan — cli-error-format

## CLIError type

```go
// cmd/clierror.go
type CLIErrorKind string

const (
    KindAuthRequired      CLIErrorKind = "auth_required"
    KindMFARequired       CLIErrorKind = "mfa_required"
    KindForbidden         CLIErrorKind = "forbidden"
    KindNotFound          CLIErrorKind = "not_found"
    KindConflict          CLIErrorKind = "conflict"
    KindRateLimited       CLIErrorKind = "rate_limited"
    KindValidation        CLIErrorKind = "validation"
    KindServerUnavailable CLIErrorKind = "server_unavailable"
    KindServerError       CLIErrorKind = "server_error"
    KindTimeout           CLIErrorKind = "timeout"
    KindNetwork           CLIErrorKind = "network"
    KindDryRun            CLIErrorKind = "dry_run"
    KindInternal          CLIErrorKind = "internal"
)

type CLIError struct {
    Kind        CLIErrorKind
    Message     string
    HTTPStatus  int
    RetryAfter  int
    Hint        string
    Details     map[string]any
}

func (e *CLIError) Error() string { return e.Message }
```

## errmap rewrite

`FriendlyError(err error) error` becomes:

```go
func FriendlyError(err error) error {
    if err == nil { return nil }
    if ce, ok := unwrapCLIError(err); ok { return ce } // already classified

    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return &CLIError{Kind: KindNetwork, Message: "...", Hint: "https://status.src.land"}
    }
    // ... one branch per kind, mirroring today's switch
    return &CLIError{Kind: KindInternal, Message: err.Error()}
}
```

The existing `errmap_test.go` tests stay green by checking `err.Error()` strings — those bytes are unchanged.

## main.go top-level

```go
func run(args []string, stderr io.Writer) int {
    root := newRootCmd()
    root.SetArgs(args)
    err := root.Execute()
    if err == nil { return 0 }

    if errors.Is(err, cmd.ErrToolCallFailed) { return 2 }

    cli := cmd.FriendlyError(err)
    var ce *cmd.CLIError
    if !errors.As(cli, &ce) {
        ce = &cmd.CLIError{Kind: cmd.KindInternal, Message: cli.Error()}
    }

    if jsonOutputResolved(root, args) {
        _ = json.NewEncoder(os.Stdout).Encode(envelope{Error: ce})
    } else {
        fmt.Fprintf(stderr, "Error: %s\n", ce.Message)
    }
    return kindToExitCode(ce.Kind)
}
```

`jsonOutputResolved` walks the cobra subcommand tree post-parse to find the matched leaf and reads its `--output` flag. Falls back to false if no leaf matched.

## Exit-code helper

```go
func kindToExitCode(k CLIErrorKind) int {
    switch k {
    case KindValidation, KindDryRun: return 2
    case KindAuthRequired, KindMFARequired, KindForbidden: return 3
    case KindNotFound: return 4
    case KindConflict: return 5
    case KindRateLimited: return 6
    case KindServerUnavailable, KindServerError, KindNetwork, KindTimeout: return 7
    }
    return 1
}
```

---
050900ZMAY26
