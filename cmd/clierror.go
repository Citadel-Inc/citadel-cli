package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/spf13/cobra"
)

// CLIErrorKind is a stable v1 error class for scripts (--output json) and exit codes.
type CLIErrorKind string

const (
	KindAuthRequired       CLIErrorKind = "auth_required"
	KindMFARequired        CLIErrorKind = "mfa_required"
	KindForbidden          CLIErrorKind = "forbidden"
	KindNotFound           CLIErrorKind = "not_found"
	KindConflict           CLIErrorKind = "conflict"
	KindRateLimited        CLIErrorKind = "rate_limited"
	KindValidation         CLIErrorKind = "validation"
	KindServerUnavailable  CLIErrorKind = "server_unavailable"
	KindServerError        CLIErrorKind = "server_error"
	KindTimeout            CLIErrorKind = "timeout"
	KindNetwork            CLIErrorKind = "network"
	KindDryRun             CLIErrorKind = "dry_run"
	KindInternal           CLIErrorKind = "internal"
)

const statusSrcLandHint = "https://status.src.land"

// CLIError is the typed CLI failure surface. Only Kind and the exit-code mapping
// are contractual; Message may evolve.
type CLIError struct {
	Kind        CLIErrorKind
	Message     string
	HTTPStatus  int
	RetryAfter  int
	Hint        string
	Details     map[string]any
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// KindToExitCode maps Kind to the process exit code contract (v1).
func KindToExitCode(k CLIErrorKind) int {
	switch k {
	case KindValidation, KindDryRun:
		return 2
	case KindAuthRequired, KindMFARequired, KindForbidden:
		return 3
	case KindNotFound:
		return 4
	case KindConflict:
		return 5
	case KindRateLimited:
		return 6
	case KindServerUnavailable, KindServerError, KindNetwork, KindTimeout:
		return 7
	default:
		return 1
	}
}

// DeepestCLIError returns the innermost *CLIError in err's unwrap chain, if any.
func DeepestCLIError(err error) (*CLIError, bool) {
	var last *CLIError
	for e := err; e != nil; e = errors.Unwrap(e) {
		var c *CLIError
		if errors.As(e, &c) {
			last = c
		}
	}
	return last, last != nil
}

func pickDisplayMessage(execErr, friendly error) string {
	_, direct := execErr.(*CLIError)
	var ignore *CLIError
	if !direct && errors.As(execErr, &ignore) {
		return execErr.Error()
	}
	return friendly.Error()
}

// ResolveCLIExit classifies the error from cobra after FriendlyError mapping.
// friendly should be FriendlyError(execErr).
func ResolveCLIExit(execErr, friendly error) (*CLIError, int) {
	var fe *CLIError
	if errors.As(friendly, &fe) {
		out := *fe
		out.Message = pickDisplayMessage(execErr, friendly)
		return &out, KindToExitCode(out.Kind)
	}
	msg := friendly.Error()
	return &CLIError{Kind: KindInternal, Message: msg}, KindToExitCode(KindInternal)
}

type cliErrorWire struct {
	Kind                CLIErrorKind   `json:"kind"`
	Message             string         `json:"message"`
	HTTPStatus          int            `json:"http_status,omitempty"`
	RetryAfterSeconds   int            `json:"retry_after_seconds,omitempty"`
	Hint                string         `json:"hint,omitempty"`
	Details             map[string]any `json:"details,omitempty"`
}

type errorWireEnvelope struct {
	Error *cliErrorWire `json:"error"`
}

func toWire(e *CLIError) *cliErrorWire {
	if e == nil {
		return nil
	}
	w := &cliErrorWire{
		Kind:    e.Kind,
		Message: e.Message,
		Hint:    e.Hint,
		Details: e.Details,
	}
	if e.HTTPStatus != 0 {
		w.HTTPStatus = e.HTTPStatus
	}
	if e.RetryAfter > 0 {
		w.RetryAfterSeconds = e.RetryAfter
	}
	return w
}

// WriteErrorEnvelope writes the v1 JSON error envelope to w for machine-readable modes.
func WriteErrorEnvelope(w io.Writer, format string, e *CLIError) error {
	env := errorWireEnvelope{Error: toWire(e)}
	format = strings.TrimSpace(strings.ToLower(format))
	switch format {
	case "ndjson":
		b, err := json.Marshal(env)
		if err != nil {
			return err
		}
		_, err = w.Write(append(b, '\n'))
		return err
	case "yaml":
		enc := yaml.NewEncoder(w)
		defer func() { _ = enc.Close() }()
		return enc.Encode(env)
	default: // json and unknown → json
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(env)
	}
}

// WantsMachineErrorEnvelope reports whether --output requests a structured error envelope.
func WantsMachineErrorEnvelope(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	o, _ := cmd.Flags().GetString("output")
	switch strings.TrimSpace(strings.ToLower(o)) {
	case "json", "yaml", "ndjson":
		return true
	default:
		return false
	}
}
