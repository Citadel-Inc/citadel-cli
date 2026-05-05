package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestLiveAuditSessions_list_optIn hits GET /audit/sessions when
// CITADEL_TEST_AUDIT_SESSIONS_LIVE=1 and CITADEL_ACCESS_TOKEN are set.
// Requires a readable namespace for --ns (pass CITADEL_TEST_AUDIT_SESSIONS_NS).
func TestLiveAuditSessions_list_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_AUDIT_SESSIONS_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_AUDIT_SESSIONS_LIVE=1 for live audit sessions integration")
	}
	ns := strings.TrimSpace(os.Getenv("CITADEL_TEST_AUDIT_SESSIONS_NS"))
	if ns == "" {
		t.Skip("CITADEL_TEST_AUDIT_SESSIONS_NS unset — provide a namespace slug you have audit:read on")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live audit sessions API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://api.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"audit", "sessions", "list", "--ns", ns, "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
