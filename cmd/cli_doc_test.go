package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCLIDocParity(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	docPath := filepath.Join(filepath.Dir(thisFile), "..", "docs", "cli.md")
	data, err := os.ReadFile(docPath) //nolint:gosec // path is this test file's ../docs/cli.md
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)

	var missing []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if !strings.Contains(doc, sub.Name()) {
				missing = append(missing, sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(NewRootCmd())
	if len(missing) > 0 {
		t.Fatalf("registered commands missing from docs/cli.md: %s", strings.Join(missing, ", "))
	}
}
