package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
	"github.com/Rethunk-Tech/citadel-cli/internal/mcpclient"
)

func TestFriendlyError_NilPassthrough(t *testing.T) {
	if got := FriendlyError(nil); got != nil {
		t.Fatalf("nil → nil; got %v", got)
	}
}

func TestFriendlyError_DNSFailure(t *testing.T) {
	in := &net.DNSError{Err: "no such host", Name: "api.src.land"}
	got := FriendlyError(in)
	if !strings.Contains(got.Error(), "hostname lookup failed") {
		t.Fatalf("DNS map: %v", got)
	}
}

func TestFriendlyError_DeadlineExceeded(t *testing.T) {
	if got := FriendlyError(context.DeadlineExceeded); !strings.Contains(got.Error(), "timed out") {
		t.Fatalf("deadline map: %v", got)
	}
}

func TestFriendlyError_HTTPStatusBranches(t *testing.T) {
	cases := map[int]string{
		401: "session expired",
		403: "forbidden",
		429: "rate limit",
		503: "temporarily unavailable",
		502: "upstream",
		504: "upstream",
		500: "citadel server error (HTTP 500)",
	}
	for code, want := range cases {
		err := FriendlyError(&apiclient.HTTPError{StatusCode: code, Body: ""})
		if !strings.Contains(err.Error(), want) {
			t.Errorf("HTTP %d → want %q, got %v", code, want, err)
		}
	}
}

func TestFriendlyError_PassthroughForUnknown(t *testing.T) {
	in := errors.New("some bespoke verb error")
	if got := FriendlyError(in); got != in {
		t.Fatalf("unmapped errors must pass through unchanged: got %v", got)
	}
}

func TestFriendlyError_EOFConnectionCut(t *testing.T) {
	in := errors.New("request failed: read tcp: EOF")
	got := FriendlyError(in)
	if !strings.Contains(got.Error(), "connection cut") {
		t.Fatalf("EOF map: %v", got)
	}
}

func TestKindToExitCode(t *testing.T) {
	// Full matrix for CLIErrorKind — stable contract for scripts (`--output json` + exit codes).
	cases := []struct {
		k    CLIErrorKind
		want int
	}{
		{KindInternal, 1},
		{KindDryRun, 2},
		{KindValidation, 2},
		{KindAuthRequired, 3},
		{KindMFARequired, 3},
		{KindForbidden, 3},
		{KindNotFound, 4},
		{KindConflict, 5},
		{KindRateLimited, 6},
		{KindServerUnavailable, 7},
		{KindServerError, 7},
		{KindTimeout, 7},
		{KindNetwork, 7},
	}
	for _, tc := range cases {
		if got := KindToExitCode(tc.k); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.k, got, tc.want)
		}
	}
}

func TestResolveCLIExit_WrappedExistingCLIError(t *testing.T) {
	inner := &CLIError{Kind: KindAuthRequired, Message: "session expired — run `citadel-cli auth login` again, or pass --token / set CITADEL_AGENT_TOKEN"}
	wrapped := fmt.Errorf("namespace get: %w", inner)
	f := FriendlyError(wrapped)
	ce, code := ResolveCLIExit(wrapped, f)
	if code != 3 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.HasPrefix(ce.Message, "namespace get:") {
		t.Fatalf("message = %q want prefix from wrap", ce.Message)
	}
}

func TestWriteErrorEnvelope_JSON429(t *testing.T) {
	in := &apiclient.HTTPError{StatusCode: 429, RetryAfter: 60, Body: ""}
	f := FriendlyError(in)
	ce, _ := ResolveCLIExit(in, f)
	var buf bytes.Buffer
	if err := WriteErrorEnvelope(&buf, "json", ce); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	inner, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %v", got)
	}
	if inner["kind"] != string(KindRateLimited) {
		t.Fatalf("kind = %v", inner["kind"])
	}
	ra, ok := inner["retry_after_seconds"].(float64)
	if !ok || int(ra) != 60 {
		t.Fatalf("retry_after_seconds = %v (%T)", inner["retry_after_seconds"], inner["retry_after_seconds"])
	}
}

func TestFriendlyError_NotAuthenticatedString(t *testing.T) {
	in := errors.New("not authenticated; run 'citadel-cli auth login' first")
	got := FriendlyError(in)
	var ce *CLIError
	if !errors.As(got, &ce) || ce.Kind != KindAuthRequired {
		t.Fatalf("got %v", got)
	}
}

func TestFriendlyError_MCPUnauthorized(t *testing.T) {
	in := &mcpclient.Error{Kind: mcpclient.KindUnauthorized, Message: "bad token", JSONRPCCode: -32001}
	got := FriendlyError(in)
	var ce *CLIError
	if !errors.As(got, &ce) || ce.Kind != KindAuthRequired {
		t.Fatalf("got %#v", got)
	}
}

