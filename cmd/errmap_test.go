package cmd

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/citadel-cli/internal/apiclient"
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
