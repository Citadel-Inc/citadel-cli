package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestLiveAccount_passkeyList_optIn performs GET /account/passkeys when
// CITADEL_TEST_ACCOUNT_SECURITY_LIVE=1 and CITADEL_ACCESS_TOKEN are set.
func TestLiveAccount_passkeyList_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_ACCOUNT_SECURITY_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_ACCOUNT_SECURITY_LIVE=1 for live account/security integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live account API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://mcp.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"account", "passkey", "list", "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}

// TestLiveAccount_deviceList_optIn performs GET /auth/devices under the same gate.
func TestLiveAccount_deviceList_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_ACCOUNT_SECURITY_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_ACCOUNT_SECURITY_LIVE=1 for live account/security integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live account API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://mcp.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"account", "device", "list", "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
