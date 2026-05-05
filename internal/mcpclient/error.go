package mcpclient

import (
	"errors"
	"fmt"
)

// Kind classifies typed mcpclient errors so callers can map to user copy
// + exit codes without string-matching.
type Kind int

const (
	// KindUnknown is the catch-all for unclassified server errors.
	KindUnknown Kind = iota
	// KindUnauthorized maps HTTP 401 + JSON-RPC -32001.
	KindUnauthorized
	// KindMethodNotFound maps JSON-RPC -32601 (tool / method missing).
	KindMethodNotFound
	// KindInvalidParams maps JSON-RPC -32602.
	KindInvalidParams
	// KindInvalidRequest maps JSON-RPC -32600.
	KindInvalidRequest
	// KindParseError maps JSON-RPC -32700.
	KindParseError
	// KindVersionMismatch is emitted by Initialize when server's
	// protocolVersion doesn't match the client's compiled-in version.
	KindVersionMismatch
)

// Error is the typed error surface for mcpclient. JSONRPCCode is set for
// errors originating from a JSON-RPC error envelope (zero otherwise).
type Error struct {
	Kind        Kind
	Message     string
	JSONRPCCode int
}

func (e *Error) Error() string { return e.Message }

// IsUnauthorized is the canonical narrowing helper for the auth-failure
// branch. Cobra cmd code uses this to decide whether to print
// "Run `citadel-cli auth login` to refresh your session." Uses errors.As
// so wrapped errors classify correctly.
func IsUnauthorized(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == KindUnauthorized
}

// classifyJSONRPCError maps server-side JSON-RPC error codes to typed
// Error values. Unknown codes pass through with KindUnknown so callers
// can still surface the server's message.
func classifyJSONRPCError(code int, message string) *Error {
	kind := KindUnknown
	pretty := message
	switch code {
	case -32700:
		kind = KindParseError
	case -32600:
		kind = KindInvalidRequest
	case -32601:
		kind = KindMethodNotFound
		if pretty == "" {
			pretty = "method not found"
		}
	case -32602:
		kind = KindInvalidParams
	case -32001:
		kind = KindUnauthorized
		if pretty == "" {
			pretty = "unauthorized"
		}
	default:
		if pretty == "" {
			pretty = fmt.Sprintf("MCP error %d", code)
		}
	}
	return &Error{Kind: kind, Message: pretty, JSONRPCCode: code}
}
