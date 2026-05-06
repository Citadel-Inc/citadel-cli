package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestLiveKg_extended_optIn exercises kg search when CITADEL_TEST_KG_EXTENDED_LIVE=1.
func TestLiveKg_extended_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_KG_EXTENDED_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_KG_EXTENDED_LIVE=1 for live KG HTTP integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live KG API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://mcp.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"kg", "search", "func", "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
