package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

// TestLiveSearch_optIn performs GET /api/search when CITADEL_TEST_SEARCH_LIVE=1
// and valid credentials are present (same gate pattern as other live tests).
func TestLiveSearch_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_SEARCH_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_SEARCH_LIVE=1 for live search integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live search API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://api.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	root := NewRootCmd()
	root.SetArgs([]string{"search", "te", "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
