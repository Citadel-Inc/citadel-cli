package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/mcpclient"
)

// FriendlyError translates infrastructure-flavored errors (raw HTTP status
// codes, dial-tcp DNS failures, deadline-exceeded contexts) into actionable
// CLI-user-facing messages. It returns *CLIError for every classified branch
// so callers can branch on Kind / exit codes; unmapped errors pass through
// unchanged so errors.Is / errors.As on the original chain keep working.
func FriendlyError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, errSessionExpired) {
		return &CLIError{
			Kind:    KindAuthRequired,
			Message: errSessionExpired.Error(),
			Hint:    statusSrcLandHint,
		}
	}

	if deepest, ok := DeepestCLIError(err); ok {
		return deepest
	}

	var me *mcpclient.Error
	if errors.As(err, &me) {
		return mcpErrorToCLI(me)
	}

	if s := err.Error(); strings.Contains(s, "not authenticated") && strings.Contains(s, "auth login") {
		return &CLIError{
			Kind:    KindAuthRequired,
			Message: "not authenticated; run 'citadel-cli auth login' first",
			Hint:    statusSrcLandHint,
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &CLIError{
			Kind:    KindNetwork,
			Message: "cannot reach Citadel server: hostname lookup failed — check your internet connection or override the server with --server <url> (or the CITADEL_SERVER env var)",
			Hint:    statusSrcLandHint,
		}
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return &CLIError{
			Kind:    KindNetwork,
			Message: "cannot reach Citadel server: connection failed — is the server URL reachable from this host? Override with --server / CITADEL_SERVER",
			Hint:    statusSrcLandHint,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &CLIError{
			Kind:    KindTimeout,
			Message: "request timed out — server took too long to respond; retry, or check https://status.src.land",
			Hint:    statusSrcLandHint,
		}
	}

	var he *apiclient.HTTPError
	if errors.As(err, &he) {
		if ce := httpErrorToCLI(he); ce != nil {
			return ce
		}
		return err
	}

	if strings.Contains(err.Error(), "EOF") && !strings.Contains(err.Error(), "decode") {
		return &CLIError{
			Kind:    KindServerUnavailable,
			Message: "connection cut by the server; retry, and if the problem persists check https://status.src.land",
			Hint:    statusSrcLandHint,
		}
	}

	return err
}

func mcpErrorToCLI(me *mcpclient.Error) *CLIError {
	var details map[string]any
	if me.JSONRPCCode != 0 {
		details = map[string]any{"jsonrpc_code": me.JSONRPCCode}
	}
	var kind CLIErrorKind
	switch me.Kind {
	case mcpclient.KindUnauthorized:
		kind = KindAuthRequired
	case mcpclient.KindMethodNotFound:
		kind = KindNotFound
	case mcpclient.KindInvalidParams, mcpclient.KindInvalidRequest, mcpclient.KindParseError, mcpclient.KindVersionMismatch:
		kind = KindValidation
	default:
		kind = KindInternal
	}
	return &CLIError{Kind: kind, Message: me.Message, Details: details}
}

func curatedValidationDetails(body string) map[string]any {
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil || len(m) == 0 {
		return nil
	}
	keys := []string{"error", "message", "detail", "details", "code", "field"}
	out := make(map[string]any)
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
		if len(out) >= 6 {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func httpErrorToCLI(he *apiclient.HTTPError) *CLIError {
	hint := statusSrcLandHint
	retry := he.RetryAfter

	switch he.StatusCode {
	case http.StatusUnauthorized:
		return &CLIError{
			Kind:       KindAuthRequired,
			Message:    "session expired — run `citadel-cli auth login` again, or pass --token / set CITADEL_AGENT_TOKEN",
			HTTPStatus: he.StatusCode,
			Hint:       hint,
		}
	case http.StatusForbidden:
		return &CLIError{
			Kind:       KindForbidden,
			Message:    "forbidden: the server rejected your token for this resource — check namespace ownership or token scope",
			HTTPStatus: he.StatusCode,
			Hint:       hint,
		}
	case http.StatusNotFound:
		return &CLIError{
			Kind:       KindNotFound,
			Message:    "not found: the Citadel API returned HTTP 404 — verify the resource id or path",
			HTTPStatus: he.StatusCode,
			Hint:       hint,
		}
	case http.StatusBadRequest:
		details := curatedValidationDetails(he.Body)
		return &CLIError{
			Kind:       KindValidation,
			Message:    "request validation failed — fix the payload and retry",
			HTTPStatus: he.StatusCode,
			Hint:       hint,
			Details:    details,
		}
	case http.StatusConflict:
		return &CLIError{
			Kind:       KindConflict,
			Message:    "conflict: the server rejected the change — resolve the condition described in the response or retry",
			HTTPStatus: he.StatusCode,
			Hint:       hint,
			Details:    curatedValidationDetails(he.Body),
		}
	case http.StatusPreconditionFailed:
		return &CLIError{
			Kind:       KindMFARequired,
			Message:    "MFA or recent verification required — re-authenticate with MFA or use the web app, then retry",
			HTTPStatus: he.StatusCode,
			Hint:       hint,
		}
	case http.StatusTooManyRequests:
		return &CLIError{
			Kind:       KindRateLimited,
			Message:    "rate limit exceeded — slow down or wait a few minutes before retrying",
			HTTPStatus: he.StatusCode,
			RetryAfter: retry,
			Hint:       hint,
		}
	case http.StatusServiceUnavailable:
		return &CLIError{
			Kind:       KindServerUnavailable,
			Message:    "citadel server is temporarily unavailable; retry in a few seconds, or check https://status.src.land",
			HTTPStatus: he.StatusCode,
			RetryAfter: retry,
			Hint:       hint,
		}
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return &CLIError{
			Kind:       KindServerUnavailable,
			Message:    "upstream server unreachable; retry in a few seconds, or check https://status.src.land",
			HTTPStatus: he.StatusCode,
			RetryAfter: retry,
			Hint:       hint,
		}
	default:
		if he.StatusCode >= 500 {
			return &CLIError{
				Kind:       KindServerError,
				Message:    fmt.Sprintf("citadel server error (HTTP %d); retry, and if the problem persists check https://status.src.land", he.StatusCode),
				HTTPStatus: he.StatusCode,
				RetryAfter: retry,
				Hint:       hint,
			}
		}
		if he.StatusCode >= 400 {
			return &CLIError{
				Kind:       KindValidation,
				Message:    fmt.Sprintf("request rejected by the server (HTTP %d)", he.StatusCode),
				HTTPStatus: he.StatusCode,
				Details:    curatedValidationDetails(he.Body),
				Hint:       hint,
			}
		}
	}
	return nil
}
