package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// CompletionCmd is the parent for `citadel-cli completion <shell>`. The
// subcommands emit a shell completion script to stdout suitable for
// sourcing. Cobra ships the generators; this file is a thin facade.
var CompletionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Emits a shell completion script to stdout.

Bash:
  Linux / Homebrew:
    citadel-cli completion bash > /etc/bash_completion.d/citadel-cli
  In-shell (current session):
    source <(citadel-cli completion bash)

Zsh:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  citadel-cli completion zsh > "${fpath[1]}/_citadel-cli"

Fish:
  citadel-cli completion fish > ~/.config/fish/completions/citadel-cli.fish

PowerShell:
  citadel-cli completion powershell | Out-String | Invoke-Expression`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := cmd.Root()
		out := cmd.OutOrStdout()
		switch args[0] {
		case "bash":
			return root.GenBashCompletionV2(out, true)
		case "zsh":
			return root.GenZshCompletion(out)
		case "fish":
			return root.GenFishCompletion(out, true)
		case "powershell":
			return root.GenPowerShellCompletionWithDesc(out)
		}
		return fmt.Errorf("unknown shell %q", args[0])
	},
}
