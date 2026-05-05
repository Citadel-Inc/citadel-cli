package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// ManCmd writes one nroff(7) man page per verb to a target directory.
// Distros + Homebrew formula consume the output via `make install`.
var ManCmd = &cobra.Command{
	Use:   "man <out-dir>",
	Short: "Generate man pages for every verb",
	Long: "Writes one nroff(7) man page per verb to the target directory.\n\n" +
		"Outputs files like:\n\n" +
		"  citadel-cli.1\n" +
		"  citadel-cli-auth.1\n" +
		"  citadel-cli-auth-login.1\n" +
		"  ...\n\n" +
		"Then 'man -M <out-dir> citadel-cli-auth-login' (or move them under\n" +
		"/usr/share/man/man1 for system-wide install).",
	Args: cobra.ExactArgs(1),
	RunE: runMan,
}

func runMan(cmd *cobra.Command, args []string) error {
	dir := args[0]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	header := &doc.GenManHeader{
		Title:   "CITADEL-CLI",
		Section: "1",
		Source:  "Citadel",
		Manual:  "Citadel CLI Manual",
	}
	if err := doc.GenManTree(cmd.Root(), header, dir); err != nil {
		return fmt.Errorf("generate man pages: %w", err)
	}
	return nil
}
