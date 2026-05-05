package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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
		401: "authentication failed",
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
	cases := []struct {
		k    CLIErrorKind
		want int
	}{
		{KindInternal, 1},
		{KindValidation, 2},
		{KindAuthRequired, 3},
		{KindNotFound, 4},
		{KindConflict, 5},
		{KindRateLimited, 6},
		{KindNetwork, 7},
	}
	for _, tc := range cases {
		if got := KindToExitCode(tc.k); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.k, got, tc.want)
		}
	}
}

func TestResolveCLIExit_WrappedExistingCLIError(t *testing.T) {
	inner := &CLIError{Kind: KindAuthRequired, Message: "authentication failed: run `citadel-cli auth login` to refresh your session, or pass --token / set CITADEL_AGENT_TOKEN"}
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
