package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestLiveProjectGraph_pinChain_optIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("CITADEL_TEST_PROJECTGRAPH_LIVE")) != "1" {
		t.Skip("set CITADEL_TEST_PROJECTGRAPH_LIVE=1 for live project graph integration")
	}
	tok := strings.TrimSpace(os.Getenv("CITADEL_ACCESS_TOKEN"))
	if tok == "" {
		t.Skip("CITADEL_ACCESS_TOKEN unset — cannot exercise live project graph API")
	}
	base := strings.TrimSuffix(strings.TrimSpace(os.Getenv("CITADEL_SERVER")), "/")
	if base == "" {
		base = "https://mcp.src.land"
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CITADEL_SERVER", base)
	t.Setenv("CITADEL_ACCESS_TOKEN", tok)

	slug := strings.TrimSpace(os.Getenv("CITADEL_TEST_PROJECTGRAPH_SLUG"))
	if slug == "" {
		t.Skip("set CITADEL_TEST_PROJECTGRAPH_SLUG to a namespace path (e.g. org/repo) for live pin-chain")
	}

	root := NewRootCmd()
	root.SetArgs([]string{"project", "pin-chain", slug, "--output", "json"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
}
