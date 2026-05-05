package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestMan_GeneratesPages(t *testing.T) {
	dir := t.TempDir()
	root := &cobra.Command{Use: "citadel-cli"}
	root.AddCommand(cmd.AuthCmd)
	root.AddCommand(cmd.ManCmd)
	root.SetArgs([]string{"man", dir})
	root.SilenceErrors = true
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		t.Fatalf("man: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("no man pages emitted")
	}
	// Verify the root + at least one subcommand page landed.
	wantSeen := map[string]bool{
		"citadel-cli.1":            false,
		"citadel-cli-auth.1":       false,
		"citadel-cli-auth-login.1": false,
	}
	for _, e := range entries {
		if _, ok := wantSeen[e.Name()]; ok {
			wantSeen[e.Name()] = true
		}
	}
	for name, seen := range wantSeen {
		if !seen {
			t.Errorf("missing %s", name)
		}
	}

	// Spot-check one page renders nroff-shaped content.
	page, err := os.ReadFile(filepath.Join(dir, "citadel-cli-auth-login.1"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), ".SH NAME") {
		t.Errorf("auth-login man page missing .SH NAME header")
	}
}

func TestMan_RejectsMissingArg(t *testing.T) {
	root := &cobra.Command{Use: "citadel-cli"}
	root.AddCommand(cmd.ManCmd)
	root.SetArgs([]string{"man"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when out-dir is missing")
	}
}