func TestFriendlyError_HTTP404409412400(t *testing.T) {
	cases := []struct {
		code int
		body string
		kind CLIErrorKind
		sub  string
	}{
		{http.StatusNotFound, "", KindNotFound, "not found"},
		{http.StatusConflict, `{"error":"x"}`, KindConflict, "conflict"},
		{http.StatusPreconditionFailed, "", KindMFARequired, "MFA"},
		{http.StatusBadRequest, `{"error":"bad","field":"name"}`, KindValidation, "validation"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.code), func(t *testing.T) {
			got := FriendlyError(&apiclient.HTTPError{StatusCode: tc.code, Body: tc.body})
			var ce *CLIError
			if !errors.As(got, &ce) || ce.Kind != tc.kind {
				t.Fatalf("got %v want kind %s", got, tc.kind)
			}
			if !strings.Contains(strings.ToLower(ce.Error()), strings.ToLower(tc.sub)) {
				t.Fatalf("message %q missing %q", ce.Message, tc.sub)
			}
			if ce.HTTPStatus != tc.code {
				t.Fatalf("HTTPStatus = %d", ce.HTTPStatus)
			}
		})
	}
}

func TestFriendlyError_HTTPUnknownPassesThrough(t *testing.T) {
	in := &apiclient.HTTPError{StatusCode: 418, Body: "teapot"}
	got := FriendlyError(in)
	if got != in {
		t.Fatalf("want pass-through, got %T %v", got, got)
	}
}

func TestFriendlyError_validationDetailsCurated(t *testing.T) {
	body := `{"error":"invalid","detail":"d0","extra":"drop","nested":{"x":1}}`
	got := FriendlyError(&apiclient.HTTPError{StatusCode: http.StatusBadRequest, Body: body})
	var ce *CLIError
	if !errors.As(got, &ce) || ce.Details == nil {
		t.Fatalf("got %#v", got)
	}
	if _, ok := ce.Details["error"]; !ok {
		t.Fatalf("details = %#v", ce.Details)
	}
	if _, bad := ce.Details["nested"]; bad {
		t.Fatal("nested should not be in curated subset")
	}
}

func TestCuratedValidationDetails_invalidJSON(t *testing.T) {
	if got := curatedValidationDetails("not json"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestCuratedValidationDetails_emptyObject(t *testing.T) {
	if got := curatedValidationDetails("{}"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestCuratedValidationDetails_noCuratedKeys(t *testing.T) {
	if got := curatedValidationDetails(`{"only_unknown":1}`); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestCuratedValidationDetails_hitsKeyCap(t *testing.T) {
	body := `{"error":"e","message":"m","detail":"d","details":"x","code":1,"field":"f","extra":"z"}`
	got := curatedValidationDetails(body)
	if got == nil || len(got) != 6 {
		t.Fatalf("want 6 curated keys, got %#v", got)
	}
	if _, ok := got["field"]; !ok {
		t.Fatalf("missing field: %#v", got)
	}
	if _, bad := got["extra"]; bad {
		t.Fatal("extra should not appear when cap reached at 6 keys")
	}
}

func TestMcpErrorToCLI_methodNotFound(t *testing.T) {
	in := &mcpclient.Error{Kind: mcpclient.KindMethodNotFound, Message: "no tool", JSONRPCCode: -32601}
	ce := mcpErrorToCLI(in)
	if ce.Kind != KindNotFound {
		t.Fatalf("kind %s", ce.Kind)
	}
	if ce.Details == nil {
		t.Fatal("expected details")
	}
	if ce.Details["jsonrpc_code"] != -32601 {
		t.Fatalf("jsonrpc_code: %#v", ce.Details["jsonrpc_code"])
	}
}

func TestMcpErrorToCLI_validationKinds(t *testing.T) {
	cases := []struct {
		kind mcpclient.Kind
		want CLIErrorKind
	}{
		{mcpclient.KindInvalidParams, KindValidation},
		{mcpclient.KindInvalidRequest, KindValidation},
		{mcpclient.KindParseError, KindValidation},
		{mcpclient.KindVersionMismatch, KindValidation},
	}
	for _, tc := range cases {
		ce := mcpErrorToCLI(&mcpclient.Error{Kind: tc.kind, Message: "x", JSONRPCCode: -32602})
		if ce.Kind != tc.want {
			t.Errorf("%v → %s want %s", tc.kind, ce.Kind, tc.want)
		}
	}
}

func TestMcpErrorToCLI_internalDefault(t *testing.T) {
	ce := mcpErrorToCLI(&mcpclient.Error{Kind: mcpclient.KindUnknown, Message: "weird", JSONRPCCode: 99})
	if ce.Kind != KindInternal {
		t.Fatalf("kind %s", ce.Kind)
	}
	if ce.Details == nil || ce.Details["jsonrpc_code"] != 99 {
		t.Fatalf("details: %#v", ce.Details)
	}
}

func TestMcpErrorToCLI_zeroJSONRPCCodeOmitsDetails(t *testing.T) {
	ce := mcpErrorToCLI(&mcpclient.Error{Kind: mcpclient.KindUnknown, Message: "x", JSONRPCCode: 0})
	if ce.Details != nil {
		t.Fatalf("want nil details, got %#v", ce.Details)
	}
}
