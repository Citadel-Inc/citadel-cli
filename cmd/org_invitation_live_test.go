package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestLiveOrgInvitation_pending_optIn hits GET /invitations/pending on the
// configured API host when CITADEL_TEST_ORG_INVITATIONS_LIVE=1 and
// CITADEL_ACCESS_TOKEN are set. CI skips by default.
func TestLiveOrgInvitation_pending_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_ORG_INVITATIONS_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_ORG_INVITATIONS_LIVE=1 for live org invitation integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live org invitations")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://api.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"org", "invitation", "pending", "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
