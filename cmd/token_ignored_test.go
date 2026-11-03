package cmd

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGistListRefusesToReportEmptyUnderConfinedToken(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Citadel-Token-Ignored", "confined")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"gists":[]}`))
	})
	out, err := executeGistTestCommand(t, "gist", "list")
	if err == nil || strings.Contains(out, "No gists") {
		t.Fatalf("confined token reported as an empty list: %q %v", out, err)
	}
	cliErr := FriendlyError(err)
	var ce *CLIError
	if !errors.As(cliErr, &ce) || ce.Kind != KindForbidden || !strings.Contains(ce.Message, "confined") {
		t.Fatalf("mapped to %v", cliErr)
	}
}

func TestNotFoundNamesTheScopeTheTokenLacks(t *testing.T) {
	testServerTokenEnv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Citadel-Token-Ignored", "code:read")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not_found"}`))
	})
	_, err := executeGistTestCommand(t, "gist", "view", "00000000-0000-0000-0000-000000000001")
	var ce *CLIError
	if !errors.As(FriendlyError(err), &ce) || ce.Kind != KindForbidden || !strings.Contains(ce.Message, "code:read") {
		t.Fatalf("got %v", err)
	}
}
