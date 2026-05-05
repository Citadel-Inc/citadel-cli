package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Rethunk-Tech/citadel-cli/cmd"
)

func TestCompletion_AllShells(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			root := &cobra.Command{Use: "citadel-cli"}
			root.AddCommand(cmd.AuthCmd)
			root.AddCommand(cmd.CompletionCmd)
			root.SetArgs([]string{"completion", shell})

			buf := &bytes.Buffer{}
			root.SetOut(buf)
			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s: %v", shell, err)
			}
			if buf.Len() == 0 {
				t.Errorf("completion %s produced empty output", shell)
			}
		})
	}
}

func TestCompletion_RejectsUnknownShell(t *testing.T) {
	root := &cobra.Command{Use: "citadel-cli"}
	root.AddCommand(cmd.CompletionCmd)
	root.SetArgs([]string{"completion", "tcsh"})
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "tcsh") {
		t.Fatalf("want unknown-shell error, got %v", err)
	}
}
