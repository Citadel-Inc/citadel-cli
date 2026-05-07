package cmd

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags. Local/source builds default to
// "dev" so `citadel-cli --version` still returns a useful non-empty value.
var Version = "dev"

// NewRootCmd builds the citadel-cli cobra root with every subcommand and the
// persistent flags that handlers expect. Mirrors main wiring so integration
// tests and shell completion exercise the same tree as the binary.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "citadel-cli",
		Short: "Citadel CLI — authentication, token, and MCP agent interface",
		Long: `citadel-cli is the command-line client for managing authentication, agent tokens,
and MCP tool interactions against the Citadel server (the server binary is "citadel", cmd/citadel).

Server URL defaults to https://mcp.src.land; override with CITADEL_SERVER or --server.`,
		Version: Version,
		// Did-you-mean: cobra emits "Did you mean ...?" hints on unknown
		// subcommand. Distance 2 catches typos like `rpo` → `repo` while
		// staying tight enough to avoid noise on genuinely-new verbs.
		SuggestionsMinimumDistance: 2,
	}

	root.PersistentFlags().String("server", "", "Server URL (overrides CITADEL_SERVER env and stored config)")
	root.PersistentFlags().BoolP("verbose", "v", false, "Print one METHOD URL → STATUS line per HTTP call to stderr")
	root.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-essential stderr output (overrides --verbose)")
	root.PersistentFlags().Bool("debug-http", false, "Dump full HTTP request/response (Authorization redacted) to stderr")
	root.PersistentFlags().String("color", "auto", "Color output: auto|always|never (honors NO_COLOR)")
	root.PersistentFlags().Bool("no-pager", false, "Do not pipe stdout through $PAGER (CITADEL_PAGER > GIT_PAGER > PAGER > less -FRX)")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		maybeEagerMigrateLegacyJWT(cmd)
		return nil
	}

	root.AddCommand(AuthCmd)
	root.AddCommand(AccountCmd)
	root.AddCommand(TokenCmd)
	root.AddCommand(McpCmd)
	root.AddCommand(KgCmd)
	root.AddCommand(RepoCmd)
	root.AddCommand(IssueCmd)
	root.AddCommand(NamespaceCmd)
	root.AddCommand(OrgCmd)
	root.AddCommand(AgentCmd)
	root.AddCommand(OauthCmd)
	root.AddCommand(SSHKeyCmd)
	root.AddCommand(CompletionCmd)
	root.AddCommand(DoctorCmd)
	root.AddCommand(ManCmd)
	root.AddCommand(AuditCmd)
	root.AddCommand(SearchCmd)
	root.AddCommand(ProjectCmd)
	root.AddCommand(NotificationCmd)

	return root
}
