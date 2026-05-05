package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestLiveSshKeys_list_optIn performs GET /account/ssh-keys when
// CITADEL_TEST_SSH_KEYS_LIVE=1 and CITADEL_ACCESS_TOKEN are set.
func TestLiveSshKeys_list_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_SSH_KEYS_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_SSH_KEYS_LIVE=1 for live ssh-key integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live ssh-key API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://api.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"ssh-key", "list", "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
